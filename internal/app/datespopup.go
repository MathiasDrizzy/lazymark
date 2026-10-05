package app

import (
	"errors"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// datesPopup es el popup "Fechas" de una tarea: dos campos, Inicio y Vence, que aceptan AAAA-MM-DD, hoy, mañana, +3d, +1w y días de la semana
// (ParseDateInput). Muestra la fecha resuelta al lado de cada campo antes de guardar; vacío quita la fecha; Enter guarda las dos en una sola
// escritura de la línea de la tarea (con el formato Obsidian Tasks: los emojis quedan solo en el archivo); Esc cancela.
type datesPopup struct {
	c       *core
	path    string
	line    int
	text    string
	orig    storage.Dates
	inputs  [2]textinput.Model // 0: inicio, 1: vence
	focus   int
	err     string
	today   time.Time // "hoy" al abrir el popup: la vista previa y lo que se escribe usan el mismo, aunque pase la medianoche
	onSaved func()
}

const (
	dateStart = 0
	dateDue   = 1
)

func newDatesPopup(c *core, path string, line int, text string, d storage.Dates, onSaved func()) *datesPopup {
	p := &datesPopup{c: c, path: path, line: line, text: text, orig: d, onSaved: onSaved}
	p.today = readToday()
	for i, v := range [2]string{d.Start, d.Due} {
		ti := textinput.New()
		ti.SetStyles(themedInputStyles())
		ti.Prompt = ""
		ti.CharLimit = 24
		ti.SetValue(v)
		ti.CursorEnd()
		ti.Blur()
		p.inputs[i] = ti
	}
	p.inputs[p.focus].Focus()
	return p
}

// openDates abre el popup de fechas de la tarea; onSaved se llama tras guardar (o tras recargar si la nota cambió por fuera).
func (c *core) openDates(path string, line int, text string, d storage.Dates, onSaved func()) {
	c.push(newDatesPopup(c, path, line, text, d, onSaved))
}

func (p *datesPopup) contexts() []Context { return nil }
func (p *datesPopup) bottomRight() bool   { return false }

// readToday es "hoy" (storage.Today) como fecha; si no se puede leer, el día del reloj.
func readToday() time.Time {
	t, err := time.ParseInLocation("2006-01-02", storage.Today(), time.Local)
	if err != nil {
		return time.Now()
	}
	return t
}

// resolve devuelve la fecha que sale del campo i ("" = sin fecha) o el error de no entenderla.
func (p *datesPopup) resolve(i int) (string, error) {
	return storage.ParseDateInput(p.inputs[i].Value(), p.today, i18n.CurrentLanguage())
}

func (p *datesPopup) setFocus(i int) {
	p.focus = i
	for n := range p.inputs {
		if n == i {
			p.inputs[n].Focus()
		} else {
			p.inputs[n].Blur()
		}
	}
}

func (p *datesPopup) handle(_ Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "tab", "down", "shift+tab", "up":
		p.setFocus(1 - p.focus)
		return nil, false
	case "enter":
		return p.save()
	}
	var cmd tea.Cmd
	p.inputs[p.focus], cmd = p.inputs[p.focus].Update(msg)
	p.err = ""
	return cmd, false
}

// paste inserta en el campo con el foco el texto pegado.
func (p *datesPopup) paste(msg tea.PasteMsg) tea.Cmd {
	var cmd tea.Cmd
	p.inputs[p.focus], cmd = p.inputs[p.focus].Update(msg)
	p.err = ""
	return cmd
}

// click: un clic en una fila de campo le da el foco.
func (p *datesPopup) click(_, y int) (tea.Cmd, bool) {
	switch y {
	case datesRowStart:
		p.setFocus(dateStart)
	case datesRowStart + 1:
		p.setFocus(dateDue)
	}
	return nil, false
}

// filas del popup (relativas a su esquina) donde están los campos: borde, tarea, vacía, Inicio, Vence.
const datesRowStart = 3

// save valida los dos campos y escribe solo lo que cambió, en una sola escritura de la línea.
func (p *datesPopup) save() (tea.Cmd, bool) {
	var vals [2]string
	for i := range vals {
		v, err := p.resolve(i)
		if err != nil {
			p.setFocus(i)
			p.err = i18n.T("No entiendo esa fecha: ", "Not a date I understand: ") + textwidth.NoControl(strings.TrimSpace(p.inputs[i].Value()))
			return nil, false
		}
		vals[i] = v
	}
	var start, due *string
	if vals[dateStart] != p.orig.Start {
		start = &vals[dateStart]
	}
	if vals[dateDue] != p.orig.Due {
		due = &vals[dateDue]
	}
	if start == nil && due == nil {
		return nil, true // nada cambió
	}
	err := p.c.store.SetTaskDates(p.path, p.line, start, due, p.noteTime())
	switch {
	case errors.Is(err, storage.ErrNoteChanged):
		p.c.reload()
		p.onSaved()
		p.c.setStatus("%s", i18n.T("La nota cambió por fuera: se recargó, vuelve a intentarlo", "The note changed outside: reloaded, try again"))
		return nil, true
	case err != nil:
		p.c.errStatus("No se pudieron guardar las fechas", "Could not save the dates", err)
		return nil, true
	}
	p.c.reload()
	p.onSaved()
	p.c.setStatus("%s", i18n.T("Fechas guardadas", "Dates saved"))
	p.c.dateNotice()
	return nil, true
}

// noteTime es el mtime con el que se cargó la nota (la comprobación de X10 al escribir).
func (p *datesPopup) noteTime() time.Time {
	for _, n := range p.c.notes {
		if n.Path == p.path {
			return n.ModTime
		}
	}
	return time.Time{}
}

// resolved es lo que se muestra al lado del campo i: la fecha resuelta con su día de la semana, "sin fecha" o que no se entiende.
func (p *datesPopup) resolved(i int) string {
	v, err := p.resolve(i)
	switch {
	case err != nil:
		return lipgloss.NewStyle().Foreground(theme.ColorRed).Render("✗ " + i18n.T("no entiendo", "not understood"))
	case v == "":
		return dim("→ " + i18n.T("sin fecha", "no date"))
	}
	d, _ := time.Parse("2006-01-02", v) // en UTC: el día de la semana no depende de la zona horaria
	return lipgloss.NewStyle().Foreground(theme.ColorGreen).Render("→ " + v + " (" + storage.WeekdayName(d.Weekday(), i18n.CurrentLanguage()) + ")")
}

func (p *datesPopup) render(l Layout) string {
	w := popupWidth(l, 62)
	label := func(i int, name string) string {
		marker := "  "
		if p.focus == i {
			marker = accent("▸ ")
		}
		p.inputs[i].SetWidth(12)
		field := lipgloss.NewStyle().Foreground(theme.ColorText).Render(p.inputs[i].View())
		return marker + dim(textwidth.Pad(name, 7)) + field + "  " + p.resolved(i)
	}
	lines := []string{
		dim(i18n.T("Tarea: ", "Task: ")) + textwidth.Truncate(textwidth.NoControl(storage.CleanTaskText(p.text)), w-14, "…"),
		"",
		label(dateStart, i18n.T("Inicio", "Start")),
		label(dateDue, i18n.T("Vence", "Due")),
		"",
		dim(textwidth.Truncate(i18n.T("hoy · mañana · +3d · +1w · lunes · vacío: quitar", "today · tomorrow · +3d · +1w · monday · empty: remove"), w-6, "…")),
	}
	switch {
	case p.err != "":
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorRed).Render(textwidth.Truncate(p.err, w-6, "…")))
	case p.startAfterDue():
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorYellow).Render(textwidth.Truncate(i18n.T("Aviso: el inicio es posterior al vencimiento (se guarda igual)", "Warning: the start is after the due date (it is saved anyway)"), w-6, "…")))
	default:
		lines = append(lines, "")
	}
	return theme.RenderPopup(i18n.T("Fechas", "Dates"), "[Tab] "+i18n.T("cambiar", "switch")+" · [Enter] OK · "+escHint, lines, w)
}

// startAfterDue dice si, con lo escrito, el inicio queda después del vencimiento: es un aviso, no impide guardar.
func (p *datesPopup) startAfterDue() bool {
	start, err1 := p.resolve(dateStart)
	due, err2 := p.resolve(dateDue)
	return err1 == nil && err2 == nil && start != "" && due != "" && start > due
}
