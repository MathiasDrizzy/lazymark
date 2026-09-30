package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderNoteList genera el panel izquierdo con el explorador de notas y carpetas en árbol (Tree View)
func RenderNoteList(entries []storage.NoteEntry, selectedPaths map[string]bool, selectedIndex int, width, height int, active bool, ht *mouse.HitTester, offsetY int) string {
	var rows []string

	// Cabecera con título de sección y contador de selección múltiple
	headerTitle := "   notes"
	selCount := len(selectedPaths)
	if selCount > 0 {
		selBadge := theme.SelectedItem.Copy().Foreground(theme.ColorTeal).Render(fmt.Sprintf(" [%d %s]", selCount, i18n.T("sel", "sel")))
		headerTitle += selBadge
	}
	header := theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true).Render(headerTitle)
	rows = append(rows, header)

	if len(entries) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render(i18n.T("  (Carpeta vacía. 'c': nueva nota, 'F': carpeta)", "  (Empty folder. 'c': new note, 'F': folder)"))
		rows = append(rows, emptyMsg)
	}

	usableHeight := height - 3
	if usableHeight < 1 {
		usableHeight = 1
	}

	startIdx := 0
	if selectedIndex >= usableHeight {
		startIdx = selectedIndex - usableHeight + 1
	}
	endIdx := startIdx + usableHeight
	if endIdx > len(entries) {
		endIdx = len(entries)
	}

	for i := startIdx; i < endIdx; i++ {
		entry := entries[i]
		isSelected := i == selectedIndex

		cursor := "  "
		itemStyle := theme.NormalItem
		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
			itemStyle = theme.SelectedItem
		}

		// Sangría proporcional al nivel de profundidad del árbol
		indent := strings.Repeat("  ", entry.Depth)

		maxNameLen := width - len(indent) - 18
		if maxNameLen < 5 {
			maxNameLen = 5
		}

		var rowText string

		if entry.Type == storage.EntryFolder {
			// Flecha indicadora de árbol
			arrow := "▾ "
			if !entry.Expanded {
				arrow = "▸ "
			}
			arrowStyled := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(arrow)
			folderIcon := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(" ")
			badge := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("(%d)", entry.Children))

			name := entry.Name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			rowText = fmt.Sprintf("%s%s%s%s%-*s %s", cursor, indent, arrowStyled, folderIcon, maxNameLen, itemStyle.Render(name), badge)
		} else {
			// Indicador de selección múltiple
			selBox := " "
			if selectedPaths != nil && selectedPaths[entry.Path] {
				selBox = theme.SelectedItem.Copy().Foreground(theme.ColorGreen).Bold(true).Render("✓ ")
			}

			// Icono Markdown estilo nerd font en color teal
			noteIcon := theme.NormalItem.Copy().Foreground(theme.ColorTeal).Bold(true).Render("󰍔 ")
			timeStr := entry.ModTime.Format("02 Jan")
			badge := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr)

			name := entry.Name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			rowText = fmt.Sprintf("%s%s %s%s%-*s %s", cursor, indent, selBox, noteIcon, maxNameLen, itemStyle.Render(name), badge)
		}

		// Registrar zona de clic del mouse
		if ht != nil {
			rowY := offsetY + (i - startIdx) + 2
			ht.Register(fmt.Sprintf("entry-%d", i), mouse.ZoneNote, 0, rowY, width, rowY, i, entry.Path)
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
