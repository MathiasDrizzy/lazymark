package app

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
	uv "github.com/charmbracelet/ultraviolet"
)

// AppModel es el modelo raíz: decide el foco, rutea teclas y clics al panel o
// popup que corresponde y mantiene el Layout. El resto vive en los paneles.
type AppModel struct {
	c      *core
	layout Layout
	w, h   int

	focus        panelID
	lastLeft     panelID
	zoom         bool
	kanbanOn     bool
	imgTickSel   int
	mascot       mascotState
	cellW, cellH int // tamaño de celda en píxeles que dijo la terminal (0 = no contestó) // selección para la que ya se armó el tick de carga de imágenes
	ratio        float64

	notes   *notesPanel
	tasks   tasksPanel
	tags    tagsPanel
	preview previewPanel
	kanban  kanbanSheet

	// imgNote es la nota cuyas imágenes están en la terminal; emit envía una
	// secuencia cruda a la terminal (se cambia en los tests para registrarlas).
	imgNote string
	emit    func(seq any) tea.Cmd

	// ht guarda las zonas que se registran durante el render (footer y Kanban).
	ht       *mouse.HitTester
	dragging bool
	clicks   doubleClick
	quitting bool
}

// New crea el modelo con la configuración dada y carga las notas.
func New(cfg *config.Config) (*AppModel, error) {
	storage.MaxNoteBytes = cfg.MaxNoteBytes()
	store := storage.New(cfg.NotesDir)
	if _, err := store.ListNotes(); err != nil {
		return nil, err
	}
	i18n.SetLanguage(cfg.Language) // "auto" o vacío: el idioma del sistema
	views.DateIcons = cfg.NerdFont
	if cfg.Theme != "" {
		theme.ApplyThemeByName(cfg.Theme)
	}
	theme.PopupSolid = cfg.PopupBackground == config.PopupBackgroundTheme
	c := &core{
		cfg:   cfg,
		store: store,
		kitty: image.New(),
		clip:  clipboard.New(),
		keys:  NewKeymap(cfg),
	}
	c.kitty.Resolve = func(p string) (string, bool) { // las imágenes de la vista previa: solo archivos regulares de la carpeta de notas
		real, err := c.store.ResolveFile(p)
		return real, err == nil
	}
	if cfg.HideCompletedTasks {
		c.taskFilter = views.TaskFilterPending
	}
	m := &AppModel{c: c, ratio: cfg.SidebarRatio, ht: mouse.NewHitTester(), emit: tea.Raw}
	m.preview.imgs = c.kitty
	m.preview.c = c
	m.notes = newNotesPanel(c)
	m.tasks = tasksPanel{c: c}
	m.tags = tagsPanel{c: c}
	m.kanban = kanbanSheet{c: c}
	c.setStatus("%s", loadedStatus(len(c.notes)))
	return m, nil
}

// loadedStatus es el aviso de arranque, con el singular cuando hay una sola nota.
func loadedStatus(n int) string {
	if n == 1 {
		return fmt.Sprintf(i18n.T("%d nota cargada", "%d note loaded"), n)
	}
	return fmt.Sprintf(i18n.T("%d notas cargadas", "%d notes loaded"), n)
}

// Init consulta a la terminal si soporta gráficos Kitty (a=q + DA1). Mientras
// no conteste afirmativamente, las imágenes se muestran como texto.
func (m *AppModel) Init() tea.Cmd {
	return m.emit(image.QuerySequence() + mascotRequestSequence())
}

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
	m.syncPreviewToTask()
}

// syncPreviewToTask posiciona el preview en la línea de la tarea seleccionada
// mientras el foco está en el panel Tareas (H2-4). Al pasar el foco al preview
// ya no se toca el scroll, así se puede leer y desplazar desde ahí.
func (m *AppModel) syncPreviewToTask() {
	if m.focus != panelTasks || m.kanbanOn || m.layout.TooSmall {
		return
	}
	t := m.tasks.current()
	if t == nil {
		m.preview.reset()
		return
	}
	note := m.noteByPath(t.NotePath)
	inner, h := m.layout.Preview.W-2, m.layout.Preview.H-2
	if note == nil || inner < 8 || h < 1 {
		return
	}
	line := taskRenderedLine(m.preview.lines(note, inner-1), note, *t)
	m.preview.scrollX = 0
	m.preview.scrollY = max(0, line-h/3) // la tarea queda en el tercio superior, con contexto arriba
}

// Update procesa el mensaje y después deja los gráficos coherentes con lo que
// se va a dibujar, enviando antes que nada las secuencias que hagan falta.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.update(msg)
	if raws := m.prepareImages(); len(raws) > 0 {
		if cmd != nil {
			raws = append(raws, cmd)
		}
		cmd = tea.Sequence(raws...)
	}
	return m, cmd
}

func (m *AppModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imageTickMsg:
		return m, m.startImageJobs(msg)
	case imageLoadedMsg:
		m.c.kitty.Done(msg.sel, msg.job, msg.tmpl, msg.err)
	case uv.CellSizeEvent:
		// la contesta la terminal: una celda mide unas decenas de píxeles; un valor absurdo no debe pedir una imagen enorme
		if msg.Width > 0 && msg.Height > 0 && msg.Width <= maxCellPx && msg.Height <= maxCellPx {
			m.cellW, m.cellH = msg.Width, msg.Height
		}
	case mascotTickMsg:
		return m, m.mascotTick(msg)
	case searchTickMsg:
		for _, p := range m.c.popups {
			if sp, ok := p.(*searchPopup); ok {
				return m, sp.run(msg.gen)
			}
		}
	case searchDoneMsg:
		for _, p := range m.c.popups {
			if sp, ok := p.(*searchPopup); ok {
				sp.done(msg)
			}
		}
	case searchJumpMsg:
		m.jumpTo(msg.m)
	case uv.KittyGraphicsEvent:
		if image.IsSupportReply(msg) {
			m.c.kitty.SetSupported(true)
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.relayout()
	case EditorFinishedMsg:
		m.c.editing = false
		m.c.kitty.Reset() // al volver del editor las imágenes se transmiten de nuevo
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
	case tea.PasteMsg:
		return m, m.handlePaste(msg)
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
	if m.kanbanOn && key == "esc" && m.kanban.cancelDrag() { // Esc cancela un arrastre en curso
		return nil
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
	if m.focus == panelPreview && !m.kanbanOn && m.preview.sel > 0 {
		switch key { // con un enlace seleccionado, Enter lo sigue (e sigue editando) y Esc lo suelta
		case "enter":
			return m.followLink(m.preview.sel - 1)
		case "esc":
			m.preview.sel = 0
			return nil
		}
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
			closePopup(p)
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
		m.c.push(newSettingsPopup(m.c, m.onSettingsChange, m.openNotesPicker))
		return nil
	case actTrash:
		m.c.push(newTrashPopup(m.c, m.afterChange))
		return nil
	case actSearch:
		m.c.push(newSearchPopup(m.c))
		return nil
	case actDaily:
		return m.openDaily()
	case actKanban:
		m.kanbanOn = !m.kanbanOn
		m.relayout()
		return nil
	case actPaste:
		m.attachImage(m.c.clip.SaveFromClipboard)
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
		if a == actEnter { // como en lazygit: Enter entra al panel de la derecha
			m.setFocus(panelPreview)
			return nil
		}
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
		case actNextLink:
			m.moveLink(1)
		case actPrevLink:
			m.moveLink(-1)
		case actEdit:
			if m.lastLeft == panelTasks {
				if t := m.tasks.current(); t != nil {
					return m.c.openEditor(t.NotePath, t.Line)
				}
			}
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
	theme.PopupSolid = m.c.cfg.PopupBackground == config.PopupBackgroundTheme
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
	if p == panelTasks {
		m.syncPreviewToTask()
	}
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
	m.c.kitty.Reset() // no dejar imágenes en la terminal al salir
	return tea.Quit
}
