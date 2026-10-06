package app

import (
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
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

// TestSearchFragmentsKeepLinksRaw (ORD-017 rev 3): al buscar `due`, el fragmento de [[due:: …]] y de [due:: …](url) se ve tal cual (son un wikilink y un link, no fechas) y
// el campo de verdad de otra línea sí se dibuja con glifo.
func TestSearchFragmentsKeepLinksRaw(t *testing.T) {
	m := newTestModel(t, 200, 40)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	path := filepath.Join(dir, "lk.md")
	os.WriteFile(path, []byte("# Lk\n\n- [ ] A [[due:: 2026-05-10]] fin\n- [ ] B [due:: 2026-05-11](https://x.y) fin\n- [ ] C [due:: 2026-05-12] fin\n"), 0o644)
	m.c.reload()
	m.afterChange()
	press(m, "/")
	typeSearch(m, "due")
	scr := plain(m)
	for _, want := range []string{"[[due:: 2026-05-10]]", "[due:: 2026-05-11](https://x.y)"} {
		if !strings.Contains(scr, want) {
			t.Errorf("el fragmento debe mostrar %q tal cual:\n%s", want, scr)
		}
	}
	if strings.Contains(scr, "[due:: 2026-05-12]") || !strings.Contains(scr, "2026-05-12") {
		t.Errorf("el campo de verdad se dibuja como glifo + fecha:\n%s", scr)
	}
}

// TestNoC1ControlsReachTheTerminal (ORD-019 C.1 / H1): ningún carácter U+0080–U+009F (el C1 de 8 bits: U+009D es OSC y U+009B es CSI) llega a la terminal desde el
// texto de una nota: vista previa, Tareas, Kanban, búsqueda, avisos de la barra de estado y títulos de nota.
func TestNoC1ControlsReachTheTerminal(t *testing.T) {
	m := newTestModel(t, 200, 40)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	evil := "\u009d52;c;cHduZWQ=\u009c\u009d0;PWNED\u009c\u009b31m"
	path := filepath.Join(dir, "evil.md")
	os.WriteFile(path, []byte("# Título "+evil+"\n\ntexto "+evil+" fin #etiqueta"+evil+"\n\n- [ ] TAREA_MALA "+evil+" fin\n- [ ] otra #kb/doing "+evil+"\n"), 0o644)
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(path)
	check := func(where string) {
		t.Helper()
		for _, r := range m.View().Content {
			if r >= 0x80 && r <= 0x9f {
				t.Fatalf("%s: llega a la terminal U+%04X", where, r)
			}
		}
	}
	check("vista previa")
	press(m, "2")
	check("Tareas")
	press(m, "space")
	check("aviso al marcar")
	press(m, "space")
	press(m, "W")
	check("Kanban")
	press(m, "W", "/")
	typeSearch(m, "TAREA_MALA")
	check("búsqueda")
	press(m, "esc")
	m.c.setStatus("%s", "aviso "+evil)
	check("barra de estado")
}

// TestSettingsDateColors (ORD-020 K3): en Ajustes se apaga date_colors, se cambia due_soon_days y el color de un estado; se aplica a la vista al momento y se guarda en el
// archivo de configuración.
func TestSettingsDateColors(t *testing.T) {
	m := newTestModel(t, 120, 40)
	old := views.DateColors
	t.Cleanup(func() { views.DateColors = old })
	p := newSettingsPopup(m.c, func() {}, func() {})
	if !m.c.cfg.DateColors || m.c.cfg.DueSoonDays != 0 || m.c.cfg.DateColorNames["soon"] != "warning" {
		t.Fatalf("por defecto: colores sí, 0 días, por vencer = warning: %+v", m.c.cfg)
	}
	p.change(setDueSoon, 1)
	p.change(setDueSoon, 1)
	if m.c.cfg.DueSoonDays != 2 || views.DateColors.SoonDays != 2 {
		t.Errorf("due_soon_days = %d (vista %d), se esperaba 2", m.c.cfg.DueSoonDays, views.DateColors.SoonDays)
	}
	p.change(setDueSoon, -1)
	p.change(setDueSoon, -1)
	p.change(setDueSoon, -1) // de 0 vuelve al último: 30
	if m.c.cfg.DueSoonDays != 30 {
		t.Errorf("la vuelta al principio da 30: %d", m.c.cfg.DueSoonDays)
	}
	p.change(setColorSoon, 1) // warning → orange
	if m.c.cfg.DateColorNames["soon"] != "orange" || views.DateColors.Names["soon"] != "orange" {
		t.Errorf("color de por vencer = %q", m.c.cfg.DateColorNames["soon"])
	}
	p.change(setDateColors, 1)
	if m.c.cfg.DateColors || views.DateColors.Enabled {
		t.Error("date_colors apagado")
	}
	for _, id := range []settingID{setDateColors, setDueSoon, setColorOverdue, setColorSoon, setColorOnTime, setColorStarted, setColorNotStarted, setColorDone} {
		if p.label(id) == "" || strings.TrimSpace(p.value(id)) == "" {
			t.Errorf("la fila %d de Ajustes debe tener texto y valor", id)
		}
	}
	// el archivo de configuración lo dice
	data, err := os.ReadFile(m.c.cfg.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"date_colors": false`, `"due_soon_days": 30`, `"soon": "orange"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el archivo de configuración debe llevar %s:\n%s", want, data)
		}
	}
}

// TestPreviewDatesAreColoredByState (ORD-020): en la vista previa las fechas llevan el color de su estado (vencida, por vencer, en fecha) y, apagados los colores, ninguno.
func TestPreviewDatesAreColoredByState(t *testing.T) {
	old := storage.Today
	storage.Today = func() string { return "2026-10-06" }
	t.Cleanup(func() { storage.Today = old })
	oldC := views.DateColors
	t.Cleanup(func() { views.DateColors = oldC })
	views.DateColors = views.DateColorSettings{Enabled: true, Names: config.DefaultDateColorNames()}
	theme.ApplyThemeByName("catppuccin-mocha")
	lines := views.ColorDateLines([]string{
		"[ ] vencida " + views.DateGlyph(storage.DateDue) + " 2026-10-01",
		"[ ] hoy " + views.DateGlyph(storage.DateDue) + " 2026-10-06",
		"[ ] lejos " + views.DateGlyph(storage.DateDue) + " 2026-12-01",
		"[✓] hecha " + string(views.DoneMark) + views.DateGlyph(storage.DateDue) + " 2026-10-01", // la tarea hecha lleva el marcador del dato parseado
		"texto sin fechas",
	}, "2026-10-06")
	want := []string{"38;2;243;139;168", "38;2;249;226;175", "38;2;137;180;250", ""} // rojo, amarillo y azul de Catppuccin Mocha; la tarea hecha, neutra
	for i, w := range want[:3] {
		if !strings.Contains(lines[i], w) {
			t.Errorf("línea %d: debe llevar el color %s: %q", i, w, lines[i])
		}
	}
	if strings.Contains(lines[3], "38;2;243;139;168") {
		t.Errorf("el vencimiento de una tarea hecha no va en rojo: %q", lines[3])
	}
	if lines[4] != "texto sin fechas" {
		t.Errorf("una línea sin fechas no cambia: %q", lines[4])
	}
}

// TestPreviewDoneComesFromTheParsedTask (ORD-022 C.1 / L19-a): que las fechas de una tarea se vean neutras (hecha) lo decide el estado de la tarea, no el texto "[✓]" de la
// línea: una tarea PENDIENTE que lleva "[✓]" en su descripción conserva el color de su estado, y una tarea hecha va neutra aunque su descripción no lo diga.
func TestPreviewDoneComesFromTheParsedTask(t *testing.T) {
	m := newTestModel(t, 200, 40)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	oldC := views.DateColors
	t.Cleanup(func() { views.DateColors = oldC })
	views.DateColors = views.DateColorSettings{Enabled: true, Names: config.DefaultDateColorNames()}
	theme.ApplyThemeByName("catppuccin-mocha")
	path := filepath.Join(dir, "estado.md")
	os.WriteFile(path, []byte("# Estado\n\n- [ ] PENDIENTE con [✓] en el texto 📅 2020-01-01\n- [x] HECHA sin marca 📅 2020-01-01 ✅ 2020-01-02\n- [ ] pendiente normal 📅 2020-01-01\n"), 0o644)
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(path)
	var pending, done, normal string
	for _, l := range strings.Split(m.View().Content, "\n") {
		switch {
		case strings.Contains(ansi.Strip(l), "[ ] PENDIENTE"): // las filas de la vista previa (el panel Tareas dibuja ☐ y ☑)
			pending = l
		case strings.Contains(ansi.Strip(l), "[✓] HECHA"):
			done = l
		case strings.Contains(ansi.Strip(l), "[ ] pendiente normal"):
			normal = l
		}
	}
	const red = "38;2;243;139;168" // el rojo de Catppuccin Mocha: vencida
	if pending == "" || done == "" || normal == "" {
		t.Fatalf("faltan filas en la vista previa: %q %q %q", pending, done, normal)
	}
	if !strings.Contains(pending, red) || !strings.Contains(normal, red) {
		t.Errorf("una tarea pendiente vencida va en rojo, también con [✓] en su texto:\npendiente con [✓]: %q\npendiente normal: %q", pending, normal)
	}
	if strings.Contains(done, red) {
		t.Errorf("el vencimiento de una tarea hecha no va en rojo aunque su texto no diga [✓]: %q", done)
	}
	for _, l := range []string{pending, done, normal} {
		if strings.ContainsRune(l, views.DoneMark) {
			t.Errorf("el marcador interno no debe llegar a la pantalla: %q", l)
		}
	}
}

// TestHandWrittenGlyphsOfBothSetsAreColored (ORD-022 C.2 / L19-b): un glifo de fecha escrito a mano con el OTRO juego (texto con Nerd Font activo, o Nerd Font con el de texto) también
// se colorea según su campo: lazymark dibuja lo que hay en la nota tal cual, así que el caso se reproducía.
func TestHandWrittenGlyphsOfBothSetsAreColored(t *testing.T) {
	oldC, oldIcons := views.DateColors, views.DateIcons
	t.Cleanup(func() { views.DateColors, views.DateIcons = oldC, oldIcons })
	views.DateColors = views.DateColorSettings{Enabled: true, Names: config.DefaultDateColorNames()}
	theme.ApplyThemeByName("catppuccin-mocha")
	const red, blue = "38;2;243;139;168", "38;2;137;180;250"
	for _, icons := range []bool{true, false} {
		views.DateIcons = icons
		for _, c := range []struct{ glyph, date, want, what string }{
			{"◷", "2026-10-01", red, "calendario de texto, vencida"}, {"\uf073", "2026-10-01", red, "calendario Nerd Font, vencida"},
			{"◷", "2999-01-01", blue, "calendario de texto, en fecha"}, {"\uf073", "2999-01-01", blue, "calendario Nerd Font, en fecha"},
		} {
			got := views.ColorDateLines([]string{"- [ ] t " + c.glyph + " " + c.date}, "2026-10-06")[0]
			if !strings.Contains(got, c.want) {
				t.Errorf("iconos=%v: %s: no se coloreó (%q)", icons, c.what, got)
			}
		}
	}
}

// TestDateWarningsOption (ORD-025 O1): con date_warnings en false el popup de Fechas no avisa que el inicio es posterior al vencimiento (y guarda igual); por defecto avisa.
func TestDateWarningsOption(t *testing.T) {
	for _, on := range []bool{true, false} {
		m, _, _ := datesRig(t)
		m.c.cfg.DateWarnings = on
		press(m, "d")
		typeText(m, "2026-10-08")
		press(m, "tab")
		typeText(m, "2026-10-05")
		if got := strings.Contains(plain(m), "posterior al vencimiento"); got != on {
			t.Errorf("date_warnings=%v: aviso visible=%v", on, got)
		}
	}
}

// TestDateFormatNoticeOption (ORD-025 O2): con date_format_notice en false el aviso único del vault con emojis no aparece nunca; por defecto aparece una vez.
func TestDateFormatNoticeOption(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "n.md"), []byte("- [ ] a 📅 2026-01-01\n- [ ] b\n"), 0o644)
	for _, off := range []bool{false, true} {
		s := storage.New(dir)
		s.DateNoticeOff = off
		s.WriteDateFormat() // en un vault solo con emojis activa la excepción y prepara el aviso
		got := s.TakeDateFormatNotice()
		if off && got != "" {
			t.Errorf("apagado no avisa nunca: %q", got)
		}
		if !off && got == "" {
			t.Error("por defecto avisa una vez")
		}
	}
}

// TestSettingsOptionRows (ORD-025 UI): Ajustes muestra las 13 filas nuevas (la lista se desplaza), cada una cambia su valor, se aplica a la capa de datos y queda en el archivo
// de configuración; los de texto validan lo escrito (uno inválido no cambia nada) y el prefijo del tablero avisa que no migra las notas.
func TestSettingsOptionRows(t *testing.T) {
	oldG, oldTD, oldKT := views.DateGlyphs, storage.TrashDays, storage.KanbanTag
	t.Cleanup(func() { views.DateGlyphs, storage.TrashDays, storage.KanbanTag = oldG, oldTD, oldKT })
	m := newTestModel(t, 120, 40)
	press(m, ",")
	sp := m.c.top().(*settingsPopup)
	cfg := m.c.cfg
	// las filas existen, con texto y valor, y se alcanzan con el cursor (la lista se desplaza)
	for id := setDateWarnings; id <= setDateGlyphs; id++ {
		if sp.label(id) == "" || strings.TrimSpace(sp.value(id)) == "" {
			t.Errorf("la fila %d de Ajustes debe tener texto y valor", id)
		}
	}
	sp.list.set(int(setDateGlyphs), sp.n)
	if out := plain(m); !strings.Contains(out, "Glifos de las fechas") {
		t.Errorf("la última fila se ve al bajar el cursor:\n%s", out)
	}
	// interruptores y presets
	sp.change(setDateWarnings, 1)
	sp.change(setDateFormatNotice, 1)
	sp.change(setHintIdle, 1)  // 20 → 30
	sp.change(setHintShow, -1) // 15 → 10
	sp.change(setHintEvery, 1) // 60 → 90
	sp.change(setTrashDays, 1) // 20 → 30
	sp.change(setNotesSort, 1)
	sp.change(setTasksSort, 1)
	if cfg.DateWarnings || cfg.DateFormatNotice || cfg.ClickHintIdleSeconds != 30 || cfg.ClickHintShowSeconds != 10 || cfg.ClickHintEverySeconds != 90 || cfg.TrashDays != 30 || cfg.NotesSort != "modified" || cfg.TasksSort != "due" {
		t.Errorf("los cambios de las filas: %+v", cfg)
	}
	if storage.TrashDays != 30 || !m.c.store.DateNoticeOff || m.c.store.NotesSort != "modified" {
		t.Errorf("se aplican a la capa de datos: días=%d aviso apagado=%v orden=%q", storage.TrashDays, m.c.store.DateNoticeOff, m.c.store.NotesSort)
	}
	// show nunca llega a every: subir show por encima de every sube every
	for i := 0; i < 12; i++ {
		sp.change(setHintShow, 1)
	}
	if cfg.ClickHintShowSeconds >= cfg.ClickHintEverySeconds {
		t.Errorf("visible (%d) debe ser menor que cada (%d)", cfg.ClickHintShowSeconds, cfg.ClickHintEverySeconds)
	}
	// texto: válido
	typeInto := func(id settingID, text string) {
		sp.change(id, 1)
		press(m, "ctrl+u")
		typeText(m, text)
		press(m, "enter")
	}
	typeInto(setDailyFolder, "diario/2026")
	typeInto(setTemplatesFolder, "moldes")
	typeInto(setDailyName, "AAAA.MM.DD")
	typeInto(setDateGlyphs, "- D - - 日")
	if cfg.DailyFolder != "diario/2026" || cfg.TemplatesFolder != "moldes" || cfg.DailyName != "AAAA.MM.DD" || cfg.DateGlyphs["due"] != "D" || cfg.DateGlyphs["created"] != "日" || len(cfg.DateGlyphs) != 2 {
		t.Errorf("los textos válidos se guardan: %+v %v", cfg, cfg.DateGlyphs)
	}
	if m.c.store.DailyFolder != "diario/2026" || views.DateGlyph(storage.DateDue) != "D" {
		t.Errorf("y se aplican: %q %q", m.c.store.DailyFolder, views.DateGlyph(storage.DateDue))
	}
	// texto: inválido (nada cambia)
	typeInto(setDailyFolder, "../fuera")
	typeInto(setDailyName, "nota")
	typeInto(setTemplatesFolder, "/abs")
	typeInto(setDateGlyphs, "ab")
	if cfg.DailyFolder != "diario/2026" || cfg.DailyName != "AAAA.MM.DD" || cfg.TemplatesFolder != "moldes" || cfg.DateGlyphs["due"] != "D" {
		t.Errorf("un valor inválido no cambia nada: %+v", cfg)
	}
	if !strings.Contains(lastRow(m), "no válido") && !strings.Contains(lastRow(m), "Invalid") {
		t.Errorf("y lo dice: %q", lastRow(m))
	}
	// prefijo del tablero: avisa del retag
	typeInto(setKanbanTag, "board")
	if cfg.KanbanTag != "board" || storage.KanbanTag != "board" || !strings.Contains(lastRow(m), "kanban retag") {
		t.Errorf("kanban_tag: %q %q fila=%q", cfg.KanbanTag, storage.KanbanTag, lastRow(m))
	}
	typeInto(setKanbanTag, "1 mal")
	if cfg.KanbanTag != "board" {
		t.Errorf("un prefijo inválido no cambia nada: %q", cfg.KanbanTag)
	}
	// quedó en el archivo y se lee de vuelta
	data, err := os.ReadFile(cfg.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"date_warnings": false`, `"date_format_notice": false`, `"click_hint_idle_seconds": 30`, `"trash_days": 30`, `"notes_sort": "modified"`, `"tasks_sort": "due"`,
		`"daily_folder": "diario/2026"`, `"daily_name": "AAAA.MM.DD"`, `"templates_folder": "moldes"`, `"kanban_tag": "board"`, `"due": "D"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el archivo de configuración debe llevar %s", want)
		}
	}
	back, err := config.LoadReadOnly(m.c.store.BaseDir)
	if err != nil || back.KanbanTag != "board" || back.TrashDays != 30 || back.DailyName != "AAAA.MM.DD" || back.DateGlyphs["due"] != "D" {
		t.Errorf("al reabrir se lee lo guardado: %v %+v", err, back)
	}
}
