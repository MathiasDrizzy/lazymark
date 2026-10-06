package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/safeio"
)

// TestRewriteLinesSyncsBeforeRename (ORD-019 C.6 / A1): el temporal se sincroniza a disco ANTES del rename (si se corta la luz entre medio, la nota no queda
// truncada): en el momento del Sync la nota todavía tiene su contenido viejo y el temporal ya tiene el nuevo completo.
func TestRewriteLinesSyncsBeforeRename(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "n.md")
	os.WriteFile(p, []byte("- [ ] a\n"), 0o644)
	calls := 0
	old := safeio.SyncFile
	t.Cleanup(func() { safeio.SyncFile = old })
	safeio.SyncFile = func(f *os.File) error {
		calls++
		if b, _ := os.ReadFile(p); string(b) != "- [ ] a\n" {
			t.Errorf("la nota ya cambió antes del Sync: %q", b)
		}
		if b, _ := os.ReadFile(f.Name()); !strings.HasPrefix(string(b), "- [x] a") || !strings.HasSuffix(string(b), "\n") {
			t.Errorf("el temporal debe estar completo al sincronizarse: %q", b)
		}
		return old(f)
	}
	if ok, err := New(dir).ToggleTaskIfUnchanged(p, 1, time.Time{}); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if calls != 1 {
		t.Errorf("Sync del temporal: %d veces, se esperaba 1", calls)
	}
	if b, _ := os.ReadFile(p); !strings.HasPrefix(string(b), "- [x] a") {
		t.Errorf("resultado: %q", b)
	}
}

// TestSafeioAtomicSyncs: WriteFileAtomic también sincroniza el temporal.
func TestSafeioAtomicSyncs(t *testing.T) {
	calls := 0
	old := safeio.SyncFile
	t.Cleanup(func() { safeio.SyncFile = old })
	safeio.SyncFile = func(f *os.File) error { calls++; return old(f) }
	if err := safeio.WriteFileAtomic(filepath.Join(t.TempDir(), "x.md"), []byte("x"), 0o644); err != nil || calls != 1 {
		t.Errorf("WriteFileAtomic: %v, Sync %d veces", err, calls)
	}
}
