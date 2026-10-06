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

// TestEditorGetsAnAbsolutePath (ORD-019 C.2 / H2): con `--dir .` el nombre de una nota puede empezar con `+` o `-` (`+!cmd.md`, `-c.md`): el editor lo tomaría como
// una opción o un comando (vim ejecuta `+!cmd`). La nota se le pasa siempre como ruta absoluta.
func TestEditorGetsAnAbsolutePath(t *testing.T) {
	m := newTestModel(t, 100, 30)
	for _, name := range []string{"+!cmd.md", "-c.md", "--servername.md", "+5.md"} {
		for _, line := range []int{1, 7} {
			cmd := m.c.editorCommand(name, line)
			last := cmd.Args[len(cmd.Args)-1]
			if !filepath.IsAbs(last) || !strings.HasSuffix(last, string(filepath.Separator)+name) {
				t.Errorf("%q (línea %d): el editor recibe %q, debe ser una ruta absoluta que termina en el nombre", name, line, last)
			}
			for _, a := range cmd.Args[1:] {
				if a == name {
					t.Errorf("%q llega tal cual al editor: %v", name, cmd.Args)
				}
			}
			found := false
			for _, kv := range cmd.Env {
				found = found || (strings.HasPrefix(kv, "LAZYMARK_NOTE=") && filepath.IsAbs(strings.TrimPrefix(kv, "LAZYMARK_NOTE=")))
			}
			if !found {
				t.Errorf("%q: LAZYMARK_NOTE debe ser absoluta", name)
			}
		}
	}
}
