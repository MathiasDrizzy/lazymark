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

// Constantes de identificación de paneles modulares estilo Lazygit
const (
	PanelNotes   = 0 // [1] Notas
	PanelTasks   = 1 // [2] Tareas
	PanelTags    = 2 // [3] Categorías / Tags
	PanelPreview = 3 // [4] Vista Previa
)

// AppModel es el modelo raíz de la aplicación Bubble Tea
type AppModel struct {
	cfg       *config.Config
	storage   *storage.Storage
	kitty     *image.Client
	clipSaver *clipboard.Saver
	hitTester *mouse.HitTester

	notes        []storage.Note
	selectedNote int
	entries      []storage.NoteEntry // Carpetas y notas navegables
	selectedEntry int
	activeTab    int
	activePanel   int  // PanelNotes, PanelTasks, PanelTags, PanelPreview
	lastLeftPanel int  // Último panel izquierdo activo (PanelNotes, PanelTasks, PanelTags)
	isMaximized   bool // Maximización del panel activo al 100%

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

	// Pantalla de configuración y cheatsheet
	showSettings   bool
	settingsItem   views.SettingsItem
	showCheatsheet bool

	// Scroll del preview derecho
	previewScrollY int
	previewScrollX int

	// Modal para mover notas a carpetas
	showMoveModal      bool
	moveFolders        []string
	selectedMoveFolder int

	// Árbol de carpetas y selección múltiple
	expandedFolders map[string]bool
	selectedPaths   map[string]bool

	// Modal de confirmación para acciones críticas
	showConfirmModal bool
	confirmTitle     string
	confirmMsg       string
	confirmAction    func() tea.Cmd

	// Modal de papelera (Trash)
	showTrashModal    bool
	trashItems        []storage.TrashItem
	selectedTrashItem int

	// Control de doble clic con ratón
	lastClickTime time.Time
	lastClickZone string

	// Proporción de ancho de paneles (Split ratio)
	sidebarRatio      float64
	isDraggingDivider bool
	hasDraggedDivider bool

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

	entries, _ := st.ListEntries()

	// Configurar idioma si está definido
	if cfg.Language != "" && cfg.Language != "auto" {
		i18n.SetLanguage(cfg.Language)
	}

	assetsDir := filepath.Join(cfg.NotesDir, "assets")
	m := &AppModel{
		cfg:             cfg,
		storage:         st,
		kitty:           image.New(),
		clipSaver:       clipboard.New(assetsDir),
		hitTester:       mouse.NewHitTester(),
		notes:           notes,
		entries:         entries,
		expandedFolders: make(map[string]bool),
		selectedPaths:   make(map[string]bool),
		selectedNote:    0,
		selectedEntry:   0,
		activeTab:       0,
		activePanel:     PanelNotes,
		lastLeftPanel:   PanelNotes,
		isMaximized:     false,
		taskFilter:      views.TaskFilterAll,
		sidebarRatio:    cfg.SidebarRatio,
		statusMsg:       fmt.Sprintf(i18n.T("%d notas cargadas", "%d notes loaded"), len(notes)),
	}
	if m.sidebarRatio < 0.15 || m.sidebarRatio > 0.75 {
		m.sidebarRatio = 0.33
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
		prevPath := ""
		if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
			prevPath = m.entries[m.selectedEntry].Path
		}
		m.reloadEntries()
		if prevPath != "" {
			for idx, ent := range m.entries {
				if ent.Path == prevPath {
					m.selectedEntry = idx
					break
				}
			}
		}
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
		if msg.Action == tea.MouseActionMotion && m.isDraggingDivider {
			m.hasDraggedDivider = true
			m.updateSidebarRatioFromMouseX(msg.X)
			return m, nil
		}
		if msg.Action == tea.MouseActionRelease {
			if m.isDraggingDivider && !m.hasDraggedDivider {
				m.cycleSidebarRatio()
			}
			m.isDraggingDivider = false
			m.hasDraggedDivider = false
			return m, nil
		}
		if msg.Action == tea.MouseActionPress {
			if msg.Button == tea.MouseButtonWheelUp {
				if m.previewScrollY > 0 {
					m.previewScrollY -= 3
					if m.previewScrollY < 0 {
						m.previewScrollY = 0
					}
				}
				return m, nil
			}
			if msg.Button == tea.MouseButtonWheelDown {
				m.previewScrollY += 3
				return m, nil
			}
			if msg.Button == tea.MouseButtonLeft {
				if !m.isModalOpen() {
					_, _, _, divX := m.calcLayout()
					if msg.X >= divX-1 && msg.X <= divX+2 && msg.Y >= 0 && msg.Y < m.height-1 {
						m.isDraggingDivider = true
						m.hasDraggedDivider = false
						return m, nil
					}
				}
				m.isDraggingDivider = false
				m.hasDraggedDivider = false
				if zone, ok := m.hitTester.Check(msg.X, msg.Y); ok {
					return m.handleZoneClick(zone)
				}
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

		// Si la ventana de confirmación para eliminar está abierta
		if m.showConfirmModal {
			switch key {
			case "y", "Y", "enter":
				m.showConfirmModal = false
				if m.confirmAction != nil {
					return m, m.confirmAction()
				}
				return m, nil
			case "n", "N", "esc", "q":
				m.showConfirmModal = false
				m.statusMsg = i18n.T("Acción cancelada", "Action cancelled")
				return m, nil
			}
			return m, nil
		}

		// Si la ventana de la papelera está abierta
		if m.showTrashModal {
			switch key {
			case "esc", "q":
				m.showTrashModal = false
				return m, nil
			case "up", "k":
				if m.selectedTrashItem > 0 {
					m.selectedTrashItem--
				} else if len(m.trashItems) > 0 {
					m.selectedTrashItem = len(m.trashItems) - 1
				}
				return m, nil
			case "down", "j":
				if m.selectedTrashItem < len(m.trashItems)-1 {
					m.selectedTrashItem++
				} else {
					m.selectedTrashItem = 0
				}
				return m, nil
			case "r":
				return m.restoreSelectedTrashItem()
			case "d":
				return m.promptDeleteTrashItem()
			case "c":
				return m.promptEmptyTrash()
			}
			return m, nil
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
			case "enter", " ", "right", "l":
				m.toggleConfigItem(m.settingsItem, true)
				return m, nil
			case "left", "h":
				m.toggleConfigItem(m.settingsItem, false)
				return m, nil
			}
			return m, nil
		}

		// Si la ventana para mover nota está abierta
		if m.showMoveModal {
			switch key {
			case "esc", "q":
				m.showMoveModal = false
				return m, nil
			case "up", "k":
				if m.selectedMoveFolder > 0 {
					m.selectedMoveFolder--
				} else if len(m.moveFolders) > 0 {
					m.selectedMoveFolder = len(m.moveFolders) - 1
				}
				return m, nil
			case "down", "j":
				if m.selectedMoveFolder < len(m.moveFolders)-1 {
					m.selectedMoveFolder++
				} else {
					m.selectedMoveFolder = 0
				}
				return m, nil
			case "enter":
				return m, m.confirmMoveNote()
			}
			return m, nil
		}

		// Si la ventana flotante de atajos (cheatsheet) está abierta
		if m.showCheatsheet {
			switch key {
			case "esc", "q", "h", "enter", "?":
				m.showCheatsheet = false
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

		case "x":
			return m, m.openTrashModal()

		case "[", "<", "-", "alt+left":
			m.adjustSidebarRatio(-0.04)
			return m, nil

		case "]", ">", "+", "=", "alt+right":
			m.adjustSidebarRatio(0.04)
			return m, nil

		case "h":
			if m.cfg.KeybindingMode == "vim" && m.activePanel == 1 {
				m.activePanel = 0
				return m, nil
			}
			m.showCheatsheet = !m.showCheatsheet
			return m, nil

		case "w":
			m.isMaximized = !m.isMaximized
			if m.isMaximized {
				m.statusMsg = i18n.T("Panel maximizado (presiona 'w' para restaurar)", "Panel maximized (press 'w' to restore)")
			} else {
				m.statusMsg = i18n.T("Panel restaurado", "Panel restored")
			}
			return m, nil

		// Navegación modular de paneles [1..4]
		case "1":
			m.activePanel = PanelNotes
			m.lastLeftPanel = PanelNotes
			m.clearKittyIfNecessary()
			return m, nil
		case "2":
			m.activePanel = PanelTasks
			m.lastLeftPanel = PanelTasks
			m.clearKittyIfNecessary()
			return m, nil
		case "3":
			m.activePanel = PanelTags
			m.lastLeftPanel = PanelTags
			m.clearKittyIfNecessary()
			return m, nil
		case "4":
			m.activePanel = PanelPreview
			return m, nil

		// Alternar panel activo con Tab / Shift+Tab o Flechas
		case "tab":
			m.activePanel = (m.activePanel + 1) % 4
			if m.activePanel < PanelPreview {
				m.lastLeftPanel = m.activePanel
			}
			return m, nil
		case "shift+tab", "backtab":
			m.activePanel = (m.activePanel + 3) % 4
			if m.activePanel < PanelPreview {
				m.lastLeftPanel = m.activePanel
			}
			return m, nil
		case "right":
			if m.activePanel < PanelPreview {
				m.activePanel = PanelPreview
				return m, nil
			}
		case "left":
			if m.activePanel == PanelPreview {
				m.activePanel = m.lastLeftPanel
				return m, nil
			}
		}

		// Delegar al panel activo
		return m.updateForActivePanel(msg)
	}

	return m, nil
}

// updateForActivePanel delega la lógica de teclado según el panel enfocado
func (m *AppModel) updateForActivePanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case PanelNotes:
		return m.updateNotesTab(msg)
	case PanelTasks:
		return m.updateTasksTab(msg)
	case PanelTags:
		return m.updateTagsTab(msg)
	case PanelPreview:
		return m.updatePreviewPanel(msg)
	}
	return m, nil
}

// ─── Panel [1] Notas ───────────────────────────────────────────────

func (m *AppModel) updateNotesTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedEntry > 0 {
			m.selectedEntry--
			m.previewScrollY = 0
			m.previewScrollX = 0
		}
		return m, nil
	case "down", "j":
		if m.selectedEntry < len(m.entries)-1 {
			m.selectedEntry++
			m.previewScrollY = 0
			m.previewScrollX = 0
		}
		return m, nil
	case "g":
		m.selectedEntry = 0
		m.previewScrollY = 0
		m.previewScrollX = 0
		return m, nil
	case "G":
		if len(m.entries) > 0 {
			m.selectedEntry = len(m.entries) - 1
			m.previewScrollY = 0
			m.previewScrollX = 0
		}
		return m, nil
	case "enter":
		if len(m.entries) == 0 {
			return m, m.createQuickNote()
		}
		entry := m.entries[m.selectedEntry]
		if entry.Type == storage.EntryFolder {
			m.toggleFolder(entry.Path)
			return m, nil
		}
		return m, m.openEditorForPath(entry.Path)
	case "e":
		return m, m.openEditor()
	case " ":
		if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
			entry := m.entries[m.selectedEntry]
			if entry.Type == storage.EntryFolder {
				m.toggleFolder(entry.Path)
			} else {
				if m.selectedPaths[entry.Path] {
					delete(m.selectedPaths, entry.Path)
				} else {
					m.selectedPaths[entry.Path] = true
				}
			}
		}
		return m, nil
	case "v":
		if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
			entry := m.entries[m.selectedEntry]
			if entry.Type == storage.EntryNote {
				if m.selectedPaths[entry.Path] {
					delete(m.selectedPaths, entry.Path)
				} else {
					m.selectedPaths[entry.Path] = true
				}
			}
		}
		return m, nil
	case "V":
		allSelected := true
		for _, ent := range m.entries {
			if ent.Type == storage.EntryNote && !m.selectedPaths[ent.Path] {
				allSelected = false
				break
			}
		}
		if allSelected {
			m.selectedPaths = make(map[string]bool)
		} else {
			for _, ent := range m.entries {
				if ent.Type == storage.EntryNote {
					m.selectedPaths[ent.Path] = true
				}
			}
		}
		return m, nil
	case "m":
		return m, m.openMoveModal()
	case "c":
		return m, m.createQuickNote()
	case "f", "F":
		return m, m.createQuickFolder()
	case "ctrl+v", "p":
		return m, m.pasteImage()
	case "d":
		return m.promptDelete()
	case "l":
		m.activePanel = PanelPreview
		return m, nil
	}
	return m, nil
}

// ─── Panel [2] Tareas ──────────────────────────────────────────────

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
	case "l":
		m.activePanel = PanelPreview
		return m, nil
	}
	return m, nil
}

// ─── Panel [3] Categorías/Tags ─────────────────────────────────────

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
	case "l":
		m.activePanel = PanelPreview
		return m, nil
	}
	return m, nil
}

// ─── Panel [4] Vista Previa ────────────────────────────────────────

func (m *AppModel) updatePreviewPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.previewScrollY > 0 {
			m.previewScrollY--
		}
		return m, nil
	case "down", "j":
		m.previewScrollY++
		return m, nil
	case "left", "h":
		if m.previewScrollX > 0 {
			m.previewScrollX -= 4
			if m.previewScrollX < 0 {
				m.previewScrollX = 0
			}
		} else {
			m.activePanel = m.lastLeftPanel
		}
		return m, nil
	case "right", "l":
		m.previewScrollX += 4
		return m, nil
	case "pageup", "ctrl+u":
		m.previewScrollY -= 8
		if m.previewScrollY < 0 {
			m.previewScrollY = 0
		}
		return m, nil
	case "pagedown", "ctrl+d":
		m.previewScrollY += 8
		return m, nil
	case "g":
		m.previewScrollY = 0
		return m, nil
	case "G":
		m.previewScrollY = 9999
		return m, nil
	case "enter", "e":
		if m.lastLeftPanel == PanelTasks && len(m.tasks) > 0 && m.selectedTask < len(m.tasks) {
			return m, m.openEditorForPath(m.tasks[m.selectedTask].NotePath)
		}
		if m.lastLeftPanel == PanelTags && len(m.tags) > 0 && m.selectedTag < len(m.tags) {
			filtered := views.NotesForTag(m.notes, m.tags[m.selectedTag].Name)
			if len(filtered) > 0 {
				return m, m.openEditorForPath(filtered[0].Path)
			}
		}
		return m, m.openEditor()
	case "m":
		return m, m.openMoveModal()
	case "c":
		return m, m.createQuickNote()
	case "f", "F":
		return m, m.createQuickFolder()
	case "d":
		return m.promptDelete()
	case "p":
		return m, m.pasteImage()
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

func (m *AppModel) isModalOpen() bool {
	return m.showSettings || m.showCheatsheet || m.showMoveModal || m.showConfirmModal || m.showTrashModal
}

// ─── Manejo de clics en zonas ──────────────────────────────────────

func (m *AppModel) handleZoneClick(zone *mouse.Zone) (tea.Model, tea.Cmd) {
	if m.showConfirmModal {
		if zone.Payload != "confirm-yes" && zone.Payload != "confirm-no" {
			return m, nil
		}
	} else if m.showTrashModal {
		if !strings.HasPrefix(zone.Payload, "select-trash:") &&
			zone.Payload != "trash-restore" &&
			zone.Payload != "trash-delete" &&
			zone.Payload != "trash-empty" &&
			zone.Payload != "trash-close" &&
			zone.Payload != "action-trash" {
			return m, nil
		}
	} else if m.showMoveModal {
		if !strings.HasPrefix(zone.Payload, "move-to:") && zone.Payload != "confirm-no" {
			return m, nil
		}
	} else if m.showSettings {
		if !strings.HasPrefix(zone.Payload, "select-config:") && zone.Payload != "action-config" {
			return m, nil
		}
	} else if m.showCheatsheet {
		if zone.Payload != "action-cheatsheet" {
			return m, nil
		}
	}

	now := time.Now()
	isDoubleClick := (zone.ID == m.lastClickZone) && (now.Sub(m.lastClickTime) < 400*time.Millisecond)
	m.lastClickTime = now
	m.lastClickZone = zone.ID

	switch zone.Type {
	case mouse.ZoneTab:
		m.activeTab = zone.Index
		m.activePanel = PanelNotes
		m.lastLeftPanel = PanelNotes
	case mouse.ZoneNote:
		m.activePanel = PanelNotes
		m.lastLeftPanel = PanelNotes
		prevSelected := m.selectedEntry
		m.selectedEntry = zone.Index
		m.previewScrollY = 0
		m.previewScrollX = 0
		if zone.Index < len(m.entries) {
			entry := m.entries[zone.Index]
			if entry.Type == storage.EntryFolder {
				if isDoubleClick || prevSelected == zone.Index {
					m.toggleFolder(entry.Path)
				}
				return m, nil
			}
			if isDoubleClick {
				return m, m.openEditorForPath(entry.Path)
			}
		}
	case mouse.ZoneTag:
		m.activePanel = PanelTags
		m.lastLeftPanel = PanelTags
		m.selectedTag = zone.Index
	case mouse.ZoneTask:
		m.activePanel = PanelTasks
		m.lastLeftPanel = PanelTasks
		if isDoubleClick && m.selectedTask < len(m.tasks) {
			return m, m.openEditorForPath(m.tasks[m.selectedTask].NotePath)
		}
		m.selectedTask = zone.Index
	case mouse.ZoneGallery:
		m.activePanel = PanelNotes
		m.lastLeftPanel = PanelNotes
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

	// Comandos de confirmación modal
	if payload == "confirm-yes" {
		m.showConfirmModal = false
		if m.confirmAction != nil {
			return m, m.confirmAction()
		}
		return m, nil
	}
	if payload == "confirm-no" {
		m.showConfirmModal = false
		m.statusMsg = i18n.T("Acción cancelada", "Action cancelled")
		return m, nil
	}

	// Comandos de configuración
	if strings.HasPrefix(payload, "select-config:") {
		var id int
		if _, err := fmt.Sscanf(payload, "select-config:%d", &id); err == nil {
			targetItem := views.SettingsItem(id)
			if m.settingsItem != targetItem {
				// Primer clic: solo mover el cursor '>' al ítem seleccionado sin cambiar su valor
				m.settingsItem = targetItem
			} else {
				// Segundo clic en el mismo ítem ya seleccionado: cambiar/ciclar valor hacia adelante
				m.toggleConfigItem(m.settingsItem, true)
			}
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

	if strings.HasPrefix(payload, "move-to:") {
		var idx int
		if _, err := fmt.Sscanf(payload, "move-to:%d", &idx); err == nil {
			m.selectedMoveFolder = idx
			return m, m.confirmMoveNote()
		}
		return m, nil
	}

	if strings.HasPrefix(payload, "select-trash:") {
		var idx int
		if _, err := fmt.Sscanf(payload, "select-trash:%d", &idx); err == nil {
			m.selectedTrashItem = idx
		}
		return m, nil
	}

	switch payload {
	case "trash-restore":
		return m.restoreSelectedTrashItem()
	case "trash-delete":
		return m.promptDeleteTrashItem()
	case "trash-empty":
		return m.promptEmptyTrash()
	case "trash-close":
		m.showTrashModal = false
		return m, nil
	case "[", "action-shrink-panel":
		m.adjustSidebarRatio(-0.04)
		return m, nil
	case "]", "action-expand-panel":
		m.adjustSidebarRatio(0.04)
		return m, nil
	case "action-divider-click":
		m.cycleSidebarRatio()
		return m, nil
	case "action-trash":
		return m, m.openTrashModal()
	case "focus-preview":
		m.activePanel = PanelPreview
		return m, nil
	case "focus-notes":
		m.activePanel = PanelNotes
		m.lastLeftPanel = PanelNotes
		return m, nil
	case "focus-tasks":
		m.activePanel = PanelTasks
		m.lastLeftPanel = PanelTasks
		return m, nil
	case "focus-tags":
		m.activePanel = PanelTags
		m.lastLeftPanel = PanelTags
		return m, nil
	case "focus-list":
		m.activePanel = m.lastLeftPanel
		return m, nil
	case "action-zoom":
		m.isMaximized = !m.isMaximized
		return m, nil
	case "c", "action-new":
		return m, m.createQuickNote()
	case "F", "action-folder":
		return m, m.createQuickFolder()
	case "action-select":
		if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
			entry := m.entries[m.selectedEntry]
			if entry.Type == storage.EntryNote {
				if m.selectedPaths[entry.Path] {
					delete(m.selectedPaths, entry.Path)
				} else {
					m.selectedPaths[entry.Path] = true
				}
			}
		}
		return m, nil
	case "m", "action-move":
		return m, m.openMoveModal()
	case "e/Enter", "Enter", "e", "action-edit":
		return m, m.openEditor()
	case "d", "action-delete":
		return m.promptDelete()
	case "p", "action-paste":
		return m, m.pasteImage()
	case "h", "action-cheatsheet":
		m.showCheatsheet = !m.showCheatsheet
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
	case "?", "action-config":
		m.showSettings = !m.showSettings
	case "t":
		newTheme := theme.NextTheme()
		m.cfg.Theme = newTheme
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Tema", "Theme"), newTheme)
	case "q", "action-quit":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *AppModel) openMoveModal() tea.Cmd {
	if len(m.selectedPaths) == 0 {
		if len(m.entries) == 0 {
			return nil
		}
		entry := m.entries[m.selectedEntry]
		if entry.Type != storage.EntryNote {
			m.statusMsg = i18n.T("Solo se pueden mover notas", "Only notes can be moved")
			return nil
		}
	}
	folders, err := m.storage.ListFolders()
	if err != nil || len(folders) == 0 {
		m.statusMsg = i18n.T("No hay carpetas disponibles", "No folders available")
		return nil
	}
	m.moveFolders = folders
	m.selectedMoveFolder = 0
	m.showMoveModal = true
	return nil
}

func (m *AppModel) confirmMoveNote() tea.Cmd {
	if !m.showMoveModal || m.selectedMoveFolder >= len(m.moveFolders) {
		m.showMoveModal = false
		return nil
	}
	targetFolder := m.moveFolders[m.selectedMoveFolder]
	m.showMoveModal = false

	rel, _ := filepath.Rel(m.storage.BaseDir, targetFolder)
	if rel == "." || rel == "" {
		rel = "/"
	}

	// Caso de selección múltiple
	if len(m.selectedPaths) > 0 {
		count := 0
		for path := range m.selectedPaths {
			if err := m.storage.MoveNote(path, targetFolder); err == nil {
				count++
			}
		}
		m.selectedPaths = make(map[string]bool)
		m.reloadEntries()
		m.statusMsg = fmt.Sprintf(i18n.T("%d notas movidas a '%s'", "%d notes moved to '%s'"), count, rel)
		return nil
	}

	// Caso de nota individual
	if len(m.entries) == 0 || m.selectedEntry >= len(m.entries) {
		return nil
	}
	entry := m.entries[m.selectedEntry]
	err := m.storage.MoveNote(entry.Path, targetFolder)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error al mover nota", "Error moving note"), err)
		return nil
	}
	m.reloadEntries()
	m.statusMsg = fmt.Sprintf("%s '%s'  '%s'", i18n.T("Nota movida", "Note moved"), entry.Name, rel)
	return nil
}

func (m *AppModel) clearKittyIfNecessary() {
	if m.kitty != nil && m.kitty.Supported {
		fmt.Print(m.kitty.ClearAllCommand())
	}
}

func (m *AppModel) toggleConfigItem(item views.SettingsItem, forward bool) {
	switch item {
	case views.ItemLanguage:
		i18n.ToggleLanguage()
		m.cfg.Language = string(i18n.CurrentLanguage())
		m.refreshVisibleTabs()
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Idioma", "Language"), m.cfg.Language)

	case views.ItemEditor:
		m.cycleEditor(forward)

	case views.ItemTheme:
		var newTheme string
		if forward {
			newTheme = theme.NextTheme()
		} else {
			newTheme = theme.PrevTheme()
		}
		m.cfg.Theme = newTheme
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Tema", "Theme"), newTheme)

	case views.ItemKeybindings:
		modes := []string{"dual", "lazygit", "vim"}
		currIdx := 0
		for i, mode := range modes {
			if strings.EqualFold(m.cfg.KeybindingMode, mode) {
				currIdx = i
				break
			}
		}
		if forward {
			currIdx = (currIdx + 1) % len(modes)
		} else {
			currIdx = (currIdx - 1 + len(modes)) % len(modes)
		}
		m.cfg.KeybindingMode = modes[currIdx]
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Atajos", "Keys"), m.cfg.KeybindingMode)

	case views.ItemSidebarRatio:
		ratios := []float64{0.25, 0.33, 0.45, 0.55, 0.65}
		currIdx := 1
		diff := 1.0
		for i, r := range ratios {
			d := m.cfg.SidebarRatio - r
			if d < 0 {
				d = -d
			}
			if d < diff {
				diff = d
				currIdx = i
			}
		}
		if forward {
			currIdx = (currIdx + 1) % len(ratios)
		} else {
			currIdx = (currIdx - 1 + len(ratios)) % len(ratios)
		}
		m.cfg.SidebarRatio = ratios[currIdx]
		m.sidebarRatio = m.cfg.SidebarRatio
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("%s: %d%%", i18n.T("Panel izquierdo", "Left panel"), int(m.sidebarRatio*100))

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

	case views.ItemConfirmDelete:
		m.cfg.ConfirmDelete = !m.cfg.ConfirmDelete
		_ = m.cfg.Save()
		state := i18n.T("Activa", "Active")
		if !m.cfg.ConfirmDelete {
			state = i18n.T("Desactivada", "Disabled")
		}
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Confirmar borrado", "Confirm deletion"), state)
	}
}

// calcLayout calcula de forma unificada el ancho disponible, ancho de los paneles y la columna del divisor
func (m *AppModel) calcLayout() (availableWidth, leftWidth, rightWidth, divX int) {
	if m.width < 30 {
		m.width = 30
	}

	ratio := m.sidebarRatio
	if ratio < 0.15 || ratio > 0.75 {
		ratio = 0.33
		m.sidebarRatio = ratio
	}

	leftWidth = int(float64(m.width) * ratio)
	if leftWidth < 16 {
		leftWidth = 16
	}
	if leftWidth > m.width-16 {
		leftWidth = m.width - 16
	}
	rightWidth = m.width - leftWidth
	divX = leftWidth
	availableWidth = m.width
	return
}

// calcStackedHeights calcula las alturas de los 3 paneles apilados de la columna izquierda
func (m *AppModel) calcStackedHeights(totalH int) (hNotes, hTasks, hTags int) {
	if totalH < 9 {
		if totalH >= 6 {
			return totalH - 3, 3, 0
		}
		return totalH, 0, 0
	}
	if m.isMaximized {
		switch m.activePanel {
		case PanelNotes:
			return totalH, 0, 0
		case PanelTasks:
			return 0, totalH, 0
		case PanelTags:
			return 0, 0, totalH
		}
	}
	hNotes = (totalH * 50) / 100
	hTasks = (totalH * 25) / 100
	hTags = totalH - hNotes - hTasks

	// Altura mínima de 3 filas para cada panel visible
	minH := 3
	if hNotes < minH {
		hNotes = minH
	}
	if hTasks < minH {
		hTasks = minH
	}
	if hTags < minH {
		hTags = minH
	}
	diff := (hNotes + hTasks + hTags) - totalH
	if diff > 0 {
		hNotes -= diff
	}
	return hNotes, hTasks, hTags
}

func (m *AppModel) updateSidebarRatioFromMouseX(mouseX int) {
	if m.width < 30 {
		return
	}
	newRatio := float64(mouseX) / float64(m.width)
	if newRatio < 0.15 {
		newRatio = 0.15
	}
	if newRatio > 0.75 {
		newRatio = 0.75
	}
	m.sidebarRatio = newRatio
	m.cfg.SidebarRatio = newRatio
	_ = m.cfg.Save()
	m.statusMsg = fmt.Sprintf(i18n.T("Panel izquierdo: %d%%", "Left panel: %d%%"), int(m.sidebarRatio*100))
}

func (m *AppModel) adjustSidebarRatio(delta float64) {
	m.sidebarRatio += delta
	if m.sidebarRatio < 0.15 {
		m.sidebarRatio = 0.15
	}
	if m.sidebarRatio > 0.75 {
		m.sidebarRatio = 0.75
	}
	m.cfg.SidebarRatio = m.sidebarRatio
	_ = m.cfg.Save()
	m.statusMsg = fmt.Sprintf(i18n.T("Panel izquierdo: %d%%", "Left panel: %d%%"), int(m.sidebarRatio*100))
}

func (m *AppModel) cycleSidebarRatio() {
	switch {
	case m.sidebarRatio < 0.28:
		m.sidebarRatio = 0.33
	case m.sidebarRatio < 0.38:
		m.sidebarRatio = 0.50
	case m.sidebarRatio < 0.55:
		m.sidebarRatio = 0.65
	case m.sidebarRatio < 0.70:
		m.sidebarRatio = 0.25
	default:
		m.sidebarRatio = 0.33
	}
	m.cfg.SidebarRatio = m.sidebarRatio
	_ = m.cfg.Save()
	m.statusMsg = fmt.Sprintf(i18n.T("Panel izquierdo: %d%%", "Left panel: %d%%"), int(m.sidebarRatio*100))
}

func (m *AppModel) cycleEditor(forward bool) {
	editors := config.DetectInstalledEditors()
	if len(editors) == 0 {
		m.cfg.Editor = "micro"
		_ = m.cfg.Save()
		m.statusMsg = fmt.Sprintf("Editor: %s", m.cfg.Editor)
		return
	}
	for i, ed := range editors {
		if strings.Contains(strings.ToLower(m.cfg.Editor), ed) {
			var next string
			if forward {
				next = editors[(i+1)%len(editors)]
			} else {
				next = editors[(i-1+len(editors))%len(editors)]
			}
			m.cfg.Editor = next
			_ = m.cfg.Save()
			m.statusMsg = fmt.Sprintf("Editor: %s", next)
			return
		}
	}
	m.cfg.Editor = editors[0]
	_ = m.cfg.Save()
	m.statusMsg = fmt.Sprintf("Editor: %s", m.cfg.Editor)
}

// ─── Acciones ──────────────────────────────────────────────────────

func (m *AppModel) openEditor() tea.Cmd {
	if len(m.entries) == 0 {
		return nil
	}
	entry := m.entries[m.selectedEntry]
	if entry.Type == storage.EntryNote {
		return m.openEditorForPath(entry.Path)
	}
	return nil
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
	targetDir := m.storage.BaseDir
	if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
		current := m.entries[m.selectedEntry]
		if current.Type == storage.EntryFolder {
			targetDir = current.Path
			m.expandedFolders[current.Path] = true
		} else {
			targetDir = filepath.Dir(current.Path)
		}
	}

	title := fmt.Sprintf("%s %d", i18n.T("Nueva Nota", "New Note"), len(m.notes)+1)
	note, err := m.storage.CreateNoteInDir(targetDir, title)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error", "Error"), err)
		return nil
	}

	m.reloadEntries()

	// Posicionar el cursor sobre la nota recién creada
	for idx, ent := range m.entries {
		if ent.Path == note.Path {
			m.selectedEntry = idx
			break
		}
	}

	m.previewScrollY = 0
	m.previewScrollX = 0
	m.statusMsg = fmt.Sprintf("%s ' %s'", i18n.T("Creada", "Created"), title)
	return m.openEditor()
}

func (m *AppModel) createQuickFolder() tea.Cmd {
	parentDir := m.storage.BaseDir
	if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
		current := m.entries[m.selectedEntry]
		if current.Type == storage.EntryFolder {
			parentDir = current.Path
			m.expandedFolders[current.Path] = true
		} else {
			parentDir = filepath.Dir(current.Path)
		}
	}

	name := fmt.Sprintf("%s-%d", i18n.T("carpeta", "folder"), len(m.entries)+1)
	createdPath, err := m.storage.CreateFolderInDir(parentDir, name)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error", "Error"), err)
		return nil
	}

	m.expandedFolders[createdPath] = true
	m.reloadEntries()

	// Posicionar el cursor sobre la carpeta recién creada
	for idx, ent := range m.entries {
		if ent.Path == createdPath {
			m.selectedEntry = idx
			break
		}
	}

	m.previewScrollY = 0
	m.previewScrollX = 0
	m.statusMsg = fmt.Sprintf("%s ' %s'", i18n.T("Carpeta creada", "Folder created"), name)
	return nil
}

func (m *AppModel) toggleFolder(path string) {
	current := true
	if val, ok := m.expandedFolders[path]; ok {
		current = val
	}
	m.expandedFolders[path] = !current
	m.reloadEntries()
	for idx, ent := range m.entries {
		if ent.Path == path {
			m.selectedEntry = idx
			break
		}
	}
}

func (m *AppModel) promptDelete() (tea.Model, tea.Cmd) {
	// Caso 1: Hay múltiples notas seleccionadas
	if len(m.selectedPaths) > 0 {
		count := len(m.selectedPaths)
		if !m.cfg.ConfirmDelete {
			deleted := 0
			for path := range m.selectedPaths {
				if _, err := m.storage.MoveToTrash(path); err == nil {
					deleted++
				}
			}
			m.selectedPaths = make(map[string]bool)
			m.reloadEntries()
			m.statusMsg = fmt.Sprintf(i18n.T("%d notas movidas a la papelera", "%d notes moved to trash"), deleted)
			return m, nil
		}
		m.confirmTitle = i18n.T("󰀪  Eliminar Notas Seleccionadas", "󰀪  Delete Selected Notes")
		m.confirmMsg = fmt.Sprintf(i18n.T("¿Deseas mover las %d notas seleccionadas a la papelera?", "Do you want to move the %d selected notes to trash?"), count)
		m.confirmAction = func() tea.Cmd {
			deleted := 0
			for path := range m.selectedPaths {
				if _, err := m.storage.MoveToTrash(path); err == nil {
					deleted++
				}
			}
			m.selectedPaths = make(map[string]bool)
			m.reloadEntries()
			m.statusMsg = fmt.Sprintf(i18n.T("%d notas movidas a la papelera", "%d notes moved to trash"), deleted)
			return nil
		}
		m.showConfirmModal = true
		return m, nil
	}

	// Caso 2: Sin elementos en la lista
	if len(m.entries) == 0 || m.selectedEntry >= len(m.entries) {
		return m, nil
	}

	entry := m.entries[m.selectedEntry]

	// Caso 3: Es una carpeta
	if entry.Type == storage.EntryFolder {
		itemCount := m.storage.CountFolderItems(entry.Path)
		if itemCount > 0 {
			m.confirmTitle = i18n.T("󰀪  Eliminar Carpeta con Contenido", "󰀪  Delete Folder with Items")
			m.confirmMsg = fmt.Sprintf(i18n.T("La carpeta '%s' contiene %d elemento(s).\n¿Deseas moverla a la papelera junto con sus notas?", "The folder '%s' contains %d item(s).\nDo you want to move it and all its notes to trash?"), entry.Name, itemCount)
			folderPath := entry.Path
			folderName := entry.Name
			m.confirmAction = func() tea.Cmd {
				_, _ = m.storage.MoveToTrash(folderPath)
				delete(m.expandedFolders, folderPath)
				m.reloadEntries()
				m.statusMsg = fmt.Sprintf(i18n.T("Carpeta '%s' movida a la papelera", "Folder '%s' moved to trash"), folderName)
				return nil
			}
			m.showConfirmModal = true
			return m, nil
		}

		// Carpeta vacía
		_, _ = m.storage.MoveToTrash(entry.Path)
		delete(m.expandedFolders, entry.Path)
		m.reloadEntries()
		m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Carpeta movida a la papelera", "Folder moved to trash"), entry.Name)
		return m, nil
	}

	// Caso 4: Nota individual
	if m.cfg.ConfirmDelete {
		m.confirmTitle = i18n.T("󰀪  Eliminar Nota", "󰀪  Delete Note")
		m.confirmMsg = fmt.Sprintf(i18n.T("¿Deseas mover '%s' a la papelera?", "Do you want to move '%s' to trash?"), entry.Name)
		notePath := entry.Path
		noteName := entry.Name
		m.confirmAction = func() tea.Cmd {
			_, _ = m.storage.MoveToTrash(notePath)
			delete(m.selectedPaths, notePath)
			m.reloadEntries()
			m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Movido a papelera", "Moved to trash"), noteName)
			return nil
		}
		m.showConfirmModal = true
		return m, nil
	}

	// Nota individual sin modal de confirmación
	_, _ = m.storage.MoveToTrash(entry.Path)
	delete(m.selectedPaths, entry.Path)
	m.reloadEntries()
	m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Movido a papelera", "Moved to trash"), entry.Name)
	return m, nil
}

func (m *AppModel) openTrashModal() tea.Cmd {
	items, err := m.storage.ListTrash()
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error al abrir papelera", "Error opening trash"), err)
		return nil
	}
	m.trashItems = items
	m.selectedTrashItem = 0
	m.showTrashModal = true
	return nil
}

func (m *AppModel) restoreSelectedTrashItem() (tea.Model, tea.Cmd) {
	if len(m.trashItems) > 0 && m.selectedTrashItem < len(m.trashItems) {
		item := m.trashItems[m.selectedTrashItem]
		if err := m.storage.RestoreTrashItem(item.ID); err != nil {
			m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Error al restaurar", "Error restoring"), err)
		} else {
			m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Restaurado", "Restored"), item.Name)
		}
		m.trashItems, _ = m.storage.ListTrash()
		if m.selectedTrashItem >= len(m.trashItems) && len(m.trashItems) > 0 {
			m.selectedTrashItem = len(m.trashItems) - 1
		}
		m.reloadEntries()
	}
	return m, nil
}

func (m *AppModel) promptDeleteTrashItem() (tea.Model, tea.Cmd) {
	if len(m.trashItems) > 0 && m.selectedTrashItem < len(m.trashItems) {
		item := m.trashItems[m.selectedTrashItem]
		m.confirmTitle = i18n.T("󰀪  Eliminar Definitivamente", "󰀪  Permanently Delete")
		m.confirmMsg = fmt.Sprintf(i18n.T("¿Estás seguro de que deseas eliminar permanentemente '%s'?\nEsta acción no se puede deshacer.", "Are you sure you want to permanently delete '%s'?\nThis action cannot be undone."), item.Name)
		m.confirmAction = func() tea.Cmd {
			_ = m.storage.DeleteTrashItem(item.ID)
			m.trashItems, _ = m.storage.ListTrash()
			if m.selectedTrashItem >= len(m.trashItems) && len(m.trashItems) > 0 {
				m.selectedTrashItem = len(m.trashItems) - 1
			}
			m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Eliminado permanente", "Permanently deleted"), item.Name)
			return nil
		}
		m.showConfirmModal = true
	}
	return m, nil
}

func (m *AppModel) promptEmptyTrash() (tea.Model, tea.Cmd) {
	count := len(m.trashItems)
	if count > 0 {
		m.confirmTitle = i18n.T("󰀪  Vaciar Papelera", "󰀪  Empty Trash")
		m.confirmMsg = fmt.Sprintf(i18n.T("¿Estás seguro de que deseas vaciar la papelera (%d elementos)?\nTodos los archivos se eliminarán permanentemente.", "Are you sure you want to empty the trash (%d items)?\nAll files will be permanently deleted."), count)
		m.confirmAction = func() tea.Cmd {
			_ = m.storage.EmptyTrash()
			m.trashItems = nil
			m.selectedTrashItem = 0
			m.statusMsg = fmt.Sprintf(i18n.T("Papelera vaciada (%d elementos)", "Trash emptied (%d items)"), count)
			return nil
		}
		m.showConfirmModal = true
	}
	return m, nil
}

func (m *AppModel) pasteImage() tea.Cmd {
	if len(m.entries) == 0 {
		m.statusMsg = i18n.T("Crea una nota primero para adjuntar imágenes", "Create a note first to attach images")
		return nil
	}
	entry := m.entries[m.selectedEntry]
	if entry.Type != storage.EntryNote {
		m.statusMsg = i18n.T("Selecciona una nota para pegar la imagen", "Select a note to paste image")
		return nil
	}
	noteID := filepath.Base(entry.Path)
	imgRef, err := m.clipSaver.PasteImage(noteID)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s: %v", i18n.T("Portapapeles", "Clipboard"), err)
		return nil
	}

	appendContent := fmt.Sprintf("\n\n![%s](%s)\n", i18n.T("Imagen", "Image"), imgRef)
	f, err := os.OpenFile(entry.Path, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(appendContent)
		_ = f.Close()
	}

	m.reloadEntries()
	m.statusMsg = fmt.Sprintf("%s: %s", i18n.T("Imagen guardada en", "Image saved to"), imgRef)
	return nil
}

func (m *AppModel) deleteCurrentEntry() tea.Cmd {
	_, cmd := m.promptDelete()
	return cmd
}

func (m *AppModel) reloadEntries() {
	entries, err := m.storage.ListTreeEntries(m.expandedFolders)
	if err == nil {
		m.entries = entries
	}
	if m.selectedEntry >= len(m.entries) && len(m.entries) > 0 {
		m.selectedEntry = len(m.entries) - 1
	}
	m.reloadNotes()
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

	// 1. Dimensiones verticales (reserva 1 fila inferior para el footer)
	totalH := m.height - 1
	if totalH < 3 {
		totalH = 3
	}

	// 2. Dimensiones horizontales
	_, leftWidth, rightWidth, divX := m.calcLayout()

	// 3. Alturas apiladas de la columna izquierda (50% notas, 25% tareas, 25% tags)
	hNotes, hTasks, hTags := m.calcStackedHeights(totalH)

	// 4. Registro de zonas de fondo base en HitTester (REVERSE ORDER LAYERING - Incidente #04)
	// Se registran PRIMERO para tener menor prioridad z-index que los ítems internos
	if m.hitTester != nil && !m.isModalOpen() {
		if m.isMaximized && m.activePanel == PanelPreview {
			m.hitTester.Register("panel-preview", mouse.ZoneAction, 0, 0, m.width, totalH-1, 0, "focus-preview")
		} else {
			m.hitTester.Register("panel-preview", mouse.ZoneAction, divX, 0, m.width, totalH-1, 0, "focus-preview")
			m.hitTester.Register("panel-divider", mouse.ZoneAction, divX-1, 0, divX+1, totalH-1, 0, "action-divider-click")

			currY := 0
			if hNotes > 0 {
				m.hitTester.Register("panel-notes", mouse.ZoneAction, 0, currY, leftWidth, currY+hNotes-1, 0, "focus-notes")
				currY += hNotes
			}
			if hTasks > 0 {
				m.hitTester.Register("panel-tasks", mouse.ZoneAction, 0, currY, leftWidth, currY+hTasks-1, 0, "focus-tasks")
				currY += hTasks
			}
			if hTags > 0 {
				m.hitTester.Register("panel-tags", mouse.ZoneAction, 0, currY, leftWidth, currY+hTags-1, 0, "focus-tags")
			}
		}
	}

	// 5. Renderizado de vistas apiladas en la columna izquierda
	var leftStacked []string
	currY := 0

	if hNotes > 0 {
		notesView := views.RenderNoteList(m.entries, m.selectedPaths, m.selectedEntry, leftWidth, hNotes, m.activePanel == PanelNotes, m.hitTester, currY)
		leftStacked = append(leftStacked, notesView)
		currY += hNotes
	}

	if hTasks > 0 {
		tasksView := views.RenderTaskList(m.tasks, m.selectedTask, m.taskFilter, leftWidth, hTasks, m.activePanel == PanelTasks, m.hitTester, currY)
		leftStacked = append(leftStacked, tasksView)
		currY += hTasks
	}

	if hTags > 0 {
		tagsView := views.RenderTagList(m.tags, m.selectedTag, leftWidth, hTags, m.activePanel == PanelTags, m.hitTester, currY)
		leftStacked = append(leftStacked, tagsView)
	}

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, leftStacked...)

	// 6. Renderizado del panel derecho de Vista Previa (Preview)
	var rightView string
	previewW := rightWidth
	if m.isMaximized && m.activePanel == PanelPreview {
		previewW = m.width
	}

	previewContext := m.activePanel
	if previewContext == PanelPreview {
		previewContext = m.lastLeftPanel
	}

	switch previewContext {
	case PanelTasks:
		var currentTask *views.FlatTask
		if len(m.tasks) > 0 && m.selectedTask < len(m.tasks) {
			currentTask = &m.tasks[m.selectedTask]
		}
		rightView = views.RenderTaskPreview(currentTask, previewW, totalH, m.activePanel == PanelPreview)

	case PanelTags:
		var selectedTagName string
		if m.selectedTag < len(m.tags) {
			selectedTagName = m.tags[m.selectedTag].Name
		}
		rightView = views.RenderTagPreview(m.notes, selectedTagName, previewW, totalH, m.activePanel == PanelPreview)

	default: // PanelNotes
		var currentNote *storage.Note
		if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
			currentNote = m.entries[m.selectedEntry].Note
		}
		rightView = views.RenderPreview(currentNote, previewW, totalH, m.activePanel == PanelPreview, m.kitty, m.previewScrollY, m.previewScrollX)
	}

	// 7. Combinar columnas
	var mainView string
	if m.isMaximized && m.activePanel == PanelPreview {
		mainView = rightView
	} else {
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightView)
	}

	// 8. Footer contextual dinámico según el panel actualmente enfocado
	var actions []views.ActionBtn
	switch m.activePanel {
	case PanelNotes:
		actions = views.GetNotesActions()
	case PanelTasks:
		actions = views.GetTaskActions()
	case PanelTags:
		actions = views.GetTagActions()
	case PanelPreview:
		actions = views.GetPreviewActions()
	}

	footerView := views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg, m.storage.CountTrash(), actions)

	fullView := lipgloss.JoinVertical(lipgloss.Left, mainView, footerView)

	currentBase := fullView

	// 9. Modales superpuestos
	if m.showTrashModal {
		trashModal := views.RenderTrashModal(m.trashItems, m.selectedTrashItem, m.width, m.height, m.hitTester)
		currentBase = views.OverlayLayers(currentBase, trashModal, m.width, m.height, false)
	}

	if m.showMoveModal {
		noteName := ""
		if len(m.selectedPaths) > 1 {
			noteName = fmt.Sprintf(i18n.T("%d notas seleccionadas", "%d selected notes"), len(m.selectedPaths))
		} else if len(m.entries) > 0 && m.selectedEntry < len(m.entries) {
			noteName = m.entries[m.selectedEntry].Name
		}
		moveModal := views.RenderMoveModal(m.moveFolders, m.storage.BaseDir, m.selectedMoveFolder, noteName, m.width, m.height, m.hitTester)
		currentBase = views.OverlayLayers(currentBase, moveModal, m.width, m.height, false)
	}

	// Si la ventana de confirmación está activa, mostrarla superpuesta centrada en primer plano
	if m.showConfirmModal {
		confirmModal := views.RenderConfirmModal(m.confirmTitle, m.confirmMsg, m.width, m.height, m.hitTester)
		return views.OverlayLayers(currentBase, confirmModal, m.width, m.height, false)
	}

	if m.showTrashModal || m.showMoveModal {
		return currentBase
	}

	// Si la pantalla de configuración está activa, mostrar modal superpuesto en vivo abajo a la derecha
	if m.showSettings {
		modal := views.RenderSettingsModal(m.cfg, views.SettingsItem(m.settingsItem), m.width, m.height, m.hitTester)
		return views.OverlayLayers(fullView, modal, m.width, m.height, true)
	}

	// Si la ventana de atajos está activa, mostrarla superpuesta abajo a la derecha
	if m.showCheatsheet {
		cheatsheet := views.RenderCheatsheet(m.width, m.height)
		return views.OverlayLayers(fullView, cheatsheet, m.width, m.height, true)
	}

	return fullView
}
