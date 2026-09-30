package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderNoteList genera el panel izquierdo con carpetas y notas, usando iconos NerdFont
func RenderNoteList(entries []storage.NoteEntry, currentSubDir string, selectedIndex int, width, height int, active bool, ht *mouse.HitTester, offsetY int) string {
	var rows []string

	// Cabecera con ruta actual
	headerPath := "notes"
	if currentSubDir != "" {
		headerPath = fmt.Sprintf("notes/%s", currentSubDir)
	}
	header := theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true).
		Render(fmt.Sprintf("   %s", headerPath))
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

		maxNameLen := width - 15
		if maxNameLen < 5 {
			maxNameLen = 5
		}

		var icon string
		var badge string
		var name string

		if entry.Type == storage.EntryFolder {
			icon = theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(" ")
			name = entry.Name + "/"
			if entry.Name == ".." {
				name = ".."
			}
			badge = theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render("[DIR]")
		} else {
			icon = theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render(" ")
			name = entry.Name
			timeStr := entry.ModTime.Format("02 Jan")
			badge = theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr)
		}

		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		rowText := fmt.Sprintf("%s%s%-*s %s", cursor, icon, maxNameLen, itemStyle.Render(name), badge)

		// Registrar zona de clic
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
