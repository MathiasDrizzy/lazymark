package app

import (
	"github.com/charmbracelet/x/ansi"
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
	m.c.store.DateFormatPref = "emoji" // estas pruebas miran el archivo en formato de emojis; la de Dataview lo dice aparte
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

// TestNoDateEmojiOnScreen (Mathias, 2026-10-05: "te dije que era sin emojis"): ninguna pantalla muestra los emojis de fecha del archivo (🛫 📅 ✅), tampoco
// la barra de estado al marcar o reabrir una tarea con fechas: el aviso usa el texto sin fechas y, por si acaso, la barra reemplaza cualquier emoji de fecha.
func TestNoDateEmojiOnScreen(t *testing.T) {
	m := newTestModel(t, 200, 40)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	line := "- [ ] Publicar tag v0.1.0 para releases automáticas con GoReleaser 🛫 2026-10-08 📅 2026-10-05"
	path := filepath.Join(dir, "pub.md")
	os.WriteFile(path, []byte("# Pub\n\n"+line+"\n"), 0o644)
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(path)
	noEmoji := func(where string) {
		t.Helper()
		for i, l := range screen(m) {
			if strings.ContainsAny(ansi.Strip(l), "🛫📅✅") {
				t.Errorf("%s: emoji de fecha en la fila %d: %q", where, i, ansi.Strip(l))
			}
		}
	}
	noEmoji("notas")
	press(m, "2")
	noEmoji("Tareas")
	press(m, "space") // marcar: "Tarea completada: …"
	noEmoji("aviso al marcar")
	press(m, "space") // reabrir
	noEmoji("aviso al reabrir")
	press(m, "d")
	noEmoji("popup de fechas")
	press(m, "esc", "W")
	noEmoji("Kanban")
	m.c.setStatus("%s", "cualquier aviso con 📅 2026-10-05 y 🛫 y ✅")
	noEmoji("un aviso cualquiera con emojis de fecha")
}

// TestDatesPopupWritesDataviewByDefault (ORD-017 F2): sin date_format el popup escribe el formato Dataview (sin emojis en el archivo) y la pantalla lo dibuja con glifos.
func TestDatesPopupWritesDataviewByDefault(t *testing.T) {
	m, path, line := datesRig(t)
	m.c.store.DateFormatPref = "" // el valor por defecto
	press(m, "d")
	typeText(m, "+1w")
	press(m, "tab")
	typeText(m, "viernes")
	press(m, "enter")
	if got := fileLines(t, path)[line-1]; got != "- [ ] Pan [start:: 2026-10-12] [due:: 2026-10-09]" {
		t.Fatalf("en el archivo: %q", got)
	}
	m.c.reload()
	m.afterChange()
	press(m, "W")
	scr := plain(m)
	if strings.Contains(scr, "::") || strings.ContainsAny(scr, "🛫📅✅⏳➕") || !strings.Contains(scr, "2026-10-09") {
		t.Errorf("la pantalla dibuja las fechas Dataview con glifos, sin su sintaxis:\n%s", scr)
	}
}

// TestNoTasksEmojiOrDataviewSyntaxOnScreen (ORD-017 F4 / L9): ⏳ (programada) y ➕ (creada) y los campos Dataview llegan a la pantalla como glifos, en
// todas partes: Tareas, Kanban, vista previa, avisos y resultados de búsqueda; nunca el emoji ni la sintaxis [clave:: fecha].
func TestNoTasksEmojiOrDataviewSyntaxOnScreen(t *testing.T) {
	m := newTestModel(t, 200, 40)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	body := "# Mix\n\n- [ ] Programada ⏳ 2026-10-09 ➕ 2026-10-01\n- [ ] Dataview [scheduled:: 2026-10-09] [created:: 2026-10-01] [due:: 2026-10-12] [start:: 2026-10-05]\n- [ ] Paréntesis (due:: 2026-10-20)\n"
	path := filepath.Join(dir, "mix.md")
	os.WriteFile(path, []byte(body), 0o644)
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(path)
	bad := func(where string) {
		t.Helper()
		for i, l := range screen(m) {
			p := ansi.Strip(l)
			if strings.ContainsAny(p, "🛫📅✅⏳➕") || strings.Contains(p, "::") {
				t.Errorf("%s: emoji o sintaxis Dataview en la fila %d: %q", where, i, p)
			}
		}
	}
	bad("notas (vista previa)")
	press(m, "2")
	bad("Tareas")
	press(m, "space")
	bad("aviso al marcar")
	press(m, "space")
	press(m, "d")
	bad("popup de fechas")
	press(m, "esc", "W")
	bad("Kanban")
	if scr := plain(m); !strings.Contains(scr, "2026-10-09") || !strings.Contains(scr, "2026-10-12") {
		t.Errorf("las fechas Dataview y las de emoji se ven igual en el tablero:\n%s", scr)
	}
	press(m, "W", "/")
	typeSearch(m, "Programada")
	if !strings.Contains(plain(m), "mix.md:3") {
		t.Fatalf("la búsqueda debe mostrar el resultado de mix.md línea 3:\n%s", plain(m))
	}
	bad("resultados de la búsqueda")
	// R2-5 (ORD-017 rev 2): buscar el nombre del campo o parte de la fecha tampoco deja la sintaxis cruda en los fragmentos
	for _, q := range []string{"due", "2026", "start"} {
		press(m, "esc", "/")
		typeSearch(m, q)
		if !strings.Contains(plain(m), ".md:") {
			t.Fatalf("la búsqueda %q debe dar resultados:\n%s", q, plain(m))
		}
		bad("búsqueda de " + q)
	}
}

// TestDateFormatNoticeOnce (ORD-017 F2): en un vault que solo tiene fechas con emojis, el popup escribe con emojis y avisa una sola vez cómo cambiarlo (y lo
// recuerda en la config); el ajuste "Formato de fechas" cambia el formato que se escribe.
func TestDateFormatNoticeOnce(t *testing.T) {
	old := storage.Today
	storage.Today = func() string { return "2026-10-05" }
	t.Cleanup(func() { storage.Today = old })
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	path := filepath.Join(dir, "n.md")
	os.WriteFile(path, []byte("# N\n- [ ] a 📅 2026-01-01\n- [ ] b\n- [ ] c\n"), 0o644)
	m.c.reload()
	m.afterChange()
	press(m, "2")
	edit := func(line int, keys string) {
		m.tasks.selectTask(path, line)
		press(m, "d")
		typeText(m, keys)
		press(m, "enter")
	}
	edit(3, "+1d")
	if got := fileLines(t, path)[2]; got != "- [ ] b 🛫 2026-10-06" {
		t.Fatalf("en un vault solo con emojis se escribe con emojis: %q", got)
	}
	if !strings.Contains(lastRow(m), "emojis") || !m.c.cfg.DateFormatNoticeShown {
		t.Errorf("avisa una vez cómo cambiarlo y lo recuerda: %q (visto=%v)", lastRow(m), m.c.cfg.DateFormatNoticeShown)
	}
	edit(4, "+2d")
	if strings.Contains(lastRow(m), "emojis") {
		t.Errorf("el aviso no se repite: %q", lastRow(m))
	}
	// el ajuste: dataview fijo → una tarea sin fechas se escribe en Dataview aunque el vault tenga emojis
	b, _ := os.ReadFile(path)
	os.WriteFile(path, append(b, []byte("- [ ] d\n")...), 0o644)
	m.c.reload()
	m.afterChange()
	press(m, ",")
	sp := m.c.top().(*settingsPopup)
	sp.list.set(int(setDateFormat), sp.n)
	press(m, "right") // "" → dataview
	press(m, "esc")
	if m.c.cfg.DateFormat != "dataview" {
		t.Fatalf("el ajuste debe quedar en dataview: %q", m.c.cfg.DateFormat)
	}
	edit(5, "+1d")
	if got := fileLines(t, path)[4]; got != "- [ ] d [start:: 2026-10-06]" {
		t.Errorf("con dataview fijo se escribe Dataview: %q", got)
	}
}

// TestDatesPopupWarnsStartAfterDue (ORD-017 F5 / L8): si el inicio queda después del vencimiento el popup avisa (con lo escrito, antes de guardar) y guarda igual.
func TestDatesPopupWarnsStartAfterDue(t *testing.T) {
	m, path, line := datesRig(t)
	press(m, "d")
	typeText(m, "2026-10-08")
	if strings.Contains(plain(m), "posterior al vencimiento") {
		t.Error("sin vencimiento no hay nada que avisar")
	}
	press(m, "tab")
	typeText(m, "2026-10-05")
	if out := plain(m); !strings.Contains(out, "posterior al vencimiento") {
		t.Errorf("debe avisar que el inicio es posterior al vencimiento:\n%s", out)
	}
	press(m, "enter")
	if got := fileLines(t, path)[line-1]; got != "- [ ] Pan 🛫 2026-10-08 📅 2026-10-05" {
		t.Errorf("el aviso no impide guardar: %q", got)
	}
	// corregido: el aviso desaparece
	press(m, "d")
	press(m, "ctrl+u")
	typeText(m, "2026-10-01")
	if strings.Contains(plain(m), "posterior al vencimiento") {
		t.Error("con el inicio antes del vencimiento no hay aviso")
	}
}
