package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

var TabTitles = []string{
	"Notas",
	"Categorías/Tags",
	"Tareas",
	"Imágenes/Adjuntos",
}

// RenderTabs genera la barra superior de pestañas y registra sus zonas de clic
func RenderTabs(activeTab int, totalWidth int, ht *mouse.HitTester) string {
	var renderedTabs []string
	currentX := 0

	for i, title := range TabTitles {
		tabLabel := fmt.Sprintf("[%d] %s", i+1, title)
		var tabStr string

		if i == activeTab {
			tabStr = theme.TabActive.Render(tabLabel)
		} else {
			tabStr = theme.TabInactive.Render(tabLabel)
		}

		tabWidth := len(tabLabel) + 4 // padding 2 a cada lado
		if ht != nil {
			ht.Register(fmt.Sprintf("tab-%d", i), mouse.ZoneTab, currentX, 0, currentX+tabWidth, 0, i, title)
		}
		currentX += tabWidth + 1
		renderedTabs = append(renderedTabs, tabStr)
	}

	tabsLine := strings.Join(renderedTabs, " ")
	return tabsLine
}
