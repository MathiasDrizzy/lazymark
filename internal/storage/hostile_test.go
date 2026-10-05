package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func within(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: se colgó (un archivo que no es regular se abrió)", what)
	}
}

// TestHostileNotesFolder (ORD-015 C.5 S2 y S3): un FIFO llamado x.md no cuelga el listado (ListNotes y ListTreeEntries lo ignoran), y una nota
// de más del tope (MaxNoteBytes) se lista sin leerla (TooLarge, sin contenido) en vez de gastar memoria.
func TestHostileNotesFolder(t *testing.T) {
	old := MaxNoteBytes
	MaxNoteBytes = 1 << 20
	t.Cleanup(func() { MaxNoteBytes = old })
	s, _ := kanbanNote(t, "# ok\n- [ ] tarea\n")
	if runtime.GOOS != "windows" {
		if err := syscall.Mkfifo(filepath.Join(s.BaseDir, "trampa.md"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	big := filepath.Join(s.BaseDir, "enorme.md")
	f, _ := os.Create(big)
	f.Truncate(MaxNoteBytes + 1)
	f.Close()
	var notes []Note
	within(t, "ListNotes", func() { notes, _ = s.ListNotes() })
	got := map[string]Note{}
	for _, n := range notes {
		got[filepath.Base(n.Path)] = n
	}
	if _, ok := got["trampa.md"]; ok {
		t.Error("el FIFO no es una nota")
	}
	if n, ok := got["enorme.md"]; !ok || !n.TooLarge || n.Content != "" || n.Size <= MaxNoteBytes {
		t.Errorf("la nota enorme se lista sin leerla: %+v", got["enorme.md"].TooLarge)
	}
	if n := got["n.md"]; strings.TrimSpace(n.Content) == "" || n.TooLarge {
		t.Errorf("la normal sí se lee: %+v", n)
	}
	var entries []NoteEntry
	within(t, "ListTreeEntries", func() { entries, _ = s.ListTreeEntries(map[string]bool{}) })
	for _, e := range entries {
		if e.Name == "trampa.md" {
			t.Error("el árbol tampoco muestra el FIFO")
		}
		if e.Name == "enorme.md" && (e.Note == nil || !e.Note.TooLarge) {
			t.Error("el árbol marca la nota enorme")
		}
	}
	// reescribir una nota enorme o un FIFO tampoco se intenta
	if _, err := s.ToggleTask(big, 1); err == nil {
		t.Error("no se edita una nota sobre el tope")
	}
}
