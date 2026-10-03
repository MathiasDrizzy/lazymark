package views

import (
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
	nerdStart, nerdDue, nerdDone = "", "", ""
	textStart, textDue, textDone = "▸", "◷", "✓"
)

// DateIcons indica si se dibujan los glifos de Nerd Font (true) o los símbolos de texto (false). La fija la app según la
// config (nerd_font).
var DateIcons = true

// DateGlyph es el glifo con el que se dibuja el campo f.
func DateGlyph(f storage.DateField) string {
	g := [3][2]string{{nerdStart, textStart}, {nerdDue, textDue}, {nerdDone, textDone}}[f]
	if DateIcons {
		return g[0]
	}
	return g[1]
}

var emojiGlyph = [...]storage.DateField{storage.DateStart, storage.DateDue, storage.DateDone}

// ReplaceDateEmoji cambia los emojis de fecha (🛫 📅 ✅, con su selector de variación U+FE0F) de un texto que se va a dibujar por
// los glifos monocromos, antes de medirlo (un emoji ocupa 2 celdas y el glifo 1). El color lo da el texto que lo rodea.
func ReplaceDateEmoji(s string) string {
	if !strings.ContainsAny(s, "🛫📅✅") {
		return s
	}
	pairs := make([]string, 0, 12)
	for _, f := range emojiGlyph {
		pairs = append(pairs, f.Emoji()+"️", DateGlyph(f), f.Emoji(), DateGlyph(f))
	}
	return strings.NewReplacer(pairs...).Replace(s)
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

// PanelDates arma las fechas que se muestran junto al texto en el panel Tareas: el vencimiento (en rojo si venció) y, si la
// tarea está hecha, cuándo; "" si no tiene ninguna.
func PanelDates(d storage.Dates, done bool, today string) string {
	var parts []string
	if d.Due != "" {
		parts = append(parts, DatePart(storage.DateDue, d.Due, storage.Overdue(done, d.Due, today)))
	}
	if done && d.Done != "" {
		parts = append(parts, DatePart(storage.DateDone, d.Done, false))
	}
	return strings.Join(parts, " ")
}
