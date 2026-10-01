package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// TabItem define una pestaña individual
type TabItem struct {
	ID    string
	Title string
}

// DefaultTabs devuelve las pestañas por defecto localizadas
func DefaultTabs(showTags, showTasks bool) []TabItem {
	tabs := []TabItem{
		{ID: "notes", Title: i18n.T("Notas", "Notes")},
	}
	if showTags {
		tabs = append(tabs, TabItem{ID: "tags", Title: i18n.T("Categorías/Tags", "Tags/Categories")})
	}
	if showTasks {
		tabs = append(tabs, TabItem{ID: "tasks", Title: i18n.T("Tareas", "Tasks")})
	}
	return tabs
}

// RenderTabs genera la barra superior de pestañas y registra sus zonas de clic
func RenderTabs(activeTab int, totalWidth int, ht *mouse.HitTester, customTabs ...[]TabItem) string {
	var tabs []TabItem
	if len(customTabs) > 0 && len(customTabs[0]) > 0 {
		tabs = customTabs[0]
	} else {
		tabs = DefaultTabs(true, true)
	}

	var renderedTabs []string
	currentX := 0

	for i, item := range tabs {
		tabLabel := fmt.Sprintf("[%d] %s", i+1, item.Title)
		var tabStr string

		if i == activeTab {
			tabStr = theme.TabActive.Render(tabLabel)
		} else {
			tabStr = theme.TabInactive.Render(tabLabel)
		}

		tabWidth := len(tabLabel) + 4 // padding 2 a cada lado
		if ht != nil {
			ht.Register(fmt.Sprintf("tab-%d", i), mouse.ZoneTab, currentX, 0, currentX+tabWidth, 0, i, item.ID)
		}
		currentX += tabWidth + 1
		renderedTabs = append(renderedTabs, tabStr)
	}

	tabsLine := strings.Join(renderedTabs, " ")
	return tabsLine
}
