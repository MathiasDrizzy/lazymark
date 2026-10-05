// Package search es la búsqueda de texto completo de lazymark: recorre las notas de la carpeta de notas en varias goroutines, sin
// índice persistente, con un tope de tamaño por archivo y cancelable (en la TUI se cancela al seguir escribiendo). Solo ve lo que
// ve el resto de lazymark: notas .md dentro de la carpeta, con la regla de enlaces simbólicos de storage.NotePaths.
package search

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// Límites por defecto.
const (
	DefaultMaxFileBytes = 2 << 20 // las notas más grandes no se leen (se cuentan en Result.Skipped)
	DefaultMaxResults   = 500
	DefaultTimeout      = 10 * time.Second
	// maxRepeat es la mayor repetición ({n}) que acepta una expresión regular: una como [\p{L}\p{N}]{1000} tarda decenas de segundos en 2 MB.
	maxRepeat  = 100
	maxPerFile = 50
	// ContextWidth es el ancho (en caracteres) de la línea que se devuelve alrededor de la coincidencia.
	ContextWidth = 140
)

// Options ajusta una búsqueda. El texto se busca sin distinguir mayúsculas (con los acentos significativos: "cafe" no es "café")
// salvo CaseSensitive; con Regex, query es una expresión regular (RE2).
type Options struct {
	Regex         bool
	CaseSensitive bool
	MaxFileBytes  int64 // 0: DefaultMaxFileBytes
	MaxResults    int   // 0: DefaultMaxResults
	// Timeout es el tiempo máximo de la búsqueda (0: DefaultTimeout). Pasado ese tiempo devuelve lo hallado con Result.TimedOut.
	Timeout time.Duration
}

// Match es una coincidencia: la línea (recortada alrededor de lo hallado) y dónde está lo hallado dentro de ella.
type Match struct {
	Path  string // ruta absoluta de la nota
	Rel   string // ruta relativa a la carpeta de notas, con "/"
	Title string // título de la nota (el nombre del archivo, como en el resto de la app)
	Line  int    // desde 1
	Text  string // la línea, sin espacios al principio y recortada con … si es larga
	Start int    // byte donde empieza lo hallado dentro de Text
	End   int    // byte donde termina
}

// Result es lo que devuelve Run.
type Result struct {
	Matches   []Match
	Files     int  // notas revisadas
	Skipped   int  // notas que no se leyeron por pasar el tope de tamaño
	Truncated bool // hubo más coincidencias que MaxResults
	TimedOut  bool // se cortó por pasar Options.Timeout: puede haber más coincidencias
	Elapsed   time.Duration
}

// Compile arma la expresión de la búsqueda; con Regex devuelve el error de sintaxis tal cual.
func Compile(query string, opts Options) (*regexp.Regexp, error) {
	if query == "" {
		return nil, fmt.Errorf("la búsqueda está vacía")
	}
	expr := query
	if !opts.Regex {
		expr = regexp.QuoteMeta(query)
	}
	if !opts.CaseSensitive {
		expr = "(?i)" + expr
	}
	if opts.Regex {
		if tree, err := syntax.Parse(expr, syntax.Perl); err == nil && repeatTooBig(tree) {
			return nil, fmt.Errorf("expresión regular demasiado costosa: una repetición pasa de %d", maxRepeat)
		}
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("expresión regular inválida: %w", err)
	}
	return re, nil
}

// Run busca query en las notas de store. Si ctx se cancela devuelve lo hallado hasta ese momento y ctx.Err().
func Run(ctx context.Context, store *storage.Storage, query string, opts Options) (Result, error) {
	start := time.Now()
	re, err := Compile(query, opts)
	if err != nil {
		return Result{}, err
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	parent := ctx
	var cancelTimeout context.CancelFunc
	ctx, cancelTimeout = context.WithTimeout(ctx, timeout)
	defer cancelTimeout()
	maxBytes, maxResults := opts.MaxFileBytes, opts.MaxResults
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFileBytes
	}
	if maxResults <= 0 {
		maxResults = DefaultMaxResults
	}
	paths := dedupe(store.NotePaths())
	base := store.BaseDir

	var (
		mu         sync.Mutex
		res        Result
		wg         sync.WaitGroup
		jobs       = make(chan string)
		cctx, stop = context.WithCancel(ctx)
	)
	defer stop()
	workers := min(runtime.NumCPU(), max(1, len(paths)))
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				matches, skipped, capped := searchFileCtx(cctx, p, base, re, !opts.Regex, maxBytes)
				mu.Lock()
				res.Files++
				if skipped {
					res.Skipped++
				}
				if capped {
					res.Truncated = true
				}
				res.Matches = append(res.Matches, matches...)
				if len(res.Matches) >= maxResults+1 {
					stop() // ya hay de sobra: se corta el resto
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for _, p := range paths {
		select {
		case jobs <- p:
		case <-cctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	sort.Slice(res.Matches, func(i, j int) bool {
		if res.Matches[i].Rel != res.Matches[j].Rel {
			return res.Matches[i].Rel < res.Matches[j].Rel
		}
		return res.Matches[i].Line < res.Matches[j].Line
	})
	if len(res.Matches) > maxResults {
		res.Matches, res.Truncated = res.Matches[:maxResults], true
	}
	res.Elapsed = time.Since(start)
	res.TimedOut = ctx.Err() == context.DeadlineExceeded && parent.Err() == nil // se cortó por tiempo, no por el llamador
	return res, parent.Err()                                                    // solo el contexto del llamador: parar por haber llegado al tope o por tiempo no es un error
}

// searchFile busca en un archivo; skipped indica que no se leyó por su tamaño.
func searchFile(ctx context.Context, path, base string, re *regexp.Regexp, prefilter bool, maxBytes int64) (matches []Match, skipped, capped bool) {
	if ctx.Err() != nil {
		return nil, false, false
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return nil, false, false
	}
	if fi.Size() > maxBytes {
		return nil, true, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, false
	}
	// sin ninguna coincidencia en el archivo no hace falta recorrer las líneas; solo con texto literal: una expresión con ^, $ o \A
	// se evalúa línea por línea y mirar el archivo entero la descartaría mal
	if prefilter && !re.Match(data) {
		return nil, false, false
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		rel = path
	}
	rel = noControl(filepath.ToSlash(rel)) // el nombre del archivo también es contenido no confiable
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	title = noControl(strings.NewReplacer("-", " ", "_", " ").Replace(title))

	line := 0
	for len(data) > 0 {
		line++
		var l []byte
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			l, data = data[:i], data[i+1:]
		} else {
			l, data = data, nil
		}
		if line%2000 == 0 && ctx.Err() != nil {
			return matches, false, false
		}
		l = bytes.TrimSuffix(l, []byte("\r"))
		loc := re.FindIndex(l)
		if loc == nil || loc[0] == loc[1] { // sin coincidencia, o una expresión que solo coincide vacía
			continue
		}
		text, s, e := window(string(l), loc[0], loc[1], ContextWidth)
		matches = append(matches, Match{Path: path, Rel: rel, Title: title, Line: line, Text: text, Start: s, End: e})
		if len(matches) >= maxPerFile {
			capped = len(data) > 0 // quedaban líneas por mirar: hay coincidencias de más que no se devuelven
			break
		}
	}
	return matches, false, capped
}

// window recorta la línea a como mucho width caracteres alrededor de la coincidencia [s,e): le quita la sangría, y si sobra por
// los lados la corta con …. Devuelve el texto y dónde queda la coincidencia dentro de él (en bytes). No convierte la línea entera a
// runas: una línea de megabytes cuesta lo mismo que una corta.
func window(line string, s, e, width int) (string, int, int) {
	trim := len(line) - len(strings.TrimLeft(line, " \t"))
	line, s, e = line[trim:], max(0, s-trim), max(0, e-trim)
	line = strings.TrimRight(line, " \t")
	e = min(e, len(line))
	s = min(s, e) // una coincidencia que cayó en los espacios finales recortados queda vacía al final de la línea
	if runesUpTo(line, width+1) <= width {
		return sanitize(line, &s, &e)
	}
	// pasa de width: se deja la coincidencia con algo de contexto a cada lado, alineado a runas
	matchLen := min(runesUpTo(line[s:e], width), width)
	room := max(0, width-matchLen)
	from := s
	for back := room / 3; back > 0 && from > 0; back-- {
		_, size := utf8.DecodeLastRuneInString(line[:from])
		from -= size
	}
	to := from
	for n := 0; n < width && to < len(line); n++ {
		_, size := utf8.DecodeRuneInString(line[to:])
		to += size
	}
	if to-from < len(line)-from && runesUpTo(line[from:], width+1) <= width { // sobró sitio por la derecha: se retrocede para llenar el ancho
		to = len(line)
	}
	if to < e { // una coincidencia más larga que el ancho: se muestra su comienzo
		e = to
	}
	prefix, suffix := "", ""
	if from > 0 {
		prefix = "…"
	}
	if to < len(line) {
		suffix = "…"
	}
	out := prefix + line[from:to] + suffix
	return sanitize(out, ptr(len(prefix)+s-from), ptr(len(prefix)+e-from))
}

func ptr(n int) *int { return &n }

// runesUpTo cuenta los caracteres de s, sin pasar de limit (para no recorrer una línea enorme entera).
func runesUpTo(s string, limit int) int {
	n := 0
	for range s {
		n++
		if n >= limit {
			break
		}
	}
	return n
}

// noControl cambia los caracteres de control de un nombre por "?".
func noControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return '?'
		}
		return r
	}, s)
}

// sanitize quita de la línea los caracteres de control (una nota no es confiable y esto va a la terminal) ajustando las posiciones.
func sanitize(s string, start, end *int) (string, int, int) {
	var b strings.Builder
	ns, ne := *start, *end
	for i, r := range s {
		if r < 0x20 && r != '\t' || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			if i < *start {
				ns -= utf8.RuneLen(r)
			}
			if i < *end {
				ne -= utf8.RuneLen(r)
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String(), ns, ne
}

// dedupe quita las rutas que son enlaces simbólicos a otra nota de la lista: la misma nota no debe salir dos veces.
func dedupe(paths []string) []string {
	real := make(map[string]bool, len(paths))
	for _, p := range paths {
		if fi, err := os.Lstat(p); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if r, err := filepath.EvalSymlinks(p); err == nil {
			real[r] = true
		}
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if fi, err := os.Lstat(p); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			if r, err := filepath.EvalSymlinks(p); err == nil && real[r] {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// repeatTooBig dice si la expresión tiene una repetición {n,m} con n o m mayores que maxRepeat.
func repeatTooBig(re *syntax.Regexp) bool {
	if re.Op == syntax.OpRepeat && (re.Min > maxRepeat || re.Max > maxRepeat) {
		return true
	}
	for _, sub := range re.Sub {
		if repeatTooBig(sub) {
			return true
		}
	}
	return false
}

// searchFileCtx es searchFile que se abandona al cancelarse ctx: una coincidencia de regex sobre una línea enorme no se puede interrumpir por dentro,
// así que se corre aparte y, si ctx termina antes, se deja de esperarla (acaba sola y su resultado se descarta).
func searchFileCtx(ctx context.Context, path, base string, re *regexp.Regexp, prefilter bool, maxBytes int64) (matches []Match, skipped, capped bool) {
	type out struct {
		m       []Match
		sk, cap bool
	}
	ch := make(chan out, 1)
	go func() {
		m, sk, cp := searchFile(ctx, path, base, re, prefilter, maxBytes)
		ch <- out{m, sk, cp}
	}()
	select {
	case o := <-ch:
		return o.m, o.sk, o.cap
	case <-ctx.Done():
		return nil, false, false
	}
}
