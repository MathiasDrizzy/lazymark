package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderMoveModal genera un popup modal para seleccionar la carpeta de destino al mover una nota
func RenderMoveModal(folders []string, baseDir string, selectedIdx int, noteName string, width, height int, ht *mouse.HitTester) string {
	modalWidth := 50
	if width < 54 {
		modalWidth = width - 4
	}

	title := theme.SelectedItem.Copy().Bold(true).Render(i18n.T("  Mover Nota a Carpeta", "  Move Note to Folder"))
	divider := theme.NormalItem.Copy().Foreground(theme.ColorSurface1).Render(strings.Repeat("─", modalWidth-4))

	var rows []string
	rows = append(rows, fmt.Sprintf("  %s", title))
	targetNoteStr := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Render(fmt.Sprintf("   %s", noteName))
	rows = append(rows, targetNoteStr)
	rows = append(rows, fmt.Sprintf("  %s", divider))

	startX := (width - modalWidth) / 2
	startY := (height - 12) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for idx, f := range folders {
		isSelected := idx == selectedIdx

		cursor := "  "
		itemStyle := theme.NormalItem
		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
			itemStyle = theme.SelectedItem.Copy().Bold(true)
		}

		relPath, err := filepath.Rel(baseDir, f)
		if err != nil || relPath == "." {
			relPath = "/ (" + i18n.T("Raíz", "Root") + ")"
		}

		display := fmt.Sprintf("%s  %s", cursor, itemStyle.Render(relPath))

		if ht != nil {
			rowY := startY + idx + 4
			ht.Register(fmt.Sprintf("move-folder-%d", idx), mouse.ZoneAction, startX+2, rowY, startX+modalWidth-2, rowY, idx, fmt.Sprintf("move-to:%d", idx))
		}

		rows = append(rows, display)
	}

	rows = append(rows, fmt.Sprintf("  %s", divider))
	hint := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
		Render(i18n.T("↑/↓: Mover  •  Enter: Seleccionar  •  Esc: Cancelar", "↑/↓: Move  •  Enter: Select  •  Esc: Cancel"))
	rows = append(rows, fmt.Sprintf("  %s", hint))

	body := strings.Join(rows, "\n")
	return theme.ActivePanelBorder.
		Width(modalWidth + 2).
		Render(body)
}
