package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, content string) (*Storage, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "n.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return New(dir), path
}

func modTime(t *testing.T, path string) time.Time {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime()
}

// TestAppendToNote: la referencia se agrega al final, separada por una línea en
// blanco, sin tocar nada de lo anterior (X10), con o sin salto final.
func TestAppendToNote(t *testing.T) {
	for _, c := range []struct{ name, orig, want string }{
		{"con salto final", "# T\n\ntexto\n", "# T\n\ntexto\n\n![](assets/a.png)\n"},
		{"sin salto final", "# T\n\ntexto", "# T\n\ntexto\n\n![](assets/a.png)\n"},
		{"CRLF", "# T\r\n\r\ntexto\r\n", "# T\r\n\r\ntexto\r\n\r\n![](assets/a.png)\r\n"},
		{"vacía", "", "![](assets/a.png)\n"},
	} {
		s, path := write(t, c.orig)
		if err := s.AppendToNote(path, "![](assets/a.png)", modTime(t, path)); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
		if !bytes.HasPrefix(got, []byte(c.orig)) && c.name != "sin salto final" {
			t.Errorf("%s: el contenido original cambió", c.name)
		}
	}
}

// TestInsertAfterLine: la referencia queda debajo de la línea indicada y el
// resto del archivo no cambia.
func TestInsertAfterLine(t *testing.T) {
	for _, c := range []struct {
		name, orig string
		line       int
		want       string
	}{
		{"en el medio", "a\n- [ ] tarea\nb\n", 2, "a\n- [ ] tarea\n\n![](x.png)\nb\n"},
		{"última línea", "a\n- [ ] tarea", 2, "a\n- [ ] tarea\n\n![](x.png)"},
		{"CRLF", "a\r\n- [ ] tarea\r\nb\r\n", 2, "a\r\n- [ ] tarea\r\n\r\n![](x.png)\r\nb\r\n"},
	} {
		s, path := write(t, c.orig)
		if err := s.InsertAfterLine(path, c.line, "![](x.png)", modTime(t, path)); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got, _ := os.ReadFile(path); string(got) != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
	s, path := write(t, "a\n")
	if err := s.InsertAfterLine(path, 9, "![](x.png)", time.Time{}); err == nil {
		t.Error("una línea fuera de rango debería fallar")
	}
}

// TestInsertRefusesStaleNote (X10): si la nota cambió por fuera, no se escribe.
func TestInsertRefusesStaleNote(t *testing.T) {
	s, path := write(t, "uno\n")
	loaded := modTime(t, path)
	external := []byte("uno\nagregado por otro editor\n")
	if err := os.WriteFile(path, external, 0o644); err != nil {
		t.Fatal(err)
	}
	later := loaded.Add(time.Hour)
	_ = os.Chtimes(path, later, later)

	if err := s.AppendToNote(path, "![](a.png)", loaded); !errors.Is(err, ErrNoteChanged) {
		t.Errorf("AppendToNote: err = %v, se esperaba ErrNoteChanged", err)
	}
	if err := s.InsertAfterLine(path, 1, "![](a.png)", loaded); !errors.Is(err, ErrNoteChanged) {
		t.Errorf("InsertAfterLine: err = %v, se esperaba ErrNoteChanged", err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, external) {
		t.Fatalf("se pisó la edición externa: %q", got)
	}
}
