package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditorFinishedMsg se emite cuando el editor externo (micro, vim, etc.) finaliza
type EditorFinishedMsg struct {
	Err error
}

// AppModel es el modelo raíz de la aplicación Bubble Tea
type AppModel struct {
	cfg       *config.Config
	storage   *storage.Storage
	kitty     *image.Client
	clipSaver *clipboard.Saver
	hitTester *mouse.HitTester

	notes        []storage.Note
	selectedNote int
	activeTab    int
	activePanel  int // 0: Lista izquierda, 1: Preview derecho

	// Pestañas dinámicas visibles
	visibleTabs []views.TabItem

	// Estado independiente de pestaña Categorías/Tags
	tags        []views.TagInfo
	selectedTag int

	// Estado independiente de pestaña Tareas
	tasks        []views.FlatTask
	selectedTask int
	taskFilter   views.TaskFilter

	// Estado independiente de pestaña Galería
	images        []views.ImageEntry
	selectedImage int

	// Pantalla de configuración y keybindings
	showSettings bool
	settingsItem views.SettingsItem

	// Control de doble clic con ratón
	lastClickTime time.Time
	lastClickZone string

	width     int
	height    int
	statusMsg string
	quitting  bool
}

// New crea e inicializa el modelo de la aplicación
func New(cfg *config.Config) (*AppModel, error) {
	st := storage.New(cfg.NotesDir)
	notes, err := st.ListNotes()
	if err != nil {
		return nil, err
	}

	// Configurar idioma si está definido
	if cfg.Language != "" && cfg.Language != "auto" {
		i18n.SetLanguage(cfg.Language)
	}

	assetsDir := filepath.Join(cfg.NotesDir, "assets")
	m := &AppModel{
		cfg:          cfg,
		storage:      st,
		kitty:        image.New(),
		clipSaver:    clipboard.New(assetsDir),
		hitTester:    mouse.NewHitTester(),
		notes:        notes,
		selectedNote: 0,
		activeTab:    0,
		activePanel:  0,
		taskFilter:   views.TaskFilterAll,
		statusMsg:    fmt.Sprintf(i18n.T("%d notas cargadas", "%d notes loaded"), len(notes)),
	}

	// Aplicar el tema configurado
	if cfg.Theme != "" {
		theme.ApplyThemeByName(cfg.Theme)
	}

	m.refreshVisibleTabs()
	m.rebuildDerivedData()
	return m, nil
}

// refreshVisibleTabs actualiza la lista de pestañas visibles según la configuración
func (m *AppModel) refreshVisibleTabs() {
	m.visibleTabs = views.DefaultTabs(m.cfg.ShowTagsTab, m.cfg.ShowTasksTab, m.cfg.ShowGalleryTab)
	if m.activeTab >= len(m.visibleTabs) {
		m.activeTab = len(m.visibleTabs) - 1
		if m.activeTab < 0 {
			m.activeTab = 0
		}
	}
}

// rebuildDerivedData recalcula tags, tareas e imágenes a partir de las notas actuales
func (m *AppModel) rebuildDerivedData() {
	m.tags = views.CollectTags(m.notes)
	m.tasks = views.CollectTasks(m.notes, m.taskFilter)
	m.images = views.CollectImages(m.notes)

	if m.selectedTag >= len(m.tags) {
		if len(m.tags) > 0 {
			m.selectedTag = len(m.tags) - 1
		} else {
			m.selectedTag = 0
		}
	}
	if m.selectedTask >= len(m.tasks) {
		if len(m.tasks) > 0 {
			m.selectedTask = len(m.tasks) - 1
		} else {
			m.selectedTask = 0
		}
	}
	if m.selectedImage >= len(m.images) {
		if len(m.images) > 0 {
			m.selectedImage = len(m.images) - 1
		} else {
			m.selectedImage = 0
		}
	}
}

func (m *AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case EditorFinishedMsg:
		m.reloadNotes()
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error en editor", "Editor error"), msg.Err)
		} else {
			m.statusMsg = i18n.T("Nota actualizada", "Note updated")
		}

		cmds := []tea.Cmd{tea.ClearScreen}
		if m.cfg.MouseClick {
			cmds = append(cmds, tea.EnableMouseCellMotion)
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMsg:
		if !m.cfg.MouseClick {
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if zone, ok := m.hitTester.Check(msg.X, msg.Y); ok {
				return m.handleZoneClick(zone)
			}
		}

	case tea.KeyMsg:
		key := msg.String()

		// Salir de la aplicación
		if key == "ctrl+c" {
			m.quitting = true
			if m.kitty != nil && m.kitty.Supported {
				fmt.Print(m.kitty.ClearAllCommand())
			}
			return m, tea.Quit
		}

		// Si la pantalla de configuración está abierta
		if m.showSettings {
			switch key {
			case "esc", "?", "q":
				m.showSettings = false
				_ = m.cfg.Save()
				return m, nil
			case "up", "k":
				if m.settingsItem > 0 {
					m.settingsItem--
				} else {
					m.settingsItem = views.TotalSettingsItems - 1
				}
				return m, nil
			case "down", "j":
				if m.settingsItem < views.TotalSettingsItems-1 {
					m.settingsItem++
				} else {
					m.settingsItem = 0
				}
				return m, nil
			case "enter", " ", "left", "right":
				m.toggleConfigItem(m.settingsItem)
				return m, nil
			}
			return m, nil
		}

		// Atajos globales fuera de configuración
		switch key {
		case "q", "esc":
			m.quitting = true
			if m.kitty != nil && m.kitty.Supported {
				fmt.Print(m.kitty.ClearAllCommand())
			}
			return m, tea.Quit

		case "?", "F2":
			m.showSettings = true
			return m, nil

		// Navegación de pestañas [1..N]
		case "1":
			if len(m.visibleTabs) >= 1 {
				m.activeTab = 0
				m.activePanel = 0
				m.clearKittyIfNecessary()
			}
			return m, nil
		case "2":
			if len(m.visibleTabs) >= 2 {
				m.activeTab = 1
				m.activePanel = 0
				m.clearKittyIfNecessary()
			}
			return m, nil
		case "3":
			if len(m.visibleTabs) >= 3 {
				m.activeTab = 2
				m.activePanel = 0
				m.clearKittyIfNecessary()
			}
			return m, nil
		case "4":
			if len(m.visibleTabs) >= 4 {
				m.activeTab = 3
				m.activePanel = 0
				m.clearKittyIfNecessary()
			}
			return m, nil

		// Alternar panel activo con Tab, Flechas o Vim h/l
		case "tab", "right", "l":
			m.activePanel = 1
			return m, nil
		case "shift+tab", "left", "h":
			m.activePanel = 0
			return m, nil
		}

		// Delegar al handler de la pestaña activa
		return m.updateForCurrentTab(msg)
	}

	return m, nil
}

// currentTabID devuelve el identificador de la pestaña activa ("notes", "tags", "tasks", "gallery")
func (m *AppModel) currentTabID() string {
	if m.activeTab >= 0 && m.activeTab < len(m.visibleTabs) {
		return m.visibleTabs[m.activeTab].ID
	}
	return "notes"
}

// updateForCurrentTab delega la lógica de teclado según la pestaña activa
func (m *AppModel) updateForCurrentTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentTabID() {
	case "notes":
		return m.updateNotesTab(msg)
	case "tags":
		return m.updateTagsTab(msg)
	case "tasks":
		return m.updateTasksTab(msg)
	case "gallery":
		return m.updateGalleryTab(msg)
	}
	return m, nil
}

// ─── Pestaña [1] Notas ─────────────────────────────────────────────

func (m *AppModel) updateNotesTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedNote > 0 {
			m.selectedNote--
		}
		return m, nil
	case "down", "j":
		if m.selectedNote < len(m.notes)-1 {
			m.selectedNote++
		}
		return m, nil
	case "g":
		m.selectedNote = 0
		return m, nil
	case "G":
		if len(m.notes) > 0 {
			m.selectedNote = len(m.notes) - 1
		}
		return m, nil
	case "enter", "e":
		return m, m.openEditor()
	case "c":
		return m, m.createQuickNote()
	case "ctrl+v", "p":
		return m, m.pasteImage()
	case "d":
		return m, m.deleteCurrentNote()
	}
	return m, nil
}

// ─── Pestaña [2] Categorías/Tags ───────────────────────────────────

func (m *AppModel) updateTagsTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedTag > 0 {
			m.selectedTag--
		}
		return m, nil
	case "down", "j":
		if m.selectedTag < len(m.tags)-1 {
			m.selectedTag++
		}
		return m, nil
	case "g":
		m.selectedTag = 0
		return m, nil
	case "G":
		if len(m.tags) > 0 {
			m.selectedTag = len(m.tags) - 1
		}
		return m, nil
	case "enter", "e":
		if m.selectedTag < len(m.tags) {
			tag := m.tags[m.selectedTag]
			filtered := views.NotesForTag(m.notes, tag.Name)
			if len(filtered) > 0 {
				return m, m.openEditorForPath(filtered[0].Path)
			}
		}
		return m, nil
	}
	return m, nil
}

// ─── Pestaña [3] Tareas ────────────────────────────────────────────

func (m *AppModel) updateTasksTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedTask > 0 {
			m.selectedTask--
		}
		return m, nil
	case "down", "j":
		if m.selectedTask < len(m.tasks)-1 {
			m.selectedTask++
		}
		return m, nil
	case "g":
		m.selectedTask = 0
		return m, nil
	case "G":
		if len(m.tasks) > 0 {
			m.selectedTask = len(m.tasks) - 1
		}
		return m, nil
	case "enter", "e":
		if m.selectedTask < len(m.tasks) {
			task := m.tasks[m.selectedTask]
			return m, m.openEditorForPath(task.NotePath)
		}
		return m, nil
	case "f":
		switch m.taskFilter {
		case views.TaskFilterAll:
			m.taskFilter = views.TaskFilterPending
		case views.TaskFilterPending:
			m.taskFilter = views.TaskFilterDone
		case views.TaskFilterDone:
			m.taskFilter = views.TaskFilterAll
		}
		m.selectedTask = 0
		m.tasks = views.CollectTasks(m.notes, m.taskFilter)
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Filtro", "Filter"), views.TaskFilterLabel(m.taskFilter))
		return m, nil
	}
	return m, nil
}

// ─── Pestaña [4] Galería ───────────────────────────────────────────

func (m *AppModel) updateGalleryTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedImage > 0 {
			m.selectedImage--
		}
		return m, nil
	case "down", "j":
		if m.selectedImage < len(m.images)-1 {
			m.selectedImage++
		}
		return m, nil
	case "g":
		m.selectedImage = 0
		return m, nil
	case "G":
		if len(m.images) > 0 {
			m.selectedImage = len(m.images) - 1
		}
		return m, nil
	case "enter", "e":
		if m.selectedImage < len(m.images) {
			entry := m.images[m.selectedImage]
			return m, m.openEditorForPath(entry.NotePath)
		}
		return m, nil
	case "p":
		return m, m.pasteImage()
	}
	return m, nil
}

// ─── Manejo de clics en zonas ──────────────────────────────────────

func (m *AppModel) handleZoneClick(zone *mouse.Zone) (tea.Model, tea.Cmd) {
	now := time.Now()
	isDoubleClick := (zone.ID == m.lastClickZone) && (now.Sub(m.lastClickTime) < 400*time.Millisecond)
	m.lastClickTime = now
	m.lastClickZone = zone.ID

	switch zone.Type {
	case mouse.ZoneTab:
		m.activeTab = zone.Index
		m.activePanel = 0
	case mouse.ZoneNote:
		if isDoubleClick {
			// Doble clic abre el editor
			return m, m.openEditor()
		}
		// Clic simple solo selecciona la nota
		m.selectedNote = zone.Index
	case mouse.ZoneTag:
		m.selectedTag = zone.Index
	case mouse.ZoneTask:
		if isDoubleClick && m.selectedTask < len(m.tasks) {
			return m, m.openEditorForPath(m.tasks[m.selectedTask].NotePath)
		}
		m.selectedTask = zone.Index
	case mouse.ZoneGallery:
		if isDoubleClick && m.selectedImage < len(m.images) {
			return m, m.openEditorForPath(m.images[m.selectedImage].NotePath)
		}
		m.selectedImage = zone.Index
	case mouse.ZoneAction:
		return m.handleActionClick(zone)
	}
	return m, nil
}

// handleActionClick procesa clics en los botones de acción
func (m *AppModel) handleActionClick(zone *mouse.Zone) (tea.Model, tea.Cmd) {
	payload := zone.Payload

	// Comandos de configuración
	if strings.HasPrefix(payload, "select-config:") {
		var id int
		if _, err := fmt.Sscanf(payload, "select-config:%d", &id); err == nil {
			m.settingsItem = views.SettingsItem(id)
			m.toggleConfigItem(m.settingsItem)
		}
		return m, nil
	}

	if strings.HasPrefix(payload, "set-editor:") {
		m.cfg.Editor = strings.TrimPrefix(payload, "set-editor:")
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("Editor: %s", m.cfg.Editor)
		return m, nil
	}
	if strings.HasPrefix(payload, "set-lang:") {
		lang := strings.TrimPrefix(payload, "set-lang:")
		i18n.SetLanguage(lang)
		m.cfg.Language = lang
		m.refreshVisibleTabs()
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Idioma", "Language"), lang)
		return m, nil
	}
	if strings.HasPrefix(payload, "toggle-tab:") {
		target := strings.TrimPrefix(payload, "toggle-tab:")
		switch target {
		case "tags":
			m.cfg.ShowTagsTab = !m.cfg.ShowTagsTab
		case "tasks":
			m.cfg.ShowTasksTab = !m.cfg.ShowTasksTab
		case "gallery":
			m.cfg.ShowGalleryTab = !m.cfg.ShowGalleryTab
		}
		m.refreshVisibleTabs()
		_ = m.cfg.Save()
		return m, nil
	}

	switch payload {
	case "c":
		return m, m.createQuickNote()
	case "e/Enter", "Enter", "e":
		return m, m.openEditor()
	case "d":
		return m, m.deleteCurrentNote()
	case "p":
		return m, m.pasteImage()
	case "f":
		switch m.taskFilter {
		case views.TaskFilterAll:
			m.taskFilter = views.TaskFilterPending
		case views.TaskFilterPending:
			m.taskFilter = views.TaskFilterDone
		case views.TaskFilterDone:
			m.taskFilter = views.TaskFilterAll
		}
		m.selectedTask = 0
		m.tasks = views.CollectTasks(m.notes, m.taskFilter)
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Filtro", "Filter"), views.TaskFilterLabel(m.taskFilter))
	case "?":
		m.showSettings = !m.showSettings
	case "t":
		newTheme := theme.NextTheme()
		m.cfg.Theme = newTheme
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Tema", "Theme"), newTheme)
	case "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *AppModel) clearKittyIfNecessary() {
	if m.kitty != nil && m.kitty.Supported {
		fmt.Print(m.kitty.ClearAllCommand())
	}
}

func (m *AppModel) toggleConfigItem(item views.SettingsItem) {
	switch item {
	case views.ItemLanguage:
		i18n.ToggleLanguage()
		m.cfg.Language = string(i18n.CurrentLanguage())
		m.refreshVisibleTabs()
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Idioma", "Language"), m.cfg.Language)

	case views.ItemEditor:
		m.cycleEditor()

	case views.ItemTheme:
		newTheme := theme.NextTheme()
		m.cfg.Theme = newTheme
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Tema", "Theme"), newTheme)

	case views.ItemTabTags:
		m.cfg.ShowTagsTab = !m.cfg.ShowTagsTab
		m.refreshVisibleTabs()
		_ = m.cfg.Save()

	case views.ItemTabTasks:
		m.cfg.ShowTasksTab = !m.cfg.ShowTasksTab
		m.refreshVisibleTabs()
		_ = m.cfg.Save()

	case views.ItemTabGallery:
		m.cfg.ShowGalleryTab = !m.cfg.ShowGalleryTab
		m.refreshVisibleTabs()
		_ = m.cfg.Save()
	}
}

func (m *AppModel) cycleEditor() {
	editors := config.DetectInstalledEditors()
	for i, ed := range editors {
		if strings.Contains(strings.ToLower(m.cfg.Editor), ed) {
			next := editors[(i+1)%len(editors)]
			m.cfg.Editor = next
			_ = m.cfg.Save()
			m.statusMsg = fmt.Sprintf("Editor: %s", next)
			return
		}
	}
	if len(editors) > 0 {
		m.cfg.Editor = editors[0]
	} else {
		m.cfg.Editor = "micro"
	}
	_ = m.cfg.Save()
	m.statusMsg = fmt.Sprintf("Editor: %s", m.cfg.Editor)
}

// ─── Acciones ──────────────────────────────────────────────────────

func (m *AppModel) openEditor() tea.Cmd {
	if len(m.notes) == 0 {
		return nil
	}
	note := m.notes[m.selectedNote]
	return m.openEditorForPath(note.Path)
}

func (m *AppModel) openEditorForPath(filePath string) tea.Cmd {
	editorBin := config.ResolveEditorBin(m.cfg.Editor)

	// Limpiar buffer gráfico antes de abrir editor
	if m.kitty != nil && m.kitty.Supported {
		fmt.Print(m.kitty.ClearAllCommand())
	}

	c := exec.Command(editorBin, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func (m *AppModel) createQuickNote() tea.Cmd {
	title := fmt.Sprintf("%s %d", i18n.T("Nueva Nota", "New Note"), len(m.notes)+1)
	_, err := m.storage.CreateNote(title)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error", "Error"), err)
		return nil
	}
	m.reloadNotes()
	m.selectedNote = 0
	m.statusMsg = fmt.Sprintf("%s '%s'", i18n.T("Creada", "Created"), title)
	return m.openEditor()
}

func (m *AppModel) pasteImage() tea.Cmd {
	if len(m.notes) == 0 {
		m.statusMsg = i18n.T("Crea una nota primero para adjuntar imágenes", "Create a note first to attach images")
		return nil
	}
	note := m.notes[m.selectedNote]
	imgRef, err := m.clipSaver.PasteImage(note.ID)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Portapapeles", "Clipboard"), err)
		return nil
	}

	appendContent := fmt.Sprintf("\n\n![%s](%s)\n", i18n.T("Imagen", "Image"), imgRef)
	f, err := os.OpenFile(note.Path, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(appendContent)
		_ = f.Close()
	}

	m.reloadNotes()
	m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Imagen guardada en", "Image saved to"), imgRef)
	return nil
}

func (m *AppModel) deleteCurrentNote() tea.Cmd {
	if len(m.notes) == 0 {
		return nil
	}
	note := m.notes[m.selectedNote]
	_ = m.storage.DeleteNote(note.Path)
	m.reloadNotes()
	if m.selectedNote >= len(m.notes) && len(m.notes) > 0 {
		m.selectedNote = len(m.notes) - 1
	}
	m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Borrada", "Deleted"), note.Title)
	return nil
}

func (m *AppModel) reloadNotes() {
	notes, err := m.storage.ListNotes()
	if err == nil {
		m.notes = notes
	}
	m.rebuildDerivedData()
}

// ─── View ──────────────────────────────────────────────────────────

func (m *AppModel) View() string {
	if m.quitting {
		return i18n.T("¡Hasta luego!\n", "Goodbye!\n")
	}

	m.hitTester.Clear()

	// 1. Renderizar pestañas
	tabsView := views.RenderTabs(m.activeTab, m.width, m.hitTester, m.visibleTabs)

	// Dimensiones de paneles
	panelHeight := m.height - 4
	if panelHeight < 5 {
		panelHeight = 5
	}

	leftWidth := m.width / 3
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := m.width - leftWidth - 3
	if rightWidth < 30 {
		rightWidth = 30
	}

	// Si la pantalla de configuración está activa, mostrar el modal centrado
	if m.showSettings {
		return views.RenderSettingsModal(m.cfg, views.SettingsItem(m.settingsItem), m.width, m.height, m.hitTester)
	}

	var leftView, rightView, footerView string

	switch m.currentTabID() {
	case "notes":
		leftView = views.RenderNoteList(m.notes, m.selectedNote, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var currentNote *storage.Note
		if len(m.notes) > 0 && m.selectedNote < len(m.notes) {
			currentNote = &m.notes[m.selectedNote]
		}
		rightView = views.RenderPreview(currentNote, rightWidth, panelHeight, m.activePanel == 1, m.kitty)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.GetNotesActions())

	case "tags":
		leftView = views.RenderTagList(m.tags, m.selectedTag, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var selectedTagName string
		if m.selectedTag < len(m.tags) {
			selectedTagName = m.tags[m.selectedTag].Name
		}
		rightView = views.RenderTagPreview(m.notes, selectedTagName, rightWidth, panelHeight, m.activePanel == 1)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.GetTagActions())

	case "tasks":
		leftView = views.RenderTaskList(m.tasks, m.selectedTask, m.taskFilter, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var currentTask *views.FlatTask
		if len(m.tasks) > 0 && m.selectedTask < len(m.tasks) {
			currentTask = &m.tasks[m.selectedTask]
		}
		rightView = views.RenderTaskPreview(currentTask, rightWidth, panelHeight, m.activePanel == 1)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.GetTaskActions())

	case "gallery":
		leftView = views.RenderGalleryList(m.images, m.selectedImage, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var currentImage *views.ImageEntry
		if len(m.images) > 0 && m.selectedImage < len(m.images) {
			currentImage = &m.images[m.selectedImage]
		}
		rightView = views.RenderGalleryPreview(currentImage, rightWidth, panelHeight, m.activePanel == 1, m.kitty)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.GetGalleryActions())
	}

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

	return lipgloss.JoinVertical(lipgloss.Left,
		tabsView,
		mainView,
		footerView,
	)
}
