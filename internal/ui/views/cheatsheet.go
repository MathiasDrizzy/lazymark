package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// RenderCheatsheetPopup genera una tarjeta flotante con los atajos rápidos
func RenderCheatsheetPopup(width, height int) string {
	boxWidth := 46
	if width < 50 {
		boxWidth = width - 4
	}

	title := theme.SelectedItem.Copy().Bold(true).Render(i18n.T(" 󰌌  Cheatsheet & Atajos ", " 󰌌  Cheatsheet & Keys "))
	divider := theme.NormalItem.Copy().Foreground(theme.ColorSurface1).Render(strings.Repeat("─", boxWidth-4))

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"↑/↓ • j/k", i18n.T("Navegar lista / Scroll nota", "Navigate list / Scroll note")},
		{"Enter", i18n.T("Abrir nota / Entrar carpeta", "Open note / Enter folder")},
		{"c", i18n.T("Crear nueva nota", "Create new note")},
		{"F", i18n.T("Crear nueva carpeta", "Create new folder")},
		{"m", i18n.T("Mover nota a carpeta", "Move note to folder")},
		{"d", i18n.T("Eliminar nota / carpeta", "Delete note / folder")},
		{"Ctrl+V", i18n.T("Pegar imagen del portapapeles", "Paste clipboard image")},
		{"Tab • h/l", i18n.T("Alternar foco lista/preview", "Toggle list/preview focus")},
		{"1, 2, 3", i18n.T("Cambiar de pestaña", "Switch tabs")},
		{"?", i18n.T("Configuración en vivo", "Live settings")},
		{"q • Esc", i18n.T("Cerrar / Salir", "Close / Quit")},
	}

	var rows []string
	rows = append(rows, fmt.Sprintf(" %s", title))
	rows = append(rows, fmt.Sprintf(" %s", divider))

	for _, s := range shortcuts {
		k := theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Render(fmt.Sprintf("%-12s", s.key))
		d := theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render(s.desc)
		rows = append(rows, fmt.Sprintf("  %s %s", k, d))
	}

	rows = append(rows, fmt.Sprintf(" %s", divider))
	rows = append(rows, theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
		Render(i18n.T(" Presiona 'h', '?' o Esc para cerrar", " Press 'h', '?' or Esc to close")))

	body := strings.Join(rows, "\n")
	return theme.ActivePanelBorder.
		Width(boxWidth).
		Background(theme.ColorBase).
		Render(body)
}

// RenderCheatsheet es un alias para RenderCheatsheetPopup
func RenderCheatsheet(width, height int) string {
	return RenderCheatsheetPopup(width, height)
}
