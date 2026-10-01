package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveNoteDoesNotOverwrite(t *testing.T) {
	base := t.TempDir()
	s := New(base)
	sub := filepath.Join(base, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(base, "a.md")
	dst := filepath.Join(sub, "a.md")
	_ = os.WriteFile(src, []byte("origen"), 0o644)
	_ = os.WriteFile(dst, []byte("destino"), 0o644)

	if err := s.MoveNote(src, sub); err == nil {
		t.Fatal("MoveNote sobre un nombre existente debería fallar")
	}
	if b, _ := os.ReadFile(dst); string(b) != "destino" {
		t.Fatalf("la nota de destino fue sobrescrita: %q", b)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal("la nota de origen desapareció")
	}
}

func TestRenameNoteAndFolder(t *testing.T) {
	base := t.TempDir()
	s := New(base)
	note := filepath.Join(base, "vieja.md")
	_ = os.WriteFile(note, []byte("# Vieja\n"), 0o644)

	got, err := s.Rename(note, "Nota Nueva")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "nota-nueva.md"); got != want {
		t.Fatalf("Rename nota = %q, se esperaba %q", got, want)
	}
	if b, _ := os.ReadFile(got); string(b) != "# Vieja\n" {
		t.Fatalf("el contenido cambió: %q", b)
	}

	dir := filepath.Join(base, "carpeta")
	_ = os.MkdirAll(dir, 0o755)
	got, err = s.Rename(dir, "Proyectos")
	if err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(got); err != nil || !fi.IsDir() || filepath.Base(got) != "proyectos" {
		t.Fatalf("Rename carpeta = %q (%v)", got, err)
	}
}

func TestRenameRejectsCollisionAndEmpty(t *testing.T) {
	base := t.TempDir()
	s := New(base)
	a := filepath.Join(base, "a.md")
	b := filepath.Join(base, "b.md")
	_ = os.WriteFile(a, []byte("A"), 0o644)
	_ = os.WriteFile(b, []byte("B"), 0o644)

	if _, err := s.Rename(a, "b"); err == nil {
		t.Error("renombrar sobre un nombre existente debería fallar")
	}
	if data, _ := os.ReadFile(b); string(data) != "B" {
		t.Errorf("b.md fue sobrescrita: %q", data)
	}
	if _, err := s.Rename(a, "  ///  "); err == nil {
		t.Error("un nombre vacío tras limpiar debería fallar")
	}
}

func TestCreateRejectsExistingFolder(t *testing.T) {
	base := t.TempDir()
	s := New(base)
	if _, err := s.CreateFolderInDir(base, "dup"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateFolderInDir(base, "dup"); err == nil {
		t.Error("crear una carpeta que ya existe debería fallar")
	}
}
