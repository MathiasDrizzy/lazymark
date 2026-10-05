package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// datesRig abre el modelo con "hoy" fijo (lunes 2026-10-05), el panel Tareas enfocado y el cursor en la tarea "Pan" de compras.md.
func datesRig(t *testing.T) (*AppModel, string, int) {
	t.Helper()
	old := storage.Today
	storage.Today = func() string { return "2026-10-05" }
	t.Cleanup(func() { storage.Today = old })
	m := newTestModel(t, 120, 35)
	path := filepath.Join(m.c.store.BaseDir, "compras.md")
	line := 0
	for _, tk := range m.c.tasks {
		if tk.NotePath == path && tk.Text == "Pan" {
			line = tk.Line
		}
	}
	if line == 0 {
		t.Fatal("no está la tarea Pan")
	}
	press(m, "2")
	m.tasks.selectTask(path, line)
	return m, path, line
}

// TestDatesPopupSavesRelativeDates (ORD-015 C.2): con una tarea seleccionada, d abre el popup Fechas; se escribe "+1w" en Inicio y "viernes" en
// Vence, y antes de guardar se ve la fecha resuelta con su día; Enter escribe solo esa línea con el formato Obsidian Tasks (los emojis quedan
// en el archivo, no en la pantalla) y cierra el popup.
func TestDatesPopupSavesRelativeDates(t *testing.T) {
	m, path, line := datesRig(t)
	before := fileLines(t, path)
	press(m, "d")
	if _, ok := m.c.top().(*datesPopup); !ok {
		t.Fatalf("d debe abrir el popup de fechas: %T", m.c.top())
	}
	typeText(m, "+1w")
	press(m, "tab")
	typeText(m, "viernes")
	out := plain(m)
	for _, want := range []string{"Fechas", "Inicio", "Vence", "→ 2026-10-12 (lunes)", "→ 2026-10-09 (viernes)", "Pan"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en el popup:\n%s", want, out)
		}
	}
	if d := diffLines(before, fileLines(t, path)); len(d) != 0 {
		t.Fatalf("no se escribe nada hasta guardar: %v", d)
	}
	press(m, "enter")
	if m.c.top() != nil {
		t.Fatalf("Enter guarda y cierra: %T", m.c.top())
	}
	after := fileLines(t, path)
	if d := diffLines(before, after); len(d) != 1 || d[0] != line || after[line-1] != "- [ ] Pan 🛫 2026-10-12 📅 2026-10-09" {
		t.Fatalf("debía cambiar solo la línea %d con el formato Obsidian Tasks: %v\n%q", line, d, after)
	}
	if !strings.Contains(lastRow(m), "Fechas guardadas") {
		t.Errorf("aviso: %q", lastRow(m))
	}
	// en pantalla (tareas y tablero) no hay emojis a color: glifos
	press(m, "W")
	if scr := plain(m); strings.ContainsAny(scr, "🛫📅✅") || !strings.Contains(scr, "2026-10-09") {
		t.Errorf("el tablero muestra las fechas con glifos, sin emojis:\n%s", scr)
	}
}

// TestDatesPopupRemovesAndCancels (ORD-015 C.2): vacío quita la fecha; Esc cancela sin escribir; sin cambios no escribe.
func TestDatesPopupRemovesAndCancels(t *testing.T) {
	m, path, line := datesRig(t)
	os.WriteFile(path, []byte(strings.Replace(string(mustRead(t, path)), "- [ ] Pan", "- [ ] Pan 🛫 2026-05-01 📅 2026-05-10", 1)), 0o644)
	m.c.reload()
	m.afterChange()
	m.tasks.selectTask(path, line)
	// Esc: nada cambia
	before := fileLines(t, path)
	press(m, "d")
	if out := plain(m); !strings.Contains(out, "2026-05-01") || !strings.Contains(out, "2026-05-10") {
		t.Fatalf("el popup parte de las fechas de la tarea:\n%s", out)
	}
	typeText(m, "xx")
	press(m, "esc")
	if m.c.top() != nil || len(diffLines(before, fileLines(t, path))) != 0 {
		t.Fatal("Esc cancela sin escribir")
	}
	// Enter sin cambios: cierra sin escribir
	st, _ := os.Stat(path)
	press(m, "d", "enter")
	if st2, _ := os.Stat(path); m.c.top() != nil || !st2.ModTime().Equal(st.ModTime()) {
		t.Fatal("Enter sin cambios no debe escribir")
	}
	// vaciar el inicio quita solo el inicio
	press(m, "d", "ctrl+u", "enter")
	if got := fileLines(t, path)[line-1]; got != "- [ ] Pan 📅 2026-05-10" {
		t.Errorf("quitar el inicio: %q", got)
	}
	// vaciar el vencimiento también
	m.tasks.selectTask(path, line)
	press(m, "d", "tab", "ctrl+u", "enter")
	if got := fileLines(t, path)[line-1]; got != "- [ ] Pan" {
		t.Errorf("quitar el vencimiento: %q", got)
	}
}

// TestDatesPopupInvalidInput (ORD-015 C.2): una fecha que no se entiende se marca al lado del campo, Enter no cierra ni escribe y avisa cuál;
// al corregirla guarda.
func TestDatesPopupInvalidInput(t *testing.T) {
	m, path, line := datesRig(t)
	before := fileLines(t, path)
	press(m, "d")
	typeText(m, "ayer")
	if out := plain(m); !strings.Contains(out, "✗") {
		t.Errorf("la fecha inválida se marca mientras se escribe:\n%s", out)
	}
	press(m, "enter")
	if _, ok := m.c.top().(*datesPopup); !ok {
		t.Fatal("con una fecha inválida el popup sigue abierto")
	}
	if out := plain(m); !strings.Contains(out, "ayer") || !strings.Contains(out, "No entiendo") {
		t.Errorf("debe decir qué no entendió:\n%s", out)
	}
	if len(diffLines(before, fileLines(t, path))) != 0 {
		t.Fatal("una fecha inválida no escribe nada")
	}
	press(m, "ctrl+u")
	typeText(m, "mañana")
	press(m, "enter")
	if got := fileLines(t, path)[line-1]; got != "- [ ] Pan 🛫 2026-10-06" || m.c.top() != nil {
		t.Errorf("corregida guarda: %q (popup %T)", got, m.c.top())
	}
}

// TestDatesPopupFromKanbanAndKeyIsVisible (ORD-015 C.2): en el Kanban d abre el popup de la tarjeta y la tecla está en la barra y en el
// cheatsheet (también en el de Tareas).
func TestDatesPopupFromKanbanAndKeyIsVisible(t *testing.T) {
	m, path, line := datesRig(t)
	press(m, "W")
	if !strings.Contains(lastRow(m), "Fechas (d)") {
		t.Errorf("la barra del Kanban muestra la tecla: %q", lastRow(m))
	}
	for i, c := range m.c.board.ColumnCards(0) {
		if c.NotePath == path && c.Task.Line == line {
			m.kanban.selected[0] = i
		}
	}
	press(m, "d")
	if _, ok := m.c.top().(*datesPopup); !ok {
		t.Fatalf("d en el Kanban abre el popup: %T", m.c.top())
	}
	typeText(m, "+2d")
	press(m, "enter")
	if got := fileLines(t, path)[line-1]; got != "- [ ] Pan 🛫 2026-10-07" {
		t.Errorf("%q", got)
	}
	if c := m.kanban.current(); c == nil || c.CleanText != "Pan" {
		t.Errorf("el cursor sigue a la tarjeta tras guardar: %+v", c)
	}
	press(m, "W", "2", "?")
	if out := plain(m); !strings.Contains(out, "Fechas") || !strings.Contains(out, "d ") {
		t.Errorf("el cheatsheet de Tareas lista la tecla:\n%s", out)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestDatesPopupPreviewMatchesWhatIsWritten (ORD-016 L2): la fecha que se ve resuelta y la que se escribe coinciden en cualquier zona horaria y aunque
// pase la medianoche entre que se abre el popup y se guarda: "hoy" se fija al abrirlo; el día de la semana mostrado no depende de la zona.
func TestDatesPopupPreviewMatchesWhatIsWritten(t *testing.T) {
	for _, tz := range []string{"UTC", "Pacific/Kiritimati", "Pacific/Pago_Pago", "America/Sao_Paulo", "Asia/Kolkata", "Pacific/Apia"} {
		t.Run(tz, func(t *testing.T) {
			loc, err := time.LoadLocation(tz)
			if err != nil {
				t.Skip(err)
			}
			old := time.Local
			time.Local = loc
			t.Cleanup(func() { time.Local = old })
			m, path, line := datesRig(t) // hoy fijo: 2026-10-05 (lunes)
			press(m, "d")
			typeText(m, "+1d")
			out := plain(m)
			if !strings.Contains(out, "→ 2026-10-06 (martes)") {
				t.Fatalf("[%s] la vista previa de +1d (martes):\n%s", tz, out)
			}
			// pasa la medianoche con el popup abierto
			storage.Today = func() string { return "2026-10-06" }
			press(m, "enter")
			if got := fileLines(t, path)[line-1]; got != "- [ ] Pan 🛫 2026-10-06" {
				t.Errorf("[%s] se escribió algo distinto de lo que se vio (2026-10-06): %q", tz, got)
			}
		})
	}
}
