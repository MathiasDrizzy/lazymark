package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSymlinkedRelativeVaultStaysConfined (auditoría del PR #4): con la carpeta de notas como enlace simbólico y con una ruta relativa,
// ni un enlace a un archivo de fuera, ni un enlace a una carpeta de fuera, ni un ".." sacan contenido de la carpeta de notas.
func TestSymlinkedRelativeVaultStaysConfined(t *testing.T) {
	tmp := t.TempDir()
	outside, real := filepath.Join(tmp, "outside"), filepath.Join(tmp, "real")
	for _, d := range []string{outside, real} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(outside, "secret.md"), []byte("- [ ] secreto\n"), 0o644)
	os.WriteFile(filepath.Join(real, "ok.md"), []byte("hola #a\n"), 0o644)
	if os.Symlink(outside, filepath.Join(real, "link")) != nil || os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(real, "s.md")) != nil ||
		os.Symlink(real, filepath.Join(tmp, "root")) != nil {
		t.Skip("sin enlaces simbólicos en este sistema")
	}
	notes, err := New(filepath.Join(tmp, "root")).ListNotes()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if strings.Contains(n.Content, "secreto") || strings.Contains(n.Path, "secret") {
			t.Errorf("ListNotes filtra %s", n.Path)
		}
	}
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	os.Chdir(tmp)
	for _, s := range []*Storage{New(filepath.Join(tmp, "root")), New("root")} {
		for _, p := range []string{"s.md", "link/secret.md", "../outside/secret.md", filepath.Join(outside, "secret.md")} {
			if got, err := s.ResolveNote(p); err == nil {
				t.Errorf("ResolveNote(%q) con vault %q permitió %s", p, s.BaseDir, got)
			}
		}
		if _, err := s.ResolveNote("ok.md"); err != nil {
			t.Errorf("ok.md: %v", err)
		}
	}
}

// TestTagsStayLinearOnHostileInput (auditoría del PR #4): el escáner de etiquetas no se vuelve cuadrático con entradas hechas para ello.
func TestTagsStayLinearOnHostileInput(t *testing.T) {
	s := New(t.TempDir())
	for name, c := range map[string]string{
		"backticks": strings.Repeat("`a ``b ```c ", 100000),
		"parens":    strings.Repeat("](", 200000),
		"nested":    "[x](" + strings.Repeat("(", 300000),
		"hashes":    strings.Repeat("#a ", 300000),
		"fences":    strings.Repeat("```\n", 200000),
		"listfence": strings.Repeat("- ```\n    ```\n", 100000),
	} {
		t0 := time.Now()
		s.extractTags(c)
		if d := time.Since(t0); d > 3*time.Second {
			t.Errorf("%s: %v", name, d)
		}
	}
}
