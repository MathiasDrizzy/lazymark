package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// TaskFilter define el tipo de filtro aplicado a las tareas
type TaskFilter int

const (
	TaskFilterAll      TaskFilter = iota // Todas las tareas
	TaskFilterPending                    // Solo pendientes (- [ ])
	TaskFilterDone                       // Solo completadas (- [x])
)

// FlatTask es una tarea aplanada con contexto de la nota padre
type FlatTask struct {
	storage.Task
	NotePath string
}

// CollectTasks extrae todas las tareas de todas las notas y las aplana en una lista única
func CollectTasks(notes []storage.Note, filter TaskFilter) []FlatTask {
	var tasks []FlatTask
	for _, note := range notes {
		for _, task := range note.Tasks {
			switch filter {
			case TaskFilterPending:
				if task.Done {
					continue
				}
			case TaskFilterDone:
				if !task.Done {
					continue
				}
			}
			tasks = append(tasks, FlatTask{
				Task:     task,
				NotePath: note.Path,
			})
		}
	}
	return tasks
}

// TaskFilterLabel devuelve la etiqueta del filtro actual
func TaskFilterLabel(filter TaskFilter) string {
	switch filter {
	case TaskFilterPending:
		return "Pendientes"
	case TaskFilterDone:
		return "Completadas"
	default:
		return "Todas"
	}
}

// RenderTaskList genera el panel izquierdo con la lista de tareas y registra clics
func RenderTaskList(tasks []FlatTask, selectedIndex int, filter TaskFilter, width, height int, active bool, ht *mouse.HitTester, offsetY int) string {
	var rows []string

	// Cabecera con filtro actual
	filterLabel := TaskFilterLabel(filter)
	pendingCount := 0
	doneCount := 0
	for _, t := range tasks {
		if t.Done {
			doneCount++
		} else {
			pendingCount++
		}
	}

	var headerText string
	switch filter {
	case TaskFilterPending:
		headerText = fmt.Sprintf(" ☐ %s (%d)", filterLabel, len(tasks))
	case TaskFilterDone:
		headerText = fmt.Sprintf(" ☑ %s (%d)", filterLabel, len(tasks))
	default:
		headerText = fmt.Sprintf(" 📋 %s (%d)  ☐ %d  ☑ %d", filterLabel, len(tasks), pendingCount, doneCount)
	}
	header := theme.SelectedItem.Copy().Foreground(theme.ColorTeal).Render(headerText)
	rows = append(rows, header)

	if len(tasks) == 0 {
		var emptyText string
		switch filter {
		case TaskFilterPending:
			emptyText = "  ¡Sin tareas pendientes! 🎉"
		case TaskFilterDone:
			emptyText = "  (Sin tareas completadas aún)"
		default:
			emptyText = "  (Sin tareas. Usa - [ ] en tus notas)"
		}
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render(emptyText)
		rows = append(rows, emptyMsg)
	}

	usableHeight := height - 3 // -2 border -1 header
	if usableHeight < 1 {
		usableHeight = 1
	}

	startIdx := 0
	if selectedIndex >= usableHeight {
		startIdx = selectedIndex - usableHeight + 1
	}
	endIdx := startIdx + usableHeight
	if endIdx > len(tasks) {
		endIdx = len(tasks)
	}

	for i := startIdx; i < endIdx; i++ {
		task := tasks[i]
		isSelected := i == selectedIndex

		cursor := "  "
		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
		}

		var checkbox string
		var textStyle string
		if task.Done {
			checkbox = theme.NormalItem.Copy().Foreground(theme.ColorGreen).Render("☑")
			textStyle = theme.TaskDone.Render(task.Text)
		} else {
			checkbox = theme.NormalItem.Copy().Foreground(theme.ColorYellow).Render("☐")
			textStyle = theme.TaskPending.Render(task.Text)
		}

		// Truncar texto de tarea si es necesario
		maxLen := width - 12
		if maxLen < 5 {
			maxLen = 5
		}
		displayText := task.Text
		if len(displayText) > maxLen {
			displayText = displayText[:maxLen-3] + "..."
			if task.Done {
				textStyle = theme.TaskDone.Render(displayText)
			} else {
				textStyle = theme.TaskPending.Render(displayText)
			}
		}

		// Nota origen en sutil
		originLabel := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("(%s)", task.NoteTitle))

		rowText := fmt.Sprintf("%s%s %s %s", cursor, checkbox, textStyle, originLabel)

		// Registrar zona de clic
		if ht != nil {
			rowY := offsetY + (i - startIdx) + 2 // +1 border +1 header
			ht.Register(fmt.Sprintf("task-%d", i), mouse.ZoneTask, 0, rowY, width, rowY, i, task.NotePath)
		}

		rows = append(rows, rowText)
	}

	content := strings.Join(rows, "\n")

	borderStyle := theme.InactivePanelBorder
	if active {
		borderStyle = theme.ActivePanelBorder
	}

	return borderStyle.
		Width(width).
		Height(height).
		Render(content)
}

// RenderTaskPreview muestra los detalles de la tarea seleccionada y su contexto
func RenderTaskPreview(task *FlatTask, width, height int, active bool) string {
	borderStyle := theme.InactivePanelBorder
	if active {
		borderStyle = theme.ActivePanelBorder
	}

	if task == nil {
		empty := theme.NormalItem.Copy().Italic(true).Render("Selecciona una tarea para ver detalles...")
		return borderStyle.Width(width).Height(height).Render(empty)
	}

	var rows []string

	// Estado
	var statusLine string
	if task.Done {
		statusLine = theme.NormalItem.Copy().Foreground(theme.ColorGreen).Bold(true).Render("  ☑ COMPLETADA")
	} else {
		statusLine = theme.NormalItem.Copy().Foreground(theme.ColorYellow).Bold(true).Render("  ☐ PENDIENTE")
	}
	rows = append(rows, statusLine)
	rows = append(rows, "")

	// Texto de la tarea
	taskLabel := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Bold(true).Render("  Tarea:")
	rows = append(rows, taskLabel)
	rows = append(rows, fmt.Sprintf("  %s", theme.NormalItem.Render(task.Text)))
	rows = append(rows, "")

	// Nota de origen
	noteLabel := theme.NormalItem.Copy().Foreground(theme.ColorBlue).Bold(true).Render("  Nota:")
	rows = append(rows, noteLabel)
	rows = append(rows, fmt.Sprintf("  📝 %s", theme.NormalItem.Render(task.NoteTitle)))
	rows = append(rows, "")

	// Línea en el archivo
	lineLabel := theme.NormalItem.Copy().Foreground(theme.ColorMauve).Render(fmt.Sprintf("  Línea: %d", task.Line))
	rows = append(rows, lineLabel)
	rows = append(rows, "")

	// Tip
	tipLabel := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
		Render("  Presiona Enter para abrir la nota en el editor")
	rows = append(rows, tipLabel)

	content := strings.Join(rows, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}
