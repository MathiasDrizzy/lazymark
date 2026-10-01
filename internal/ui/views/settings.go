package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
)

var AvailableEditors = []string{"micro", "vim", "nvim", "nano"}

// SettingsItem representa una opción en el menú de configuración
type SettingsItem int

const (
	ItemLanguage SettingsItem = iota
	ItemEditor
	ItemTheme
	ItemKeybindings
	ItemSidebarRatio
	ItemConfirmDelete
	ItemHideDoneTasks
	ItemTabTags
	ItemTabTasks
	TotalSettingsItems
)

// RenderSettingsModal renderiza un popup modal flotante, minimalista y centrado
func RenderSettingsModal(cfg *config.Config, selectedItem SettingsItem, totalWidth, totalHeight int, ht *mouse.HitTester) string {
	modalWidth := 52
	if totalWidth < 56 {
		modalWidth = totalWidth - 4
	}

	title := theme.SelectedItem.Copy().Bold(true).Render(i18n.T("  Configuración", "  Settings"))
	divider := theme.NormalItem.Copy().Foreground(theme.ColorSurface1).Render(strings.Repeat("─", modalWidth-4))

	var rows []string
	rows = append(rows, fmt.Sprintf("  %s", title))
	rows = append(rows, fmt.Sprintf("  %s", divider))

	keyMode := cfg.KeybindingMode
	if keyMode == "" {
		keyMode = "dual"
	}

	items := []struct {
		id    SettingsItem
		label string
		val   string
	}{
		{
			id:    ItemLanguage,
			label: i18n.T("Idioma / Language", "Language / Idioma"),
			val: func() string {
				if i18n.CurrentLanguage() == i18n.LangES {
					return "◀ Español ▶"
				}
				return "◀ English ▶"
			}(),
		},
		{
			id:    ItemEditor,
			label: i18n.T("Editor de Texto", "Text Editor"),
			val:   fmt.Sprintf("◀ %s ▶", filepath.Base(cfg.Editor)),
		},
		{
			id:    ItemTheme,
			label: i18n.T("Tema de Color", "Color Theme"),
			val:   fmt.Sprintf("◀ %s ▶", theme.CurrentThemeName),
		},
		{
			id:    ItemKeybindings,
			label: i18n.T("Modo Atajos / Keys", "Keybindings Mode"),
			val: func() string {
				switch strings.ToLower(keyMode) {
				case "vim":
					return "◀ Vim ▶"
				case "lazygit":
					return "◀ Lazygit ▶"
				default:
					return fmt.Sprintf("◀ %s ▶", i18n.T("Dual (Ambos)", "Dual (Both)"))
				}
			}(),
		},
		{
			id:    ItemSidebarRatio,
			label: i18n.T("Ancho Panel Izq", "Sidebar Width"),
			val: func() string {
				pct := int(cfg.SidebarRatio * 100)
				if pct == 0 {
					pct = 33
				}
				return fmt.Sprintf("◀ %d%% ▶", pct)
			}(),
		},
		{
			id:    ItemConfirmDelete,
			label: i18n.T("Confirmar Borrado", "Confirm Delete"),
			val: func() string {
				if cfg.ConfirmDelete {
					return "[✓] " + i18n.T("Activa", "Enabled")
				}
				return "[ ] " + i18n.T("Desactivada", "Disabled")
			}(),
		},
		{
			id:    ItemHideDoneTasks,
			label: i18n.T("Ocultar Tareas Hechas", "Hide Done Tasks"),
			val: func() string {
				if cfg.HideCompletedTasks {
					return "[✓] " + i18n.T("Activa", "Enabled")
				}
				return "[ ] " + i18n.T("Desactivada", "Disabled")
			}(),
		},
		{
			id:    ItemTabTags,
			label: i18n.T("Pestaña Categorías", "Tags Tab"),
			val: func() string {
				if cfg.ShowTagsTab {
					return "[✓] " + i18n.T("Activa", "Enabled")
				}
				return "[ ] " + i18n.T("Oculta", "Disabled")
			}(),
		},
		{
			id:    ItemTabTasks,
			label: i18n.T("Pestaña Tareas", "Tasks Tab"),
			val: func() string {
				if cfg.ShowTasksTab {
					return "[✓] " + i18n.T("Activa", "Enabled")
				}
				return "[ ] " + i18n.T("Oculta", "Disabled")
			}(),
		},
	}

	// Ancho total del modal con bordes y padding para cálculo exacto de posición
	startX := totalWidth - (modalWidth + 4) - 1
	startY := totalHeight - 16
	if startX < 2 {
		startX = 2
	}
	if startY < 2 {
		startY = 2
	}

	for idx, item := range items {
		isSelected := selectedItem == item.id
		cursor := "  "
		labelStyle := theme.NormalItem.Copy().Foreground(theme.ColorText)
		valStyle := theme.NormalItem.Copy().Foreground(theme.ColorTeal)

		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
			labelStyle = theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true)
			valStyle = theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true)
		}

		maxLabelLen := modalWidth - 24
		if maxLabelLen < 10 {
			maxLabelLen = 10
		}
		truncatedLabel := item.label
		if len(truncatedLabel) > maxLabelLen {
			truncatedLabel = truncatedLabel[:maxLabelLen-3] + "..."
		}

		rowText := fmt.Sprintf("%s%-*s  %s",
			cursor,
			maxLabelLen,
			labelStyle.Render(truncatedLabel),
			valStyle.Render(item.val),
		)

		if ht != nil {
			rowY := startY + idx + 3
			ht.Register(fmt.Sprintf("modal-item-%d", idx), mouse.ZoneAction, startX+2, rowY, startX+modalWidth-2, rowY, int(item.id), fmt.Sprintf("select-config:%d", item.id))
		}

		rows = append(rows, rowText)
	}

	rows = append(rows, fmt.Sprintf("  %s", divider))

	hint := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
		Render(i18n.T("↑/↓: Mover  •  ←/→/Enter: Cambiar  •  Esc: Cerrar", "↑/↓: Move  •  ←/→/Enter: Change  •  Esc: Close"))
	rows = append(rows, fmt.Sprintf("  %s", hint))

	body := strings.Join(rows, "\n")

	return theme.ActivePanelBorder.
		Width(modalWidth).
		Render(body)
}

// OverlayLayers superpone un bloque flotante sobre la vista base preservando el fondo visible sin romper secuencias ANSI
func OverlayLayers(base, overlay string, totalWidth, totalHeight int, alignBottomRight bool) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	// Asegurar tamaño base
	for len(baseLines) < totalHeight {
		baseLines = append(baseLines, strings.Repeat(" ", totalWidth))
	}

	overlayH := len(overlayLines)
	overlayW := 0
	for _, l := range overlayLines {
		w := ansi.StringWidth(l)
		if w > overlayW {
			overlayW = w
		}
	}

	startY := (totalHeight - overlayH) / 2
	startX := (totalWidth - overlayW) / 2

	if alignBottomRight {
		startY = totalHeight - overlayH - 2
		startX = totalWidth - overlayW - 1
	}

	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	for i, oLine := range overlayLines {
		targetY := startY + i
		if targetY >= len(baseLines) {
			break
		}

		bLine := baseLines[targetY]
		bLen := ansi.StringWidth(bLine)
		if bLen > totalWidth {
			bLine = ansi.Truncate(bLine, totalWidth, "")
		} else if bLen < totalWidth {
			bLine = bLine + strings.Repeat(" ", totalWidth-bLen)
		}

		// Truncar izquierda con soporte ANSI
		left := ""
		if startX > 0 {
			left = ansi.Truncate(bLine, startX, "")
			lw := ansi.StringWidth(left)
			if lw < startX {
				left += strings.Repeat(" ", startX-lw)
			}
		}

		// Normalizar ancho del overlay: garantizar que mida exactamente overlayW
		oLen := ansi.StringWidth(oLine)
		if oLen < overlayW {
			oLine = oLine + strings.Repeat(" ", overlayW-oLen)
		}

		// Recortar derecha con soporte ANSI
		expectedRightWidth := totalWidth - (startX + overlayW)
		right := ""
		if expectedRightWidth > 0 {
			if alignBottomRight {
				right = strings.Repeat(" ", expectedRightWidth)
			} else {
				remainingStart := startX + overlayW
				right = ansi.Cut(bLine, remainingStart, totalWidth)
				rw := ansi.StringWidth(right)
				if rw < expectedRightWidth {
					right += strings.Repeat(" ", expectedRightWidth-rw)
				}
			}
		}

		baseLines[targetY] = left + "\x1b[0m" + oLine + "\x1b[0m" + right
	}

	return strings.Join(baseLines, "\n")
}
