package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderNoteList genera el panel izquierdo con la lista de notas y registra los clics
func RenderNoteList(notes []storage.Note, selectedIndex int, width, height int, active bool, ht *mouse.HitTester, offsetY int) string {
	var rows []string

	if len(notes) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render("  (No hay notas aún. Presiona 'c' para crear una)")
		rows = append(rows, emptyMsg)
	}

	usableHeight := height - 2
	if usableHeight < 1 {
		usableHeight = 1
	}

	startIdx := 0
	if selectedIndex >= usableHeight {
		startIdx = selectedIndex - usableHeight + 1
	}
	endIdx := startIdx + usableHeight
	if endIdx > len(notes) {
		endIdx = len(notes)
	}

	for i := startIdx; i < endIdx; i++ {
		note := notes[i]
		isSelected := i == selectedIndex

		cursor := "  "
		itemStyle := theme.NormalItem
		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
			itemStyle = theme.SelectedItem
		}

		// Título truncado
		maxTitleLen := width - 14
		if maxTitleLen < 5 {
			maxTitleLen = 5
		}
		title := note.Title
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		timeStr := note.ModTime.Format("02 Jan")
		timeBadge := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr)

		rowText := fmt.Sprintf("%s%-*s %s", cursor, maxTitleLen, itemStyle.Render(title), timeBadge)

		// Registrar zona de clic para esta fila
		if ht != nil {
			rowY := offsetY + (i - startIdx) + 1 // +1 por el borde superior
			ht.Register(fmt.Sprintf("note-%d", i), mouse.ZoneNote, 0, rowY, width, rowY, i, note.Path)
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
