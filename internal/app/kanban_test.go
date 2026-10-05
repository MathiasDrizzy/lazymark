package app

import (
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
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
	}{{120, 35, long}, {100, 30, short}, {80, 24, short}} {
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
		if !strings.Contains(lastRow(m), "Notas (W)") {
			t.Errorf("%dx%d Kanban: falta el botón Notas (W) en la barra: %q", sz.w, sz.h, lastRow(m))
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
	card := m.c.board.ColumnCards(0)[0]
	before, _ := os.ReadFile(card.NotePath)

	press(m, "shift+right")
	if got := len(m.c.board.ColumnCards(1)); got != 1 || m.c.board.ColumnCards(1)[0].Task.Line != card.Task.Line || m.c.board.ColumnCards(1)[0].NotePath != card.NotePath {
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
	if len(m.c.board.ColumnCards(1)) != 0 || !strings.Contains(lastRow(m), "→ Por hacer") {
		t.Errorf("Shift+← no devolvió la tarjeta a Por hacer: doing=%d, barra %q", len(m.c.board.ColumnCards(1)), lastRow(m))
	}
	press(m, "L")
	if len(m.c.board.ColumnCards(1)) != 1 {
		t.Error("L debe seguir moviendo la tarjeta")
	}
	press(m, "H")
	if len(m.c.board.ColumnCards(1)) != 0 {
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

// TestKanbanFooterBackButton (K6): la barra inferior del Kanban tiene el botón "Notas (W)", igual
// que "Kanban (W)" en notas (mismo estilo, mismo lugar entre los globales, clickeable), y vuelve a
// las notas; el botón de arriba, "← Notas (Esc)", sigue funcionando.
func TestKanbanFooterBackButton(t *testing.T) {
	m := newTestModel(t, 120, 35)
	xNotes, y, _ := cellOf(m, "Kanban (W)")
	press(m, "W")
	x, y2, ok := cellOf(m, "Notas (W)")
	if !ok || y2 != y {
		t.Fatalf("no se ve Notas (W) en la última fila: %v (%d,%d)", ok, x, y2)
	}
	if x < xNotes {
		t.Errorf("Notas (W) (x=%d) debe ir tan a la derecha como Kanban (W) (x=%d) o más: son el primer global tras las acciones", x, xNotes)
	}
	click(m, x+3, y2)
	if m.kanbanOn {
		t.Error("el clic en Notas (W) no volvió a las notas")
	}
	press(m, "W")
	bx, by, _ := cellOf(m, "← Notas (Esc)")
	click(m, bx+3, by)
	if m.kanbanOn {
		t.Error("el botón de arriba dejó de funcionar")
	}
	// mismo estilo: el texto de ambos botones usa el estilo de botón del pie
	press(m, "W")
	if !strings.Contains(m.View().Content, theme.FooterKey.Render("Notas (W)")) {
		t.Error("Notas (W) no usa el estilo de botón del pie")
	}
}

// TestKanbanCursorFollowsCard (auditoría de K4): tras mover una tarjeta (Shift+←/→, H/L o Espacio) el
// cursor queda sobre ESA tarjeta en su columna nueva, no en la última de la columna.
func TestKanbanCursorFollowsCard(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "W")
	if len(m.c.board.ColumnCards(0)) < 3 {
		t.Fatalf("la fixture necesita al menos 3 tareas por hacer, tiene %d", len(m.c.board.ColumnCards(0)))
	}
	same := func(a, b views.KanbanCard) bool { return a.NotePath == b.NotePath && a.Task.Line == b.Task.Line }
	card := *m.kanban.current()
	for step, key := range []string{"shift+right", "shift+left", "L", "H", "space", "space"} {
		press(m, key)
		cur := m.kanban.current()
		if cur == nil || !same(*cur, card) {
			t.Fatalf("paso %d (%s): el cursor está en %+v, se esperaba la tarjeta movida %q", step, key, cur, card.CleanText)
		}
	}
}

// fileLines devuelve las líneas de un archivo.
func fileLines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(b), "\n")
}

// diffLines devuelve los números (desde 1) de las líneas que difieren entre dos versiones.
func diffLines(a, b []string) []int {
	var out []int
	for i := 0; i < max(len(a), len(b)); i++ {
		var x, y string
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			out = append(out, i+1)
		}
	}
	return out
}

// TestKanbanConfiguredColumns (H4-1): las columnas salen de la config (con título libre o por idioma), el tablero
// las muestra y mover una tarjeta escribe el tag de la columna en su línea.
func TestKanbanConfiguredColumns(t *testing.T) {
	m := newTestModel(t, 140, 35)
	m.c.cfg.KanbanColumns = []config.KanbanColumn{
		{ID: "backlog", Title: "Pendientes"},
		{ID: "doing", Titles: map[string]string{"es": "En curso", "en": "Doing"}},
		{ID: "review"},
		{ID: "done"},
	}
	m.c.reload()
	press(m, "W")
	out := plain(m)
	for _, want := range []string{"[1] Pendientes", "[2] En Curso", "[3] Review", "[4] Completado"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta la cabecera %q:\n%s", want, out)
		}
	}
	if m.c.board.NumCols() != 4 {
		t.Fatalf("columnas = %d", m.c.board.NumCols())
	}
	card := *m.kanban.current()
	before := fileLines(t, card.NotePath)
	press(m, "L", "L")
	if m.kanban.col != 2 {
		t.Fatalf("la tarjeta debía quedar en la columna 3, está en %d", m.kanban.col)
	}
	after := fileLines(t, card.NotePath)
	if d := diffLines(before, after); len(d) != 1 || !strings.HasSuffix(after[d[0]-1], " #kb/review") {
		t.Errorf("debía cambiar 1 línea terminada en #kb/review: %v → %q", d, after[max(0, d[0]-1)])
	}
	press(m, "space") // a hecho
	if got := m.c.board.ColumnCards(3); len(got) == 0 || got[0].Task.Line != card.Task.Line && !hasCard(got, card) {
		t.Errorf("Espacio debía llevarla a la columna de hecho")
	}
	if strings.Contains(strings.Join(fileLines(t, card.NotePath), "\n"), "#kb/review") {
		t.Error("al pasar a hecho no debe quedar el tag de la columna anterior")
	}
	if !strings.Contains(lastRow(m), "→ Completado") {
		t.Errorf("el aviso debía decir la columna: %q", lastRow(m))
	}
}

func hasCard(cards []views.KanbanCard, c views.KanbanCard) bool {
	for _, x := range cards {
		if x.NotePath == c.NotePath && x.Task.Line == c.Task.Line {
			return true
		}
	}
	return false
}

// TestKanbanDoesNotOverwriteExternalChange (K2): si la nota cambió en disco entre que se cargó y que se mueve la
// tarjeta, no se pisa el cambio: se recarga y se avisa; al repetir, ya con la nota al día, mueve.
func TestKanbanDoesNotOverwriteExternalChange(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "W")
	card := *m.kanban.current()
	original, _ := os.ReadFile(card.NotePath)
	external := string(original) + "\nAñadido por otro programa\n"
	os.WriteFile(card.NotePath, []byte(external), 0o644)
	later := time.Now().Add(5 * time.Second)
	os.Chtimes(card.NotePath, later, later)

	press(m, "L")
	if got, _ := os.ReadFile(card.NotePath); string(got) != external {
		t.Fatalf("se pisó el cambio externo:\n%s", got)
	}
	if !strings.Contains(lastRow(m), "Cambió por fuera") {
		t.Errorf("debía avisar que la nota cambió: %q", lastRow(m))
	}
	if len(m.c.board.ColumnCards(1)) != 0 {
		t.Error("la tarjeta no debía haberse movido")
	}
	press(m, "L") // con la nota recargada sí mueve
	if got := fileLines(t, card.NotePath); !strings.Contains(strings.Join(got, "\n"), "Añadido por otro programa") || len(m.c.board.ColumnCards(1)) != 1 {
		t.Error("tras recargar, mover debe funcionar y conservar el cambio externo")
	}
}

// TestKanbanDragAndDrop (H4-2, K3): arrastrar una tarjeta con el mouse a otra columna la mueve y reescribe solo su
// línea; mientras se arrastra, la tarjeta va resaltada y la columna de destino marcada; soltar donde empezó o
// cancelar con Esc no hace nada.
func TestKanbanDragAndDrop(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "W")
	card := *m.kanban.current()
	x, y, ok := cellOf(m, card.CleanText[:12])
	if !ok {
		t.Fatalf("no se ve la tarjeta %q", card.CleanText)
	}
	before := fileLines(t, card.NotePath)
	colW := m.layout.Kanban.W / 3

	m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	if m.kanban.press == nil || m.kanban.drag.Active {
		t.Fatal("al apretar el botón sobre una tarjeta no empieza aún un arrastre")
	}
	m.Update(tea.MouseMotionMsg{X: colW + colW/2, Y: y + 4, Button: tea.MouseLeft})
	if !m.kanban.drag.Active || m.kanban.drag.Target != 1 {
		t.Fatalf("el arrastre debía apuntar a la columna 2: %+v", m.kanban.drag)
	}
	out := plain(m)
	if !strings.Contains(out, "⇢") || !strings.Contains(out, "▸ [2]") {
		t.Errorf("durante el arrastre la tarjeta lleva ⇢ y la columna de destino ▸:\n%s", out)
	}
	if got := fileLines(t, card.NotePath); len(diffLines(before, got)) != 0 {
		t.Error("el archivo no debe cambiar hasta soltar")
	}
	m.Update(tea.MouseReleaseMsg{X: colW + colW/2, Y: y + 4, Button: tea.MouseLeft})
	if m.kanban.drag.Active || m.kanban.press != nil {
		t.Error("al soltar termina el arrastre")
	}
	after := fileLines(t, card.NotePath)
	d := diffLines(before, after)
	if len(d) != 1 || d[0] != card.Task.Line || !strings.HasSuffix(after[d[0]-1], " #kb/doing") {
		t.Fatalf("debía cambiar solo la línea %d con #kb/doing: %v", card.Task.Line, d)
	}
	if !hasCard(m.c.board.ColumnCards(1), card) || m.kanban.col != 1 || !strings.Contains(lastRow(m), "→ En progreso") {
		t.Errorf("la tarjeta debía estar en la columna 2 con el aviso: col=%d %q", m.kanban.col, lastRow(m))
	}

	// soltar en la misma columna: nada
	x, y, _ = cellOf(m, card.CleanText[:12])
	snapshot := fileLines(t, card.NotePath)
	m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: x + 3, Y: y + 1, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x + 3, Y: y + 1, Button: tea.MouseLeft})
	if len(diffLines(snapshot, fileLines(t, card.NotePath))) != 0 {
		t.Error("soltar en la misma columna no debe escribir")
	}
	// Esc cancela (un segundo clic seguido en la misma tarjeta sería un doble clic: se espera)
	time.Sleep(450 * time.Millisecond)
	m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: 2, Y: y, Button: tea.MouseLeft})
	if !m.kanban.drag.Active {
		t.Fatal("debía haber un arrastre hacia la columna 1")
	}
	press(m, "esc")
	m.Update(tea.MouseReleaseMsg{X: 2, Y: y, Button: tea.MouseLeft})
	if m.kanbanOn == false || len(diffLines(snapshot, fileLines(t, card.NotePath))) != 0 {
		t.Error("Esc debe cancelar el arrastre sin salir del Kanban ni escribir")
	}
}

// TestHostileTrashJSONDoesNotDeleteOnOpen (S1): abrir la app con un trash.json hostil en la carpeta de notas no
// borra nada fuera y avisa de las entradas ignoradas.
func TestHostileTrashJSONDoesNotDeleteOnOpen(t *testing.T) {
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	victim := filepath.Join(filepath.Dir(dir), "victima-trash")
	os.MkdirAll(victim, 0o755)
	t.Cleanup(func() { os.RemoveAll(victim) })
	os.WriteFile(filepath.Join(victim, "importante.txt"), []byte("dato"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".trash"), 0o755)
	rel, _ := filepath.Rel(filepath.Join(dir, ".trash"), victim)
	meta := `[{"id":"` + filepath.ToSlash(rel) + `","name":"x","original_path":"/nonexistent/x","deleted_at":"2000-01-01T00:00:00Z","is_dir":true}]`
	os.WriteFile(filepath.Join(dir, ".trash", "trash.json"), []byte(meta), 0o644)
	m.c.reload()
	if _, err := os.Stat(filepath.Join(victim, "importante.txt")); err != nil {
		t.Fatalf("abrir la app borró una carpeta de fuera: %v", err)
	}
	if !strings.Contains(m.c.status, "trash.json") {
		t.Errorf("debía avisar: %q", m.c.status)
	}
}

// TestKanbanCardsWithDates (C.6): las tarjetas muestran las fechas de la nota (con la vencida en color de error), un clic
// en cualquier fila de la tarjeta (no solo en su texto) la selecciona, marcarla como hecha con Espacio agrega ✅ a su línea
// (solo a esa) y devolverla lo quita; la opción "Tarjetas: compactas" vuelve a la vista de una fila.
func TestKanbanCardsWithDates(t *testing.T) {
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	p := filepath.Join(dir, "plazos.md")
	os.WriteFile(p, []byte("# Plazos\n\n- [ ] AAA_PRIMERA 📅 2020-01-02 🛫 2019-12-01\n- [ ] BBB_SEGUNDA 📅 2099-01-01\n"), 0o644)
	m.c.reload()
	press(m, "W")
	out := plain(m)
	for _, want := range []string{"AAA_PRIMERA", "\uf135 2019-12-01 \uf073 2020-01-02", "\uf073 2099-01-01", "╭──"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	// clic en la fila de las fechas de BBB (no en su texto): selecciona esa tarjeta y abre el popup de fechas (ORD-015 C.2); Esc lo cierra
	x, y, ok := cellOf(m, "\uf073 2099-01-01")
	if !ok {
		t.Fatalf("no se ve la fecha de BBB:\n%s", out)
	}
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	if cur := m.kanban.current(); cur == nil || !strings.Contains(cur.CleanText, "BBB_SEGUNDA") {
		t.Fatalf("el clic en la fila de fechas debía seleccionar la tarjeta BBB: %+v", cur)
	}
	if _, ok := m.c.top().(*datesPopup); !ok {
		t.Fatalf("el clic en la fecha de la tarjeta debe abrir el popup de fechas: %T", m.c.top())
	}
	press(m, "esc")
	if m.c.top() != nil {
		t.Fatal("Esc cierra el popup")
	}
	// Espacio la marca como hecha: ✅ hoy en esa línea y solo en esa
	before := fileLines(t, p)
	press(m, "space")
	after := fileLines(t, p)
	d := diffLines(before, after)
	if len(d) != 1 || !strings.Contains(after[d[0]-1], "BBB_SEGUNDA 📅 2099-01-01 ✅ "+storage.Today()) || !strings.Contains(after[d[0]-1], "- [x]") {
		t.Fatalf("debía cambiar solo la línea de BBB con [x] y ✅ hoy: %v\n%v", d, after)
	}
	if !strings.Contains(plain(m), "\uf073 2099-01-01 \uf00c "+storage.Today()) {
		t.Errorf("la tarjeta debe mostrar la fecha de completada:\n%s", plain(m))
	}
	// Espacio otra vez la devuelve a la primera columna y quita el ✅
	press(m, "space")
	if got := fileLines(t, p); !reflect.DeepEqual(got, before) {
		t.Errorf("al desmarcarla la nota debe quedar como estaba:\n%v", got)
	}
	// la opción compactas
	m.c.cfg.KanbanCards = config.KanbanCardsCompact
	if got := plain(m); strings.Count(got, "╭") > 3 || !strings.Contains(got, "☐ AAA_PRIMERA") {
		t.Errorf("con compactas debe verse una fila por tarea:\n%s", got)
	}
}

// TestKanbanCardsSetting (C.6): "Tarjetas" está en Ajustes (rectángulos | compactas), se cambia con ← y →, se aplica al
// momento y se guarda; por defecto rectángulos.
func TestKanbanCardsSetting(t *testing.T) {
	m := newTestModel(t, 120, 35)
	if m.c.cfg.KanbanCards != config.KanbanCardsRects {
		t.Fatalf("por defecto = %q", m.c.cfg.KanbanCards)
	}
	press(m, ",")
	sp := m.c.top().(*settingsPopup)
	sp.list.set(int(setKanbanCards), sp.n)
	if got := plain(m); !strings.Contains(got, "Tarjetas") || !strings.Contains(got, "rectángulos") {
		t.Errorf("el ajuste no aparece con su valor:\n%s", got)
	}
	press(m, "right")
	if m.c.cfg.KanbanCards != config.KanbanCardsCompact || !strings.Contains(plain(m), "compactas") {
		t.Fatalf("tras → debía ser compact: %q", m.c.cfg.KanbanCards)
	}
	if data, err := os.ReadFile(m.c.cfg.Path()); err != nil || !strings.Contains(string(data), `"kanban_cards": "compact"`) {
		t.Errorf("no se guardó (%v):\n%s", err, data)
	}
	press(m, "left")
	if m.c.cfg.KanbanCards != config.KanbanCardsRects {
		t.Errorf("tras ← debía volver a cards")
	}
}

// TestNoColorEmojiOnScreen (C.8): con una nota con fechas, la salida de View no lleva ningún emoji a color (U+1F000–U+1FAFF,
// U+2705, U+FE0F) en el Kanban (tarjetas y compacto), en el panel Tareas ni en la vista previa; el archivo conserva 🛫 📅 ✅.
func TestNoColorEmojiOnScreen(t *testing.T) {
	m := newTestModel(t, 140, 40)
	dir := m.c.store.BaseDir
	for _, e := range m.c.notes { // solo la nota de prueba, sin los emojis propios de las fixtures
		os.Remove(e.Path)
	}
	p := filepath.Join(dir, "plazos.md")
	body := "# Plazos\n\n- [ ] vencida 📅 2020-01-02 🛫 2019-12-01\n- [x] hecha ✅ 2026-09-30 📅 2026-09-29\n- [ ] repetida 📅 2026-13-45 y 📅 2030-01-01\n"
	os.WriteFile(p, []byte(body), 0o644)
	m.c.reload()
	m.afterChange()
	check := func(what string) {
		t.Helper()
		out := plain(m)
		for _, r := range out {
			if r == 0x1F6EB || r == 0x1F4C5 || r == 0x2705 || r == 0xFE0F || (r >= 0x1F000 && r <= 0x1FAFF) {
				t.Fatalf("%s: aparece el emoji %U en pantalla:\n%s", what, r, out)
			}
		}
	}
	m.notes.selectPath(p)
	check("notas con vista previa")
	if out := plain(m); !strings.Contains(out, " 2020-01-02") && !strings.Contains(out, "") {
		t.Errorf("la vista previa debe mostrar los glifos de fecha:\n%s", out)
	}
	press(m, "2") // panel Tareas
	check("panel Tareas")
	if out := plain(m); !strings.Contains(out, " 2020-01-02") {
		t.Errorf("el panel Tareas debe mostrar el vencimiento con su glifo:\n%s", out)
	}
	press(m, "esc")
	press(m, "W")
	check("Kanban con tarjetas")
	m.c.cfg.KanbanCards = config.KanbanCardsCompact
	check("Kanban compacto")
	if b, _ := os.ReadFile(p); string(b) != body {
		t.Errorf("mirar no debe cambiar la nota:\n%s", b)
	}
	// sin Nerd Font: símbolos de texto, tampoco emojis
	views.DateIcons = false
	defer func() { views.DateIcons = true }()
	m.c.cfg.KanbanCards = config.KanbanCardsRects
	check("Kanban sin Nerd Font")
	if out := plain(m); !strings.Contains(out, "◷ 2020-01-02") {
		t.Errorf("sin Nerd Font el vencimiento lleva ◷:\n%s", out)
	}
}

// TestTasksPanelRowAfterTicking (C.8): al marcar una tarea con Espacio se le agrega ✅ hoy en el archivo, pero la fila del panel
// Tareas sigue siendo "☑ texto … nota" (el panel solo muestra el vencimiento, no la fecha de completada).
func TestTasksPanelRowAfterTicking(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "2", "space")
	for _, l := range strings.Split(plain(m), "\n") {
		if strings.Contains(l, "☑") && strings.Contains(l, "") {
			t.Errorf("la fila de una tarea marcada no lleva la fecha de completada: %q", l)
		}
	}
	if !strings.Contains(plain(m), "☑") {
		t.Errorf("la tarea marcada debe verse como hecha:\n%s", plain(m))
	}
}
