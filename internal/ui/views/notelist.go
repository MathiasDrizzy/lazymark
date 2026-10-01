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
func RenderNoteList(entries []storage.NoteEntry, selectedPaths map[string]bool, selectedIndex int, width, height int, active bool, ht *mouse.HitTester, offsetY int, filterTag ...string) string {
	var rows []string

	title := i18n.T("[1] Notas", "[1] Notes")
	if len(filterTag) > 0 && filterTag[0] != "" {
		title = fmt.Sprintf("%s (#%s)", title, filterTag[0])
	}
	selCount := len(selectedPaths)
	badge := "0 of 0"
	if len(entries) > 0 {
		badge = fmt.Sprintf("%d of %d", selectedIndex+1, len(entries))
		if selCount > 0 {
			badge = fmt.Sprintf("[%d sel] %d of %d", selCount, selectedIndex+1, len(entries))
		}
	}

	if len(entries) == 0 {
		var emptyMsgText string
		if len(filterTag) > 0 && filterTag[0] != "" {
			emptyMsgText = fmt.Sprintf(i18n.T("  (Sin notas con tag #%s. 'Esc': limpiar)", "  (No notes with tag #%s. 'Esc': clear)"), filterTag[0])
		} else {
			emptyMsgText = i18n.T("  (Carpeta vacía. 'c': nueva nota, 'F': carpeta)", "  (Empty folder. 'c': new note, 'F': folder)")
		}
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render(emptyMsgText)
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

	selStyle := theme.SelectedLineInactive
	if active {
		selStyle = theme.SelectedLineActive
	}

	for i := startIdx; i < endIdx; i++ {
		entry := entries[i]
		isSelected := i == selectedIndex

		// Sangría proporcional al nivel de profundidad del árbol
		indent := strings.Repeat("  ", entry.Depth)

		var rowText string

		if entry.Type == storage.EntryFolder {
			arrow := "▾ "
			if !entry.Expanded {
				arrow = "▸ "
			}

			folderBadge := ""
			badgeLen := 0
			if contentWidth-len(indent) >= 18 {
				cntStr := fmt.Sprintf("(%d)", entry.Children)
				folderBadge = cntStr
				badgeLen = len(cntStr) + 1
			}

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

			if isSelected {
				cursor := "▸ "
				rawLine := fmt.Sprintf("%s%s%s %s", cursor, effIndent, arrow, name)
				if folderBadge != "" {
					rawLine += " " + folderBadge
				}
				rawLine = ansi.Truncate(rawLine, contentWidth, "")
				lineW := ansi.StringWidth(rawLine)
				if lineW < contentWidth {
					rawLine += strings.Repeat(" ", contentWidth-lineW)
				}
				rowText = selStyle.Render(rawLine)
			} else {
				cursor := "  "
				arrowStyled := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(arrow)
				folderIcon := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(" ")
				styledName := theme.NormalItem.Render(name)
				if folderBadge != "" {
					badgeStyled := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(folderBadge)
					rowText = fmt.Sprintf("%s%s%s%s%s %s", cursor, effIndent, arrowStyled, folderIcon, styledName, badgeStyled)
				} else {
					rowText = fmt.Sprintf("%s%s%s%s%s", cursor, effIndent, arrowStyled, folderIcon, styledName)
				}
				rowText = ansi.Truncate(rowText, contentWidth, "")
			}
		} else {
			// Nota Markdown
			selBox := ""
			selLen := 0
			if selectedPaths != nil && selectedPaths[entry.Path] {
				selBox = "✓ "
				selLen = 2
			}

			timeStr := ""
			badgeLen := 0
			if contentWidth-len(indent) >= 22 {
				timeStr = entry.ModTime.Format("02 Jan")
				badgeLen = 7 // " " + "02 Jan"
			}

			effIndent := indent
			fixedPrefix := 2 + selLen + 2 + badgeLen
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

			if isSelected {
				cursor := "▸ "
				rawLine := fmt.Sprintf("%s%s%s󰍔 %s", cursor, effIndent, selBox, name)
				if timeStr != "" {
					rawLine += " " + timeStr
				}
				rawLine = ansi.Truncate(rawLine, contentWidth, "")
				lineW := ansi.StringWidth(rawLine)
				if lineW < contentWidth {
					rawLine += strings.Repeat(" ", contentWidth-lineW)
				}
				rowText = selStyle.Render(rawLine)
			} else {
				cursor := "  "
				styledSel := ""
				if selBox != "" {
					styledSel = theme.SelectedItem.Copy().Foreground(theme.ColorGreen).Bold(true).Render(selBox)
				}
				noteIcon := theme.NormalItem.Copy().Foreground(theme.ColorTeal).Bold(true).Render("󰍔 ")
				styledName := theme.NormalItem.Render(name)
				if timeStr != "" {
					badgeStyled := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr)
					rowText = fmt.Sprintf("%s%s%s%s%s %s", cursor, effIndent, styledSel, noteIcon, styledName, badgeStyled)
				} else {
					rowText = fmt.Sprintf("%s%s%s%s%s", cursor, effIndent, styledSel, noteIcon, styledName)
				}
				rowText = ansi.Truncate(rowText, contentWidth, "")
			}
		}

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
