package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// corpus arma una carpeta de notas pequeña con acentos, japonés, CRLF, una línea larga, carpetas que se ignoran, un enlace que sale
// de la carpeta y una nota enorme.
func corpus(t *testing.T) (*storage.Storage, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "notas")
	write(t, root, "a.md", "# A\nprimera línea\nLa canción de cuna es bonita\n- [ ] comprar CAFÉ y leche\n")
	write(t, root, "sub/b.md", "# B\r\nTexto en windows con zorzalino\r\notra línea\r\n")
	write(t, root, "sub/deep/c.md", "# C\n検索テスト en japonés\nzorzalino otra vez\n")
	write(t, root, "largo.md", "inicio "+strings.Repeat("palabra ", 60)+"AGUJA "+strings.Repeat("relleno ", 60)+"fin\n")
	write(t, root, "ctrl.md", "antes \x1b]52;c;cHduZWQ=\x07 zorzalino después\n")
	write(t, root, ".trash/x.md", "zorzalino en la papelera\n")
	write(t, root, "assets/y.md", "zorzalino en assets\n")
	write(t, root, "enorme.md", "zorzalino\n"+strings.Repeat("x", 3<<20))
	outside := filepath.Join(filepath.Dir(root), "fuera.md")
	os.WriteFile(outside, []byte("zorzalino fuera de la carpeta\n"), 0o644)
	if runtime.GOOS != "windows" {
		os.Symlink(outside, filepath.Join(root, "enlace-fuera.md"))
		os.Symlink(filepath.Join(root, "a.md"), filepath.Join(root, "enlace-dentro.md"))
	}
	return storage.New(root), root
}

func rels(r Result) []string {
	var out []string
	for _, m := range r.Matches {
		out = append(out, m.Rel+":"+itoa(m.Line))
	}
	return out
}

func itoa(n int) string { return fmt.Sprintf("%02d", n) }

func TestPlainSearch(t *testing.T) {
	s, _ := corpus(t)
	res, err := Run(context.Background(), s, "zorzalino", Options{})
	if err != nil {
		t.Fatal(err)
	}
	// no ve .trash, assets, la nota enorme (se salta) ni el enlace que sale de la carpeta; sí CRLF, subcarpetas y el control
	want := []string{"ctrl.md:01", "sub/b.md:02", "sub/deep/c.md:03"}
	if got := rels(res); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("coincidencias %v, se esperaban %v", got, want)
	}
	if res.Skipped != 1 {
		t.Errorf("la nota enorme se salta: Skipped=%d", res.Skipped)
	}
	for _, m := range res.Matches {
		if m.Text[m.Start:m.End] != "zorzalino" {
			t.Errorf("%s: la posición %d:%d marca %q", m.Rel, m.Start, m.End, m.Text[m.Start:m.End])
		}
		if strings.ContainsAny(m.Text, "\x1b\x07\r") {
			t.Errorf("%s: la línea lleva caracteres de control: %q", m.Rel, m.Text)
		}
	}
}

func TestCaseAccentsAndCJK(t *testing.T) {
	s, _ := corpus(t)
	for _, c := range []struct {
		q    string
		opts Options
		want int
	}{
		{"café", Options{}, 1},                    // sin distinguir mayúsculas: CAFÉ
		{"café", Options{CaseSensitive: true}, 0}, // distinguiendo, CAFÉ no es café
		{"CAFÉ", Options{CaseSensitive: true}, 1},
		{"cafe", Options{}, 0}, // los acentos cuentan
		{"canción de cuna", Options{}, 1},
		{"検索テスト", Options{}, 1},
		{"CANCIÓN", Options{}, 1},
		{"no existe en ningún lado", Options{}, 0},
	} {
		res, err := Run(context.Background(), s, c.q, c.opts)
		if err != nil || len(res.Matches) != c.want {
			t.Errorf("%q %+v: %d coincidencias (%v), se esperaban %d", c.q, c.opts, len(res.Matches), err, c.want)
		}
	}
}

func TestRegexSearch(t *testing.T) {
	s, _ := corpus(t)
	res, err := Run(context.Background(), s, `zorz\w+ (otra|después)`, Options{Regex: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(res); strings.Join(got, ",") != "ctrl.md:01,sub/deep/c.md:03" {
		t.Errorf("regex: %v", got)
	}
	// sin Regex, los metacaracteres son texto
	if res, _ := Run(context.Background(), s, `zorz\w+`, Options{}); len(res.Matches) != 0 {
		t.Errorf("sin --regex la barra y la w son texto: %v", rels(res))
	}
	for _, bad := range []string{`(`, `[a-`, `*x`} {
		if _, err := Run(context.Background(), s, bad, Options{Regex: true}); err == nil {
			t.Errorf("%q debía ser un error de sintaxis", bad)
		}
	}
	if _, err := Run(context.Background(), s, "", Options{}); err == nil {
		t.Error("una búsqueda vacía es un error")
	}
	// una expresión que solo coincide vacía no devuelve todas las líneas
	if res, _ := Run(context.Background(), s, `x*`, Options{Regex: true}); len(res.Matches) > 5 {
		for _, m := range res.Matches {
			if m.Start == m.End {
				t.Errorf("coincidencia vacía: %+v", m)
			}
		}
	}
}

func TestLongLineWindow(t *testing.T) {
	s, _ := corpus(t)
	res, _ := Run(context.Background(), s, "AGUJA", Options{})
	if len(res.Matches) != 1 {
		t.Fatalf("%v", rels(res))
	}
	m := res.Matches[0]
	if r := []rune(m.Text); len(r) > ContextWidth+2 || !strings.HasPrefix(m.Text, "…") || !strings.HasSuffix(m.Text, "…") || m.Text[m.Start:m.End] != "AGUJA" {
		t.Errorf("la línea larga se recorta alrededor de lo hallado: %q (%d:%d)", m.Text, m.Start, m.End)
	}
}

func TestTruncatesAndOrders(t *testing.T) {
	root := filepath.Join(t.TempDir(), "n")
	for i := 0; i < 30; i++ {
		write(t, root, "n"+string(rune('a'+i%26))+string(rune('a'+i/26))+".md", "x\nhallado\nhallado\n")
	}
	s := storage.New(root)
	res, err := Run(context.Background(), s, "hallado", Options{MaxResults: 10})
	if err != nil || len(res.Matches) != 10 || !res.Truncated {
		t.Fatalf("tope: %d truncated=%v err=%v", len(res.Matches), res.Truncated, err)
	}
	for i := 1; i < len(res.Matches); i++ {
		a, b := res.Matches[i-1], res.Matches[i]
		if a.Rel > b.Rel || (a.Rel == b.Rel && a.Line >= b.Line) {
			t.Fatalf("no está ordenado por nota y línea: %s:%d antes de %s:%d", a.Rel, a.Line, b.Rel, b.Line)
		}
	}
	all, _ := Run(context.Background(), s, "hallado", Options{})
	if len(all.Matches) != 60 || all.Truncated || all.Files != 30 {
		t.Errorf("sin tope: %d coincidencias, %d notas", len(all.Matches), all.Files)
	}
}

func TestCancel(t *testing.T) {
	s, _ := corpus(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	res, err := Run(ctx, s, "zorzalino", Options{})
	if err != context.Canceled {
		t.Errorf("se esperaba context.Canceled: %v", err)
	}
	if time.Since(start) > time.Second || len(res.Matches) > 3 {
		t.Errorf("una búsqueda cancelada no trabaja de más: %v, %d", time.Since(start), len(res.Matches))
	}
}

func TestWindowIsRuneSafe(t *testing.T) {
	line := strings.Repeat("日本語のテキスト", 40) + "検索" + strings.Repeat("日本語のテキスト", 40)
	s := strings.Index(line, "検索")
	text, a, b := window(line, s, s+len("検索"), 40)
	if text[a:b] != "検索" {
		t.Errorf("la coincidencia debe quedar marcada: %q %d:%d", text, a, b)
	}
	if !strings.HasPrefix(text, "…") || !strings.HasSuffix(text, "…") {
		t.Errorf("recortada por los dos lados: %q", text)
	}
}

// BenchmarkSearchCorpus mide la latencia sobre un corpus grande: LAZYMARK_CORPUS=<carpeta> go test ./internal/search -bench Corpus.
func BenchmarkSearchCorpus(b *testing.B) {
	dir := os.Getenv("LAZYMARK_CORPUS")
	if dir == "" {
		b.Skip("sin LAZYMARK_CORPUS")
	}
	s := storage.New(dir)
	for i := 0; i < b.N; i++ {
		if _, err := Run(context.Background(), s, "zorzalino", Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

// TestRegexAnchorsAndTrailingSpaces (hallazgos de la segunda opinión): `^` y `$` valen por línea (el prefiltro de archivo solo se usa con
// texto literal), y una coincidencia en los espacios finales de una línea no hace entrar en pánico a window().
func TestRegexAnchorsAndTrailingSpaces(t *testing.T) {
	root := filepath.Join(t.TempDir(), "n")
	write(t, root, "a.md", "# Título\nObjetivo del día\nfin\n")
	write(t, root, "b.md", strings.Repeat("a", 150)+"   \ncorta   \n")
	s := storage.New(root)
	res, err := Run(context.Background(), s, `^Objetivo`, Options{Regex: true})
	if err != nil || len(res.Matches) != 1 || res.Matches[0].Line != 2 {
		t.Errorf("`^Objetivo` debía hallar la línea 2: %v %v", rels(res), err)
	}
	res, _ = Run(context.Background(), s, `día$`, Options{Regex: true})
	if len(res.Matches) != 1 {
		t.Errorf("`día$`: %v", rels(res))
	}
	// `\s+$` coincide en los espacios del final, que window() recorta: no debe haber pánico ni rangos al revés
	res, err = Run(context.Background(), s, `\s+$`, Options{Regex: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range res.Matches {
		if m.Start > m.End || m.End > len(m.Text) {
			t.Errorf("%s:%d: rango inválido %d:%d en %q", m.Rel, m.Line, m.Start, m.End, m.Text)
		}
	}
	for _, l := range []struct {
		line string
		s, e int
	}{{strings.Repeat("a", 150) + "   ", 150, 153}, {"corta   ", 5, 8}, {"", 0, 0}, {"   ", 0, 3}} {
		text, a, b := window(l.line, l.s, l.e, 140) // no debe entrar en pánico
		if a > b || b > len(text) {
			t.Errorf("window(%q): %q %d:%d", l.line, text, a, b)
		}
	}
}

// TestPerFileCapIsReported: si una nota tiene más coincidencias que el tope por archivo, Truncated lo dice.
func TestPerFileCapIsReported(t *testing.T) {
	root := filepath.Join(t.TempDir(), "n")
	write(t, root, "mucho.md", strings.Repeat("alerta aquí\n", 70))
	write(t, root, "justo.md", strings.Repeat("alerta aquí\n", maxPerFile))
	s := storage.New(root)
	res, _ := Run(context.Background(), s, "alerta", Options{MaxResults: 1000})
	if len(res.Matches) != 2*maxPerFile || !res.Truncated {
		t.Errorf("%d coincidencias, truncado=%v: una nota con 70 debía marcar el recorte", len(res.Matches), res.Truncated)
	}
	only := filepath.Join(t.TempDir(), "m")
	write(t, only, "justo.md", strings.Repeat("alerta aquí\n", maxPerFile))
	if res, _ := Run(context.Background(), storage.New(only), "alerta", Options{}); len(res.Matches) != maxPerFile || res.Truncated {
		t.Errorf("una nota con exactamente %d coincidencias no es un recorte: %d %v", maxPerFile, len(res.Matches), res.Truncated)
	}
}

// TestNamesAreSanitized: el nombre de la nota (contenido no confiable) no lleva caracteres de control a la terminal.
func TestNamesAreSanitized(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows no admite esos nombres de archivo")
	}
	root := filepath.Join(t.TempDir(), "n")
	write(t, root, "mala\x1b[31mnota.md", "hallado aquí\n")
	res, _ := Run(context.Background(), storage.New(root), "hallado", Options{})
	if len(res.Matches) != 1 || strings.ContainsAny(res.Matches[0].Rel+res.Matches[0].Title, "\x1b") {
		t.Errorf("%+v", res.Matches)
	}
}

// TestWindowOnHugeLine: una línea de megabytes se recorta sin convertirla entera a runas (poca memoria) y sin perder la coincidencia.
func TestWindowOnHugeLine(t *testing.T) {
	line := strings.Repeat("日本語 ", 400_000) + "AGUJA" + strings.Repeat("語 ", 400_000)
	s := strings.Index(line, "AGUJA")
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	text, a, b := window(line, s, s+5, 140)
	runtime.ReadMemStats(&after)
	if text[a:b] != "AGUJA" {
		t.Errorf("la coincidencia debe quedar marcada: %q", text[a:b])
	}
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 1<<20 {
		t.Errorf("recortar una línea enorme reservó %d KB", grown>>10)
	}
}

// TestSearchTimeout (ORD-015 C.5 S4): una regex lenta sobre una nota grande no deja la búsqueda colgada: pasado el tiempo máximo devuelve lo que
// tenga con TimedOut, sin error; y una repetición enorme ({1000}) se rechaza al compilar.
func TestSearchTimeout(t *testing.T) {
	store, dir := corpus(t)
	line := strings.Repeat("abcdefghij", 200000) // 2 MB en una sola línea
	os.WriteFile(filepath.Join(dir, "larga.md"), []byte(line+"\n"), 0o644)
	start := time.Now()
	res, err := Run(context.Background(), store, `(\p{L}|\p{N}){60}x`, Options{Regex: true, Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("la búsqueda tardó %v con un máximo de 100 ms", d)
	}
	if !res.TimedOut {
		t.Errorf("debe avisar que se cortó por tiempo: %+v", res)
	}
	if _, err := Run(context.Background(), store, `[\p{L}\p{N}]{1000}x`, Options{Regex: true}); err == nil {
		t.Error("una repetición de 1000 se rechaza")
	}
	// una búsqueda normal no se corta
	if res, err := Run(context.Background(), store, "abcdef", Options{Timeout: 5 * time.Second}); err != nil || res.TimedOut || len(res.Matches) == 0 {
		t.Errorf("búsqueda normal: %v %+v", err, res.TimedOut)
	}
}

// TestTooManyAbandonedSearches (ORD-015 segunda opinión): si ya hay muchas coincidencias lentas abandonadas corriendo, una búsqueda nueva con regex
// no lanza más: avisa que se cortó en vez de acumular goroutines.
func TestTooManyAbandonedSearches(t *testing.T) {
	store, _ := corpus(t)
	abandoned.Store(10000)
	defer abandoned.Store(0)
	res, err := Run(context.Background(), store, `a.*b`, Options{Regex: true})
	if err != nil || !res.TimedOut || len(res.Matches) != 0 {
		t.Errorf("con búsquedas lentas pendientes debe cortarse: %v %+v", err, res)
	}
	if res, err := Run(context.Background(), store, "a", Options{}); err != nil || res.TimedOut {
		t.Errorf("el texto literal no se ve afectado: %v %v", err, res.TimedOut)
	}
}
