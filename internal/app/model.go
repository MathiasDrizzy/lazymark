package app

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// AppModel es el modelo raíz: decide el foco, rutea teclas y clics al panel o
// popup que corresponde y mantiene el Layout. El resto vive en los paneles.
type AppModel struct {
	c      *core
	layout Layout
	w, h   int

	focus    panelID
	lastLeft panelID
	zoom     bool
	kanbanOn bool
	ratio    float64

	notes   *notesPanel
	tasks   tasksPanel
	tags    tagsPanel
	preview previewPanel
	kanban  kanbanSheet

	// ht guarda las zonas que se registran durante el render (footer y Kanban).
	ht       *mouse.HitTester
	dragging bool
	clicks   doubleClick
	quitting bool
}

// New crea el modelo con la configuración dada y carga las notas.
func New(cfg *config.Config) (*AppModel, error) {
	store := storage.New(cfg.NotesDir)
	if _, err := store.ListNotes(); err != nil {
		return nil, err
	}
	if cfg.Language != "" && cfg.Language != "auto" {
		i18n.SetLanguage(cfg.Language)
	}
	if cfg.Theme != "" {
		theme.ApplyThemeByName(cfg.Theme)
	}
	c := &core{
		cfg:   cfg,
		store: store,
		kitty: image.New(),
		clip:  clipboard.New(filepath.Join(cfg.NotesDir, "assets")),
		keys:  NewKeymap(cfg),
	}
	if cfg.HideCompletedTasks {
		c.taskFilter = views.TaskFilterPending
	}
	m := &AppModel{c: c, ratio: cfg.SidebarRatio, ht: mouse.NewHitTester()}
	m.notes = newNotesPanel(c)
	m.tasks = tasksPanel{c: c}
	m.tags = tagsPanel{c: c}
	m.kanban = kanbanSheet{c: c}
	c.setStatus(i18n.T("%d notas cargadas", "%d notes loaded"), len(c.notes))
	return m, nil
}

func (m *AppModel) Init() tea.Cmd { return nil }

// relayout recalcula la geometría. Se llama solo ante un cambio de tamaño o de
// modo (foco con zoom, Kanban, proporción, paneles visibles, cantidad de tags).
func (m *AppModel) relayout() {
	m.layout = computeLayout(layoutInput{
		W: m.w, H: m.h, Ratio: m.ratio, Zoom: m.zoom, Focus: m.focus,
		TagCount: len(m.c.tags), ShowTasks: m.c.cfg.ShowTasksTab, ShowTags: m.c.cfg.ShowTagsTab,
		Kanban: m.kanbanOn,
	})
	if m.layout.TooSmall {
		return
	}
	// Si el panel enfocado quedó escondido, el foco vuelve a Notas.
	if (m.focus == panelTasks && m.layout.Tasks.Empty()) || (m.focus == panelTags && m.layout.Tags.Empty()) {
		m.setFocus(panelNotes)
	}
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.relayout()
	case EditorFinishedMsg:
		m.notes.reload()
		m.notes.selectPath(msg.Path)
		m.preview.cacheKey = ""
		m.relayout()
		if msg.Err != nil {
			m.c.errStatus("Error del editor", "Editor error", msg.Err)
		} else {
			m.c.setStatus("%s", i18n.T("Nota actualizada", "Note updated"))
		}
		return m, tea.ClearScreen
	case tea.KeyPressMsg:
		return m, m.handleKey(msg)
	case tea.MouseClickMsg:
		return m, m.handleClick(msg)
	case tea.MouseMotionMsg:
		m.handleMotion(msg)
	case tea.MouseReleaseMsg:
		m.handleRelease()
	case tea.MouseWheelMsg:
		m.handleWheel(msg)
	}
	return m, nil
}

func (m *AppModel) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	if key == "ctrl+c" {
		return m.quit()
	}
	if p := m.c.top(); p != nil {
		if key == "esc" { // Esc cierra cualquier popup.
			m.c.pop()
			return nil
		}
		cmd, done := p.handle(m.c.keys.Lookup(key, p.contexts()...), msg)
		if done {
			m.removePopup(p)
		}
		m.relayout()
		return cmd
	}
	return m.do(m.c.keys.Lookup(key, m.contexts()...))
}

// contexts devuelve los contextos de atajos activos, del más específico al global.
func (m *AppModel) contexts() []Context {
	if m.kanbanOn {
		return []Context{ctxKanban, ctxNav, ctxGlobal}
	}
	return []Context{m.panelContext(), ctxNav, ctxGlobal}
}

func (m *AppModel) panelContext() Context {
	switch m.focus {
	case panelTasks:
		return ctxTasks
	case panelTags:
		return ctxTags
	case panelPreview:
		return ctxPreview
	}
	return ctxNotes
}

// removePopup saca p de la pila aunque ya no esté arriba (p. ej. abrió una
// confirmación propia antes de cerrarse).
func (m *AppModel) removePopup(p popup) {
	for i, q := range m.c.popups {
		if q == p {
			m.c.popups = append(m.c.popups[:i], m.c.popups[i+1:]...)
			return
		}
	}
}

// do ejecuta una acción en el contexto actual. Lo usan el teclado y los
// botones del footer.
func (m *AppModel) do(a Action) tea.Cmd {
	switch a {
	case actNone:
		return nil
	case actQuit:
		return m.quit()
	case actCheatsheet:
		m.c.push(&cheatsheetPopup{keys: m.c.keys, ctx: m.contexts()[0]})
		return nil
	case actSettings:
		m.c.push(newSettingsPopup(m.c, m.onSettingsChange))
		return nil
	case actTrash:
		m.c.push(newTrashPopup(m.c, m.afterChange))
		return nil
	case actKanban:
		m.kanbanOn = !m.kanbanOn
		m.relayout()
		return nil
	}
	if m.kanbanOn {
		return m.kanban.key(a)
	}
	switch a {
	case actPanelNotes, actPanelTasks, actPanelTags, actPanelPreview:
		m.setFocus(panelID(a - actPanelNotes))
	case actNextPanel:
		m.cycleFocus(1)
	case actPrevPanel:
		m.cycleFocus(-1)
	case actZoom:
		m.zoom = !m.zoom
		m.relayout()
	case actShrink:
		m.setRatio(m.ratio - 0.04)
	case actGrow:
		m.setRatio(m.ratio + 0.04)
	case actEscape:
		if m.zoom && m.notes.tagFilter == "" && len(m.notes.selected) == 0 {
			m.zoom = false
			m.relayout()
			return nil
		}
		return m.afterPanel(m.notes.key(actEscape))
	case actRight:
		if m.focus != panelPreview {
			m.setFocus(panelPreview)
			return nil
		}
		m.preview.scroll(0, 4)
	case actLeft:
		if m.focus == panelPreview {
			if m.preview.scrollX > 0 {
				m.preview.scroll(0, -4)
			} else {
				m.setFocus(m.lastLeft)
			}
		}
	default:
		return m.panelAction(a)
	}
	return nil
}

// panelAction delega la acción al panel enfocado.
func (m *AppModel) panelAction(a Action) tea.Cmd {
	switch m.focus {
	case panelNotes:
		before := m.notes.list.cursor
		cmd := m.notes.key(a)
		if m.notes.list.cursor != before {
			m.preview.reset()
		}
		return m.afterPanel(cmd)
	case panelTasks:
		return m.afterPanel(m.tasks.key(a, m.afterChange))
	case panelTags:
		return m.afterPanel(m.tags.key(a, m.filterTag))
	case panelPreview:
		h := m.layout.Preview.H - 2
		switch a {
		case actUp:
			m.preview.scroll(-1, 0)
		case actDown:
			m.preview.scroll(1, 0)
		case actPageUp:
			m.preview.scroll(-h, 0)
		case actPageDown:
			m.preview.scroll(h, 0)
		case actTop:
			m.preview.scrollY = 0
		case actBottom:
			m.preview.scroll(1<<20, 0)
		case actEdit:
			if note := m.previewNote(); note != nil {
				return m.c.openEditor(note.Path, 1)
			}
		}
	}
	return nil
}

// afterPanel recalcula el layout por si cambió la cantidad de tags.
func (m *AppModel) afterPanel(cmd tea.Cmd) tea.Cmd {
	m.relayout()
	return cmd
}

// afterChange relee las notas tras un cambio en disco hecho fuera del árbol.
func (m *AppModel) afterChange() {
	m.notes.reload()
	m.kanban.clampSelection()
	m.relayout()
}

func (m *AppModel) onSettingsChange() {
	m.c.keys = NewKeymap(m.c.cfg)
	m.c.reload()
	m.tasks.list.set(m.tasks.list.cursor, len(m.c.tasks))
	m.relayout()
}

func (m *AppModel) filterTag(tag string) {
	m.notes.setFilter(tag)
	m.preview.reset()
	m.setFocus(panelNotes)
}

func (m *AppModel) setFocus(p panelID) {
	if (p == panelTasks && m.layout.Tasks.Empty() && !m.zoom) || (p == panelTags && m.layout.Tags.Empty() && !m.zoom) {
		return
	}
	m.focus = p
	if p != panelPreview {
		m.lastLeft = p
	}
	if m.zoom {
		m.relayout()
	}
}

func (m *AppModel) cycleFocus(dir int) {
	for i := 1; i <= 4; i++ {
		next := panelID((int(m.focus) + dir*i + 8) % 4)
		if next == panelPreview || (next == panelNotes) ||
			(next == panelTasks && !m.layout.Tasks.Empty()) || (next == panelTags && !m.layout.Tags.Empty()) {
			m.setFocus(next)
			return
		}
	}
}

func (m *AppModel) setRatio(r float64) {
	m.ratio = min(0.75, max(0.15, r))
	m.c.cfg.SidebarRatio = m.ratio
	m.relayout()
	m.c.setStatus(i18n.T("Columna izquierda: %d%%", "Left column: %d%%"), int(m.ratio*100))
}

// previewNote es la nota que muestra el preview según el último panel izquierdo.
func (m *AppModel) previewNote() *storage.Note {
	switch m.lastLeft {
	case panelTasks:
		if t := m.tasks.current(); t != nil {
			return m.noteByPath(t.NotePath)
		}
		return nil
	case panelTags:
		if tag := m.tags.current(); tag != "" {
			if notes := views.NotesForTag(m.c.notes, tag); len(notes) > 0 {
				return &notes[0]
			}
		}
		return nil
	}
	return m.notes.currentNote()
}

func (m *AppModel) noteByPath(path string) *storage.Note {
	for i := range m.c.notes {
		if m.c.notes[i].Path == path {
			return &m.c.notes[i]
		}
	}
	return nil
}

func (m *AppModel) quit() tea.Cmd {
	m.quitting = true
	_ = m.c.cfg.Save()
	if m.c.kitty != nil && m.c.kitty.Supported {
		fmt.Print(m.c.kitty.ClearAllCommand())
	}
	return tea.Quit
}
