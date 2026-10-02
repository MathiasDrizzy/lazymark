package app

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestEditorEnv (P2): el editor se abre con LAZYMARK_NOTE y con la carpeta del ejecutable
// en el PATH, para que sus plugins llamen a `lazymark paste`.
func TestEditorEnv(t *testing.T) {
	exe := filepath.Join("opt", "lazymark", "bin", "lazymark")
	dir := filepath.Dir(exe)
	sep := string(filepath.ListSeparator)

	got := editorEnv([]string{"HOME=/h", "PATH=/usr/bin" + sep + "/bin", "LAZYMARK_NOTE=viejo"}, "/n/nota.md", exe)
	want := map[string]bool{"HOME=/h": true, "LAZYMARK_NOTE=/n/nota.md": true, "PATH=" + dir + sep + "/usr/bin" + sep + "/bin": true}
	if len(got) != len(want) {
		t.Fatalf("entorno = %q", got)
	}
	for _, kv := range got {
		if !want[kv] {
			t.Errorf("variable inesperada %q en %q", kv, got)
		}
	}

	// si la carpeta ya está en el PATH no se repite; sin PATH se crea
	got = editorEnv([]string{"PATH=/x" + sep + dir}, "/n.md", exe)
	if strings.Count(strings.Join(got, "\n"), dir) != 1 {
		t.Errorf("la carpeta se duplicó en el PATH: %q", got)
	}
	got = editorEnv(nil, "/n.md", exe)
	if len(got) != 2 || got[1] != "PATH="+dir {
		t.Errorf("sin PATH = %q", got)
	}

	// y el comando real los lleva
	m := newTestModel(t, 100, 30)
	m.c.cfg.Editor = "true"
	cmd := m.c.editorCommand("/n/nota.md", 1)
	if !strings.Contains(strings.Join(cmd.Env, "\n"), "LAZYMARK_NOTE=/n/nota.md") {
		t.Errorf("el comando del editor no lleva LAZYMARK_NOTE: %q", cmd.Env)
	}
}
