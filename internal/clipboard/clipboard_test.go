package clipboard

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeReader simula el portapapeles: nunca se toca el del sistema.
type fakeReader struct {
	data []byte
	err  error
}

func (f fakeReader) ReadImage(dest string) error {
	if f.err != nil {
		_ = os.WriteFile(dest, []byte("a medias"), 0o644) // un fallo real puede dejar un archivo parcial
		return f.err
	}
	return os.WriteFile(dest, f.data, 0o644)
}

var fixedNow = func() time.Time { return time.Date(2026, 10, 1, 15, 30, 45, 0, time.UTC) }

func newSaver(r Reader) *Saver { return &Saver{Reader: r, now: fixedNow} }

func TestSaveFromClipboard(t *testing.T) {
	dir := t.TempDir()
	noteDir := filepath.Join(dir, "proyectos")
	s := newSaver(fakeReader{data: []byte("PNGDATA")})

	got, err := s.SaveFromClipboard(noteDir, "Mi Nota: plan?.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := "assets/mi-nota-plan-20261001-153045.png"; got != want {
		t.Fatalf("referencia = %q, se esperaba %q", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(noteDir, filepath.FromSlash(got))); string(b) != "PNGDATA" {
		t.Errorf("la imagen no quedó en assets/ de la carpeta de la nota: %q", b)
	}

	// el mismo segundo no pisa la anterior
	got2, err := s.SaveFromClipboard(noteDir, "Mi Nota: plan?.md")
	if err != nil || got2 == got || !strings.HasSuffix(got2, "-2.png") {
		t.Errorf("segunda imagen = %q (%v), se esperaba otro nombre terminado en -2.png", got2, err)
	}
}

func TestSaveFromClipboardFailureLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	for name, r := range map[string]fakeReader{
		"error del sistema": {err: errors.New("sin imagen")},
		"archivo vacío":     {data: nil},
	} {
		_, err := newSaver(r).SaveFromClipboard(dir, "n.md")
		if err == nil {
			t.Errorf("%s: debería fallar", name)
		}
		if left, _ := filepath.Glob(filepath.Join(dir, "assets", "*")); len(left) != 0 {
			t.Errorf("%s: quedaron archivos %v", name, left)
		}
	}
}

func TestImportFileCopiesAndKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Foto Grande.PNG")
	_ = os.WriteFile(src, []byte("ORIGINAL"), 0o644)
	noteDir := filepath.Join(dir, "notas")

	got, err := newSaver(nil).ImportFile(noteDir, "diario.md", src)
	if err != nil {
		t.Fatal(err)
	}
	if want := "assets/diario-20261001-153045.png"; got != want {
		t.Fatalf("referencia = %q, se esperaba %q (extensión en minúsculas)", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(noteDir, filepath.FromSlash(got))); string(b) != "ORIGINAL" {
		t.Errorf("copia = %q", b)
	}
	if b, _ := os.ReadFile(src); string(b) != "ORIGINAL" {
		t.Error("el original cambió")
	}
	if _, err := newSaver(nil).ImportFile(noteDir, "diario.md", filepath.Join(dir, "no-existe.png")); err == nil {
		t.Error("importar un archivo inexistente debería fallar")
	}
}

func TestParsePastedPaths(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	plain := mk("foto.png")
	spaced := mk("mi foto (1).JPG")
	text := mk("notas.txt")
	_ = os.Mkdir(filepath.Join(dir, "carpeta.png"), 0o755)

	// Ghostty/macOS y Linux escapan con "\"; Windows Terminal entrega la ruta entre comillas.
	esc := strings.NewReplacer(" ", `\ `, "(", `\(`, ")", `\)`).Replace(spaced)
	if runtime.GOOS == "windows" {
		esc = `"` + spaced + `"`
	}
	fileURL := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(spaced), "/")}).String()
	cases := []struct {
		name, in string
		want     []string
	}{
		{"ruta simple", plain, []string{plain}},
		{"con salto final", plain + "\n", []string{plain}},
		{"espacios escapados", esc, []string{spaced}},
		{"entre comillas", `'` + spaced + `'`, []string{spaced}},
		{"url file://", fileURL, []string{spaced}},
		{"varias", plain + "\n" + esc, []string{plain, spaced}},
		{"no es imagen", text, nil},
		{"una buena y una mala", plain + "\n" + text, nil},
		{"texto normal", "hola mundo", nil},
		{"no existe", filepath.Join(dir, "fantasma.png"), nil},
		{"carpeta con nombre de imagen", filepath.Join(dir, "carpeta.png"), nil},
		{"ruta relativa", "foto.png", nil},
		{"vacío", "  \n ", nil},
	}
	for _, c := range cases {
		got := ParsePastedPaths(c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%s: ParsePastedPaths(%q) = %q, se esperaba %q", c.name, c.in, got, c.want)
		}
	}
}

// TestFilePathFromURL prueba la conversión de Windows en cualquier sistema.
func TestFilePathFromURL(t *testing.T) {
	cases := []struct {
		in      string
		windows bool
		want    string
	}{
		{"/C:/Users/yo/mi foto.png", true, `C:\Users\yo\mi foto.png`},
		{"/D:/x/y.jpg", true, `D:\x\y.jpg`},
		{"/unidad-de-red/x.png", true, `\unidad-de-red\x.png`},
		{"/Users/yo/mi foto.png", false, "/Users/yo/mi foto.png"},
		{"/C:/no/cambia.png", false, "/C:/no/cambia.png"},
	}
	for _, c := range cases {
		if got := filePathFromURL(c.in, c.windows); got != c.want {
			t.Errorf("filePathFromURL(%q, windows=%v) = %q, se esperaba %q", c.in, c.windows, got, c.want)
		}
	}
}
