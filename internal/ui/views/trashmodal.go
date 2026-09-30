package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

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

	numItemRows := 1
	if len(items) > 0 {
		numItemRows = endIdx - startIdx
	}

	// 1 (borde sup) + 1 (título) + 1 (divisor) + numItemRows + 1 (divisor) + 2 (filas de acción) + 1 (borde inf)
	modalH := 1 + 1 + 1 + numItemRows + 1 + 2 + 1
	totalModalW := modalWidth + 4
	startX := (width - totalModalW) / 2
	startY := (height - modalH) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	if len(items) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).
			Render(i18n.T("  (La papelera está vacía. Los archivos borrados se guardan aquí)", "  (Trash is empty. Deleted files are stored here)"))
		rows = append(rows, emptyMsg)
	} else {
		for i, idx := 0, startIdx; idx < endIdx; i, idx = i+1, idx+1 {
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

			rowY := startY + 1 + 2 + i
			if ht != nil {
				ht.Register(fmt.Sprintf("trash-item-%d", idx), mouse.ZoneAction, startX+2, rowY, startX+modalWidth+1, rowY, idx, fmt.Sprintf("select-trash:%d", idx))
			}

			rows = append(rows, rowText)
		}
	}

	rows = append(rows, fmt.Sprintf("  %s", divider))

	// Botones de acción formateados en dos filas limpias y compactas
	rBtn := theme.FooterKey.Render("[ r ]") + " " + theme.NormalItem.Render(i18n.T("Restaurar", "Restore"))
	dBtn := theme.FooterKey.Render("[ d ]") + " " + theme.NormalItem.Render(i18n.T("Definitivo", "Permanent"))
	cBtn := theme.FooterKey.Render("[ c ]") + " " + theme.NormalItem.Render(i18n.T("Vaciar", "Empty"))
	escBtn := theme.FooterKey.Render("[ Esc ]") + " " + theme.NormalItem.Render(i18n.T("Cerrar", "Close"))

	rLen := ansi.StringWidth(rBtn)
	dLen := ansi.StringWidth(dBtn)
	cLen := ansi.StringWidth(cBtn)
	escLen := ansi.StringWidth(escBtn)

	colWidth := (modalWidth - 6) / 2
	if colWidth < 18 {
		colWidth = 18
	}

	pad1 := colWidth - rLen
	if pad1 < 2 {
		pad1 = 2
	}
	rowAction1 := fmt.Sprintf("  %s%s%s", rBtn, strings.Repeat(" ", pad1), dBtn)

	pad2 := colWidth - cLen
	if pad2 < 2 {
		pad2 = 2
	}
	rowAction2 := fmt.Sprintf("  %s%s%s", cBtn, strings.Repeat(" ", pad2), escBtn)

	rows = append(rows, rowAction1)
	rows = append(rows, rowAction2)

	if ht != nil {
		row1Y := startY + 1 + len(rows) - 2
		row2Y := startY + 1 + len(rows) - 1

		ht.Register("trash-restore", mouse.ZoneAction, startX+4, row1Y, startX+4+rLen-1, row1Y, 0, "trash-restore")
		ht.Register("trash-delete", mouse.ZoneAction, startX+4+rLen+pad1, row1Y, startX+4+rLen+pad1+dLen-1, row1Y, 0, "trash-delete")
		ht.Register("trash-empty", mouse.ZoneAction, startX+4, row2Y, startX+4+cLen-1, row2Y, 0, "trash-empty")
		ht.Register("trash-close", mouse.ZoneAction, startX+4+cLen+pad2, row2Y, startX+4+cLen+pad2+escLen-1, row2Y, 0, "trash-close")
	}

	body := strings.Join(rows, "\n")
	return theme.ActivePanelBorder.
		Width(modalWidth).
		Render(body)
}
