package views

import (
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
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

// glyphSets son los dos juegos de glifos de cada campo (índice de DateField): [0] Nerd Font, [1] texto. Se dibuja uno según DateIcons, pero se reconocen los dos.
var glyphSets = [2][5]string{
	{nerdStart, nerdDue, nerdDone, nerdScheduled, nerdCreated},
	{textStart, textDue, textDone, textScheduled, textCreated},
}

// DateGlyphs son los glifos propios de la configuración (`date_glyphs`), por campo; un vacío es "sin glifo propio": se usa el de Nerd Font o el de texto.
var DateGlyphs [5]string

// DateGlyph es el glifo con el que se dibuja el campo f.
func DateGlyph(f storage.DateField) string {
	if g := DateGlyphs[f]; g != "" {
		return g
	}
	g := [2]string{glyphSets[0][f], glyphSets[1][f]}
	if DateIcons {
		return g[0]
	}
	return g[1]
}

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
	if strings.Contains(s, "::") { // solo los campos que de verdad se leen como fecha (no los wikilinks, ni los links, ni el código en línea)
		spans := storage.DataviewSpans(s)
		var sb strings.Builder
		last := 0
		for _, sp := range spans {
			sb.WriteString(s[last:sp.Start])
			sb.WriteString(DateGlyph(sp.Field) + " " + sp.Date)
			last = sp.End
		}
		sb.WriteString(s[last:])
		s = sb.String()
	}
	return s
}

// DatePart arma una fecha para dibujarla: el glifo y la fecha, con el color de su estado (datecolors.go). done: la tarea está hecha; today: hoy (AAAA-MM-DD).
func DatePart(f storage.DateField, date string, done bool, today string) string {
	return DateStyleFor(f, date, done, today).Render(DateGlyph(f) + " " + date)
}

// PanelDates arma las fechas que se muestran junto al texto en el panel Tareas: solo el vencimiento (con el color de su estado). El
// panel es angosto y la fecha de completada, que se agrega sola al marcar la tarea, se ve en las tarjetas y en la vista previa;
// "" si no tiene vencimiento.
func PanelDates(d storage.Dates, done bool, today string) string {
	if d.Due == "" {
		return ""
	}
	return DatePart(storage.DateDue, d.Due, done, today)
}
