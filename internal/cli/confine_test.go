package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

func outsideLayout(t *testing.T) (notes, note, secret string) {
	t.Helper()
	root := t.TempDir()
	notes = filepath.Join(root, "notas")
	os.MkdirAll(notes, 0o755)
	note = filepath.Join(notes, "a.md")
	os.WriteFile(note, []byte("# A\n- [ ] dentro\n"), 0o644)
	secret = filepath.Join(root, "secreto.md")
	os.WriteFile(secret, []byte("clave\n- [ ] fuera\n"), 0o644)
	return
}

// TestNoteGetAndTaskToggleStayInsideNotes (seguridad): `note get` no lee y `task toggle` no escribe archivos de
// fuera de la carpeta de notas (ruta absoluta, "..", symlink): exit 2, nada en stdout y el archivo intacto.
// Antes, `note get /ruta/cualquiera.md` la imprimía y `task toggle` le cambiaba la línea.
func TestNoteGetAndTaskToggleStayInsideNotes(t *testing.T) {
	notes, note, secret := outsideLayout(t)
	paths := map[string]string{
		"absoluta": secret,
		"con ..":   filepath.Join(notes, "..", "secreto.md"),
		"relativa": filepath.Join("..", "secreto.md"),
		"/etc":     "/etc/hosts",
	}
	if runtime.GOOS != "windows" {
		os.Symlink(secret, filepath.Join(notes, "enlace.md"))
		paths["symlink"] = filepath.Join(notes, "enlace.md")
	}
	before, _ := os.ReadFile(secret)
	for name, p := range paths {
		var out bytes.Buffer
		err := RunNoteWithWriter(&out, []string{"get", p, "--dir", notes}, notes)
		if ExitCode(err) != ExitUsage || out.Len() != 0 {
			t.Errorf("note get (%s): exit=%d stdout=%q err=%v", name, ExitCode(err), out.String(), err)
		}
		out.Reset()
		err = RunTaskWithWriter(&out, []string{"toggle", "--path", p, "--line", "2", "--dir", notes}, notes)
		if ExitCode(err) != ExitUsage || !errors.Is(err, storage.ErrOutsideNotes) {
			t.Errorf("task toggle (%s): exit=%d err=%v", name, ExitCode(err), err)
		}
	}
	if after, _ := os.ReadFile(secret); string(after) != string(before) {
		t.Errorf("se modificó un archivo de fuera: %q", after)
	}
	// dentro funciona: ruta absoluta y relativa a la carpeta
	for _, p := range []string{note, "a.md"} {
		var out bytes.Buffer
		if err := RunNoteWithWriter(&out, []string{"get", p, "--dir", notes}, notes); err != nil || out.Len() == 0 {
			t.Errorf("note get %q dentro: %v", p, err)
		}
	}
	var out bytes.Buffer
	if err := RunTaskWithWriter(&out, []string{"toggle", "--path", note, "--line", "2", "--dir", notes}, notes); err != nil {
		t.Errorf("task toggle dentro: %v", err)
	}
}
