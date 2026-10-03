package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// KanbanOptions son las opciones de dibujo del tablero.
type KanbanOptions struct {
	Cards bool   // tarjetas como rectángulos con fechas (false: la vista compacta, una fila por tarea)
	Today string // la fecha de hoy (AAAA-MM-DD), para resaltar las vencidas
}

// Tamaños mínimos para que las tarjetas quepan: por debajo se usa la vista compacta.
const (
	minCardInner  = 22 // ancho útil de la columna (sin su borde ni su margen)
	minCardHeight = 10 // alto del tablero
)

// cardIndent son las columnas que ocupan el cursor (▸ ) y la casilla (☐ ) antes del texto de una tarjeta.
const cardIndent = 4

// cardsFit indica si las tarjetas caben en columnas de ese ancho útil y en un tablero de esa altura.
func cardsFit(innerWidth, height int) bool {
	return innerWidth >= minCardInner && height >= minCardHeight
}

// wrapLines parte text en como mucho max líneas de ancho width: por palabras en las primeras y, en la última, lo que sobre
// entero, cortado con "…" si no cabe. Una palabra más ancha que la línea (o un texto sin espacios, como el japonés) se corta por
// caracteres.
func wrapLines(text string, width, max int) []string {
	words := strings.Fields(text)
	var lines []string
	i := 0
	for len(lines) < max-1 && i < len(words) {
		line := ""
		for i < len(words) {
			w := words[i]
			if line == "" {
				if textwidth.Width(w) > width {
					head, rest := splitWidth(w, width)
					line, words[i] = head, rest
					break
				}
				line = w
				i++
				continue
			}
			if textwidth.Width(line)+1+textwidth.Width(w) > width {
				break
			}
			line += " " + w
			i++
		}
		lines = append(lines, line)
	}
	if rest := strings.Join(words[i:], " "); rest != "" || len(lines) == 0 {
		lines = append(lines, textwidth.Truncate(rest, width, "…"))
	}
	return lines
}

// splitWidth parte s en el prefijo más largo que cabe en width columnas y el resto.
func splitWidth(s string, width int) (head, rest string) {
	w := 0
	for i, r := range s {
		rw := textwidth.Width(string(r))
		if w+rw > width {
			return s[:i], s[i:]
		}
		w += rw
	}
	return s, ""
}

// cardDates arma la línea de fechas de una tarjeta (glifo y fecha: inicio, vencimiento, completada), con el vencimiento en el color de
// error del tema si está vencida; "" si no tiene fechas. width es el ancho disponible: si no caben todas, se quita
// primero la de completada y después la de inicio.
func cardDates(d storage.Dates, done bool, today string, width int) string {
	overdue := storage.Overdue(done, d.Due, today)
	part := func(f storage.DateField, date string, over bool) string {
		if date == "" {
			return ""
		}
		return DatePart(f, date, over)
	}
	build := func(start, due, completed bool) string {
		var parts []string
		if start {
			if p := part(storage.DateStart, d.Start, false); p != "" {
				parts = append(parts, p)
			}
		}
		if due {
			if p := part(storage.DateDue, d.Due, overdue); p != "" {
				parts = append(parts, p)
			}
		}
		if completed {
			if p := part(storage.DateDone, d.Done, false); p != "" {
				parts = append(parts, p)
			}
		}
		return strings.Join(parts, " ")
	}
	for _, combo := range [][3]bool{{true, true, true}, {true, true, false}, {false, true, false}, {false, false, true}} {
		if s := build(combo[0], combo[1], combo[2]); s != "" && textwidth.Width(s) <= width {
			return s
		}
	}
	return ""
}

// cardTextWidth es el ancho del texto de una tarjeta de ancho w: sin el borde, el espacio de cada lado, el cursor y la casilla.
func cardTextWidth(w int) int { return w - 4 - cardIndent }

// cardHeight es el alto de una tarjeta de ancho w: 2 de borde, 1 o 2 líneas de texto, la nota de origen y, si tiene fechas, su línea.
func cardHeight(c KanbanCard, w int) int {
	h := 2 + len(wrapLines(ReplaceDateEmoji(c.CleanText), cardTextWidth(w), 2)) + 1
	if c.Task.Dates != (storage.Dates{}) {
		h++
	}
	return h
}

// renderCard dibuja una tarjeta de ancho w como un rectángulo de borde redondeado: el texto (hasta 2 líneas), la nota de origen
// y las fechas. Devuelve sus filas.
func renderCard(c KanbanCard, w int, doneCol bool, midCol bool, selected, active, dragged bool, today string) []string {
	inner := w - 4 // borde y un espacio a cada lado
	borderColor := theme.ColorOverlay0
	switch {
	case dragged:
		borderColor = theme.ColorPeach
	case selected && active:
		borderColor = theme.ColorBlue
	case selected:
		borderColor = theme.ColorSubtext0
	}
	border := lipgloss.NewStyle().Foreground(borderColor)
	if selected || dragged {
		border = border.Bold(true)
	}
	muted := lipgloss.NewStyle().Foreground(theme.ColorOverlay0)

	mark, markColor := "☐", theme.ColorSubtext0
	switch {
	case doneCol:
		mark, markColor = "☑", theme.ColorGreen
	case midCol:
		mark, markColor = "◓", theme.ColorYellow
	}
	cursor := "  " // la tarjeta seleccionada lleva ▸ (también sin color), la que se arrastra ⇢
	switch {
	case dragged:
		cursor = "⇢ "
	case selected:
		cursor = "▸ "
	}
	textStyle := lipgloss.NewStyle().Foreground(theme.ColorText)
	switch {
	case doneCol:
		textStyle = theme.TaskDone
	case midCol:
		textStyle = lipgloss.NewStyle().Foreground(theme.ColorPeach)
	}
	if selected || dragged {
		textStyle = textStyle.Bold(true)
	}

	row := func(content string) string {
		return border.Render("│") + " " + textwidth.Pad(content, inner) + " " + border.Render("│")
	}
	rows := []string{border.Render("╭" + strings.Repeat("─", w-2) + "╮")}
	for i, l := range wrapLines(ReplaceDateEmoji(c.CleanText), cardTextWidth(w), 2) {
		prefix := "    "
		if i == 0 {
			prefix = border.Render(cursor) + lipgloss.NewStyle().Foreground(markColor).Render(mark) + " "
		}
		rows = append(rows, row(prefix+textStyle.Render(l)))
	}
	rows = append(rows, row(muted.Render(textwidth.Truncate("· "+ReplaceDateEmoji(textwidth.NoControl(c.NoteTitle)), inner, "…"))))
	if c.Task.Dates != (storage.Dates{}) {
		rows = append(rows, row(cardDates(c.Task.Dates, c.Task.Done, today, inner)))
	}
	rows = append(rows, border.Render("╰"+strings.Repeat("─", w-2)+"╯"))
	return rows
}

// renderCardColumn dibuja las tarjetas de una columna en un área de usable filas: arranca de la primera tarjeta que deja
// la seleccionada a la vista. Registra una zona de clic del alto de cada tarjeta visible.
func renderCardColumn(cards []KanbanCard, col, sel int, width, usable int, doneCol, midCol, active bool, drag KanbanDrag, today string, ht *mouse.HitTester, x0, y0 int) []string {
	inner := width - 4 // el ancho que RenderBoxWithTitle deja al contenido, que es el de cada tarjeta
	start := 0
	for start < sel {
		h := 0
		for i := start; i <= sel; i++ {
			h += cardHeight(cards[i], inner)
		}
		if h <= usable {
			break
		}
		start++
	}
	var lines []string
	for i := start; i < len(cards); i++ {
		h := cardHeight(cards[i], inner)
		if len(lines)+h > usable {
			if len(lines) == 0 { // una tarjeta más alta que el área: se recorta, y sigue siendo clicable
				card := renderCard(cards[i], inner, doneCol, midCol, i == sel, active, drag.Active && col == drag.Col && i == drag.Idx, today)
				lines = append(lines, card[:min(len(card), usable)]...)
				if ht != nil {
					ht.Register(fmt.Sprintf("kanban-card-%d-%d", col, i), mouse.ZoneKanbanCard, x0+2, y0, x0+width-3, y0+len(lines)-1, i,
						fmt.Sprintf("%d|%d|%s", col, cards[i].Task.Line, cards[i].NotePath))
				}
			}
			break
		}
		dragged := drag.Active && col == drag.Col && i == drag.Idx
		if ht != nil {
			y := y0 + len(lines)
			ht.Register(fmt.Sprintf("kanban-card-%d-%d", col, i), mouse.ZoneKanbanCard, x0+2, y, x0+width-3, y+h-1, i,
				fmt.Sprintf("%d|%d|%s", col, cards[i].Task.Line, cards[i].NotePath))
		}
		lines = append(lines, renderCard(cards[i], inner, doneCol, midCol, i == sel || (drag.Active && col == drag.Col && drag.Target == col && i == drag.TargetIdx && i != drag.Idx), active, dragged, today)...)
	}
	return lines
}
