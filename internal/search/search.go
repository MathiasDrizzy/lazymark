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
	maxPerFile          = 50
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
				matches, skipped := searchFile(cctx, p, base, re, maxBytes)
				mu.Lock()
				res.Files++
				if skipped {
					res.Skipped++
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
	return res, ctx.Err() // solo el contexto del llamador: parar por haber llegado al tope no es un error
}

// searchFile busca en un archivo; skipped indica que no se leyó por su tamaño.
func searchFile(ctx context.Context, path, base string, re *regexp.Regexp, maxBytes int64) (matches []Match, skipped bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return nil, false
	}
	if fi.Size() > maxBytes {
		return nil, true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	if !re.Match(data) { // sin ninguna coincidencia en el archivo: no hace falta recorrer las líneas
		return nil, false
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	title = strings.NewReplacer("-", " ", "_", " ").Replace(title)

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
			return matches, false
		}
		l = bytes.TrimSuffix(l, []byte("\r"))
		loc := re.FindIndex(l)
		if loc == nil || loc[0] == loc[1] { // sin coincidencia, o una expresión que solo coincide vacía
			continue
		}
		text, s, e := window(string(l), loc[0], loc[1], ContextWidth)
		matches = append(matches, Match{Path: path, Rel: rel, Title: title, Line: line, Text: text, Start: s, End: e})
		if len(matches) >= maxPerFile {
			break
		}
	}
	return matches, false
}

// window recorta la línea a como mucho width caracteres alrededor de la coincidencia [s,e): le quita la sangría, y si sobra por
// los lados la corta con …. Devuelve el texto y dónde queda la coincidencia dentro de él (en bytes).
func window(line string, s, e, width int) (string, int, int) {
	trim := len(line) - len(strings.TrimLeft(line, " \t"))
	line, s, e = line[trim:], max(0, s-trim), max(0, e-trim)
	line = strings.TrimRight(line, " \t")
	if e > len(line) {
		e = len(line)
	}
	if utf8.RuneCountInString(line) <= width {
		return sanitize(line, &s, &e)
	}
	// pasa de width: se deja la coincidencia con algo de contexto a cada lado, alineado a runas
	runes := []rune(line)
	rs := utf8.RuneCountInString(line[:s])
	re := utf8.RuneCountInString(line[:e])
	matchLen := re - rs
	room := max(0, width-matchLen)
	from := max(0, rs-room/3)
	to := min(len(runes), from+width)
	if to-from < width {
		from = max(0, to-width)
	}
	if re > to { // una coincidencia más larga que el ancho: se muestra su comienzo
		re = to
	}
	prefix, suffix := "", ""
	if from > 0 {
		prefix = "…"
	}
	if to < len(runes) {
		suffix = "…"
	}
	out := prefix + string(runes[from:to]) + suffix
	ns := len(prefix) + len(string(runes[from:rs]))
	ne := len(prefix) + len(string(runes[from:re]))
	return sanitize(out, &ns, &ne)
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
