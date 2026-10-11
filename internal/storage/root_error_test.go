package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListNotesReportsUnreadableRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := New(missing).ListNotes(); err == nil {
		t.Fatal("ListNotes on a missing notes folder returned no error")
	}
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission bits do not block reads here")
	}
	locked := t.TempDir()
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if _, err := New(locked).ListNotes(); err == nil {
		t.Fatal("ListNotes on an unreadable notes folder returned no error")
	}
}

func TestListNotesSkipsUnreadableSubfolder(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission bits do not block reads here")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "locked")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o700) })
	notes, err := New(root).ListNotes()
	if err != nil || len(notes) != 1 {
		t.Fatalf("got %d notes, err %v; want the readable note and no error", len(notes), err)
	}
}
