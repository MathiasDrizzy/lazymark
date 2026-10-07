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

// TestChangingTrashDaysNeverPurgesWhatIsThere (ORD-026 N2, repro de cerebro): cambiar trash_days no borra lo que ya está en la papelera: con 20 hay 1; con 0 (los borrados nuevos
// no pasan por la papelera) sigue habiendo 1; al volver a 20, 1; bajar a 3 tampoco lo purga (caduca con los 20 días con que se borró); y lo borrado con 3 días caduca a los 3.
func TestChangingTrashDaysNeverPurgesWhatIsThere(t *testing.T) {
	old := TrashDays
	t.Cleanup(func() { TrashDays = old })
	s := New(t.TempDir())
	note := func(name string) string {
		p := filepath.Join(s.BaseDir, name)
		os.WriteFile(p, []byte("x"), 0o644)
		return p
	}
	TrashDays = 20
	if _, err := s.MoveToTrash(note("viejo.md")); err != nil {
		t.Fatal(err)
	}
	// hace 10 días
	items, _ := s.readTrashMeta()
	items[0].DeletedAt = time.Now().Add(-10 * 24 * time.Hour)
	s.saveTrashMeta(items)
	count := func(where string, want int) {
		t.Helper()
		if got, _ := s.ListTrash(); len(got) != want {
			t.Errorf("%s: %d en la papelera, se esperaba %d", where, len(got), want)
		}
	}
	count("con trash_days=20", 1)
	TrashDays = 0
	count("con trash_days=0 al listar", 1)
	TrashDays = 20
	count("volviendo a 20", 1)
	TrashDays = 3
	count("bajando a 3 (lo que ya estaba conserva sus 20 días)", 1)
	got, _ := s.ListTrash()
	if got[0].DaysRemaining() != 10 {
		t.Errorf("le quedan 10 días (20 - 10): %d", got[0].DaysRemaining())
	}
	// lo borrado con 3 días caduca a los 3
	if _, err := s.MoveToTrash(note("nuevo.md")); err != nil {
		t.Fatal(err)
	}
	items, _ = s.readTrashMeta()
	for i := range items {
		if items[i].Name == "nuevo.md" {
			if items[i].RetentionDays != 3 {
				t.Errorf("el elemento guarda la retención con que se borró: %d", items[i].RetentionDays)
			}
			items[i].DeletedAt = time.Now().Add(-4 * 24 * time.Hour)
		}
	}
	s.saveTrashMeta(items)
	count("lo de 3 días con 4 de antigüedad caduca; el viejo no", 1)
	// una papelera de antes de este campo (sin retention_days) vale 20 días
	legacy := []byte(`[{"id":"1_a.md","name":"a.md","original_path":"/x/a.md","deleted_at":"` + time.Now().Add(-19*24*time.Hour).Format(time.RFC3339) + `","is_dir":false,"size":1}]`)
	os.MkdirAll(filepath.Join(s.BaseDir, ".trash", "1_a.md"), 0o755)
	os.WriteFile(filepath.Join(s.BaseDir, ".trash", "trash.json"), legacy, 0o644)
	TrashDays = 0
	if got, _ := s.ListTrash(); len(got) != 1 || got[0].DaysRemaining() != 1 {
		t.Errorf("un elemento de antes (sin retention_days) vale 20 días, con trash_days=0 también: %+v", got)
	}
}

// TestCountFiles (ORD-026 N1): cuenta todos los archivos, no solo las notas, y los ocultos; un archivo vale 1.
func TestCountFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755)
	for _, f := range []string{"a/x.md", "a/foto.png", "a/b/datos.csv", "a/.oculto", "a/b/z.md"} {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}
	if got := CountFiles(filepath.Join(dir, "a")); got != 5 {
		t.Errorf("5 archivos (no solo notas): %d", got)
	}
	if got := CountFiles(filepath.Join(dir, "a", "x.md")); got != 1 {
		t.Errorf("un archivo vale 1: %d", got)
	}
	os.MkdirAll(filepath.Join(dir, "vacia"), 0o755)
	if got := CountFiles(filepath.Join(dir, "vacia")); got != 0 {
		t.Errorf("carpeta vacía: %d", got)
	}
}
