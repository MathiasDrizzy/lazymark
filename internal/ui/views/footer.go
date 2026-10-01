package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

type ActionBtn struct {
	Key    string
	Action string
	ID     string
}

// GetNotesActions devuelve los atajos para el panel de Notas
func GetNotesActions(hasTagFilter ...bool) []ActionBtn {
	var actions []ActionBtn
	if len(hasTagFilter) > 0 && hasTagFilter[0] {
		actions = append(actions, ActionBtn{Key: "Esc", Action: i18n.T("Limpiar filtro", "Clear filter"), ID: "action-clear-filter"})
	}
	actions = append(actions,
		ActionBtn{Key: "c", Action: i18n.T("+Nota", "+Note"), ID: "action-new"},
		ActionBtn{Key: "F", Action: i18n.T("+Carpeta", "+Folder"), ID: "action-folder"},
		ActionBtn{Key: "v", Action: i18n.T("Sel", "Sel"), ID: "action-select"},
		ActionBtn{Key: "m", Action: i18n.T("Mover", "Move"), ID: "action-move"},
		ActionBtn{Key: "e", Action: i18n.T("Editar", "Edit"), ID: "action-edit"},
		ActionBtn{Key: "d", Action: i18n.T("Borrar", "Delete"), ID: "action-delete"},
		ActionBtn{Key: "W", Action: i18n.T("Kanban", "Kanban"), ID: "action-toggle-kanban"},
		ActionBtn{Key: "w", Action: i18n.T("Zoom", "Zoom"), ID: "action-zoom"},
		ActionBtn{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		ActionBtn{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		ActionBtn{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	)
	return actions
}

// GetTagActions devuelve los atajos contextuales del panel Categorías/Tags
func GetTagActions(hasTagFilter ...bool) []ActionBtn {
	filterAction := i18n.T("Filtrar notas", "Filter notes")
	if len(hasTagFilter) > 0 && hasTagFilter[0] {
		filterAction = i18n.T("Quitar filtro", "Remove filter")
	}
	return []ActionBtn{
		{Key: "Enter", Action: filterAction, ID: "action-view-tag"},
		{Key: "e", Action: i18n.T("Editar", "Edit"), ID: "action-edit"},
		{Key: "W", Action: i18n.T("Kanban", "Kanban"), ID: "action-toggle-kanban"},
		{Key: "w", Action: i18n.T("Zoom", "Zoom"), ID: "action-zoom"},
		{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// GetTaskActions devuelve los atajos contextuales del panel Tareas
func GetTaskActions(hideCompleted bool) []ActionBtn {
	hideLabel := i18n.T("Ocultar hechas", "Hide done")
	if hideCompleted {
		hideLabel = i18n.T("Mostrar hechas", "Show done")
	}
	return []ActionBtn{
		{Key: "space", Action: i18n.T("Completar", "Toggle"), ID: "action-toggle-task"},
		{Key: "h", Action: hideLabel, ID: "action-hide-tasks"},
		{Key: "Enter", Action: i18n.T("Abrir nota", "Open note"), ID: "action-open-task"},
		{Key: "W", Action: i18n.T("Kanban", "Kanban"), ID: "action-toggle-kanban"},
		{Key: "w", Action: i18n.T("Zoom", "Zoom"), ID: "action-zoom"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// GetKanbanActions devuelve los atajos contextuales del tablero Kanban (Hoja 2)
func GetKanbanActions() []ActionBtn {
	return []ActionBtn{
		{Key: "h/l", Action: i18n.T("Columna", "Column"), ID: "action-kanban-col"},
		{Key: "H/L", Action: i18n.T("Mover", "Move"), ID: "action-kanban-move"},
		{Key: "space", Action: i18n.T("Completar", "Toggle"), ID: "action-toggle-task"},
		{Key: "Enter", Action: i18n.T("Editar", "Edit"), ID: "action-open-task"},
		{Key: "W", Action: i18n.T("Notas (Hoja 1)", "Notes (Sheet 1)"), ID: "action-toggle-kanban"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// GetPreviewActions devuelve los atajos contextuales del panel de Vista Previa
func GetPreviewActions() []ActionBtn {
	return []ActionBtn{
		{Key: "e", Action: i18n.T("Editar", "Edit"), ID: "action-edit"},
		{Key: "d", Action: i18n.T("Borrar", "Delete"), ID: "action-delete"},
		{Key: "p", Action: i18n.T("Pegar img", "Paste img"), ID: "action-paste"},
		{Key: "w", Action: i18n.T("Zoom", "Zoom"), ID: "action-zoom"},
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
		{Key: "w", Action: i18n.T("Zoom", "Zoom"), ID: "action-zoom"},
		{Key: "h", Action: i18n.T("Atajos", "Keys"), ID: "action-cheatsheet"},
		{Key: "?", Action: i18n.T("Ajustes", "Settings"), ID: "action-config"},
		{Key: "q", Action: i18n.T("Salir", "Quit"), ID: "action-quit"},
	}
}

// formatKeycap formatea un botón con el diseño estético de Lazygit
func formatKeycap(key, action string) (string, int) {
	bracketStyle := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0)
	keyStyle := theme.NormalItem.Copy().Foreground(theme.ColorMauve).Bold(true)
	actionStyle := theme.NormalItem.Copy().Foreground(theme.ColorText)

	rendered := fmt.Sprintf("%s%s%s %s",
		bracketStyle.Render("["),
		keyStyle.Render(key),
		bracketStyle.Render("]"),
		actionStyle.Render(action),
	)
	visualLen := 1 + len(key) + 1 + 1 + len(action)
	return rendered, visualLen
}

// RenderFooter renderiza la barra inferior de atajos estilo Lazygit con status a la derecha
func RenderFooter(width int, ht *mouse.HitTester, posY int, statusMsg string, trashCount int, actions ...[]ActionBtn) string {
	var elements []string
	currentX := 1

	// Botón interactivo de papelera estilo Lazygit
	bracketStyle := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0)
	xKeyStyle := theme.NormalItem.Copy().Foreground(theme.ColorMauve).Bold(true)
	trashStyle := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Bold(true)
	trashLabel := fmt.Sprintf("󰩹 (%d)", trashCount)
	trashBtn := fmt.Sprintf("%s%s%s %s",
		bracketStyle.Render("["),
		xKeyStyle.Render("x"),
		bracketStyle.Render("]"),
		trashStyle.Render(trashLabel),
	)
	trashBtnLen := 3 + 1 + 2 + 1 + len(fmt.Sprintf("(%d)", trashCount)) // [x] + " " + 󰩹 (N)

	if ht != nil {
		ht.Register("action-trash", mouse.ZoneAction, currentX, posY, currentX+trashBtnLen, posY, 0, "action-trash")
	}
	currentX += trashBtnLen + 2
	elements = append(elements, trashBtn)

	actionsToRender := GetNotesActions()
	if len(actions) > 0 && actions[0] != nil {
		actionsToRender = actions[0]
	}

	for _, act := range actionsToRender {
		btnStr, btnLen := formatKeycap(act.Key, act.Action)
		if ht != nil {
			ht.Register(act.ID, mouse.ZoneAction, currentX, posY, currentX+btnLen, posY, 0, act.Key)
		}
		currentX += btnLen + 2
		elements = append(elements, btnStr)
	}

	leftPart := strings.Join(elements, "  ")

	var statusBadge string
	statusLen := 0
	if statusMsg != "" {
		statusBadge = theme.SelectedItem.Copy().Foreground(theme.ColorPeach).Bold(true).Render(fmt.Sprintf(" %s", statusMsg))
		statusLen = ansi.StringWidth(statusBadge)
	}

	available := width - 2
	if available < 10 {
		available = 10
	}

	leftLen := ansi.StringWidth(leftPart)

	var finalLine string
	if statusBadge != "" && available > leftLen+statusLen+3 {
		spaces := available - leftLen - statusLen
		finalLine = fmt.Sprintf("%s%s%s", leftPart, strings.Repeat(" ", spaces), statusBadge)
	} else {
		finalLine = ansi.Truncate(leftPart, available, "")
	}

	return theme.FooterBar.Render(finalLine)
}
