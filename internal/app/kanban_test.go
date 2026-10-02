package app

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// cellOf devuelve la celda (x, y) donde empieza sub en la pantalla; ok=false si no está.
func cellOf(m *AppModel, sub string) (x, y int, ok bool) {
	for y, line := range screen(m) {
		plainLine := ansi.Strip(line)
		if i := strings.Index(plainLine, sub); i >= 0 {
			return ansi.StringWidth(plainLine[:i]), y, true
		}
	}
	return 0, 0, false
}

func lastRow(m *AppModel) string {
	lines := screen(m)
	return ansi.Strip(lines[len(lines)-1])
}

// TestKanbanActionsAreVisible (K2): nada de teclas escondidas. En notas se ve el
// botón "Kanban (W)"; en el Kanban, "← Notas (Esc)" arriba y las 4 acciones de la
// tarjeta con su tecla abajo, completas a 120x35 y en su forma corta a 80x24.
func TestKanbanActionsAreVisible(t *testing.T) {
	long := []string{"Mover ← (H)", "Mover → (L)", "Listo/Por hacer (Espacio)", "Editar (Enter)"}
	short := []string{"← (H)", "→ (L)", "Listo (Espacio)", "Editar (Enter)"}
	for _, sz := range []struct {
		w, h int
		want []string
	}{{120, 35, long}, {100, 30, long}, {80, 24, short}} {
		m := newTestModel(t, sz.w, sz.h)
		if !strings.Contains(lastRow(m), "Kanban (W)") {
			t.Errorf("%dx%d notas: falta el botón Kanban (W) en la barra: %q", sz.w, sz.h, lastRow(m))
		}
		press(m, "W")
		if !m.kanbanOn {
			t.Fatal("W no abrió el Kanban")
		}
		if row0 := ansi.Strip(screen(m)[0]); !strings.Contains(row0, "← Notas (Esc)") {
			t.Errorf("%dx%d Kanban: falta el botón de volver arriba: %q", sz.w, sz.h, row0)
		}
		for _, label := range sz.want {
			if !strings.Contains(lastRow(m), label) {
				t.Errorf("%dx%d Kanban: la barra no muestra %q: %q", sz.w, sz.h, label, lastRow(m))
			}
		}
	}
}

// TestKanbanButtonsClick (K3): el mouse abre el Kanban con "Kanban (W)" y vuelve con "← Notas (Esc)".
func TestKanbanButtonsClick(t *testing.T) {
	m := newTestModel(t, 120, 35)
	x, y, ok := cellOf(m, "Kanban (W)")
	if !ok {
		t.Fatal("no se ve Kanban (W)")
	}
	click(m, x+2, y)
	if !m.kanbanOn {
		t.Fatal("el clic en Kanban (W) no abrió el Kanban")
	}
	x, y, ok = cellOf(m, "← Notas (Esc)")
	if !ok {
		t.Fatal("no se ve ← Notas (Esc)")
	}
	click(m, x+3, y)
	if m.kanbanOn {
		t.Error("el clic en ← Notas (Esc) no volvió a las notas")
	}
}

// TestKanbanShiftArrowsMoveCard (K4): Shift+→ y Shift+← mueven la tarjeta como L y H, el
// aviso dice la columna de destino y en la nota cambia solo esa línea.
func TestKanbanShiftArrowsMoveCard(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "W")
	card := m.c.board.Todo[0]
	before, _ := os.ReadFile(card.NotePath)

	press(m, "shift+right")
	if got := len(m.c.board.Doing); got != 1 || m.c.board.Doing[0].Task.Line != card.Task.Line || m.c.board.Doing[0].NotePath != card.NotePath {
		t.Fatalf("Shift+→ no pasó la tarjeta a En progreso: doing=%d", got)
	}
	if !strings.Contains(lastRow(m), "→ En progreso") {
		t.Errorf("falta el aviso de destino: %q", lastRow(m))
	}
	after, _ := os.ReadFile(card.NotePath)
	bl, al := strings.Split(string(before), "\n"), strings.Split(string(after), "\n")
	if len(bl) != len(al) {
		t.Fatalf("cambió el número de líneas: %d → %d", len(bl), len(al))
	}
	changed := 0
	for i := range bl {
		if bl[i] != al[i] {
			changed++
			if i != card.Task.Line-1 && i != card.Task.Line {
				t.Errorf("cambió la línea %d, que no es la de la tarea (%d)", i, card.Task.Line)
			}
		}
	}
	if changed != 1 {
		t.Errorf("deben cambiar solo 1 línea, cambiaron %d", changed)
	}

	press(m, "shift+left")
	if len(m.c.board.Doing) != 0 || !strings.Contains(lastRow(m), "→ Por hacer") {
		t.Errorf("Shift+← no devolvió la tarjeta a Por hacer: doing=%d, barra %q", len(m.c.board.Doing), lastRow(m))
	}
	press(m, "L")
	if len(m.c.board.Doing) != 1 {
		t.Error("L debe seguir moviendo la tarjeta")
	}
	press(m, "H")
	if len(m.c.board.Doing) != 0 {
		t.Error("H debe seguir moviendo la tarjeta")
	}
}

// TestKanbanCheatsheetHasKanbanSection (K5): ? dentro del Kanban muestra la sección Kanban.
func TestKanbanCheatsheetHasKanbanSection(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "W", "?")
	screenText := plain(m)
	for _, want := range []string{"Atajos · Kanban", "Mover a la izquierda", "shift+←", "Volver a notas"} {
		if !strings.Contains(screenText, want) {
			t.Errorf("el cheatsheet del Kanban no muestra %q:\n%s", want, screenText)
		}
	}
	press(m, "esc", "esc")
	press(m, "?")
	if strings.Contains(plain(m), "Atajos · Kanban") {
		t.Error("el cheatsheet de la vista de notas no debe decir Kanban en el título")
	}
}

// TestKanbanToastFitsAt80 (K4): a 80 columnas el aviso de destino también se lee completo,
// sin quitar ninguno de los 4 botones.
func TestKanbanToastFitsAt80(t *testing.T) {
	m := newTestModel(t, 80, 24)
	press(m, "W", "shift+right")
	row := lastRow(m)
	for _, want := range []string{"→ En progreso", "← (H)", "→ (L)", "Listo (Espacio)", "Editar (Enter)"} {
		if !strings.Contains(row, want) {
			t.Errorf("la barra a 80 columnas no muestra %q: %q", want, row)
		}
	}
}
