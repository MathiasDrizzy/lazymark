package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// ─── Settings ──────────────────────────────────────────────────────

type settingID int

const (
	setTheme settingID = iota
	setLanguage
	setEditor
	setKeys
	setTaskScope
	setConfirmDelete
	setScreenBg
	setPopupBg
	setMascot
	setKanbanCards
	setNotesDir
	settingCount
)

// settingsPopup es el único popup de ajustes. Cada cambio se aplica y se
// guarda al momento, así se ve en vivo detrás del popup.
type settingsPopup struct {
	popupList
	c          *core
	onChange   func()
	openPicker func() // abre el selector de la carpeta de notas
}

func newSettingsPopup(c *core, onChange, openPicker func()) *settingsPopup {
	p := &settingsPopup{c: c, onChange: onChange, openPicker: openPicker}
	p.n = int(settingCount)
	return p
}

func (p *settingsPopup) contexts() []Context { return []Context{ctxPopup, ctxNav} }
func (p *settingsPopup) bottomRight() bool   { return false }

func (p *settingsPopup) handle(a Action, _ tea.KeyPressMsg) (tea.Cmd, bool) {
	switch a {
	case actConfirm, actRight:
		p.change(settingID(p.list.cursor), 1)
	case actLeft:
		p.change(settingID(p.list.cursor), -1)
	case actSettings:
		return nil, true
	default:
		p.nav(a)
	}
	return nil, false
}

func (p *settingsPopup) click(_, y int) (tea.Cmd, bool) {
	if i, again, ok := p.clickRow(y); ok && again {
		p.change(settingID(i), 1)
	}
	return nil, false
}

func (p *settingsPopup) label(id settingID) string {
	switch id {
	case setTheme:
		return i18n.T("Tema", "Theme")
	case setLanguage:
		return i18n.T("Idioma", "Language")
	case setEditor:
		return i18n.T("Editor", "Editor")
	case setKeys:
		return i18n.T("Atajos", "Keybindings")
	case setTaskScope:
		return i18n.T("Alcance de tareas", "Task scope")
	case setConfirmDelete:
		return i18n.T("Confirmar al borrar", "Confirm deletion")
	case setScreenBg:
		return i18n.T("Fondo de pantalla", "Screen background")
	case setPopupBg:
		return i18n.T("Fondo de popups", "Popup background")
	case setMascot:
		return i18n.T("Mascota", "Mascot")
	case setKanbanCards:
		return i18n.T("Tarjetas", "Cards")
	case setNotesDir:
		return i18n.T("Carpeta de notas", "Notes folder")
	}
	return ""
}

func (p *settingsPopup) value(id settingID) string {
	cfg := p.c.cfg
	switch id {
	case setTheme:
		return theme.CurrentThemeName
	case setLanguage:
		if cfg.Language == "" || cfg.Language == "auto" {
			return i18n.T("auto", "auto") + " · " + i18n.CurrentLanguage().Name()
		}
		return i18n.CurrentLanguage().Name()
	case setEditor:
		return filepath.Base(cfg.Editor)
	case setKeys:
		if cfg.KeybindingMode == config.KeybindingModeLazy {
			return "Lazy"
		}
		return "Lazy + Vim (hjkl)"
	case setTaskScope:
		return scopeLabel(cfg.TaskScope)
	case setConfirmDelete:
		if cfg.ConfirmDelete {
			return i18n.T("Sí", "Yes")
		}
		return "No"
	case setScreenBg:
		if cfg.ScreenBackground == config.ScreenBackgroundTerminal {
			return "terminal"
		}
		return i18n.T("tema", "theme")
	case setMascot:
		if cfg.Mascot {
			return i18n.T("sí", "yes")
		}
		return "no"
	case setKanbanCards:
		if cfg.KanbanCards == config.KanbanCardsCompact {
			return i18n.T("compactas", "compact")
		}
		return i18n.T("rectángulos", "rectangles")
	case setPopupBg:
		if cfg.PopupBackground == config.PopupBackgroundTheme {
			return i18n.T("tema", "theme")
		}
		return "terminal"
	case setNotesDir:
		return leftTruncate(shortPath(cfg.NotesDir), 24)
	}
	return ""
}

// change aplica el valor siguiente (dir=1) o anterior (dir=-1) del ajuste.
func (p *settingsPopup) change(id settingID, dir int) {
	if id == setNotesDir { // no se recorre: abre el selector de carpeta
		p.openPicker()
		return
	}
	cfg := p.c.cfg
	switch id {
	case setTheme:
		if dir > 0 {
			cfg.Theme = theme.NextTheme()
		} else {
			cfg.Theme = theme.PrevTheme()
		}
	case setLanguage:
		// auto (el idioma del sistema) y después cada idioma soportado, en orden
		opts := []string{"auto"}
		for _, l := range i18n.Languages {
			opts = append(opts, string(l))
		}
		cfg.Language = cycle(opts, cfg.Language, dir)
		i18n.SetLanguage(cfg.Language)
	case setEditor:
		cfg.Editor = cycle(config.DetectInstalledEditors(), filepath.Base(cfg.Editor), dir)
	case setKeys:
		if cfg.KeybindingMode == config.KeybindingModeLazy {
			cfg.KeybindingMode = config.KeybindingModeDual
		} else {
			cfg.KeybindingMode = config.KeybindingModeLazy
		}
	case setTaskScope:
		cfg.TaskScope = cycle(p.scopes(), cfg.TaskScope, dir)
	case setConfirmDelete:
		cfg.ConfirmDelete = !cfg.ConfirmDelete
	case setScreenBg:
		if cfg.ScreenBackground == config.ScreenBackgroundTerminal {
			cfg.ScreenBackground = config.ScreenBackgroundTheme
		} else {
			cfg.ScreenBackground = config.ScreenBackgroundTerminal
		}
	case setKanbanCards:
		if cfg.KanbanCards == config.KanbanCardsCompact {
			cfg.KanbanCards = config.KanbanCardsRects
		} else {
			cfg.KanbanCards = config.KanbanCardsCompact
		}
	case setMascot:
		cfg.Mascot = !cfg.Mascot
		p.c.kitty.Reset() // si se apaga, sus placeholders ya no se piden: se borra lo transmitido
	case setPopupBg:
		if cfg.PopupBackground == config.PopupBackgroundTheme {
			cfg.PopupBackground = config.PopupBackgroundTerminal
		} else {
			cfg.PopupBackground = config.PopupBackgroundTheme
		}
	}
	p.c.save()
	p.c.setStatus("%s: %s", p.label(id), p.value(id))
	p.onChange()
}

// scopes devuelve los alcances de tareas posibles: todas, cada tag y cada
// carpeta de primer nivel.
func (p *settingsPopup) scopes() []string {
	out := []string{"all"}
	for _, t := range p.c.tags {
		out = append(out, "tag:"+t.Name)
	}
	if entries, err := os.ReadDir(p.c.store.BaseDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && e.Name() != "assets" {
				out = append(out, "folder:"+e.Name())
			}
		}
	}
	return out
}

func scopeLabel(scope string) string {
	switch {
	case strings.HasPrefix(scope, "tag:"):
		return "#" + strings.TrimPrefix(scope, "tag:")
	case strings.HasPrefix(scope, "folder:"):
		return "/" + strings.TrimPrefix(scope, "folder:")
	}
	return i18n.T("Todas las notas", "All notes")
}

// cycle devuelve el elemento de opts a dir posiciones de current.
func cycle(opts []string, current string, dir int) string {
	if len(opts) == 0 {
		return current
	}
	for i, o := range opts {
		if o == current {
			return opts[(i+dir+len(opts))%len(opts)]
		}
	}
	return opts[0]
}

func (p *settingsPopup) render(l Layout) string {
	w := popupWidth(l, 56)
	p.top, p.height = 1, p.n
	labelW := 22
	var lines []string
	lines = append(lines, p.rows(w-3, func(i int) string {
		id := settingID(i)
		val := "‹ " + p.value(id) + " ›"
		return lipgloss.NewStyle().Foreground(theme.ColorText).Render(textwidth.Pad(p.label(id), labelW)) + accent(val)
	})...)
	lines = append(lines, dim(i18n.T("←/→ cambiar · clic: seleccionar, 2.º clic: cambiar", "←/→ change · click: select, 2nd click: change")))
	return theme.RenderPopup(i18n.T("Ajustes", "Settings"), escHint, lines, w)
}

// ─── Papelera ──────────────────────────────────────────────────────

// trashPopup lista la papelera con los días que le quedan a cada elemento.
// Restaurar es directo; borrar definitivo y vaciar piden confirmación.
type trashPopup struct {
	popupList
	c          *core
	items      []storage.TrashItem
	onChange   func()
	buttonsRow int
	buttonEnds [3]int // columna final (exclusiva) de cada botón
}

func newTrashPopup(c *core, onChange func()) *trashPopup {
	p := &trashPopup{c: c, onChange: onChange}
	p.refresh()
	return p
}

func (p *trashPopup) refresh() {
	p.items, _ = p.c.store.ListTrash()
	p.n = len(p.items)
	p.list.set(p.list.cursor, p.n)
}

func (p *trashPopup) contexts() []Context { return []Context{ctxTrash, ctxPopup, ctxNav} }
func (p *trashPopup) bottomRight() bool   { return false }

func (p *trashPopup) current() *storage.TrashItem {
	if p.list.cursor < len(p.items) {
		return &p.items[p.list.cursor]
	}
	return nil
}

func (p *trashPopup) handle(a Action, _ tea.KeyPressMsg) (tea.Cmd, bool) {
	if p.nav(a) {
		return nil, false
	}
	switch a {
	case actRestore, actConfirm:
		p.restore()
	case actDelete:
		p.deleteForever()
	case actEmptyTrash:
		p.empty()
	case actTrash:
		return nil, true
	}
	return nil, false
}

func (p *trashPopup) restore() {
	it := p.current()
	if it == nil {
		return
	}
	if err := p.c.store.RestoreTrashItem(it.ID); err != nil {
		p.c.errStatus("No se pudo restaurar", "Could not restore", err)
		return
	}
	p.c.setStatus(i18n.T("Restaurado: %s", "Restored: %s"), it.Name)
	p.refresh()
	p.onChange()
}

func (p *trashPopup) deleteForever() {
	it := p.current()
	if it == nil {
		return
	}
	id, name := it.ID, it.Name
	msg := fmt.Sprintf(i18n.T("¿Borrar '%s' para siempre? No se puede deshacer.", "Delete '%s' forever? This cannot be undone."), name)
	p.c.confirm(i18n.T("Borrar definitivo", "Delete forever"), msg, true, func() tea.Cmd {
		if err := p.c.store.DeleteTrashItem(id); err != nil {
			p.c.errStatus("No se pudo borrar", "Could not delete", err)
		}
		p.refresh()
		p.onChange()
		return nil
	})
}

func (p *trashPopup) empty() {
	if len(p.items) == 0 {
		return
	}
	msg := fmt.Sprintf(i18n.T("¿Vaciar la papelera (%d elementos)? No se puede deshacer.", "Empty the trash (%d items)? This cannot be undone."), len(p.items))
	p.c.confirm(i18n.T("Vaciar papelera", "Empty trash"), msg, true, func() tea.Cmd {
		if err := p.c.store.EmptyTrash(); err != nil {
			p.c.errStatus("No se pudo vaciar", "Could not empty", err)
		}
		p.refresh()
		p.onChange()
		return nil
	})
}

func (p *trashPopup) click(x, y int) (tea.Cmd, bool) {
	if y == p.buttonsRow {
		switch {
		case x < p.buttonEnds[0]:
			p.restore()
		case x < p.buttonEnds[1]:
			p.deleteForever()
		case x < p.buttonEnds[2]:
			p.empty()
		}
		return nil, false
	}
	p.clickRow(y)
	return nil, false
}

func (p *trashPopup) render(l Layout) string {
	w := popupWidth(l, 56)
	p.top = 2
	p.height = clamp(p.n, 1, max(1, l.H-12))
	lines := []string{dim(fmt.Sprintf(i18n.T("Se borra sola a los %d días", "Auto-deleted after %d days"), storage.TrashRetentionDays))}
	if p.n == 0 {
		lines = append(lines, "  "+dim(i18n.T("La papelera está vacía", "Trash is empty")))
	} else {
		lines = append(lines, p.rows(w-3, func(i int) string {
			it := p.items[i]
			icon := iconNote
			if it.IsDir {
				icon = iconFolderClosed
			}
			left := textwidth.Fit(icon+" "+it.Name, w-14)
			return left + " " + dim(fmt.Sprintf("%2d %s", it.DaysRemaining(), i18n.T("días", "days")))
		})...)
	}
	lines = append(lines, "")
	p.buttonsRow = len(lines) + 1
	buttons := []string{"[r] " + i18n.T("Restaurar", "Restore"), "[d] " + i18n.T("Borrar", "Delete"), "[D] " + i18n.T("Vaciar", "Empty")}
	x := 2 // borde + margen del popup
	for i, b := range buttons {
		x += textwidth.Width(b) + 3
		p.buttonEnds[i] = x - 1
		buttons[i] = accent(b)
	}
	lines = append(lines, strings.Join(buttons, "   "))
	return theme.RenderPopup(i18n.T("Papelera", "Trash"), escHint, lines, w)
}
