package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
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
		return i18n.T("Pendientes", "Pending")
	case TaskFilterDone:
		return i18n.T("Completadas", "Completed")
	default:
		return i18n.T("Todas", "All")
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
			emptyText = i18n.T("  ¡Sin tareas pendientes! 🎉", "  No pending tasks! 🎉")
		case TaskFilterDone:
			emptyText = i18n.T("  (Sin tareas completadas aún)", "  (No completed tasks yet)")
		default:
			emptyText = i18n.T("  (Sin tareas. Usa - [ ] en tus notas)", "  (No tasks. Use - [ ] in your notes)")
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

		contentWidth := width - 2
		if contentWidth < 4 {
			contentWidth = 4
		}
		rowText := fmt.Sprintf("%s%s %s %s", cursor, checkbox, textStyle, originLabel)
		rowText = ansi.Truncate(rowText, contentWidth, "")

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
		empty := theme.NormalItem.Copy().Italic(true).Render(i18n.T("Selecciona una tarea para ver detalles...", "Select a task to view details..."))
		return borderStyle.Width(width).Height(height).Render(empty)
	}

	var rows []string

	// Estado
	var statusLine string
	if task.Done {
		statusLine = theme.NormalItem.Copy().Foreground(theme.ColorGreen).Bold(true).Render("  " + i18n.T("☑ COMPLETADA", "☑ COMPLETED"))
	} else {
		statusLine = theme.NormalItem.Copy().Foreground(theme.ColorYellow).Bold(true).Render("  " + i18n.T("☐ PENDIENTE", "☐ PENDING"))
	}
	rows = append(rows, statusLine)
	rows = append(rows, "")

	// Texto de la tarea
	taskLabel := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Bold(true).Render("  " + i18n.T("Tarea:", "Task:"))
	rows = append(rows, taskLabel)
	rows = append(rows, fmt.Sprintf("  %s", theme.NormalItem.Render(task.Text)))
	rows = append(rows, "")

	// Nota de origen
	noteLabel := theme.NormalItem.Copy().Foreground(theme.ColorBlue).Bold(true).Render("  " + i18n.T("Nota:", "Note:"))
	rows = append(rows, noteLabel)
	rows = append(rows, fmt.Sprintf("    %s", theme.NormalItem.Render(task.NoteTitle)))
	rows = append(rows, "")

	// Línea en el archivo
	lineLabel := theme.NormalItem.Copy().Foreground(theme.ColorMauve).Render(fmt.Sprintf("  %s: %d", i18n.T("Línea", "Line"), task.Line))
	rows = append(rows, lineLabel)
	rows = append(rows, "")

	// Tip
	tipLabel := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
		Render(i18n.T("  Presiona Enter para abrir la nota en el editor", "  Press Enter to open note in editor"))
	rows = append(rows, tipLabel)

	content := strings.Join(rows, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}
