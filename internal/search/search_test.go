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
