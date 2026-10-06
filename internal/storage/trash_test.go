package storage

import (
	"os"
	"path/filepath"
	"strings"
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

// TestTrashDaysOption (ORD-025 O4): trash_days manda en la retención (por defecto 20); con 0 no hay papelera: MoveToTrash borra de inmediato y no deja nada en .trash.
func TestTrashDaysOption(t *testing.T) {
	old := TrashDays
	t.Cleanup(func() { TrashDays = old })
	if TrashDays != 20 {
		t.Fatalf("el defecto es 20 días: %d", TrashDays)
	}
	s := New(t.TempDir())
	note := func(name string) string {
		p := filepath.Join(s.BaseDir, name)
		os.WriteFile(p, []byte("x"), 0o644)
		return p
	}
	// 3 días: lo que lleva 4 se purga, lo que lleva 2 queda con 1 día restante
	TrashDays = 3
	for _, n := range []string{"viejo.md", "nuevo.md"} {
		if _, err := s.MoveToTrash(note(n)); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := s.readTrashMeta()
	for i := range items {
		age := 2
		if items[i].Name == "viejo.md" {
			age = 4
		}
		items[i].DeletedAt = time.Now().Add(-time.Duration(age) * 24 * time.Hour)
	}
	s.saveTrashMeta(items)
	got, _ := s.ListTrash()
	if len(got) != 1 || got[0].Name != "nuevo.md" || got[0].DaysRemaining() != 1 {
		t.Errorf("con 3 días solo queda lo de 2 días (1 restante): %+v", got)
	}
	// 0: sin papelera
	TrashDays = 0
	p := note("borrame.md")
	it, err := s.MoveToTrash(p)
	if err != nil || it == nil || it.Name != "borrame.md" {
		t.Fatalf("MoveToTrash con 0: %v %+v", err, it)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("con trash_days=0 el archivo se borra de verdad")
	}
	if entries, _ := os.ReadDir(filepath.Join(s.BaseDir, ".trash")); len(entries) > 1 { // solo el meta de antes (nuevo.md ya estaba)
		for _, e := range entries {
			if strings.Contains(e.Name(), "borrame") {
				t.Errorf("no deja copia en .trash: %s", e.Name())
			}
		}
	}
}
