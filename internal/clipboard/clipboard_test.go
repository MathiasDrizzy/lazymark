package clipboard

import (
	"errors"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
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

// TestPSQuote (S7, INFERIDO en Windows): todas las comillas simples de PowerShell, también las tipográficas, se
// duplican; el resto de la ruta queda igual.
func TestPSQuote(t *testing.T) {
	for in, want := range map[string]string{
		`C:\notas\a.png`:           `C:\notas\a.png`,
		`it's.png`:                 `it''s.png`,
		"a\u2019);calc;(\u2019.md": "a\u2019\u2019);calc;(\u2019\u2019.md",
		"\u2018\u201a\u201b":       "\u2018\u2018\u201a\u201a\u201b\u201b",
	} {
		if got := psQuote(in); got != want {
			t.Errorf("psQuote(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

// TestPasteDoesNotFollowSymlinks (ORD-019 C.5 / H5): si assets/ es un enlace simbólico no se escribe (ni por él se llega a otra carpeta), y un enlace colgante con
// el nombre que tocaba cuenta como ocupado (se usa otro nombre) en vez de escribir en su destino.
func TestPasteDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "fuera")
	os.MkdirAll(outside, 0o755)
	noteDir := filepath.Join(root, "notas")
	os.MkdirAll(noteDir, 0o755)
	if err := os.Symlink(outside, filepath.Join(noteDir, "assets")); err != nil {
		t.Skip("sin enlaces simbólicos:", err)
	}
	s := newSaver(fakeReader{data: []byte("PNG")})
	if _, err := s.SaveFromClipboard(noteDir, "n.md"); err == nil || !strings.Contains(err.Error(), "enlace") {
		t.Errorf("assets/ es un enlace simbólico: debe rechazarse con un error claro: %v", err)
	}
	src := filepath.Join(root, "foto.png")
	os.WriteFile(src, []byte("PNG"), 0o644)
	if _, err := s.ImportFile(noteDir, "n.md", src); err == nil {
		t.Error("ImportFile tampoco escribe a través de un assets/ enlazado")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("no se escribió nada fuera de la carpeta de la nota: %v", entries)
	}

	// un enlace colgante con el nombre que tocaba
	noteDir2 := filepath.Join(root, "notas2")
	os.MkdirAll(filepath.Join(noteDir2, "assets"), 0o755)
	victim := filepath.Join(root, "victima.png")
	os.Symlink(victim, filepath.Join(noteDir2, "assets", "n-20261001-153045.png"))
	got, err := s.SaveFromClipboard(noteDir2, "n.md")
	if err != nil || !strings.HasSuffix(got, "-2.png") {
		t.Errorf("el nombre ocupado por un enlace colgante se salta: %q %v", got, err)
	}
	if _, err := os.Lstat(victim); err == nil {
		t.Error("se creó el destino del enlace colgante")
	}
	got, err = s.ImportFile(noteDir2, "n.md", src)
	if err != nil || strings.HasSuffix(got, "153045.png") {
		t.Errorf("ImportFile: %q %v", got, err)
	}
	if _, err := os.Lstat(victim); err == nil {
		t.Error("ImportFile creó el destino del enlace colgante")
	}
}

// TestPasteRefEscapesBrackets (ORD-019 C.5 / H6): la referencia escapa ( ) [ ] y los espacios del nombre para no romper el markdown.
func TestPasteRefEscapesBrackets(t *testing.T) {
	dir := t.TempDir()
	s := newSaver(fakeReader{data: []byte("PNG")})
	got, err := s.SaveFromClipboard(dir, "Foto (1) [x].md")
	if err != nil {
		t.Fatal(err)
	}
	if want := `assets/foto-\(1\)-\[x\]-20261001-153045.png`; got != want {
		t.Errorf("referencia = %q, se esperaba %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "foto-(1)-[x]-20261001-153045.png")); err != nil {
		t.Errorf("el archivo se llama sin escapes: %v", err)
	}
	if ref("a b.png") != "assets/a%20b.png" {
		t.Errorf("un espacio va como %%20: %q", ref("a b.png"))
	}
	if storage.UnescapeRef(got) != "assets/foto-(1)-[x]-20261001-153045.png" || storage.UnescapeRef("assets/a%20b.png") != "assets/a b.png" {
		t.Errorf("UnescapeRef deshace el escape: %q", storage.UnescapeRef(got))
	}
}
