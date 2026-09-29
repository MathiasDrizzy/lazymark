package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

type ActionBtn struct {
	Key    string
	Action string
	ID     string
}

var DefaultActions = []ActionBtn{
	{Key: "c", Action: "Nueva", ID: "action-new"},
	{Key: "e/Enter", Action: "Editar en micro", ID: "action-edit"},
	{Key: "d", Action: "Borrar", ID: "action-delete"},
	{Key: "/", Action: "Buscar", ID: "action-search"},
	{Key: "p", Action: "Pegar imagen", ID: "action-paste"},
	{Key: "q", Action: "Salir", ID: "action-quit"},
}

// TagActions son los atajos contextuales de la pestaña Categorías/Tags
var TagActions = []ActionBtn{
	{Key: "Enter", Action: "Ver notas", ID: "action-view-tag"},
	{Key: "e", Action: "Abrir nota", ID: "action-edit"},
	{Key: "q", Action: "Salir", ID: "action-quit"},
}

// TaskActions son los atajos contextuales de la pestaña Tareas
var TaskActions = []ActionBtn{
	{Key: "f", Action: "Filtro", ID: "action-filter"},
	{Key: "Enter", Action: "Abrir nota", ID: "action-open-task"},
	{Key: "q", Action: "Salir", ID: "action-quit"},
}

// GalleryActions son los atajos contextuales de la pestaña Galería
var GalleryActions = []ActionBtn{
	{Key: "p", Action: "Pegar imagen", ID: "action-paste"},
	{Key: "Enter", Action: "Abrir nota", ID: "action-open-gallery"},
	{Key: "q", Action: "Salir", ID: "action-quit"},
}

// RenderFooter renderiza la barra inferior de atajos y registra los botones clickeables
// Si actions es nil, usa DefaultActions
func RenderFooter(width int, ht *mouse.HitTester, posY int, statusMsg string, actions ...[]ActionBtn) string {
	var elements []string
	currentX := 1

	actionsToRender := DefaultActions
	if len(actions) > 0 && actions[0] != nil {
		actionsToRender = actions[0]
	}

	for _, act := range actionsToRender {
		keyStr := theme.FooterKey.Render(fmt.Sprintf("[%s]", act.Key))
		actStr := theme.NormalItem.Copy().Foreground(theme.ColorSubtext0).Render(act.Action)
		btnStr := fmt.Sprintf("%s %s", keyStr, actStr)

		btnLen := len(act.Key) + len(act.Action) + 4
		if ht != nil {
			ht.Register(act.ID, mouse.ZoneAction, currentX, posY, currentX+btnLen, posY, 0, act.Key)
		}
		currentX += btnLen + 2
		elements = append(elements, btnStr)
	}

	actionsLine := strings.Join(elements, "  ")

	if statusMsg != "" {
		statusBadge := theme.SelectedItem.Render(fmt.Sprintf(" ℹ️  %s", statusMsg))
		actionsLine = fmt.Sprintf("%s  |  %s", actionsLine, statusBadge)
	}

	return theme.FooterBar.Width(width).Render(actionsLine)
}
