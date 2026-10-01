package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTrashRetention: lo que lleva más de TrashRetentionDays en la papelera se
// purga del disco al listarla; lo más reciente se conserva con sus días restantes.
func TestTrashRetention(t *testing.T) {
	base := t.TempDir()
	s := New(base)
	for _, n := range []string{"vieja.md", "reciente.md"} {
		_ = os.WriteFile(filepath.Join(base, n), []byte(n), 0o644)
		if _, err := s.MoveToTrash(filepath.Join(base, n)); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := s.readTrashMeta()
	for i := range items {
		switch items[i].Name {
		case "vieja.md":
			items[i].DeletedAt = time.Now().Add(-(TrashRetentionDays + 1) * 24 * time.Hour)
		case "reciente.md":
			items[i].DeletedAt = time.Now().Add(-(TrashRetentionDays - 2) * 24 * time.Hour)
		}
	}
	if err := s.saveTrashMeta(items); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListTrash()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "reciente.md" {
		t.Fatalf("tras purgar quedaron %+v", got)
	}
	if d := got[0].DaysRemaining(); d != 2 {
		t.Errorf("DaysRemaining = %d, se esperaba 2", d)
	}
	for _, it := range items {
		_, err := os.Stat(filepath.Join(s.trashDir(), it.ID))
		if it.Name == "vieja.md" && !os.IsNotExist(err) {
			t.Error("el archivo vencido sigue en .trash")
		}
		if it.Name == "reciente.md" && err != nil {
			t.Error("el archivo reciente desapareció de .trash")
		}
	}
	if TrashRetentionDays != 20 {
		t.Errorf("TrashRetentionDays = %d, la SPEC H1-6 pide 20", TrashRetentionDays)
	}
}
