package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestToggleTaskRewritesOnlyThatLine (H2-2, C3): alternar una tarea cambia
// únicamente los 3 bytes de su casilla; el resto del archivo queda idéntico
// byte a byte (CRLF, tabuladores, espacios finales y sin salto final incluidos).
func TestToggleTaskRewritesOnlyThatLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.md")
	orig := []byte("# Título\r\n\r\n- [ ] primera  \r\n\t- [ ] anidada con tab\r\n- [x] hecha\r\n* [ ] con asterisco #doing\n\nfinal sin salto")
	if err := os.WriteFile(path, orig, 0o640); err != nil {
		t.Fatal(err)
	}
	s := New(dir)
	s.DateFormatPref = "emoji" // estas pruebas ejercitan el formato de emojis
	cases := []struct {
		line int
		old  string
		new  string
		done bool
	}{
		{3, "- [ ] primera", "- [x] primera", true},
		{4, "\t- [ ] anidada", "\t- [x] anidada", true},
		{5, "- [x] hecha", "- [ ] hecha", false},
		{6, "* [ ] con", "* [x] con", true},
	}
	for _, c := range cases {
		before, _ := os.ReadFile(path)
		got, err := s.ToggleTask(path, c.line)
		if err != nil {
			t.Fatalf("línea %d: %v", c.line, err)
		}
		if got != c.done {
			t.Errorf("línea %d: done=%v, se esperaba %v", c.line, got, c.done)
		}
		after, _ := os.ReadFile(path)
		// marcar agrega ✅ <hoy> a esa línea y desmarcar lo quita; fuera de eso, solo cambia la casilla
		stamp := []byte(" ✅ 2026-10-02")
		if n := bytes.Count(after, stamp) - bytes.Count(before, stamp); (c.done && n != 1) || (!c.done && n != 0) {
			t.Fatalf("línea %d: la fecha de completada cambió en %d (done=%v): %q", c.line, n, c.done, after)
		}
		after = bytes.ReplaceAll(after, stamp, nil)
		want := bytes.Replace(bytes.ReplaceAll(before, stamp, nil), []byte(c.old), []byte(c.new), 1)
		if !bytes.Equal(after, want) {
			t.Fatalf("línea %d: el archivo cambió más que la casilla:\nantes:   %q\ndespués: %q\nesperado:%q", c.line, before, after, want)
		}
	}
	// Windows no tiene permisos Unix (todo archivo figura como 0666): solo se comprueban en el resto.
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o640 {
			t.Errorf("los permisos cambiaron: %v", fi.Mode().Perm())
		}
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("quedó un .tmp")
	}
	if _, err := s.ToggleTask(path, 1); err == nil {
		t.Error("alternar una línea que no es tarea debería fallar")
	}
	if _, err := s.ToggleTask(path, 99); err == nil {
		t.Error("alternar una línea fuera de rango debería fallar")
	}
}

// TestToggleTaskRefusesStaleNote (X10, C7): si la nota cambió por fuera desde
// que se cargó, no se pisa.
func TestToggleTaskRefusesStaleNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.md")
	loaded := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	if err := os.WriteFile(path, []byte("- [ ] tarea\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, loaded, loaded)
	s := New(dir)
	s.DateFormatPref = "emoji" // estas pruebas ejercitan el formato de emojis

	// Otro programa edita la nota después de que lazymark la cargó.
	external := []byte("- [ ] tarea\nlínea agregada por otro editor\n")
	if err := os.WriteFile(path, external, 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, loaded.Add(time.Hour), loaded.Add(time.Hour))

	_, err := s.ToggleTaskIfUnchanged(path, 1, loaded)
	if !errors.Is(err, ErrNoteChanged) {
		t.Fatalf("err = %v, se esperaba ErrNoteChanged", err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, external) {
		t.Fatalf("se pisó la edición externa: %q", got)
	}

	// Con la fecha actual de la nota sí escribe, conservando la línea nueva.
	fi, _ := os.Stat(path)
	if _, err := s.ToggleTaskIfUnchanged(path, 1, fi.ModTime()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, []byte("- [x] tarea ✅ 2026-10-02\nlínea agregada por otro editor\n")) {
		t.Fatalf("contenido final: %q", got)
	}
}
