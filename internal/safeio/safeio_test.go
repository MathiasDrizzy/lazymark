package safeio

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestReadRegular(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	os.WriteFile(p, []byte("hola"), 0o644)
	if b, err := ReadRegular(p, 10); err != nil || string(b) != "hola" {
		t.Fatalf("%q %v", b, err)
	}
	if _, err := ReadRegular(p, 3); !errors.Is(err, ErrTooLarge) {
		t.Errorf("sobre el tope: %v", err)
	}
	if b, err := ReadRegular(p, 0); err != nil || len(b) != 4 {
		t.Errorf("sin tope: %v", err)
	}
	if _, err := ReadRegular(dir, 10); !errors.Is(err, ErrNotRegular) {
		t.Errorf("una carpeta: %v", err)
	}
}

// TestReadRegularNeverBlocks: un FIFO, /dev/tty o /dev/zero se rechazan sin esperar (y /dev/zero no llena la memoria).
func TestReadRegularNeverBlocks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sin FIFO en Windows")
	}
	fifo := filepath.Join(t.TempDir(), "f.md")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skip(err)
	}
	for _, p := range []string{fifo, "/dev/tty", "/dev/zero"} {
		done := make(chan error, 1)
		go func() { _, err := ReadRegular(p, 1<<20); done <- err }()
		select {
		case err := <-done:
			if err == nil {
				t.Errorf("%s: debía rechazarse", p)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: se colgó", p)
		}
	}
}

// TestWriteFileAtomic (ORD-015 C.5 S5): reemplaza el archivo entero conservando sus permisos, no deja temporales y, si falla, deja el original.
func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	os.WriteFile(p, []byte("viejo"), 0o600)
	if err := WriteFileAtomic(p, []byte("nuevo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "nuevo" {
		t.Errorf("%q", b)
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
			t.Errorf("conserva los permisos del original: %v", fi.Mode().Perm())
		}
	}
	if es, _ := os.ReadDir(dir); len(es) != 1 {
		t.Errorf("no deben quedar temporales: %v", es)
	}
	if err := WriteFileAtomic(filepath.Join(dir, "no-existe", "x"), []byte("x"), 0o644); err == nil {
		t.Error("una carpeta que no existe es un error")
	}
	// un archivo nuevo usa perm
	q := filepath.Join(dir, "nuevo.json")
	if err := WriteFileAtomic(q, []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(q); string(b) != "n" {
		t.Errorf("%q", b)
	}
}
