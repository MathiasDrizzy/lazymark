package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestEditThroughInternalSymlink (ORD-019 C.3 / H3, L15): una nota que es un enlace simbólico a otra nota de la carpeta se edita escribiendo en el destino y
// conservando el enlace: marcar, mover de columna, fechas y migrate; sin falso "la nota cambió por fuera" (el mtime que se compara es el del destino).
func TestEditThroughInternalSymlink(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	real := filepath.Join(dir, "real.md")
	link := filepath.Join(dir, "enlace.md")
	os.WriteFile(real, []byte("# R\n- [ ] uno 📅 2026-05-10\n- [ ] dos\n"), 0o644)
	if err := os.Symlink("real.md", link); err != nil {
		t.Skip("sin enlaces simbólicos:", err)
	}
	s := New(dir)
	s.DateFormatPref = "emoji"
	find := func() Note {
		t.Helper()
		notes, err := s.ListNotes()
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range notes {
			if n.Path == link {
				return n
			}
		}
		t.Fatal("el enlace interno debe listarse como nota")
		return Note{}
	}
	stillLink := func(step string) {
		t.Helper()
		if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s: el enlace se reemplazó por un archivo", step)
		}
	}
	read := func() string { b, _ := os.ReadFile(real); return string(b) }

	n := find()
	if ok, err := s.ToggleTaskIfUnchanged(link, 3, n.ModTime); err != nil || !ok {
		t.Fatalf("marcar: %v %v", ok, err)
	}
	stillLink("marcar")
	if !strings.Contains(read(), "- [x] dos") {
		t.Errorf("marcar escribe en el destino:\n%s", read())
	}
	n = find()
	if err := s.SetTaskDate(link, 3, DateDue, "2026-06-01", n.ModTime); err != nil {
		t.Fatalf("fecha: %v", err)
	}
	stillLink("fecha")
	n = find()
	if err := s.MoveTask(link, 2, Columns{"todo", "doing", "done"}, 1, n.ModTime); err != nil {
		t.Fatalf("mover: %v", err)
	}
	stillLink("mover")
	if !strings.Contains(read(), "#kb/doing") {
		t.Errorf("mover escribe en el destino:\n%s", read())
	}
	done, err := s.MigrateDates(FormatDataview, false)
	if err != nil || len(done) == 0 {
		t.Fatalf("migrate: %v %v", done, err)
	}
	stillLink("migrate")
	if !strings.Contains(read(), "[due:: ") {
		t.Errorf("migrate escribe en el destino:\n%s", read())
	}
}

// TestEditThroughExternalSymlink (ORD-019 C.3): un enlace cuyo destino está fuera de la carpeta se rechaza con un error claro (no se escribe nada fuera) y migrate
// no corta la corrida por eso: las demás notas se migran.
func TestEditThroughExternalSymlink(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	dir := filepath.Join(root, "notas")
	os.MkdirAll(dir, 0o755)
	outside := filepath.Join(root, "fuera.md")
	body := "- [ ] ajena 📅 2026-05-10\n"
	os.WriteFile(outside, []byte(body), 0o644)
	link := filepath.Join(dir, "sale.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("sin enlaces simbólicos:", err)
	}
	os.WriteFile(filepath.Join(dir, "ok.md"), []byte("- [ ] mia 📅 2026-05-11\n"), 0o644)
	s := New(dir)
	for name, err := range map[string]error{
		"toggle": func() error { _, e := s.ToggleTaskIfUnchanged(link, 1, time.Time{}); return e }(),
		"fecha":  s.SetTaskDate(link, 1, DateDue, "2026-06-01", time.Time{}),
		"insert": s.InsertAfterLine(link, 1, "- [ ] x", time.Time{}),
	} {
		if !errors.Is(err, ErrOutsideNotes) {
			t.Errorf("%s: se esperaba ErrOutsideNotes, hay %v", name, err)
		}
	}
	if b, _ := os.ReadFile(outside); string(b) != body {
		t.Errorf("el archivo de fuera no se toca: %q", b)
	}
	done, err := s.MigrateDates(FormatDataview, false)
	if err != nil || len(done) != 1 || filepath.Base(done[0].Path) != "ok.md" {
		t.Errorf("migrate sigue con las demás notas: %v %v", done, err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Error("el enlace sigue siendo un enlace")
	}
}

// TestSymlinkToNonMarkdownIsNotANote (ORD-019, segunda opinión): un enlace llamado x.md cuyo destino NO es un .md (un script, por ejemplo) no es una nota: no se
// lista, no se edita y migrate no lo escribe, aunque sus líneas parezcan tareas.
func TestSymlinkToNonMarkdownIsNotANote(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	script := filepath.Join(dir, "script.sh")
	body := "#!/bin/sh\n- [ ] no soy tarea 📅 2026-05-10\n"
	os.WriteFile(script, []byte(body), 0o755)
	link := filepath.Join(dir, "tarea.md")
	if err := os.Symlink("script.sh", link); err != nil {
		t.Skip("sin enlaces simbólicos:", err)
	}
	s := New(dir)
	if _, err := s.ResolveNote(link); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("ResolveNote del enlace a un .sh: %v", err)
	}
	if err := s.SetTaskDate(link, 2, DateDue, "2026-06-01", time.Time{}); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("editar el enlace a un .sh: %v", err)
	}
	if done, err := s.MigrateDates(FormatDataview, false); err != nil || len(done) != 0 {
		t.Errorf("migrate no toca el .sh: %v %v", done, err)
	}
	if b, _ := os.ReadFile(script); string(b) != body {
		t.Errorf("el script no cambió: %q", b)
	}
	if fi, _ := os.Stat(script); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o755 { // en Windows los permisos Unix no existen
		t.Errorf("permisos intactos: %v", fi.Mode().Perm())
	}
}
