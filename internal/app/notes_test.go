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
