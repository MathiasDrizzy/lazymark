package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderConfirmModal genera un modal de diálogo para confirmar acciones críticas (ej. eliminar carpetas o notas)
func RenderConfirmModal(title, message string, width, height int, ht *mouse.HitTester) string {
	modalWidth := 52
	if width < 56 {
		modalWidth = width - 4
	}

	titleStyled := theme.SelectedItem.Copy().Foreground(theme.ColorRed).Bold(true).Render(title)
	divider := theme.NormalItem.Copy().Foreground(theme.ColorSurface1).Render(strings.Repeat("─", modalWidth-4))

	var rows []string
	rows = append(rows, fmt.Sprintf("  %s", titleStyled))
	rows = append(rows, fmt.Sprintf("  %s", divider))

	// Desglosar mensaje en líneas si tiene saltos
	msgLines := strings.Split(message, "\n")
	for _, ml := range msgLines {
		wrappedMsg := theme.NormalItem.Copy().Foreground(theme.ColorText).Render(fmt.Sprintf("  %s", ml))
		rows = append(rows, wrappedMsg)
	}

	rows = append(rows, fmt.Sprintf("  %s", divider))

	// Botones de acción
	btnYes := theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true).Render(i18n.T("[ y ] Sí, eliminar", "[ y ] Yes, delete"))
	btnNo := theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render(i18n.T("[ n ] Cancelar (Esc)", "[ n ] Cancel (Esc)"))
	btnRow := fmt.Sprintf("  %s    %s", btnYes, btnNo)
	rows = append(rows, btnRow)

	startX := (width - modalWidth) / 2
	startY := (height - len(rows) - 2) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	if ht != nil {
		btnY := startY + len(rows) // fila de los botones
		ht.Register("confirm-yes", mouse.ZoneAction, startX+2, btnY, startX+20, btnY, 0, "confirm-yes")
		ht.Register("confirm-no", mouse.ZoneAction, startX+24, btnY, startX+44, btnY, 0, "confirm-no")
	}

	body := strings.Join(rows, "\n")
	return theme.ActivePanelBorder.
		Width(modalWidth + 2).
		BorderForeground(theme.ColorRed).
		Render(body)
}
