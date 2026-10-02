package views

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// KanbanCard es una tarjeta del tablero: una tarea de una nota.
type KanbanCard struct {
	Task      storage.Task
	NotePath  string
	NoteTitle string
	CleanText string
	Column    int // la columna en la que está
}

// KanbanBoard agrupa las tarjetas por columnas. IDs y Titles tienen una entrada por columna.
type KanbanBoard struct {
	IDs    []string
	Titles []string
	Cols   [][]KanbanCard
}

// NumCols es la cantidad de columnas.
func (b *KanbanBoard) NumCols() int { return len(b.Cols) }

// ColumnCards devuelve las tarjetas de la columna col (nil si no existe).
func (b *KanbanBoard) ColumnCards(col int) []KanbanCard {
	if col < 0 || col >= len(b.Cols) {
		return nil
	}
	return b.Cols[col]
}

// TotalCards devuelve el total de tarjetas del tablero.
func (b *KanbanBoard) TotalCards() int {
	n := 0
	for _, c := range b.Cols {
		n += len(c)
	}
	return n
}

// CollectKanban extrae las tareas de todas las notas y las reparte en las columnas cols (con sus títulos).
func CollectKanban(notes []storage.Note, cols storage.Columns, titles []string) KanbanBoard {
	board := KanbanBoard{IDs: append([]string(nil), cols...), Titles: titles, Cols: make([][]KanbanCard, len(cols))}
	for len(board.Titles) < len(cols) {
		board.Titles = append(board.Titles, cols[len(board.Titles)])
	}
	for _, note := range notes {
		for _, task := range note.Tasks {
			col := cols.Of(task)
			board.Cols[col] = append(board.Cols[col], KanbanCard{
				Task:      task,
				NotePath:  note.Path,
				NoteTitle: note.Title,
				CleanText: storage.CleanTaskText(task.Text),
				Column:    col,
			})
		}
	}
	return board
}

// KanbanDrag describe un arrastre en curso: la tarjeta (Col, Idx) que se lleva y la columna de destino bajo el puntero.
type KanbanDrag struct {
	Active   bool
	Col, Idx int
	Target   int
}

// titleCase pone en mayúscula la primera letra de cada palabra ("En progreso" → "En Progreso").
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// RenderKanban genera la vista del tablero: una caja por columna, con sus tarjetas. selectedRows tiene la
// tarjeta seleccionada de cada columna y drag, el arrastre en curso (la tarjeta que se lleva va resaltada y
// la columna de destino, marcada).
func RenderKanban(board KanbanBoard, activeCol int, selectedRows []int, width, height int, ht *mouse.HitTester, offsetY int, drag KanbanDrag) string {
	n := board.NumCols()
	if n == 0 {
		return ""
	}
	width = max(width, 10*n)
	height = max(height, 5)

	colWidths := make([]int, n)
	for c := range colWidths {
		colWidths[c] = width / n
	}
	colWidths[n-1] += width - (width/n)*n // absorbe el remanente de la división entera

	usableHeight := max(height-2, 1)
	emptyMsg := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Italic(true).Render(i18n.T("  (Sin tareas)", "  (No tasks)"))

	// registrar primero las zonas de columna (menor prioridad), después las de tarjeta
	if ht != nil {
		x := 0
		for c := 0; c < n; c++ {
			ht.Register(fmt.Sprintf("kanban-col-bg-%d", c), mouse.ZoneKanbanCol, x, offsetY, x+colWidths[c]-1, offsetY+height-1, c, fmt.Sprintf("%d", c))
			x += colWidths[c]
		}
	}

	doneCol := board.doneIndex()
	var rendered []string
	colStartX := 0
	for c := 0; c < n; c++ {
		w := colWidths[c]
		cards := board.ColumnCards(c)
		isActive := c == activeCol
		target := drag.Active && c == drag.Target
		selIdx := 0
		if c < len(selectedRows) {
			selIdx = selectedRows[c]
		}
		selIdx = max(0, min(selIdx, len(cards)-1))

		badge := "0 of 0"
		if len(cards) > 0 {
			badge = fmt.Sprintf("%d of %d", selIdx+1, len(cards))
		}

		var lines []string
		if len(cards) == 0 {
			lines = append(lines, emptyMsg)
		} else {
			startIdx := 0
			if selIdx >= usableHeight {
				startIdx = selIdx - usableHeight + 1
			}
			endIdx := min(startIdx+usableHeight, len(cards))
			contentWidth := max(w-4, 4)

			for i := startIdx; i < endIdx; i++ {
				card := cards[i]
				isSelected := i == selIdx
				dragged := drag.Active && c == drag.Col && i == drag.Idx

				mark := "☐"
				markColor := theme.ColorSubtext0
				switch {
				case c == doneCol:
					mark, markColor = "☑", theme.ColorGreen
				case c != 0:
					mark, markColor = "◓", theme.ColorYellow
				}
				var rowText string
				if isSelected || dragged {
					style := theme.SelectedLineInactive
					if isActive || dragged {
						style = theme.SelectedLineActive
					}
					cursor := "▸ "
					if dragged {
						cursor = "⇢ "
					}
					raw := textwidth.Truncate(fmt.Sprintf("%s%s %s (%s)", cursor, mark, card.CleanText, card.NoteTitle), contentWidth, "")
					if lw := textwidth.Width(raw); lw < contentWidth {
						raw += strings.Repeat(" ", contentWidth-lw)
					}
					rowText = style.Render(raw)
				} else {
					text := theme.TaskPending.Render(card.CleanText)
					switch {
					case c == doneCol:
						text = theme.TaskDone.Render(card.CleanText)
					case c != 0:
						text = theme.TaskPending.Foreground(theme.ColorPeach).Render(card.CleanText)
					}
					origin := theme.NormalItem.Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("(%s)", card.NoteTitle))
					rowText = textwidth.Truncate(fmt.Sprintf("  %s %s %s", theme.NormalItem.Foreground(markColor).Render(mark), text, origin), contentWidth, "")
				}
				lines = append(lines, rowText)

				if ht != nil { // las tarjetas se registran después de las columnas: ganan al clic
					rowY := offsetY + 1 + (i - startIdx)
					ht.Register(fmt.Sprintf("kanban-card-%d-%d", c, i), mouse.ZoneKanbanCard, colStartX+1, rowY, colStartX+w-2, rowY, i,
						fmt.Sprintf("%d:%s:%d", c, card.NotePath, card.Task.Line))
				}
			}
		}

		title := fmt.Sprintf("[%d] %s", c+1, titleCase(board.Titles[c]))
		if target {
			title = "▸ " + title // la columna de destino del arrastre
		}
		box := theme.RenderBoxWithTitle(title, badge, strings.Join(lines, "\n"), w, height, isActive || target)
		rendered = append(rendered, box)
		colStartX += w
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// doneIndex es la columna de las tareas hechas ("done", o la última).
func (b *KanbanBoard) doneIndex() int { return storage.Columns(b.IDs).DoneIndex() }
