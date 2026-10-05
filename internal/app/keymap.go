package app

import (
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// Action es una acción de la interfaz, independiente de la tecla que la dispara.
type Action int

const (
	actNone Action = iota
	actQuit
	actCheatsheet
	actSettings
	actTrash
	actSearch
	actDaily
	actDates
	actNewFromTemplate
	actNextLink
	actPrevLink
	actPanelNotes
	actPanelTasks
	actPanelTags
	actPanelPreview
	actNextPanel
	actPrevPanel
	actLeft
	actRight
	actUp
	actDown
	actTop
	actBottom
	actPageUp
	actPageDown
	actZoom
	actKanban
	actShrink
	actGrow
	actEscape
	actEnter
	actEdit
	actNewNote
	actNewFolder
	actRename
	actMove
	actDelete
	actSelect
	actSelectAll
	actPaste
	actToggleTask
	actHideDone
	actTaskFilter
	actMoveCardLeft
	actMoveCardRight
	actMoveCardUp
	actMoveCardDown
	actRestore
	actEmptyTrash
	actConfirm
)

// Context agrupa los atajos que valen en una zona de la interfaz.
type Context int

const (
	ctxGlobal Context = iota
	ctxNav            // navegación común a listas y preview
	ctxNotes
	ctxTasks
	ctxTags
	ctxPreview
	ctxKanban
	ctxPopup // listas dentro de popups (mover, papelera, settings)
	ctxTrash
	ctxConfirm
)

// Binding asocia teclas a una acción dentro de un contexto. Footer indica si
// aparece en la barra inferior de ese contexto.
type Binding struct {
	Action Action
	Ctx    Context
	Keys   []string
	ES, EN string
	Footer bool
}

// Desc devuelve la descripción en el idioma activo.
func (b Binding) Desc() string { return i18n.T(b.ES, b.EN) }

// KeyLabel devuelve las teclas en formato legible ("↑/k", "ctrl+v").
func (b Binding) KeyLabel() string {
	labels := make([]string, len(b.Keys))
	for i, k := range b.Keys {
		labels[i] = keyLabel(k)
	}
	return strings.Join(labels, "/")
}

func keyLabel(k string) string {
	switch k {
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	case "space":
		return "<space>"
	case "shift+tab":
		return "shift+tab"
	case "shift+left":
		return "shift+←"
	case "shift+right":
		return "shift+→"
	}
	return k
}

// vimKeys son las teclas de compatibilidad Vim (X8): se pueden desactivar con
// el modo de atajos "lazy" y nunca son las únicas que disparan una acción.
var vimKeys = map[string]bool{"h": true, "j": true, "k": true, "l": true, "g": true, "G": true, "ctrl+u": true, "ctrl+d": true}

func defaultBindings() []Binding {
	return []Binding{
		{actQuit, ctxGlobal, []string{"q", "ctrl+c"}, "Salir", "Quit", true},
		{actCheatsheet, ctxGlobal, []string{"?"}, "Atajos", "Keybindings", true},
		{actPaste, ctxGlobal, []string{"ctrl+v"}, "Pegar imagen", "Paste image", false},
		{actSettings, ctxGlobal, []string{","}, "Ajustes", "Settings", false},
		{actTrash, ctxGlobal, []string{"x"}, "Papelera", "Trash", false},
		{actSearch, ctxGlobal, []string{"/"}, "Buscar", "Search", false},
		{actDaily, ctxGlobal, []string{"T"}, "Nota diaria", "Daily note", false},
		{actPanelNotes, ctxGlobal, []string{"1"}, "Panel Notas", "Notes panel", false},
		{actPanelTasks, ctxGlobal, []string{"2"}, "Panel Tareas", "Tasks panel", false},
		{actPanelTags, ctxGlobal, []string{"3"}, "Panel Categorías", "Categories panel", false},
		{actPanelPreview, ctxGlobal, []string{"4"}, "Panel Vista previa", "Preview panel", false},
		{actNextPanel, ctxGlobal, []string{"tab"}, "Panel siguiente", "Next panel", false},
		{actPrevPanel, ctxGlobal, []string{"shift+tab"}, "Panel anterior", "Previous panel", false},
		{actZoom, ctxGlobal, []string{"w"}, "Maximizar panel", "Zoom panel", false},
		{actKanban, ctxGlobal, []string{"W"}, "Tablero Kanban", "Kanban board", false},
		{actShrink, ctxGlobal, []string{"["}, "Achicar columna", "Narrow column", false},
		{actGrow, ctxGlobal, []string{"]"}, "Agrandar columna", "Widen column", false},
		{actEscape, ctxGlobal, []string{"esc"}, "Quitar filtro", "Clear filter", false},

		{actUp, ctxNav, []string{"up", "k"}, "Subir", "Up", false},
		{actDown, ctxNav, []string{"down", "j"}, "Bajar", "Down", false},
		{actLeft, ctxNav, []string{"left", "h"}, "Izquierda", "Left", false},
		{actRight, ctxNav, []string{"right", "l"}, "Derecha", "Right", false},
		{actTop, ctxNav, []string{"home", "g"}, "Al inicio", "Top", false},
		{actBottom, ctxNav, []string{"end", "G"}, "Al final", "Bottom", false},
		{actPageUp, ctxNav, []string{"pgup", "ctrl+u"}, "Página arriba", "Page up", false},
		{actPageDown, ctxNav, []string{"pgdown", "ctrl+d"}, "Página abajo", "Page down", false},

		{actEnter, ctxNotes, []string{"enter"}, "Abrir", "Open", false},
		{actEdit, ctxNotes, []string{"e"}, "Editar", "Edit", true},
		{actNewNote, ctxNotes, []string{"c"}, "Nueva nota", "New note", true},
		{actNewFromTemplate, ctxNotes, []string{"C"}, "Nueva nota desde plantilla", "New note from template", false},
		{actNewFolder, ctxNotes, []string{"F"}, "Nueva carpeta", "New folder", true},
		{actRename, ctxNotes, []string{"r"}, "Renombrar", "Rename", true},
		{actMove, ctxNotes, []string{"m"}, "Mover", "Move", true},
		{actDelete, ctxNotes, []string{"d"}, "Borrar", "Delete", true},
		{actSelect, ctxNotes, []string{"v"}, "Seleccionar", "Select", true},
		{actSelectAll, ctxNotes, []string{"V"}, "Seleccionar todas", "Select all", false},

		{actEnter, ctxTasks, []string{"enter"}, "Abrir nota", "Open note", true},
		{actToggleTask, ctxTasks, []string{"space"}, "Alternar tarea", "Toggle task", true},
		{actHideDone, ctxTasks, []string{"H"}, "Ocultar hechas", "Hide done", true},
		{actDates, ctxTasks, []string{"d"}, "Fechas", "Dates", true},
		{actTaskFilter, ctxTasks, []string{"f"}, "Filtrar tareas", "Filter tasks", false},

		{actEnter, ctxTags, []string{"enter"}, "Filtrar notas", "Filter notes", true},

		{actEdit, ctxPreview, []string{"enter", "e"}, "Editar", "Edit", true},
		{actNextLink, ctxPreview, []string{"n"}, "Siguiente enlace", "Next link", false},
		{actPrevLink, ctxPreview, []string{"N"}, "Enlace anterior", "Previous link", false},

		{actLeft, ctxKanban, []string{"left", "h"}, "Columna anterior", "Previous column", false},
		{actRight, ctxKanban, []string{"right", "l"}, "Columna siguiente", "Next column", false},
		{actMoveCardLeft, ctxKanban, []string{"H", "shift+left"}, "Mover a la izquierda", "Move left", true},
		{actMoveCardRight, ctxKanban, []string{"L", "shift+right"}, "Mover a la derecha", "Move right", true},
		{actMoveCardUp, ctxKanban, []string{"K", "shift+up"}, "Subir en la columna", "Move up", true},
		{actMoveCardDown, ctxKanban, []string{"J", "shift+down"}, "Bajar en la columna", "Move down", true},
		{actToggleTask, ctxKanban, []string{"space"}, "Alternar tarea", "Toggle task", true},
		{actDates, ctxKanban, []string{"d"}, "Fechas", "Dates", true},
		{actEdit, ctxKanban, []string{"enter", "e"}, "Editar", "Edit", true},
		{actKanban, ctxKanban, []string{"esc", "W"}, "Volver a notas", "Back to notes", false},

		{actConfirm, ctxPopup, []string{"enter"}, "Aceptar", "Accept", false},
		{actRestore, ctxTrash, []string{"r"}, "Restaurar", "Restore", true},
		{actDelete, ctxTrash, []string{"d"}, "Borrar definitivo", "Delete forever", true},
		{actEmptyTrash, ctxTrash, []string{"D"}, "Vaciar papelera", "Empty trash", true},
		{actConfirm, ctxConfirm, []string{"y", "enter"}, "Confirmar", "Confirm", true},
	}
}

// Keymap resuelve teclas a acciones por contexto. Es la única fuente de atajos:
// de aquí salen el ruteo, el footer y el cheatsheet.
type Keymap struct {
	bindings []Binding
}

// NewKeymap aplica los atajos personalizados de la config y el modo de
// compatibilidad Vim ("lazy" la desactiva).
func NewKeymap(cfg *config.Config) Keymap {
	kb := cfg.Keybindings
	overrides := map[Action]string{
		actNewNote:    kb.NewNote,
		actNewFolder:  kb.NewFolder,
		actEdit:       kb.Edit,
		actDelete:     kb.Delete,
		actMove:       kb.Move,
		actPaste:      kb.PasteImage,
		actNextPanel:  kb.TogglePanel,
		actSettings:   kb.Settings,
		actCheatsheet: kb.Cheatsheet,
	}
	noVim := cfg.KeybindingMode == config.KeybindingModeLazy
	var out []Binding
	for _, b := range defaultBindings() {
		if k := overrides[b.Action]; k != "" && b.Ctx != ctxTrash && b.Ctx != ctxPreview && b.Ctx != ctxKanban {
			b.Keys = []string{k}
		}
		if b.Action == actQuit && kb.Quit != "" {
			b.Keys = []string{kb.Quit, "ctrl+c"}
		}
		if noVim {
			var keys []string
			for _, k := range b.Keys {
				if !vimKeys[k] {
					keys = append(keys, k)
				}
			}
			b.Keys = keys
		}
		if len(b.Keys) > 0 {
			out = append(out, b)
		}
	}
	return Keymap{bindings: out}
}

// Lookup busca la acción de key en los contextos dados, en orden.
func (k Keymap) Lookup(key string, ctxs ...Context) Action {
	for _, c := range ctxs {
		for _, b := range k.bindings {
			if b.Ctx != c {
				continue
			}
			for _, bk := range b.Keys {
				if bk == key {
					return b.Action
				}
			}
		}
	}
	return actNone
}

// In devuelve los atajos de un contexto, en el orden en que se definieron.
func (k Keymap) In(c Context) []Binding {
	var out []Binding
	for _, b := range k.bindings {
		if b.Ctx == c {
			out = append(out, b)
		}
	}
	return out
}

// Footer devuelve los atajos que se muestran en la barra inferior de un contexto.
func (k Keymap) Footer(c Context) []Binding {
	var out []Binding
	for _, b := range k.In(c) {
		if b.Footer {
			out = append(out, b)
		}
	}
	return out
}

// Key devuelve la primera tecla de una acción en un contexto ("" si no tiene).
func (k Keymap) Key(a Action, c Context) string {
	for _, b := range k.In(c) {
		if b.Action == a {
			return b.Keys[0]
		}
	}
	return ""
}
