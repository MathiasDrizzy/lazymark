package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

type SettingsSection int

const (
	SectionEditor SettingsSection = iota
	SectionTheme
	SectionLanguage
	SectionTabs
	SectionKeybindings
)

var AvailableEditors = []string{"micro", "vim", "nvim", "nano"}

// RenderSettingsView renderiza la pantalla/modal de configuración y ayuda
func RenderSettingsView(cfg *config.Config, currentSection SettingsSection, selectedItem int, width, height int, ht *mouse.HitTester) string {
	borderStyle := theme.ActivePanelBorder.Width(width - 2).Height(height - 2)

	title := theme.SelectedItem.Copy().Bold(true).Render(i18n.T(" ⚙️  CONFIGURACIÓN & ATAJOS DE TECLADO ", " ⚙️  SETTINGS & KEYBINDINGS "))

	var rows []string
	rows = append(rows, title)
	rows = append(rows, strings.Repeat("─", width-6))

	// 1. Selector de Editor
	editorLabel := theme.FooterKey.Render("1. " + i18n.T("Editor de Texto:", "Text Editor:"))
	var editorBadges []string
	for idx, ed := range AvailableEditors {
		badge := fmt.Sprintf("[%s]", ed)
		isCurrent := strings.Contains(strings.ToLower(cfg.Editor), ed)
		isSelected := currentSection == SectionEditor && selectedItem == idx

		var style string
		if isCurrent {
			style = theme.TabActive.Render("● " + badge)
		} else if isSelected {
			style = theme.SelectedItem.Render("❯ " + badge)
		} else {
			style = theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render(badge)
		}

		if ht != nil {
			ht.Register(fmt.Sprintf("cfg-ed-%d", idx), mouse.ZoneAction, 4+idx*12, 4, 14+idx*12, 4, idx, "set-editor:"+ed)
		}
		editorBadges = append(editorBadges, style)
	}
	rows = append(rows, fmt.Sprintf("  %s  %s", editorLabel, strings.Join(editorBadges, "  ")))
	rows = append(rows, "")

	// 2. Selector de Idioma
	langLabel := theme.FooterKey.Render("2. " + i18n.T("Idioma / Language:", "Language / Idioma:"))
	langES := "[Español]"
	langEN := "[English]"
	if i18n.CurrentLanguage() == i18n.LangES {
		langES = theme.TabActive.Render("● [Español]")
		langEN = theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render("[English]")
	} else {
		langES = theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render("[Español]")
		langEN = theme.TabActive.Render("● [English]")
	}
	if ht != nil {
		ht.Register("cfg-lang-es", mouse.ZoneAction, 25, 6, 36, 6, 0, "set-lang:es")
		ht.Register("cfg-lang-en", mouse.ZoneAction, 38, 6, 49, 6, 1, "set-lang:en")
	}
	rows = append(rows, fmt.Sprintf("  %s  %s  %s", langLabel, langES, langEN))
	rows = append(rows, "")

	// 3. Selector de Tema
	themeLabel := theme.FooterKey.Render("3. " + i18n.T("Tema de Color (tecla 't'):", "Color Theme (key 't'):"))
	currentThemeBadge := theme.TagBadge.Render(theme.CurrentThemeName)
	rows = append(rows, fmt.Sprintf("  %s  %s  %s", themeLabel, currentThemeBadge,
		theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).Render(i18n.T("(Presiona 't' para ciclar temas)", "(Press 't' to cycle themes)"))))
	rows = append(rows, "")

	// 4. Configuración de Pestañas
	tabsLabel := theme.FooterKey.Render("4. " + i18n.T("Pestañas Habilitadas:", "Enabled Tabs:"))
	tagsToggle := "[ ] Categorías/Tags"
	if cfg.ShowTagsTab {
		tagsToggle = "[✓] Categorías/Tags"
	}
	tasksToggle := "[ ] Tareas"
	if cfg.ShowTasksTab {
		tasksToggle = "[✓] Tareas"
	}
	galleryToggle := "[ ] Imágenes/Galería"
	if cfg.ShowGalleryTab {
		galleryToggle = "[✓] Imágenes/Galería"
	}
	if ht != nil {
		ht.Register("cfg-tab-tags", mouse.ZoneAction, 4, 10, 24, 10, 0, "toggle-tab:tags")
		ht.Register("cfg-tab-tasks", mouse.ZoneAction, 26, 10, 40, 10, 1, "toggle-tab:tasks")
		ht.Register("cfg-tab-gallery", mouse.ZoneAction, 42, 10, 64, 10, 2, "toggle-tab:gallery")
	}
	rows = append(rows, fmt.Sprintf("  %s  %s  %s  %s", tabsLabel,
		theme.NormalItem.Render(tagsToggle),
		theme.NormalItem.Render(tasksToggle),
		theme.NormalItem.Render(galleryToggle)))
	rows = append(rows, "")

	// 5. Guía de Atajos
	keyLabel := theme.FooterKey.Render(i18n.T("5. Atajos Rápidos:", "5. Quick Shortcuts:"))
	rows = append(rows, fmt.Sprintf("  %s", keyLabel))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("1, 2, 3, 4"),
		theme.NormalItem.Render(i18n.T("Cambiar de pestaña", "Switch tabs"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("↑ / ↓  |  j / k"),
		theme.NormalItem.Render(i18n.T("Navegar listas", "Navigate lists"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("Tab    |  h / l"),
		theme.NormalItem.Render(i18n.T("Alternar entre panel izquierdo y derecho", "Toggle between left and right panel"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("Enter  |  e"),
		theme.NormalItem.Render(i18n.T("Abrir nota en editor externo", "Open note in external editor"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("c"),
		theme.NormalItem.Render(i18n.T("Crear nueva nota con plantilla", "Create new note from template"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("p"),
		theme.NormalItem.Render(i18n.T("Pegar imagen desde el portapapeles", "Paste image from clipboard"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("f"),
		theme.NormalItem.Render(i18n.T("Ciclar filtro de tareas (Todas/Pendientes/Completadas)", "Cycle task filter (All/Pending/Done)"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("t"),
		theme.NormalItem.Render(i18n.T("Ciclar entre los 7 temas disponibles", "Cycle between 7 available color themes"))))
	rows = append(rows, fmt.Sprintf("     %s : %s",
		theme.SelectedItem.Render("? / Esc"),
		theme.NormalItem.Render(i18n.T("Cerrar esta ventana de configuración", "Close this settings view"))))

	content := strings.Join(rows, "\n")
	return borderStyle.Render(content)
}
