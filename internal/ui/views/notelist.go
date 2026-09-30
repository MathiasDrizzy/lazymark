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

	title := i18n.T("[1] Notas", "[1] Notes")
	selCount := len(selectedPaths)
	badge := fmt.Sprintf("(%d)", len(entries))
	if selCount > 0 {
		badge = fmt.Sprintf("[%d sel]", selCount)
	}

	if len(entries) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render(i18n.T("  (Carpeta vacía. 'c': nueva nota, 'F': carpeta)", "  (Empty folder. 'c': new note, 'F': folder)"))
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
	if endIdx > len(entries) {
		endIdx = len(entries)
	}

	// Ancho interior disponible para el contenido (descontando 4 columnas por bordes '│ ' y ' │')
	contentWidth := width - 4
	if contentWidth < 4 {
		contentWidth = 4
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
			if contentWidth-len(indent) >= 18 {
				cntStr := fmt.Sprintf("(%d)", entry.Children)
				badge = theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(cntStr)
				badgeLen = len(cntStr) + 1
			}

			// Ajustar sangría si el espacio es muy estrecho para garantizar nombre visible
			effIndent := indent
			fixedPrefix := 2 + 4 + badgeLen // cursor(2) + arrow(2) + icon(2) + badge
			if contentWidth-fixedPrefix-len(effIndent) < 3 && len(effIndent) > 0 {
				maxIndent := contentWidth - fixedPrefix - 3
				if maxIndent < 0 {
					maxIndent = 0
				}
				if len(effIndent) > maxIndent {
					effIndent = effIndent[:maxIndent]
				}
			}

			availName := contentWidth - fixedPrefix - len(effIndent)
			if availName < 2 {
				availName = 2
			}

			name := entry.Name
			if len(name) > availName {
				if availName > 1 {
					name = name[:availName-1] + "…"
				} else {
					name = "…"
				}
			}

			if badge != "" {
				rowText = fmt.Sprintf("%s%s%s%s%s %s", cursor, effIndent, arrowStyled, folderIcon, itemStyle.Render(name), badge)
			} else {
				rowText = fmt.Sprintf("%s%s%s%s%s", cursor, effIndent, arrowStyled, folderIcon, itemStyle.Render(name))
			}
		} else {
			// Indicador de selección múltiple
			selBox := ""
			selLen := 0
			if selectedPaths != nil && selectedPaths[entry.Path] {
				selBox = theme.SelectedItem.Copy().Foreground(theme.ColorGreen).Bold(true).Render("✓ ")
				selLen = 2
			}

			// Icono Markdown estilo nerd font en color teal
			noteIcon := theme.NormalItem.Copy().Foreground(theme.ColorTeal).Bold(true).Render("󰍔 ")

			badge := ""
			badgeLen := 0
			// Solo mostrar fecha si el panel tiene ancho suficiente para que no desborde hacia abajo
			if contentWidth-len(indent) >= 22 {
				timeStr := entry.ModTime.Format("02 Jan")
				badge = theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr)
				badgeLen = 7 // " " + "02 Jan"
			}

			// Ajustar sangría si el espacio es muy estrecho para garantizar nombre visible
			effIndent := indent
			fixedPrefix := 2 + selLen + 2 + badgeLen // cursor(2) + selBox + noteIcon(2) + badge
			if contentWidth-fixedPrefix-len(effIndent) < 3 && len(effIndent) > 0 {
				maxIndent := contentWidth - fixedPrefix - 3
				if maxIndent < 0 {
					maxIndent = 0
				}
				if len(effIndent) > maxIndent {
					effIndent = effIndent[:maxIndent]
				}
			}

			availName := contentWidth - fixedPrefix - len(effIndent)
			if availName < 2 {
				availName = 2
			}

			name := entry.Name
			if len(name) > availName {
				if availName > 1 {
					name = name[:availName-1] + "…"
				} else {
					name = "…"
				}
			}

			if badge != "" {
				rowText = fmt.Sprintf("%s%s%s%s%s %s", cursor, effIndent, selBox, noteIcon, itemStyle.Render(name), badge)
			} else {
				rowText = fmt.Sprintf("%s%s%s%s%s", cursor, effIndent, selBox, noteIcon, itemStyle.Render(name))
			}
		}

		rowText = ansi.Truncate(rowText, contentWidth, "")

		// Registrar zona de clic del mouse
		if ht != nil {
			rowY := offsetY + 1 + (i - startIdx)
			ht.Register(fmt.Sprintf("entry-%d", i), mouse.ZoneNote, 0, rowY, width, rowY, i, entry.Path)
		}

		rows = append(rows, rowText)
	}

	content := strings.Join(rows, "\n")
	return theme.RenderBoxWithTitle(title, badge, content, width, height, active)
}
