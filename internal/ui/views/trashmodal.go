package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderTrashModal genera un modal flotante que muestra los elementos en la papelera con soporte de restauración
func RenderTrashModal(items []storage.TrashItem, selectedIndex int, width, height int, ht *mouse.HitTester) string {
	modalWidth := 56
	if width < 60 {
		modalWidth = width - 4
	}

	title := theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true).
		Render(i18n.T("󰩹  Papelera (Auto-eliminación en 20 días)", "󰩹  Trash Bin (20-day auto-purge)"))
	divider := theme.NormalItem.Copy().Foreground(theme.ColorSurface1).Render(strings.Repeat("─", modalWidth-4))

	var rows []string
	rows = append(rows, fmt.Sprintf("  %s", title))
	rows = append(rows, fmt.Sprintf("  %s", divider))

	if len(items) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).
			Render(i18n.T("  (La papelera está vacía. Los archivos borrados se guardan aquí)", "  (Trash is empty. Deleted files are stored here)"))
		rows = append(rows, emptyMsg)
	} else {
		maxItems := 8
		if height > 20 {
			maxItems = height - 12
		}
		if maxItems < 3 {
			maxItems = 3
		}

		startIdx := 0
		if selectedIndex >= maxItems {
			startIdx = selectedIndex - maxItems + 1
		}
		endIdx := startIdx + maxItems
		if endIdx > len(items) {
			endIdx = len(items)
		}

		for idx := startIdx; idx < endIdx; idx++ {
			item := items[idx]
			isSelected := idx == selectedIndex

			cursor := "  "
			itemStyle := theme.NormalItem
			if isSelected {
				cursor = theme.SelectedItem.Render("❯ ")
				itemStyle = theme.SelectedItem.Copy().Bold(true)
			}

			icon := theme.NormalItem.Copy().Foreground(theme.ColorTeal).Render("󰍔 ")
			if item.IsDir {
				icon = theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(" ")
			}

			daysLeft := item.DaysRemaining()
			daysBadge := theme.NormalItem.Copy().Foreground(theme.ColorYellow).
				Render(fmt.Sprintf("%dd %s", daysLeft, i18n.T("restantes", "left")))

			maxNameLen := modalWidth - 26
			if maxNameLen < 10 {
				maxNameLen = 10
			}
			name := item.Name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			rowText := fmt.Sprintf("%s%s %-*s  %s", cursor, icon, maxNameLen, itemStyle.Render(name), daysBadge)

			if ht != nil {
				startY := (height - len(rows) - 8) / 2
				rowY := startY + (idx - startIdx) + 3
				startX := (width - modalWidth) / 2
				ht.Register(fmt.Sprintf("trash-item-%d", idx), mouse.ZoneAction, startX+2, rowY, startX+modalWidth-2, rowY, idx, fmt.Sprintf("select-trash:%d", idx))
			}

			rows = append(rows, rowText)
		}
	}

	rows = append(rows, fmt.Sprintf("  %s", divider))

	actionsHint := theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).
		Render(i18n.T("[ r ] Restaurar  •  [ d ] Definitivo  •  [ c ] Vaciar  •  [ Esc ] Cerrar", "[ r ] Restore  •  [ d ] Permanent  •  [ c ] Empty  •  [ Esc ] Close"))
	rows = append(rows, fmt.Sprintf("  %s", actionsHint))

	body := strings.Join(rows, "\n")
	return theme.ActivePanelBorder.
		Width(modalWidth).
		Render(body)
}
