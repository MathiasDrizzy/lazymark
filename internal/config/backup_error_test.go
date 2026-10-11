package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// failingBackup fails its write and then locks the folder, so the partial copy cannot be removed.
type failingBackup struct{ dir string }

var errDiskFull = errors.New("disk full")

func (f failingBackup) Write([]byte) (int, error) {
	_ = os.Chmod(f.dir, 0o500)
	return 0, errDiskFull
}
func (failingBackup) Sync() error  { return nil }
func (failingBackup) Close() error { return nil }

func TestWriteBackupReportsPartialCopyItCannotRemove(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission bits do not block removal here")
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	dest := filepath.Join(dir, "config.json.bak")
	if err := os.WriteFile(dest, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := writeBackup(failingBackup{dir}, dest, []byte("data"))
	if !errors.Is(err, errDiskFull) {
		t.Fatalf("the write error must be kept, got %v", err)
	}
	if !strings.Contains(err.Error(), dest) {
		t.Fatalf("the error must say which partial copy stays behind, got %v", err)
	}
}

func TestWriteBackupRemovesPartialCopy(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json.bak")
	if err := os.WriteFile(dest, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeBackup(failingBackup{t.TempDir()}, dest, []byte("data")); !errors.Is(err, errDiskFull) {
		t.Fatalf("got %v, want the write error", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("the partial copy must be removed, stat err %v", err)
	}
}
