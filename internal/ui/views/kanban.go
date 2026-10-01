package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// KanbanCard representa una tarjeta individual en el tablero Kanban
type KanbanCard struct {
	Task      storage.Task
	NotePath  string
	NoteTitle string
	CleanText string
	Stage     storage.TaskStage
}

// KanbanBoard agrupa las tarjetas por sus columnas/etapas
type KanbanBoard struct {
	Todo  []KanbanCard
	Doing []KanbanCard
	Done  []KanbanCard
}

// ColumnCards devuelve el slice de tarjetas de la columna indicada (0=Todo, 1=Doing, 2=Done)
func (b *KanbanBoard) ColumnCards(col int) []KanbanCard {
	switch col {
	case 0:
		return b.Todo
	case 1:
		return b.Doing
	case 2:
		return b.Done
	default:
		return nil
	}
}

// TotalCards devuelve el total de tarjetas presentes en el tablero
func (b *KanbanBoard) TotalCards() int {
	return len(b.Todo) + len(b.Doing) + len(b.Done)
}

// CollectKanban extrae todas las tareas de todas las notas y las organiza en columnas Kanban
func CollectKanban(notes []storage.Note) KanbanBoard {
	var board KanbanBoard
	for _, note := range notes {
		for _, task := range note.Tasks {
			stage := storage.GetTaskStage(task)
			card := KanbanCard{
				Task:      task,
				NotePath:  note.Path,
				NoteTitle: note.Title,
				CleanText: storage.CleanTaskText(task.Text),
				Stage:     stage,
			}
			switch stage {
			case storage.StageTodo:
				board.Todo = append(board.Todo, card)
			case storage.StageDoing:
				board.Doing = append(board.Doing, card)
			case storage.StageDone:
				board.Done = append(board.Done, card)
			}
		}
	}
	return board
}

// RenderKanban genera la vista de 3 columnas del tablero Kanban estilo Taskell/Kaban
func RenderKanban(board KanbanBoard, activeCol int, selectedRows [3]int, width, height int, ht *mouse.HitTester, offsetY int) string {
	if width < 30 {
		width = 30
	}
	if height < 5 {
		height = 5
	}

	colW0 := width / 3
	colW1 := width / 3
	colW2 := width - (colW0 + colW1) // Absorber el remanente de división entera

	colWidths := [3]int{colW0, colW1, colW2}
	colTitles := [3]string{
		i18n.T("[1] Por Hacer", "[1] To Do"),
		i18n.T("[2] En Progreso", "[2] In Progress"),
		i18n.T("[3] Completado", "[3] Done"),
	}

	colEmptyMsgs := [3]string{
		i18n.T("  (Sin tareas pendientes)", "  (No pending tasks)"),
		i18n.T("  (Sin tareas en curso. Mueve con L)", "  (No tasks in progress. Move with L)"),
		i18n.T("  (Sin tareas completadas)", "  (No completed tasks)"),
	}

	usableHeight := height - 2
	if usableHeight < 1 {
		usableHeight = 1
	}

	// NOTA DE MEMORIA: En HitTester, registrar primero las zonas base de columna (menor prioridad z-index)
	currentX := 0
	if ht != nil {
		for c := 0; c < 3; c++ {
			w := colWidths[c]
			ht.Register(
				fmt.Sprintf("kanban-col-bg-%d", c),
				mouse.ZoneKanbanCol,
				currentX,
				offsetY,
				currentX+w-1,
				offsetY+height-1,
				c,
				fmt.Sprintf("%d", c),
			)
			currentX += w
		}
	}

	var renderedCols []string
	colStartX := 0

	for c := 0; c < 3; c++ {
		w := colWidths[c]
		cards := board.ColumnCards(c)
		isActive := c == activeCol
		selIdx := selectedRows[c]
		if selIdx < 0 {
			selIdx = 0
		}
		if selIdx >= len(cards) && len(cards) > 0 {
			selIdx = len(cards) - 1
		}

		badge := "0 of 0"
		if len(cards) > 0 {
			badge = fmt.Sprintf("%d of %d", selIdx+1, len(cards))
		}

		var lines []string
		if len(cards) == 0 {
			emptyStyle := theme.NormalItem.Copy().Italic(true)
			lines = append(lines, emptyStyle.Render(colEmptyMsgs[c]))
		} else {
			startIdx := 0
			if selIdx >= usableHeight {
				startIdx = selIdx - usableHeight + 1
			}
			endIdx := startIdx + usableHeight
			if endIdx > len(cards) {
				endIdx = len(cards)
			}

			contentWidth := w - 4
			if contentWidth < 4 {
				contentWidth = 4
			}

			for i := startIdx; i < endIdx; i++ {
				card := cards[i]
				isSelected := i == selIdx

				var selStyle lipgloss.Style
				if isSelected {
					if isActive {
						selStyle = theme.SelectedLineActive
					} else {
						selStyle = theme.SelectedLineInactive
					}
				}

				displayText := card.CleanText
				noteOrigin := card.NoteTitle

				var rowText string
				if isSelected {
					cursor := "▸ "
					checkbox := "☐"
					if card.Stage == storage.StageDone {
						checkbox = "☑"
					} else if card.Stage == storage.StageDoing {
						checkbox = "◓"
					}
					rawLine := fmt.Sprintf("%s%s %s (%s)", cursor, checkbox, displayText, noteOrigin)
					rawLine = ansi.Truncate(rawLine, contentWidth, "")
					lineW := ansi.StringWidth(rawLine)
					if lineW < contentWidth {
						rawLine += strings.Repeat(" ", contentWidth-lineW)
					}
					rowText = selStyle.Render(rawLine)
				} else {
					cursor := "  "
					var checkbox string
					var textStyle string
					switch card.Stage {
					case storage.StageDone:
						checkbox = theme.NormalItem.Copy().Foreground(theme.ColorGreen).Render("☑")
						textStyle = theme.TaskDone.Render(displayText)
					case storage.StageDoing:
						checkbox = theme.NormalItem.Copy().Foreground(theme.ColorYellow).Render("◓")
						textStyle = theme.TaskPending.Copy().Foreground(theme.ColorPeach).Render(displayText)
					default:
						checkbox = theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render("☐")
						textStyle = theme.TaskPending.Render(displayText)
					}
					originLabel := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("(%s)", noteOrigin))
					rowText = fmt.Sprintf("%s%s %s %s", cursor, checkbox, textStyle, originLabel)
					rowText = ansi.Truncate(rowText, contentWidth, "")
				}

				lines = append(lines, rowText)

				// Registrar zona de tarjeta (mayor prioridad z-index, registrada después de col-bg)
				if ht != nil {
					rowY := offsetY + 1 + (i - startIdx)
					ht.Register(
						fmt.Sprintf("kanban-card-%d-%d", c, i),
						mouse.ZoneKanbanCard,
						colStartX+1,
						rowY,
						colStartX+w-2,
						rowY,
						i,
						fmt.Sprintf("%d:%s:%d", c, card.NotePath, card.Task.Line),
					)
				}
			}
		}

		content := strings.Join(lines, "\n")
		colBox := theme.RenderBoxWithTitle(colTitles[c], badge, content, w, height, isActive)
		renderedCols = append(renderedCols, colBox)
		colStartX += w
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
}
