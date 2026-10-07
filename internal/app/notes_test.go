package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// TestDeleteAlwaysAsksWithoutTrash (ORD-025 O4): con trash_days = 0 (sin papelera) borrar es para siempre: pregunta aunque confirm_delete esté apagado, lo dice y borra de verdad;
// con papelera y confirm_delete apagado, borra sin preguntar (como siempre).
func TestDeleteAlwaysAsksWithoutTrash(t *testing.T) {
	old := storage.TrashDays
	t.Cleanup(func() { storage.TrashDays = old })
	for _, days := range []int{20, 0} {
		m := newTestModel(t, 120, 35)
		storage.TrashDays = days // después de crear el modelo: app.New lo toma de la configuración
		m.c.cfg.ConfirmDelete = false
		p := filepath.Join(m.c.store.BaseDir, "borrame.md")
		os.WriteFile(p, []byte("# x\n"), 0o644)
		m.afterChange()
		m.notes.selectPath(p)
		press(m, "d")
		asks := m.c.top() != nil
		if asks != (days == 0) {
			t.Errorf("trash_days=%d: pregunta=%v", days, asks)
		}
		if days == 0 {
			if !strings.Contains(plain(m), "forever") && !strings.Contains(plain(m), "para siempre") {
				t.Errorf("sin papelera la pregunta dice que es para siempre:\n%s", plain(m))
			}
			press(m, "y")
		}
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("trash_days=%d: la nota debe haber desaparecido de su sitio", days)
		}
		_, inTrash := os.Stat(filepath.Join(m.c.store.BaseDir, ".trash"))
		if days == 0 && inTrash == nil {
			if entries, _ := os.ReadDir(filepath.Join(m.c.store.BaseDir, ".trash")); len(entries) > 0 {
				t.Errorf("sin papelera no queda nada en .trash: %v", entries)
			}
		}
	}
}

// TestTasksSortOption (ORD-025 O7): el panel Tareas sigue el orden de las notas por defecto y, con tasks_sort = "due", pone primero las pendientes con vencimiento, la más vencida arriba.
func TestTasksSortOption(t *testing.T) {
	m := newTestModel(t, 120, 35)
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	p := filepath.Join(m.c.store.BaseDir, "t.md")
	os.WriteFile(p, []byte("# T\n- [ ] sin fecha\n- [ ] lejos 📅 2999-01-01\n- [ ] vencida 📅 2020-01-01\n"), 0o644)
	m.c.reload()
	order := func() string {
		var out []string
		for _, k := range m.c.tasks {
			out = append(out, strings.Fields(k.Task.Text)[0])
		}
		return strings.Join(out, ",")
	}
	if got := order(); got != "sin,lejos,vencida" {
		t.Errorf("por defecto, el orden de la nota: %s", got)
	}
	m.c.cfg.TasksSort = "due"
	m.c.reload()
	if got := order(); got != "vencida,lejos,sin" {
		t.Errorf("tasks_sort=due: %s", got)
	}
}

// TestNoTrashAlwaysAsksWithCounts (ORD-026 N1): con trash_days = 0 borrar SIEMPRE pide confirmación, también una carpeta vacía y una carpeta con archivos que no son notas (la
// que antes se borraba sin preguntar), y dice que es definitivo y cuántos archivos son; con papelera, una carpeta vacía se sigue moviendo sin preguntar.
func TestNoTrashAlwaysAsksWithCounts(t *testing.T) {
	old := storage.TrashDays
	t.Cleanup(func() { storage.TrashDays = old })
	mk := func(t *testing.T) (*AppModel, string) {
		m := newTestModel(t, 140, 40)
		m.c.cfg.ConfirmDelete = false // aun así debe preguntar sin papelera
		base := m.c.store.BaseDir
		os.MkdirAll(filepath.Join(base, "vacia"), 0o755)
		os.MkdirAll(filepath.Join(base, "fotos"), 0o755)
		os.WriteFile(filepath.Join(base, "fotos", "a.png"), []byte("x"), 0o644)
		os.WriteFile(filepath.Join(base, "fotos", "b.csv"), []byte("x"), 0o644)
		os.WriteFile(filepath.Join(base, "fotos", ".oculto"), []byte("x"), 0o644)
		m.afterChange()
		return m, base
	}
	for _, c := range []struct {
		folder string
		files  string
	}{{"vacia", "0 archivo(s)"}, {"fotos", "3 archivo(s)"}} {
		m, base := mk(t)
		storage.TrashDays = 0
		m.notes.selectPath(filepath.Join(base, c.folder))
		press(m, "d")
		if m.c.top() == nil {
			t.Fatalf("trash_days=0, carpeta %q: debe preguntar", c.folder)
		}
		out := plain(m)
		if !strings.Contains(out, "para siempre") || !strings.Contains(out, c.files) {
			t.Errorf("carpeta %q: la pregunta dice que es definitivo y cuántos archivos (%s):\n%s", c.folder, c.files, out)
		}
		if _, err := os.Stat(filepath.Join(base, c.folder)); err != nil {
			t.Errorf("carpeta %q: preguntar no borra todavía: %v", c.folder, err)
		}
		press(m, "n")
		if _, err := os.Stat(filepath.Join(base, c.folder)); err != nil {
			t.Errorf("carpeta %q: al decir que no, sigue ahí", c.folder)
		}
	}
	// con papelera: una carpeta vacía se mueve sin preguntar (como siempre)
	m, base := mk(t)
	storage.TrashDays = 20
	m.notes.selectPath(filepath.Join(base, "vacia"))
	press(m, "d")
	if m.c.top() != nil {
		t.Error("con papelera, una carpeta vacía se mueve sin preguntar")
	}
	if _, err := os.Stat(filepath.Join(base, "vacia")); !os.IsNotExist(err) {
		t.Error("y se movió a la papelera")
	}
}

// TestTrashDaysSettingDoesNotWrap (ORD-026 N2): en Ajustes ←/→ no da la vuelta en trash_days: de 365 → se queda en 365 y de 0 ← se queda en 0 (0 = sin papelera, no se llega "sin querer").
func TestTrashDaysSettingDoesNotWrap(t *testing.T) {
	if stepIntClamp(trashDaysPresets, 365, 1) != 365 || stepIntClamp(trashDaysPresets, 0, -1) != 0 {
		t.Errorf("no da la vuelta: 365→%d, 0←%d", stepIntClamp(trashDaysPresets, 365, 1), stepIntClamp(trashDaysPresets, 0, -1))
	}
	if stepIntClamp(trashDaysPresets, 20, 1) != 30 || stepIntClamp(trashDaysPresets, 20, -1) != 14 || stepIntClamp(trashDaysPresets, 1, -1) != 0 {
		t.Error("en el medio avanza y retrocede con normalidad")
	}
	m := newTestModel(t, 120, 40)
	sp := newSettingsPopup(m.c, func() {}, func() {})
	m.c.cfg.TrashDays = 365
	sp.change(setTrashDays, 1)
	if m.c.cfg.TrashDays != 365 {
		t.Errorf("la fila de Ajustes tampoco da la vuelta: %d", m.c.cfg.TrashDays)
	}
}
