package editors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// env aísla todo en un directorio temporal: nada de esto toca el HOME real (P2).
func env(t *testing.T) Env {
	t.Helper()
	home := t.TempDir()
	return Env{Home: home, Getenv: func(string) string { return "" }}
}

// snapshot devuelve ruta -> contenido de todos los archivos bajo dir.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(dir, p)
			out[rel] = string(b)
		}
		return nil
	})
	return out
}

func equal(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// TestInstallIdempotentAndUninstall (P2): instalar dos veces deja lo mismo que una vez, y
// desinstalar deja la configuración como estaba, para cada editor.
func TestInstallIdempotentAndUninstall(t *testing.T) {
	for _, editor := range []string{"micro", "vim", "nano"} {
		e := env(t)
		before := snapshot(t, e.Home)
		if _, err := Install(editor, e); err != nil {
			t.Fatalf("%s: %v", editor, err)
		}
		once := snapshot(t, e.Home)
		if equal(before, once) {
			t.Fatalf("%s: install no escribió nada", editor)
		}
		if _, err := Install(editor, e); err != nil {
			t.Fatalf("%s (2.ª vez): %v", editor, err)
		}
		if twice := snapshot(t, e.Home); !equal(once, twice) {
			t.Errorf("%s: instalar dos veces cambió algo: %v → %v", editor, once, twice)
		}
		if _, err := Uninstall(editor, e); err != nil {
			t.Fatalf("%s: %v", editor, err)
		}
		if after := snapshot(t, e.Home); !equal(before, after) {
			t.Errorf("%s: uninstall no dejó la configuración como estaba: %v", editor, after)
		}
		if _, err := Uninstall(editor, e); err != nil {
			t.Errorf("%s: desinstalar lo que no está debe ser un no-op: %v", editor, err)
		}
	}
}

// TestNanoKeepsUserRC (P2): el nanorc del usuario solo gana el bloque de lazymark; al
// desinstalar queda byte a byte como estaba, con o sin salto de línea final.
func TestNanoKeepsUserRC(t *testing.T) {
	for _, original := range []string{"set linenumbers\nset mouse\n", "set tabsize 4", ""} {
		e := env(t)
		rc := filepath.Join(e.Home, ".nanorc")
		os.WriteFile(rc, []byte(original), 0o644)
		if _, err := Install("nano", e); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(rc)
		if !strings.HasPrefix(string(got), original) || strings.Count(string(got), "bind M-7") != 1 {
			t.Fatalf("el nanorc del usuario debe conservar su contenido y ganar 1 bind:\n%s", got)
		}
		Install("nano", e)
		if again, _ := os.ReadFile(rc); string(again) != string(got) {
			t.Error("instalar otra vez cambió el nanorc")
		}
		if _, err := Uninstall("nano", e); err != nil {
			t.Fatal(err)
		}
		if after, err := os.ReadFile(rc); err != nil || string(after) != original {
			t.Errorf("uninstall dejó %q (%v), se esperaba %q", after, err, original)
		}
	}
}

// TestNeverOverwritesForeignFiles (P2): si en el lugar del plugin de micro o de vim hay un archivo del
// usuario que no es de lazymark, install falla sin tocarlo y uninstall tampoco lo borra.
func TestNeverOverwritesForeignFiles(t *testing.T) {
	cases := map[string]string{
		"micro": filepath.Join(".config", "micro", "plug", "lazymark", "lazymark.lua"),
		"vim":   filepath.Join(".vim", "pack", "lazymark", "start", "lazymark", "plugin", "lazymark.vim"),
	}
	for editor, rel := range cases {
		e := env(t)
		path := filepath.Join(e.Home, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte("mío, no de lazymark\n"), 0o644)
		before := snapshot(t, e.Home)
		if _, err := Install(editor, e); err == nil {
			t.Errorf("%s: install debía negarse a pisar un archivo ajeno", editor)
		}
		if _, err := Uninstall(editor, e); err == nil {
			t.Errorf("%s: uninstall debía negarse a borrar un archivo ajeno", editor)
		}
		if after := snapshot(t, e.Home); !equal(before, after) {
			t.Errorf("%s: se tocó un archivo del usuario: %v", editor, after)
		}
	}
}

// TestNanoRefusesToOverrideBind (P2): si el nanorc ya enlaza M-7, install se niega y no lo toca.
func TestNanoRefusesToOverrideBind(t *testing.T) {
	e := env(t)
	rc := filepath.Join(e.Home, ".nanorc")
	original := "set mouse\nbind M-7 \"{justify}\" main\n"
	os.WriteFile(rc, []byte(original), 0o644)
	if _, err := Install("nano", e); err == nil {
		t.Error("install debía negarse: M-7 ya está enlazada")
	}
	if got, _ := os.ReadFile(rc); string(got) != original {
		t.Errorf("se tocó el nanorc: %q", got)
	}
}

// TestRespectsConfigHomes (P2): micro usa MICRO_CONFIG_HOME y XDG_CONFIG_HOME, y nano el nanorc de XDG si no hay ~/.nanorc.
func TestRespectsConfigHomes(t *testing.T) {
	home, custom := t.TempDir(), t.TempDir()
	e := Env{Home: home, Getenv: func(k string) string {
		if k == "MICRO_CONFIG_HOME" {
			return custom
		}
		return ""
	}}
	if _, err := Install("micro", e); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(custom, "plug", "lazymark", "lazymark.lua")); err != nil {
		t.Errorf("micro no respetó MICRO_CONFIG_HOME: %v", err)
	}
	if snap := snapshot(t, home); len(snap) != 0 {
		t.Errorf("no debía escribir en HOME: %v", snap)
	}

	xdg := t.TempDir()
	os.MkdirAll(filepath.Join(xdg, "nano"), 0o755)
	os.WriteFile(filepath.Join(xdg, "nano", "nanorc"), []byte("set mouse\n"), 0o644)
	e = Env{Home: t.TempDir(), Getenv: func(k string) string {
		if k == "XDG_CONFIG_HOME" {
			return xdg
		}
		return ""
	}}
	if _, err := Install("nano", e); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(xdg, "nano", "nanorc")); !strings.Contains(string(b), "bind M-7") {
		t.Errorf("nano no usó el nanorc de XDG_CONFIG_HOME:\n%s", b)
	}
}

func TestUnknownEditor(t *testing.T) {
	if _, err := Install("emacs", env(t)); err == nil {
		t.Error("un editor desconocido debe dar error")
	}
}
