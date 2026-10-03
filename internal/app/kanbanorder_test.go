package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// orderModel abre el Kanban con dos notas: a.md con tareas "por hacer" (una con subtarea) y una en progreso de por medio, y b.md.
func orderModel(t *testing.T) (*AppModel, string, string) {
	t.Helper()
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	a, b := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	os.WriteFile(a, []byte("# A\n- [ ] uno\n  - [ ] uno-hijo\n- [ ] medio #kb/doing\n- [ ] dos\n- [ ] tres\n"), 0o644)
	os.WriteFile(b, []byte("# B\n- [ ] cuatro\n"), 0o644)
	old := time.Now().Add(-48 * time.Hour) // el tablero ordena las notas de la más reciente a la más antigua: a.md primero
	os.Chtimes(b, old, old)
	m.c.reload()
	m.afterChange()
	press(m, "W")
	return m, a, b
}

func todoTitles(m *AppModel) []string {
	var out []string
	for _, c := range m.c.board.ColumnCards(0) {
		out = append(out, c.NotePath[strings.LastIndex(c.NotePath, string(filepath.Separator))+1:]+":"+c.CleanText)
	}
	return out
}

// TestKanbanReorderWithKeys (C.4): J/K (y Shift+↑/↓) suben y bajan la tarjeta en su columna intercambiando su tarea con la vecina de
// la misma nota; el cursor la sigue; lo de en medio y el resto del archivo no se mueven; entre notas distintas no hace nada y avisa.
func TestKanbanReorderWithKeys(t *testing.T) {
	m, a, _ := orderModel(t)
	want := func(t *testing.T, got []string, exp ...string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(exp, ",") {
			t.Fatalf("orden de la columna:\n got %v\nwant %v", got, exp)
		}
	}
	want(t, todoTitles(m), "a.md:uno", "a.md:uno-hijo", "a.md:dos", "a.md:tres", "b.md:cuatro")
	// "uno" tiene una subtarea con otra sangría: bajar "uno" lo cambia por "uno-hijo" y no se puede
	m.kanban.selected[0] = 0
	press(m, "J")
	if got := fileLines(t, a); strings.Join(got, "|") != "# A|- [ ] uno|  - [ ] uno-hijo|- [ ] medio #kb/doing|- [ ] dos|- [ ] tres|" {
		t.Errorf("uno con su hijo no se intercambian: %q", got)
	}
	if !strings.Contains(lastRow(m), "hermanas") {
		t.Errorf("debe avisar que solo se intercambian hermanas: %q", lastRow(m))
	}
	// "dos" sube sobre "uno": uno (con su hijo) baja detrás de dos, "medio" no se mueve
	m.kanban.selected[0] = 2
	press(m, "shift+up") // dos ↔ uno-hijo: sangrías distintas
	if !strings.Contains(lastRow(m), "hermanas") {
		t.Errorf("dos y uno-hijo: %q", lastRow(m))
	}
	m.kanban.selected[0] = 3 // tres
	press(m, "K")
	want(t, todoTitles(m), "a.md:uno", "a.md:uno-hijo", "a.md:tres", "a.md:dos", "b.md:cuatro")
	if got := m.kanban.current(); got == nil || got.CleanText != "tres" || m.kanban.selected[0] != 2 {
		t.Errorf("el cursor sigue a la tarjeta movida: %+v idx=%d", got, m.kanban.selected[0])
	}
	if got := fileLines(t, a); strings.Join(got, "|") != "# A|- [ ] uno|  - [ ] uno-hijo|- [ ] medio #kb/doing|- [ ] tres|- [ ] dos|" {
		t.Errorf("solo se intercambian las dos líneas:\n%q", got)
	}
	// bajar la última de a.md sobre una de otra nota: no hace nada y lo dice
	m.kanban.selected[0] = 3 // dos, la última de a.md
	before := fileLines(t, a)
	press(m, "J")
	if len(diffLines(before, fileLines(t, a))) != 0 || !strings.Contains(lastRow(m), "notas") {
		t.Errorf("entre notas no se reordena: %q", lastRow(m))
	}
	// los extremos no hacen nada ni fallan
	m.kanban.selected[0] = 0
	press(m, "K")
	press(m, "shift+down") // uno ↔ uno-hijo: no
	if m.kanban.col != 0 {
		t.Errorf("la columna no cambia: %d", m.kanban.col)
	}
}

// TestKanbanReorderByDragging (C.4): arrastrar una tarjeta sobre otra de su columna (arrastre vertical) la reordena, un lugar a la vez
// hasta donde se pueda; durante el arrastre el archivo no cambia y la tarjeta de destino se marca.
func TestKanbanReorderByDragging(t *testing.T) {
	m, a, _ := orderModel(t)
	m.kanban.selected[0] = 2 // dos
	_, y2, ok2 := cellOf(m, "dos")
	x3, y3, ok3 := cellOf(m, "tres")
	if !ok2 || !ok3 {
		t.Fatalf("no se ven las tarjetas:\n%s", plain(m))
	}
	before := fileLines(t, a)
	m.Update(tea.MouseClickMsg{X: x3 + 2, Y: y2, Button: tea.MouseLeft})
	if m.kanban.press == nil {
		t.Fatal("el clic debe apretar sobre la tarjeta")
	}
	m.Update(tea.MouseMotionMsg{X: x3 + 2, Y: y3, Button: tea.MouseLeft})
	d := m.kanban.drag
	if !d.Active || d.Target != 0 || d.TargetIdx != 3 || d.Idx != 2 {
		t.Fatalf("el arrastre vertical debía apuntar a la tarjeta 3 de la columna 0: %+v", d)
	}
	if len(diffLines(before, fileLines(t, a))) != 0 {
		t.Error("el archivo no debe cambiar hasta soltar")
	}
	m.Update(tea.MouseReleaseMsg{X: x3 + 2, Y: y3, Button: tea.MouseLeft})
	if got := fileLines(t, a); strings.Join(got, "|") != "# A|- [ ] uno|  - [ ] uno-hijo|- [ ] medio #kb/doing|- [ ] tres|- [ ] dos|" {
		t.Fatalf("tras arrastrar dos sobre tres:\n%q", got)
	}
	if c := m.kanban.current(); c == nil || c.CleanText != "dos" {
		t.Errorf("el cursor queda en la tarjeta movida: %+v", c)
	}
	// soltar sobre sí misma o en un hueco no escribe
	snapshot := fileLines(t, a)
	x, y, _ := cellOf(m, "dos")
	m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	if len(diffLines(snapshot, fileLines(t, a))) != 0 {
		t.Error("soltar sobre la misma tarjeta no debe escribir")
	}
}
