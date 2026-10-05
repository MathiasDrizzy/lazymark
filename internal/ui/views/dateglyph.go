package views

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// En el archivo las fechas de una tarea son emojis (🛫 📅 ✅: el formato de Obsidian Tasks), pero en pantalla nunca se dibujan
// así: un emoji a color rompe la estética de la interfaz, que usa glifos monocromos de Nerd Font (los de las carpetas y las
// notas) con los colores del tema. Estos son los glifos de la misma familia (Font Awesome): un cohete para el inicio, un
// calendario para el vencimiento y un check para la completada. Sin Nerd Font (nerd_font = false) caen a símbolos de texto.
const (
	// Font Awesome de Nerd Font, por su código (los caracteres no se ven en todos los editores): cohete, calendario, check, reloj de arena y más
	nerdStart, nerdDue, nerdDone, nerdScheduled, nerdCreated = "\uf135", "\uf073", "\uf00c", "\uf252", "\uf067"
	textStart, textDue, textDone, textScheduled, textCreated = "▸", "◷", "✓", "◑", "+"
)

// DateIcons indica si se dibujan los glifos de Nerd Font (true) o los símbolos de texto (false). La fija la app según la
// config (nerd_font).
var DateIcons = true

// DateGlyph es el glifo con el que se dibuja el campo f.
func DateGlyph(f storage.DateField) string {
	g := [...][2]string{{nerdStart, textStart}, {nerdDue, textDue}, {nerdDone, textDone}, {nerdScheduled, textScheduled}, {nerdCreated, textCreated}}[f]
	if DateIcons {
		return g[0]
	}
	return g[1]
}

// dataviewField reconoce un campo de fecha en el formato Dataview, entre corchetes o paréntesis, para dibujarlo como glifo y fecha.
var dataviewField = regexp.MustCompile(`\[[ \t]*(start|due|completion|scheduled|created)[ \t]*::[ \t]*(\d{4}-\d{2}-\d{2})[ \t]*\]|\([ \t]*(start|due|completion|scheduled|created)[ \t]*::[ \t]*(\d{4}-\d{2}-\d{2})[ \t]*\)`)

// ReplaceDateEmoji cambia las fechas de un texto que se va a dibujar por los glifos monocromos, antes de medirlo (un emoji ocupa 2 celdas y el glifo
// 1): los emojis de Obsidian Tasks (🛫 📅 ✅ ⏳ ➕, con su selector de variación U+FE0F) y los campos Dataview (`[due:: 2026-05-10]` se dibuja
// `<glifo> 2026-05-10`). El color lo da el texto que lo rodea. En pantalla nunca hay emojis de fecha ni la sintaxis Dataview.
func ReplaceDateEmoji(s string) string {
	if strings.ContainsAny(s, "🛫📅✅⏳➕") {
		pairs := make([]string, 0, 20)
		for f := storage.DateStart; f <= storage.DateCreated; f++ {
			pairs = append(pairs, f.Emoji()+"️", DateGlyph(f), f.Emoji(), DateGlyph(f))
		}
		s = strings.NewReplacer(pairs...).Replace(s)
	}
	if strings.Contains(s, "::") {
		s = dataviewField.ReplaceAllStringFunc(s, func(m string) string {
			sm := dataviewField.FindStringSubmatch(m)
			key, date := sm[1], sm[2]
			if key == "" {
				key, date = sm[3], sm[4]
			}
			for f := storage.DateStart; f <= storage.DateCreated; f++ {
				if f.Key() == key {
					return DateGlyph(f) + " " + date
				}
			}
			return m
		})
	}
	return s
}

// dateStyles son los colores de cada fecha: la vencida con el color de error del tema, la completada con el de éxito y el
// resto atenuado.
func dateStyle(f storage.DateField, overdue bool) lipgloss.Style {
	switch {
	case f == storage.DateDue && overdue:
		return lipgloss.NewStyle().Foreground(theme.ColorRed).Bold(true)
	case f == storage.DateDone:
		return lipgloss.NewStyle().Foreground(theme.ColorGreen)
	}
	return lipgloss.NewStyle().Foreground(theme.ColorOverlay0)
}

// DatePart arma una fecha para dibujarla: el glifo y la fecha, con su color.
func DatePart(f storage.DateField, date string, overdue bool) string {
	return dateStyle(f, overdue).Render(DateGlyph(f) + " " + date)
}

// PanelDates arma las fechas que se muestran junto al texto en el panel Tareas: solo el vencimiento (en rojo si venció). El
// panel es angosto y la fecha de completada, que se agrega sola al marcar la tarea, se ve en las tarjetas y en la vista previa;
// "" si no tiene vencimiento.
func PanelDates(d storage.Dates, done bool, today string) string {
	if d.Due == "" {
		return ""
	}
	return DatePart(storage.DateDue, d.Due, storage.Overdue(done, d.Due, today))
}
