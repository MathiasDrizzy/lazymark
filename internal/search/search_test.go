package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// TestRegexCostCap (ORD-016 L1): el costo de una expresión (su programa compilado, con las clases Unicode pesadas) se calcula antes de ejecutarla: una
// repetición anidada que pasaba el tope de {100} se rechaza al instante con el mismo mensaje; las expresiones de uso corriente se aceptan.
func TestRegexCostCap(t *testing.T) {
	for _, bad := range []string{`([\p{L}\p{N}]{50}){20}x`, `(\p{L}{10}){10}[\p{L}\p{N}]{100}`, `(\w{50}){20}(\p{L}{100}){2}`, `[\p{L}\p{N}]{1000}x`, `(\p{L}{100}){3}`, `(a?){1000}b`, `(a?){110}b`, `.{300}x`, `\w{330}x`, `(?i)\w{330}x`, `\w{1000}x`, `(\w{50}){20}x`, `[a-z]{1000}x`, `.{1000}x`, `((a|b)?){500}c`} {
		start := time.Now()
		_, err := Compile(bad, Options{Regex: true})
		if err == nil || !strings.Contains(err.Error(), "demasiado costosa") {
			t.Errorf("%s debe rechazarse como demasiado costosa: %v", bad, err)
		}
		if d := time.Since(start); d > 200*time.Millisecond {
			t.Errorf("%s: el rechazo tardó %v", bad, d)
		}
	}
	for _, good := range []string{`[\p{L}\p{N}]{20}x`, `[\p{L}\p{N}]{100}x`, `(a|b|c){100}x`, `\w{160}x`, `.{200}x`, `[^\n]{200}x`, `(a|b|c){40}x`, `\w+@\w+\.\w{2,6}`, `(\p{L}+\s*){5}x`, `\bTODO\b`, `^#+ `, `\d{4}-\d{2}-\d{2}`, `(?i)error|warning|fatal`, `https?://[^\s)]+`} {
		if _, err := Compile(good, Options{Regex: true}); err != nil {
			t.Errorf("%s debe aceptarse: %v", good, err)
		}
	}
}

// TestCutoffFreesTheCPU (ORD-016 L1): después del corte por tiempo no queda ninguna goroutine de la búsqueda ni CPU gastándose: la coincidencia sobre
// una línea enorme se detiene sola al cancelarse (antes seguía hasta 3 s después del corte).
func TestCutoffFreesTheCPU(t *testing.T) {
	store, dir := corpus(t)
	os.WriteFile(filepath.Join(dir, "larga.md"), []byte(strings.Repeat("abcdefghij", 200000)+"\n"), 0o644)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	goBefore := runtime.NumGoroutine()
	res, err := Run(context.Background(), store, `[\p{L}\p{N}]{60}x`, Options{Regex: true, Timeout: 150 * time.Millisecond})
	if err != nil || !res.TimedOut {
		t.Fatalf("debía cortarse por tiempo: %v %+v", err, res.TimedOut)
	}
	time.Sleep(300 * time.Millisecond) // lo que tarde en notar la cancelación
	if after := runtime.NumGoroutine(); after > goBefore {
		t.Errorf("quedan goroutines tras el corte: %d antes, %d después", goBefore, after)
	}
	if c0, ok := cpuTime(); ok {
		time.Sleep(600 * time.Millisecond)
		c1, _ := cpuTime()
		if used := c1 - c0; used > 150*time.Millisecond {
			t.Errorf("la CPU sigue gastándose tras el corte: %v en 600 ms de espera", used)
		}
	}
}

// TestRegexCostAgainstOracle (ORD-016 L1): 30 patrones (útiles, con clases Unicode grandes, anidados y cerca del tope) con su veredicto, calculados por el
// agente de apoyo con una métrica de solo rangos de runas. Se exigen los claros (≤ 300 o ≥ 200 000 en esa métrica); los intermedios dependen de la métrica.
func TestRegexCostAgainstOracle(t *testing.T) {
	data, err := os.ReadFile("testdata/regex-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Pattern  string `json:"pattern"`
		Peso     int    `json:"peso_aprox"`
		Veredict string `json:"veredicto"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 25 {
		t.Fatalf("solo %d casos", len(cases))
	}
	// desacuerdos revisados a mano: el oráculo suma las dos ramas, pero Go factoriza las alternativas idénticas (peso real 54 720 = una sola rama)
	reviewed := map[string]bool{`\p{L}{80}|\p{L}{80}`: true}
	for _, c := range cases {
		if reviewed[c.Pattern] {
			continue
		}
		_, err := Compile(c.Pattern, Options{Regex: true})
		rejected := err != nil
		// el oráculo pesaba solo los rangos de runas; el costo real también cuenta los estados vivos (\w{1000} tarda 4,5 s por 500 KB con "peso" 4001):
		// solo se exigen los veredictos que no dependen de la métrica (muy baratos y muy caros)
		clear := c.Peso <= 300 || c.Peso >= 200000
		if clear && rejected != (c.Veredict == "rechaza") {
			t.Errorf("%s: el oráculo dice %s (peso ≈ %d) y Compile dio error=%v", c.Pattern, c.Veredict, c.Peso, err)
		}
	}
}

// TestSlowLinesDoNotDelayTheCut (ORD-016, segunda opinión): con muchas líneas de tamaño medio (por debajo de los 64 KB del lector con cancelación) cada
// una cuesta, pero el corte por tiempo se nota entre líneas, no cada 2000: la búsqueda vuelve cerca del tiempo máximo.
func TestSlowLinesDoNotDelayTheCut(t *testing.T) {
	store, dir := corpus(t)
	var b strings.Builder
	for i := 0; i < 60; i++ { // 1,8 MB: bajo el tope de 2 MiB por nota
		b.WriteString(strings.Repeat("abcdefghij", 3000) + "\n") // 30 KB
	}
	os.WriteFile(filepath.Join(dir, "muchas.md"), []byte(b.String()), 0o644)
	// lo que cuesta una línea en esta máquina (con -race o sin él): el corte puede tardar a lo sumo en terminar las que están en curso
	oneLine := []byte(strings.Repeat("abcdefghij", 3000))
	t0 := time.Now()
	// \w{150}x y no \w{300}x (ORD-017): el tope de costo ahora rechaza \w{300}x, y la prueba necesita un patrón aceptado que igual sea lento por línea
	regexp.MustCompile(`\w{150}x`).Find(oneLine)
	lineCost := time.Since(t0)
	start := time.Now()
	res, err := Run(context.Background(), store, `\w{150}x`, Options{Regex: true, Timeout: 300 * time.Millisecond})
	if err != nil || !res.TimedOut {
		t.Fatalf("debía cortarse por tiempo: %v %+v", err, res.TimedOut)
	}
	if d, limit := time.Since(start), 300*time.Millisecond+8*lineCost+500*time.Millisecond; d > limit {
		t.Errorf("la búsqueda tardó %v con un máximo de 300 ms (una línea cuesta %v; límite %v): el corte no se nota entre líneas", d, lineCost, limit)
	}
}

// TestAcceptedRegexIsFast (ORD-017 F8 / L11): lo que el tope acepta tarda como mucho unos 3 s sobre una línea de 2 MB (letras o "aaaa…"), y \w{330}x, que
// tardaba 5,9 s, ya no se acepta. Mide tiempo: se salta con -race y con -short.
func TestAcceptedRegexIsFast(t *testing.T) {
	if raceEnabled || testing.Short() {
		t.Skip("mide tiempo: no corre con -race ni con -short")
	}
	if _, err := Compile(`\w{330}x`, Options{Regex: true, CaseSensitive: true}); err == nil {
		t.Error(`\w{330}x --case tardaba 5,8 s y debe rechazarse`)
	}
	texts := map[string]string{"aaaa": strings.Repeat("a", 2000000)} // el peor texto para los patrones de arriba (con letras variadas tardan igual o menos)
	for _, expr := range []string{`\w{160}x`, `[\p{L}\p{N}]{100}x`, `(a?){90}b`, `(a|b|c){100}x`, `(a|b|c){40}x`} {
		re, err := Compile(expr, Options{Regex: true})
		if err != nil {
			t.Errorf("%s debe aceptarse: %v", expr, err)
			continue
		}
		for name, text := range texts {
			start := time.Now()
			re.MatchString(text)
			if d := time.Since(start); d > 4*time.Second { // 3 s medidos + margen de la máquina
				t.Errorf("%s sobre %s tardó %v: lo aceptado debe tardar ≈3 s o menos", expr, name, d)
			}
		}
	}
}
