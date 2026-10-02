package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
)

// fakeClip simula el portapapeles: P1 nunca toca el del sistema (R17).
type fakeClip struct {
	png   []byte   // imagen copiada (captura de pantalla), nil si no hay
	files []string // archivos copiados en el Finder
}

func (f fakeClip) ReadImage(dest string) error {
	if f.png == nil {
		return errors.New("no hay imagen")
	}
	return os.WriteFile(dest, f.png, 0o644)
}

func (f fakeClip) ReadFiles() ([]string, error) {
	if len(f.files) == 0 {
		return nil, errors.New("no hay archivos")
	}
	return f.files, nil
}

func pasteWith(t *testing.T, c fakeClip, args []string, env map[string]string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	saver := &clipboard.Saver{Reader: c, Files: c}
	code = RunPaste(args, func(k string) string { return env[k] }, saver, &out, &errb)
	return code, out.String(), errb.String()
}

// TestPasteWithImage (P1): con una captura en el portapapeles, la guarda en assets/
// junto a la nota e imprime solo la referencia.
func TestPasteWithImage(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "Mi nota.md")
	os.WriteFile(note, []byte("# Mi nota\n"), 0o644)

	code, out, errOut := pasteWith(t, fakeClip{png: []byte("\x89PNG-simulado")}, []string{note}, nil)
	if code != 0 || errOut != "" {
		t.Fatalf("exit=%d stderr=%q", code, errOut)
	}
	ref := strings.TrimSpace(out)
	if !strings.HasPrefix(ref, "![](assets/mi-nota-") || !strings.HasSuffix(ref, ".png)") || strings.Count(out, "\n") != 1 {
		t.Fatalf("stdout debe ser solo la referencia: %q", out)
	}
	rel := strings.TrimSuffix(strings.TrimPrefix(ref, "![]("), ")")
	if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))); err != nil || string(b) != "\x89PNG-simulado" {
		t.Errorf("la imagen no quedó en assets/ junto a la nota: %v %q", err, b)
	}
	if b, _ := os.ReadFile(note); string(b) != "# Mi nota\n" {
		t.Error("paste no debe tocar la nota: la inserta el editor")
	}
}

// TestPasteWithFile (P1): con un archivo de imagen copiado, lo copia a assets/ y el original no se toca.
func TestPasteWithFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "origen", "Foto Vacaciones.JPG")
	os.MkdirAll(filepath.Dir(src), 0o755)
	os.WriteFile(src, []byte("JPEG-simulado"), 0o644)
	note := filepath.Join(dir, "notas", "viaje.md")
	os.MkdirAll(filepath.Dir(note), 0o755)

	code, out, errOut := pasteWith(t, fakeClip{files: []string{filepath.Join(dir, "origen", "no-es-imagen.txt"), src}}, []string{note}, nil)
	if code != 0 || errOut != "" {
		t.Fatalf("exit=%d stderr=%q", code, errOut)
	}
	if ref := strings.TrimSpace(out); !strings.HasPrefix(ref, "![](assets/viaje-") || !strings.HasSuffix(ref, ".jpg)") {
		t.Fatalf("referencia inesperada: %q", out)
	}
	if b, _ := os.ReadFile(src); string(b) != "JPEG-simulado" {
		t.Error("el archivo original se modificó")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "notas", "assets"))
	if len(entries) != 1 {
		t.Errorf("assets/ debe tener 1 archivo, tiene %d", len(entries))
	}
}

// TestPasteWithoutImage (P1): sin imagen, exit ≠ 0, nada en stdout, el motivo en stderr
// y ninguna carpeta assets/ huérfana.
func TestPasteWithoutImage(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "n.md")
	code, out, errOut := pasteWith(t, fakeClip{}, []string{note}, nil)
	if code == 0 || out != "" || strings.TrimSpace(errOut) == "" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets")); !os.IsNotExist(err) {
		t.Error("un paste sin imagen no debe dejar assets/")
	}
}

// TestPasteNoteFromEnvironment (P1): sin argumento usa $LAZYMARK_NOTE; sin ninguno falla sin tocar nada.
func TestPasteNoteFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "desde-env.md")
	code, out, _ := pasteWith(t, fakeClip{png: []byte("x")}, nil, map[string]string{"LAZYMARK_NOTE": note})
	if code != 0 || !strings.Contains(out, "assets/desde-env-") {
		t.Fatalf("exit=%d stdout=%q", code, out)
	}
	code, out, errOut := pasteWith(t, fakeClip{png: []byte("x")}, nil, nil)
	if code == 0 || out != "" || !strings.Contains(errOut, "LAZYMARK_NOTE") {
		t.Errorf("sin nota debe fallar y decirlo: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
}

// TestPasteNoNewline (P1): --no-newline imprime la referencia sin salto de línea (para nano).
func TestPasteNoNewline(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := pasteWith(t, fakeClip{png: []byte("x")}, []string{"--no-newline", filepath.Join(dir, "n.md")}, nil)
	if code != 0 || !strings.HasPrefix(out, "![](assets/n-") || strings.HasSuffix(out, "\n") || !strings.HasSuffix(out, ")") {
		t.Errorf("exit=%d stdout=%q", code, out)
	}
}
