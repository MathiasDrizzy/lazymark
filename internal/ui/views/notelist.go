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

		var rowText string

		if entry.Type == storage.EntryFolder {
			// Flecha indicadora de árbol
			arrow := "▾ "
			if !entry.Expanded {
				arrow = "▸ "
			}
			arrowStyled := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(arrow)
			folderIcon := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(" ")

			badge := ""
			badgeLen := 0
			// Solo mostrar contador de carpeta si hay ancho suficiente
			if width-len(indent) >= 18 {
				cntStr := fmt.Sprintf("(%d)", entry.Children)
				badge = theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(cntStr)
				badgeLen = len(cntStr) + 1
			}

			availName := width - len(indent) - 7 - badgeLen
			if availName < 3 {
				availName = 3
			}

			name := entry.Name
			if len(name) > availName {
				name = name[:availName-1] + "…"
			}

			if badge != "" {
				rowText = fmt.Sprintf("%s%s%s%s%-*s %s", cursor, indent, arrowStyled, folderIcon, availName, itemStyle.Render(name), badge)
			} else {
				rowText = fmt.Sprintf("%s%s%s%s%s", cursor, indent, arrowStyled, folderIcon, itemStyle.Render(name))
			}
		} else {
			// Indicador de selección múltiple
			selBox := " "
			if selectedPaths != nil && selectedPaths[entry.Path] {
				selBox = theme.SelectedItem.Copy().Foreground(theme.ColorGreen).Bold(true).Render("✓ ")
			}

			// Icono Markdown estilo nerd font en color teal
			noteIcon := theme.NormalItem.Copy().Foreground(theme.ColorTeal).Bold(true).Render("󰍔 ")

			badge := ""
			badgeLen := 0
			// Solo mostrar fecha si el panel tiene ancho suficiente para que no desborde hacia abajo
			if width-len(indent) >= 25 {
				timeStr := entry.ModTime.Format("02 Jan")
				badge = theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr)
				badgeLen = 7 // " " + "02 Jan"
			}

			availName := width - len(indent) - 7 - badgeLen
			if availName < 3 {
				availName = 3
			}

			name := entry.Name
			if len(name) > availName {
				name = name[:availName-1] + "…"
			}

			if badge != "" {
				rowText = fmt.Sprintf("%s%s %s%s%-*s %s", cursor, indent, selBox, noteIcon, availName, itemStyle.Render(name), badge)
			} else {
				rowText = fmt.Sprintf("%s%s %s%s%s", cursor, indent, selBox, noteIcon, itemStyle.Render(name))
			}
		}

		rowText = ansi.Truncate(rowText, width, "")

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
