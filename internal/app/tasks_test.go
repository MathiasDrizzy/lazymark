package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// taskIndex devuelve la posición en la lista de tareas de la que contiene text.
func taskIndex(t *testing.T, m *AppModel, text string) int {
	t.Helper()
	for i, tk := range m.c.tasks {
		if strings.Contains(tk.Text, text) {
			return i
		}
	}
	t.Fatalf("no hay tarea %q en %d tareas", text, len(m.c.tasks))
	return -1
}

// TestToggleKeepsCursorOnTask (C12): al alternar, la nota se reescribe y pasa
// a ser la más reciente, así que sus tareas saltan al principio de la lista; el
// cursor tiene que seguir a la tarea alternada y no quedarse en el mismo índice.
func TestToggleKeepsCursorOnTask(t *testing.T) {
	m := newTestModel(t, 120, 35)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(m.c.store.BaseDir, "compras.md"), old, old); err != nil {
		t.Fatal(err)
	}
	m.afterChange()
	press(m, "2")
	i := taskIndex(t, m, "Pan")
	m.tasks.list.set(i, len(m.c.tasks))

	press(m, "space")

	cur := m.tasks.current()
	if cur == nil || !strings.Contains(cur.Text, "Pan") {
		got := "ninguna"
		if cur != nil {
			got = cur.Text
		}
		t.Fatalf("tras alternar el cursor quedó sobre %q (índice %d, antes %d), se esperaba 'Pan'", got, m.tasks.list.cursor, i)
	}
	if !cur.Done {
		t.Error("la tarea alternada debería estar hecha")
	}
	// y sigue siguiéndola al reabrirla
	press(m, "space")
	if cur := m.tasks.current(); cur == nil || !strings.Contains(cur.Text, "Pan") || cur.Done {
		t.Errorf("al reabrir, el cursor no siguió a 'Pan': %+v", cur)
	}
}

// TestToggleHiddenTaskClampsCursor: con las hechas ocultas, la tarea alternada
// desaparece de la lista y el cursor queda en una fila válida.
func TestToggleHiddenTaskClampsCursor(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "2", "H")
	n := len(m.c.tasks)
	m.tasks.list.set(n-1, n)
	press(m, "space")
	if len(m.c.tasks) != n-1 {
		t.Fatalf("la tarea hecha debería ocultarse: %d -> %d", n, len(m.c.tasks))
	}
	if m.tasks.current() == nil {
		t.Errorf("el cursor quedó fuera de la lista (cursor=%d, n=%d)", m.tasks.list.cursor, len(m.c.tasks))
	}
}

// TestToggleStaleNoteIsNotOverwritten (X10, C7): si otro programa cambió la
// nota después de cargarla, space no la pisa y avisa.
func TestToggleStaleNoteIsNotOverwritten(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "2")
	i := taskIndex(t, m, "Leche")
	m.tasks.list.set(i, len(m.c.tasks))
	path := filepath.Join(m.c.store.BaseDir, "compras.md")

	data, _ := os.ReadFile(path)
	external := append(data, []byte("\nescrito por otro editor\n")...)
	if err := os.WriteFile(path, external, 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	_ = os.Chtimes(path, later, later)

	press(m, "space")

	if got, _ := os.ReadFile(path); string(got) != string(external) {
		t.Fatalf("se pisó la edición externa:\n%s", got)
	}
	if !strings.Contains(m.c.status, "cambió por fuera") {
		t.Errorf("no avisó del cambio externo: %q", m.c.status)
	}
	// tras recargar, ahora sí alterna
	if tk := m.tasks.current(); tk == nil || tk.Done {
		t.Fatalf("estado inesperado tras el aviso: %+v", tk)
	}
	press(m, "space")
	if got, _ := os.ReadFile(path); !strings.Contains(string(got), "[x] Leche") || !strings.Contains(string(got), "escrito por otro editor") {
		t.Errorf("el segundo intento debía alternar conservando la edición externa:\n%s", got)
	}
}

// taskRowPos devuelve la celda (x, y) de la casilla y la del texto de la tarea.
func taskRowPos(m *AppModel, i int) (boxX, textX, y int) {
	_ = m.View() // el render fija el desplazamiento de la lista
	r := m.layout.Tasks
	return r.X + 2, r.X + 6, r.Y + 1 + i - m.tasks.list.offset
}

// TestClickOnCheckboxToggles (X7, C8): el clic en la casilla alterna la tarea;
// el clic en el texto solo la selecciona.
func TestClickOnCheckboxToggles(t *testing.T) {
	m := newTestModel(t, 120, 35)
	i := taskIndex(t, m, "Leche")
	// El panel muestra pocas filas y el orden de las notas depende de sus fechas:
	// el cursor se pone en una fila vecina para que la tarea quede visible.
	neighbor := i - 1
	if neighbor < 0 {
		neighbor = i + 1
	}
	m.tasks.list.set(neighbor, len(m.c.tasks))
	_ = m.View() // el render fija el desplazamiento de la lista
	_, textX, y := taskRowPos(m, i)

	click(m, textX, y)
	if m.focus != panelTasks || m.tasks.list.cursor != i {
		t.Fatalf("el clic en el texto no seleccionó: foco=%v cursor=%d (esperado %d)", m.focus, m.tasks.list.cursor, i)
	}
	if m.c.tasks[taskIndex(t, m, "Leche")].Done {
		t.Fatal("el clic en el texto alternó la tarea")
	}

	boxX, _, y := taskRowPos(m, i)
	click(m, boxX, y)
	data, _ := os.ReadFile(filepath.Join(m.c.store.BaseDir, "compras.md"))
	if !strings.Contains(string(data), "[x] Leche") {
		t.Fatalf("el clic en la casilla no marcó la tarea:\n%s", data)
	}
	if cur := m.tasks.current(); cur == nil || !strings.Contains(cur.Text, "Leche") || !cur.Done {
		t.Errorf("el cursor no siguió a la tarea alternada: %+v", cur)
	}

	// segundo clic en la casilla: la reabre (el orden de la lista cambió)
	j := taskIndex(t, m, "Leche")
	boxX, _, y = taskRowPos(m, j)
	click(m, boxX, y)
	data, _ = os.ReadFile(filepath.Join(m.c.store.BaseDir, "compras.md"))
	if !strings.Contains(string(data), "[ ] Leche") {
		t.Fatalf("el segundo clic en la casilla no la reabrió:\n%s", data)
	}
}

// previewText devuelve el texto plano del panel derecho.
func previewText(m *AppModel) string {
	r := m.layout.Preview
	var out []string
	for _, l := range screen(m)[r.Y : r.Y+r.H] {
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(l, r.X, r.X+r.W)), " "))
	}
	return strings.Join(out, "\n")
}

// TestTaskSelectionShowsNoteAtLine (H2-4, C5): con el cursor sobre una tarea, el
// preview muestra la nota (no una ficha de detalle) posicionada en la línea de
// la tarea; Enter pasa el foco al preview para seguir leyendo desde ahí.
func TestTaskSelectionShowsNoteAtLine(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "2")
	m.tasks.list.set(taskIndex(t, m, "TAREA_PROFUNDA_ZZ"), len(m.c.tasks))
	m.relayout()

	pv := previewText(m)
	for _, want := range []string{"TAREA_PROFUNDA_ZZ revisar la bitacora al final", "relleno 40", "relleno 41"} {
		if !strings.Contains(pv, want) {
			t.Errorf("el preview no muestra %q:\n%s", want, pv)
		}
	}
	for _, bad := range []string{"BITACORA_MARCA", "relleno 01", "Presiona Enter", "Press Enter", "PENDIENTE", "Línea:"} {
		if strings.Contains(pv, bad) {
			t.Errorf("el preview no debería mostrar %q (ni ficha de detalle ni el inicio de la nota):\n%s", bad, pv)
		}
	}

	press(m, "enter")
	if m.focus != panelPreview {
		t.Fatalf("Enter debería pasar el foco al preview, foco=%v", m.focus)
	}
	if pv := previewText(m); !strings.Contains(pv, "TAREA_PROFUNDA_ZZ") {
		t.Errorf("tras Enter el preview perdió la posición:\n%s", pv)
	}

	// la tarea del final de la nota: se ve con su contexto previo
	press(m, "2")
	m.tasks.list.set(taskIndex(t, m, "TAREA_LARGA_QQ"), len(m.c.tasks))
	m.relayout()
	pv = previewText(m)
	if !strings.Contains(pv, "TAREA_LARGA_QQ") || !strings.Contains(pv, "relleno 5") {
		t.Errorf("la tarea del final no se ve con su contexto:\n%s", pv)
	}

	// una tarea de otra nota cambia el preview a esa nota
	m.tasks.list.set(taskIndex(t, m, "Leche"), len(m.c.tasks))
	m.relayout()
	if pv := previewText(m); !strings.Contains(pv, "Leche") || strings.Contains(pv, "relleno") {
		t.Errorf("el preview no cambió a la nota de la tarea:\n%s", pv)
	}
}

// strikeCells devuelve, por fila de pantalla, las columnas con tachado activo.
func strikeCells(m *AppModel) map[int][]int {
	out := m.View().Content
	canvas := lipgloss.NewCanvas(m.w, m.h)
	canvas.Compose(lipgloss.NewLayer(out))
	cells := map[int][]int{}
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if c := canvas.CellAt(x, y); c != nil && c.Style.Attrs&uv.AttrStrikethrough != 0 {
				cells[y] = append(cells[y], x)
			}
		}
	}
	return cells
}

// TestStrikethroughStaysInRow (H2-5, C6): el tachado de las tareas hechas se
// abre y se cierra dentro de su fila; no sangra a otras filas, al borde del
// panel, al preview ni a los popups que se abren encima.
func TestStrikethroughStaysInRow(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "2")
	r := m.layout.Tasks
	// el cursor sobre una tarea hecha, para que haya alguna visible en el panel
	for i, tk := range m.c.tasks {
		if tk.Done {
			m.tasks.list.set(i, len(m.c.tasks))
			break
		}
	}
	_ = m.View() // el render fija el desplazamiento de la lista

	doneRows := map[int]bool{}
	for i, tk := range m.c.tasks {
		row := i - m.tasks.list.offset
		if tk.Done && row >= 0 && row < r.H-2 { // solo las filas visibles del panel
			doneRows[r.Y+1+row] = true
		}
	}
	if len(doneRows) == 0 {
		t.Fatal("la fixture no tiene tareas hechas")
	}
	cells := strikeCells(m)
	for y, xs := range cells {
		if !doneRows[y] {
			t.Errorf("fila %d tiene tachado y no es una tarea hecha", y)
		}
		for _, x := range xs {
			if x <= r.X || x >= r.X+r.W-1 {
				t.Errorf("tachado fuera del interior del panel, en (%d,%d)", x, y)
			}
		}
	}
	for y := range doneRows {
		if len(cells[y]) == 0 {
			t.Errorf("la tarea hecha de la fila %d no está tachada", y)
		}
	}

	for _, keys := range [][]string{{"?"}, {","}, {"x"}} {
		press(m, keys...)
		p := m.c.top()
		pr := popupRect(m.layout, p, p.render(m.layout))
		for y, xs := range strikeCells(m) {
			for _, x := range xs {
				if pr.Contains(x, y) {
					t.Errorf("tachado dentro del popup %T en (%d,%d)", p, x, y)
				}
			}
		}
		press(m, "esc")
	}
}

// TestTaskScope (H2-1, C2): por defecto el panel lista las tareas de todas las
// notas; el alcance puede ser un tag o una carpeta y se cambia desde Ajustes.
func TestTaskScope(t *testing.T) {
	m := newTestModel(t, 120, 35)
	if m.c.cfg.TaskScope != "all" {
		t.Fatalf("alcance por defecto = %q, se esperaba all", m.c.cfg.TaskScope)
	}
	all := len(m.c.tasks)
	if all < 7 {
		t.Fatalf("con el alcance por defecto debería haber tareas de todas las notas, hay %d", all)
	}

	only := func(scope string, wantPrefix string) {
		t.Helper()
		m.c.cfg.TaskScope = scope
		m.c.reload()
		if len(m.c.tasks) == 0 || len(m.c.tasks) >= all {
			t.Fatalf("%s: %d tareas (todas=%d)", scope, len(m.c.tasks), all)
		}
		for _, tk := range m.c.tasks {
			if !strings.Contains(tk.NotePath, wantPrefix) {
				t.Errorf("%s: la tarea %q viene de %s", scope, tk.Text, tk.NotePath)
			}
		}
	}
	only("tag:personal", "compras.md")
	only("folder:proyectos", string(filepath.Separator)+"proyectos"+string(filepath.Separator))

	// desde el popup de Ajustes se recorre: todas -> tag -> ... -> carpeta -> todas
	m.c.cfg.TaskScope = "all"
	m.c.reload()
	press(m, ",")
	p := m.c.top().(*settingsPopup)
	p.list.set(int(setTaskScope), p.n)
	press(m, "right")
	if got := m.c.cfg.TaskScope; got == "all" || len(m.c.tasks) >= all {
		t.Errorf("tras → el alcance sigue en %q con %d tareas", got, len(m.c.tasks))
	}
	press(m, "left")
	if m.c.cfg.TaskScope != "all" || len(m.c.tasks) != all {
		t.Errorf("tras ← no volvió a todas: %q con %d tareas", m.c.cfg.TaskScope, len(m.c.tasks))
	}
}
