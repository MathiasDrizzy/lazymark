package views

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

type ActionBtn struct {
	Key    string
	Action string
	ID     string
}

// GetNotesActions devuelve los atajos para la pestaña Notas
func GetNotesActions() []ActionBtn {
	return []ActionBtn{
		{Key: "c", Action: i18n.T("+Nota", "+Note"), ID: "action-new"},
		{Key: "F", Action: i18n.T("+Carpeta", "+Folder"), ID: "action-folder"},
		{Key: "m", Action: i18n.T("Mover", "Move"), ID: "action-move"},
		{Key: "e", Action: i18n.T("Editar", "Edit"), ID: "action-edit"},
		{Key: "d", Action: i18n.T("Borrar", "Delete"), ID: "action-delete"},
		{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// GetTagActions devuelve los atajos contextuales de la pestaña Categorías/Tags
func GetTagActions() []ActionBtn {
	return []ActionBtn{
		{Key: "Enter", Action: i18n.T("Ver notas", "View notes"), ID: "action-view-tag"},
		{Key: "e", Action: i18n.T("Editar", "Edit"), ID: "action-edit"},
		{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// GetTaskActions devuelve los atajos contextuales de la pestaña Tareas
func GetTaskActions() []ActionBtn {
	return []ActionBtn{
		{Key: "f", Action: i18n.T("Filtro", "Filter"), ID: "action-filter"},
		{Key: "Enter", Action: i18n.T("Abrir nota", "Open note"), ID: "action-open-task"},
		{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// GetGalleryActions devuelve los atajos contextuales de la pestaña Galería
func GetGalleryActions() []ActionBtn {
	return []ActionBtn{
		{Key: "p", Action: i18n.T("Pegar img", "Paste img"), ID: "action-paste"},
		{Key: "Enter", Action: i18n.T("Abrir nota", "Open note"), ID: "action-open-gallery"},
		{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// RenderFooter renderiza la barra inferior de atajos y registra los botones clickeables
func RenderFooter(width int, ht *mouse.HitTester, posY int, statusMsg string, actions ...[]ActionBtn) string {
	var elements []string
	currentX := 1

	actionsToRender := GetNotesActions()
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
		statusBadge := theme.SelectedItem.Render(fmt.Sprintf(" %s", statusMsg))
		actionsLine = fmt.Sprintf("%s   │   %s", actionsLine, statusBadge)
	}

	return theme.FooterBar.Render(actionsLine)
}
