package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditorFinishedMsg se emite cuando el editor externo (micro) finaliza
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

	// Estado independiente de pestaña [2] Categorías/Tags
	tags        []views.TagInfo
	selectedTag int

	// Estado independiente de pestaña [3] Tareas
	tasks        []views.FlatTask
	selectedTask int
	taskFilter   views.TaskFilter

	// Estado independiente de pestaña [4] Galería
	images        []views.ImageEntry
	selectedImage int

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
		statusMsg:    fmt.Sprintf("%d notas cargadas", len(notes)),
	}

	// Aplicar el tema configurado
	if cfg.Theme != "" {
		theme.ApplyThemeByName(cfg.Theme)
	}

	m.rebuildDerivedData()
	return m, nil
}

// rebuildDerivedData recalcula tags, tareas e imágenes a partir de las notas actuales
func (m *AppModel) rebuildDerivedData() {
	m.tags = views.CollectTags(m.notes)
	m.tasks = views.CollectTasks(m.notes, m.taskFilter)
	m.images = views.CollectImages(m.notes)

	// Ajustar índices de selección si están fuera de rango
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
		// Recargar notas después de salir del editor
		m.reloadNotes()
		m.statusMsg = "Nota actualizada"
		return m, nil

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
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			if m.kitty != nil && m.kitty.Supported {
				// Limpiar imágenes de la terminal al salir
				fmt.Print(m.kitty.ClearAllCommand())
			}
			return m, tea.Quit

		// Navegación de pestañas [1], [2], [3], [4]
		case "1":
			m.activeTab = 0
			m.activePanel = 0
			return m, nil
		case "2":
			m.activeTab = 1
			m.activePanel = 0
			return m, nil
		case "3":
			m.activeTab = 2
			m.activePanel = 0
			return m, nil
		case "4":
			m.activeTab = 3
			m.activePanel = 0
			return m, nil

		// Alternar panel activo con Tab, Flechas o Vim h/l
		case "tab", "right", "l":
			m.activePanel = 1
			return m, nil
		case "shift+tab", "left", "h":
			m.activePanel = 0
			return m, nil

		// Ciclar tema con 't'
		case "t":
			newTheme := theme.NextTheme()
			m.statusMsg = fmt.Sprintf("Tema: %s", newTheme)
			return m, nil
		}

		// Delegar al handler de la pestaña activa
		return m.updateForTab(msg)
	}

	return m, nil
}

// updateForTab delega la lógica de teclado según la pestaña activa
func (m *AppModel) updateForTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case 0:
		return m.updateNotesTab(msg)
	case 1:
		return m.updateTagsTab(msg)
	case 2:
		return m.updateTasksTab(msg)
	case 3:
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
	case "p":
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
		// Abrir la primera nota del tag seleccionado
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
		// Abrir la nota que contiene la tarea seleccionada
		if m.selectedTask < len(m.tasks) {
			task := m.tasks[m.selectedTask]
			return m, m.openEditorForPath(task.NotePath)
		}
		return m, nil
	case "f":
		// Ciclar filtro: All → Pending → Done → All
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
		m.statusMsg = fmt.Sprintf("Filtro: %s", views.TaskFilterLabel(m.taskFilter))
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
		// Abrir la nota que contiene la imagen seleccionada
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

// ─── Manejo de clicks en zonas ─────────────────────────────────────

func (m *AppModel) handleZoneClick(zone *mouse.Zone) (tea.Model, tea.Cmd) {
	switch zone.Type {
	case mouse.ZoneTab:
		m.activeTab = zone.Index
		m.activePanel = 0
	case mouse.ZoneNote:
		if m.selectedNote == zone.Index {
			// Doble clic o clic en la seleccionada -> abrir editor
			return m, m.openEditor()
		}
		m.selectedNote = zone.Index
	case mouse.ZoneTag:
		m.selectedTag = zone.Index
	case mouse.ZoneTask:
		if m.selectedTask == zone.Index {
			// Doble clic en tarea -> abrir nota
			if m.selectedTask < len(m.tasks) {
				return m, m.openEditorForPath(m.tasks[m.selectedTask].NotePath)
			}
		}
		m.selectedTask = zone.Index
	case mouse.ZoneGallery:
		if m.selectedImage == zone.Index {
			// Doble clic en imagen -> abrir nota
			if m.selectedImage < len(m.images) {
				return m, m.openEditorForPath(m.images[m.selectedImage].NotePath)
			}
		}
		m.selectedImage = zone.Index
	case mouse.ZoneAction:
		return m.handleActionClick(zone)
	}
	return m, nil
}

// handleActionClick procesa clics en los botones del footer
func (m *AppModel) handleActionClick(zone *mouse.Zone) (tea.Model, tea.Cmd) {
	switch zone.Payload {
	case "c":
		return m, m.createQuickNote()
	case "e/Enter", "Enter", "e":
		return m, m.openEditor()
	case "d":
		return m, m.deleteCurrentNote()
	case "p":
		return m, m.pasteImage()
	case "f":
		// Ciclar filtro de tareas desde el footer
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
		m.statusMsg = fmt.Sprintf("Filtro: %s", views.TaskFilterLabel(m.taskFilter))
	case "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
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
	editor := m.cfg.Editor
	if editor == "" {
		editor = "micro"
	}

	c := exec.Command(editor, filePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func (m *AppModel) createQuickNote() tea.Cmd {
	title := fmt.Sprintf("Nueva Nota %d", len(m.notes)+1)
	_, err := m.storage.CreateNote(title)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Error: %v", err)
		return nil
	}
	m.reloadNotes()
	m.selectedNote = 0
	m.statusMsg = fmt.Sprintf("Creada '%s'", title)
	return m.openEditor()
}

func (m *AppModel) pasteImage() tea.Cmd {
	if len(m.notes) == 0 {
		m.statusMsg = "Crea una nota primero para adjuntar imágenes"
		return nil
	}
	note := m.notes[m.selectedNote]
	imgRef, err := m.clipSaver.PasteImage(note.ID)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Portapapeles: %v", err)
		return nil
	}

	// Añadir la referencia al final de la nota
	appendContent := fmt.Sprintf("\n\n![Imagen](%s)\n", imgRef)
	f, err := os.OpenFile(note.Path, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(appendContent)
		_ = f.Close()
	}

	m.reloadNotes()
	m.statusMsg = fmt.Sprintf("Imagen guardada en %s", imgRef)
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
	m.statusMsg = fmt.Sprintf("Borrada: %s", note.Title)
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
		return "¡Hasta luego!\n"
	}

	m.hitTester.Clear()

	// 1. Renderizar pestañas
	tabsView := views.RenderTabs(m.activeTab, m.width, m.hitTester)

	// Dimensiones de paneles
	panelHeight := m.height - 4 // Tabs (1) + Spacing (1) + Footer (1) + Margin (1)
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

	var leftView, rightView, footerView string

	switch m.activeTab {
	case 0: // [1] Notas
		leftView = views.RenderNoteList(m.notes, m.selectedNote, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var currentNote *storage.Note
		if len(m.notes) > 0 && m.selectedNote < len(m.notes) {
			currentNote = &m.notes[m.selectedNote]
		}
		rightView = views.RenderPreview(currentNote, rightWidth, panelHeight, m.activePanel == 1, m.kitty)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg)

	case 1: // [2] Categorías/Tags
		leftView = views.RenderTagList(m.tags, m.selectedTag, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var selectedTagName string
		if m.selectedTag < len(m.tags) {
			selectedTagName = m.tags[m.selectedTag].Name
		}
		rightView = views.RenderTagPreview(m.notes, selectedTagName, rightWidth, panelHeight, m.activePanel == 1)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.TagActions)

	case 2: // [3] Tareas
		leftView = views.RenderTaskList(m.tasks, m.selectedTask, m.taskFilter, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var currentTask *views.FlatTask
		if len(m.tasks) > 0 && m.selectedTask < len(m.tasks) {
			currentTask = &m.tasks[m.selectedTask]
		}
		rightView = views.RenderTaskPreview(currentTask, rightWidth, panelHeight, m.activePanel == 1)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.TaskActions)

	case 3: // [4] Galería de Imágenes
		leftView = views.RenderGalleryList(m.images, m.selectedImage, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)
		var currentImage *views.ImageEntry
		if len(m.images) > 0 && m.selectedImage < len(m.images) {
			currentImage = &m.images[m.selectedImage]
		}
		rightView = views.RenderGalleryPreview(currentImage, rightWidth, panelHeight, m.activePanel == 1, m.kitty)
		footerView = views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, views.GalleryActions)
	}

	// Unir paneles horizontalmente
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

	return lipgloss.JoinVertical(lipgloss.Left,
		tabsView,
		mainView,
		footerView,
	)
}
