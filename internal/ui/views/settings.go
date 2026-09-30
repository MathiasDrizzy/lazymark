package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

var AvailableEditors = []string{"micro", "vim", "nvim", "nano"}

// SettingsItem representa una opción en el menú de configuración
type SettingsItem int

const (
	ItemLanguage SettingsItem = iota
	ItemEditor
	ItemTheme
	ItemTabTags
	ItemTabTasks
	ItemTabGallery
	TotalSettingsItems
)

// RenderSettingsModal renderiza un popup modal flotante, minimalista y centrado
func RenderSettingsModal(cfg *config.Config, selectedItem SettingsItem, totalWidth, totalHeight int, ht *mouse.HitTester) string {
	modalWidth := 56
	if totalWidth < 60 {
		modalWidth = totalWidth - 4
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	title := theme.SelectedItem.Copy().Bold(true).Render(i18n.T("⚙️  Configuración", "⚙️  Settings"))
	divider := theme.NormalItem.Copy().Foreground(theme.ColorSurface1).Render(strings.Repeat("─", modalWidth-4))

	var rows []string
	rows = append(rows, fmt.Sprintf("  %s", title))
	rows = append(rows, fmt.Sprintf("  %s", divider))

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
					return "Español"
				}
				return "English"
			}(),
		},
		{
			id:    ItemEditor,
			label: i18n.T("Editor de Texto", "Text Editor"),
			val:   filepath.Base(cfg.Editor),
		},
		{
			id:    ItemTheme,
			label: i18n.T("Tema de Color", "Color Theme"),
			val:   theme.CurrentThemeName,
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
		{
			id:    ItemTabGallery,
			label: i18n.T("Pestaña Galería", "Gallery Tab"),
			val: func() string {
				if cfg.ShowGalleryTab {
					return "[✓] " + i18n.T("Activa", "Enabled")
				}
				return "[ ] " + i18n.T("Oculta", "Disabled")
			}(),
		},
	}

	startY := (totalHeight - 16) / 2
	if startY < 2 {
		startY = 2
	}
	startX := (totalWidth - modalWidth) / 2
	if startX < 2 {
		startX = 2
	}

	for idx, item := range items {
		isSelected := selectedItem == item.id
		cursor := "  "
		labelStyle := theme.NormalItem
		valStyle := theme.TagBadge

		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
			labelStyle = theme.SelectedItem
			valStyle = theme.TabActive
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
		Render(i18n.T("↑/↓: Mover  •  Enter/Espacio: Cambiar  •  Esc: Cerrar", "↑/↓: Move  •  Enter/Space: Toggle  •  Esc: Close"))
	rows = append(rows, fmt.Sprintf("  %s", hint))

	body := strings.Join(rows, "\n")

	modal := theme.ActivePanelBorder.
		Width(modalWidth).
		Background(theme.ColorBase).
		Render(body)

	// Centrar el modal en la pantalla
	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, modal)
}
