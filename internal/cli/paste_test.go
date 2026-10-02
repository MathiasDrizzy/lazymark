package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/editors"
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
	os.WriteFile(note, nil, 0o644)

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
	os.WriteFile(note, nil, 0o644)
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
	os.WriteFile(note, nil, 0o644)
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
	os.WriteFile(filepath.Join(dir, "n.md"), nil, 0o644)
	code, out, _ := pasteWith(t, fakeClip{png: []byte("x")}, []string{"--no-newline", filepath.Join(dir, "n.md")}, nil)
	if code != 0 || !strings.HasPrefix(out, "![](assets/n-") || strings.HasSuffix(out, "\n") || !strings.HasSuffix(out, ")") {
		t.Errorf("exit=%d stdout=%q", code, out)
	}
}

// spyClip cuenta cuántas veces se mira el portapapeles: con una entrada inválida no debe mirarse nunca (R17).
type spyClip struct{ reads *int }

func (s spyClip) ReadImage(dest string) error {
	*s.reads++
	return os.WriteFile(dest, []byte("x"), 0o644)
}
func (s spyClip) ReadFiles() ([]string, error) { *s.reads++; return nil, errors.New("no") }

// TestPasteValidatesBeforeTouchingClipboard (R17): -h/--help, opciones desconocidas, argumentos de más,
// una nota que no existe o que no termina en .md fallan (o ayudan) SIN leer el portapapeles y sin
// escribir nada, ni en el cwd ni junto a la nota. Antes `--help` se tomaba como el nombre de la
// nota y escribía ./assets/--help-<fecha>.png con el portapapeles real.
func TestPasteValidatesBeforeTouchingClipboard(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Chdir(dir)
	defer os.Chdir(cwd)
	real := filepath.Join(dir, "real.md")
	os.WriteFile(real, nil, 0o644)
	txt := filepath.Join(dir, "notas.txt")
	os.WriteFile(txt, nil, 0o644)
	cases := []struct {
		name string
		args []string
		env  map[string]string
		code int
	}{
		{"--help", []string{"--help"}, nil, 0},
		{"-h", []string{"-h"}, nil, 0},
		{"opción desconocida", []string{"--foo"}, nil, 2},
		{"opción desconocida y nota", []string{"--foo", real}, nil, 2},
		{"-x", []string{"-x"}, nil, 2},
		{"dos notas", []string{real, real}, nil, 2},
		{"nota inexistente", []string{filepath.Join(dir, "no-existe.md")}, nil, 2},
		{"no termina en .md", []string{txt}, nil, 2},
		{"una carpeta .md", []string{dir + "/carpeta.md"}, nil, 2},
		{"sin nota", nil, nil, 2},
		{"$LAZYMARK_NOTE inexistente", nil, map[string]string{"LAZYMARK_NOTE": filepath.Join(dir, "x.md")}, 2},
	}
	os.Mkdir(filepath.Join(dir, "carpeta.md"), 0o755)
	for _, c := range cases {
		reads := 0
		var out, errb bytes.Buffer
		saver := &clipboard.Saver{Reader: spyClip{&reads}, Files: spyClip{&reads}}
		code := RunPaste(c.args, func(k string) string { return c.env[k] }, saver, &out, &errb)
		if code != c.code {
			t.Errorf("%s: exit = %d, se esperaba %d", c.name, code, c.code)
		}
		if reads != 0 {
			t.Errorf("%s: leyó el portapapeles %d veces", c.name, reads)
		}
		if c.code == 0 && !strings.Contains(out.String(), "lazymark paste") {
			t.Errorf("%s: la ayuda no sale por stdout: %q", c.name, out.String())
		}
		if c.code != 0 && (out.Len() != 0 || errb.Len() == 0) {
			t.Errorf("%s: stdout=%q stderr=%q", c.name, out.String(), errb.String())
		}
	}
	// nada escrito: ni assets/ en el cwd ni al lado de la nota ni archivos raros
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() == "assets" || strings.HasPrefix(e.Name(), "--") {
			t.Errorf("quedó %q escrito", e.Name())
		}
	}
}

// TestHelpAndUnknownOptionsAreSafe (R17): -h/--help ayuda y una opción desconocida se rechaza en
// editor-plugins y en task/note, sin tocar nada.
func TestHelpAndUnknownOptionsAreSafe(t *testing.T) {
	home := t.TempDir()
	env := editors.Env{Home: home, Getenv: func(string) string { return "" }}
	for _, args := range [][]string{{"--help"}, {"install", "--help"}, {"-h"}} {
		var out, errb bytes.Buffer
		if code := RunEditorPlugins(args, env, &out, &errb); code != 0 || !strings.Contains(out.String(), "editor-plugins") {
			t.Errorf("editor-plugins %v: exit=%d stdout=%q", args, code, out.String())
		}
	}
	for _, args := range [][]string{{"install", "--foo"}, {"uninstall", "-x", "micro"}} {
		var out, errb bytes.Buffer
		if code := RunEditorPlugins(args, env, &out, &errb); code != 2 || out.Len() != 0 {
			t.Errorf("editor-plugins %v: exit=%d stdout=%q", args, code, out.String())
		}
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("se escribió algo en el HOME: %v", entries)
	}
	for _, run := range []func(io.Writer, []string, string) error{RunTaskWithWriter, RunNoteWithWriter} {
		var out bytes.Buffer
		if err := run(&out, []string{"--help"}, t.TempDir()); err != nil || !strings.Contains(out.String(), "lazymark") {
			t.Errorf("--help: err=%v stdout=%q", err, out.String())
		}
	}
}
