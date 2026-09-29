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
	cfg          *config.Config
	storage      *storage.Storage
	kitty        *image.Client
	clipSaver    *clipboard.Saver
	hitTester    *mouse.HitTester

	notes        []storage.Note
	selectedNote int
	activeTab    int
	activePanel  int // 0: Lista izquierda, 1: Preview derecho

	width        int
	height       int
	statusMsg    string
	quitting     bool
}

// New crea e inicializa el modelo de la aplicación
func New(cfg *config.Config) (*AppModel, error) {
	st := storage.New(cfg.NotesDir)
	notes, err := st.ListNotes()
	if err != nil {
		return nil, err
	}

	assetsDir := filepath.Join(cfg.NotesDir, "assets")
	return &AppModel{
		cfg:          cfg,
		storage:      st,
		kitty:        image.New(),
		clipSaver:    clipboard.New(assetsDir),
		hitTester:    mouse.NewHitTester(),
		notes:        notes,
		selectedNote: 0,
		activeTab:    0,
		activePanel:  0,
		statusMsg:    fmt.Sprintf("%d notas cargadas", len(notes)),
	}, nil
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
		case "ctrl+c", "q":
			m.quitting = true
			if m.kitty != nil && m.kitty.Supported {
				// Limpiar imágenes de la terminal al salir
				fmt.Print(m.kitty.ClearAllCommand())
			}
			return m, tea.Quit

		// Navegación de pestañas [1], [2], [3], [4]
		case "1":
			m.activeTab = 0
			return m, nil
		case "2":
			m.activeTab = 1
			return m, nil
		case "3":
			m.activeTab = 2
			return m, nil
		case "4":
			m.activeTab = 3
			return m, nil

		// Navegación oficial con Flechas
		case "up":
			if m.selectedNote > 0 {
				m.selectedNote--
			}
			return m, nil

		case "down":
			if m.selectedNote < len(m.notes)-1 {
				m.selectedNote++
			}
			return m, nil

		// Alternar panel activo con Tab o Flechas izquierda/derecha
		case "tab", "right":
			m.activePanel = 1
			return m, nil
		case "shift+tab", "left":
			m.activePanel = 0
			return m, nil

		// Editar en editor externo (micro)
		case "enter", "e":
			return m, m.openEditor()

		// Crear nueva nota
		case "c":
			return m, m.createQuickNote()

		// Pegar imagen del portapapeles
		case "p":
			return m, m.pasteImage()

		// Borrar nota seleccionada
		case "d":
			return m, m.deleteCurrentNote()
		}
	}

	return m, nil
}

func (m *AppModel) handleZoneClick(zone *mouse.Zone) (tea.Model, tea.Cmd) {
	switch zone.Type {
	case mouse.ZoneTab:
		m.activeTab = zone.Index
	case mouse.ZoneNote:
		if m.selectedNote == zone.Index {
			// Doble clic o clic en la seleccionada -> abrir editor
			return m, m.openEditor()
		}
		m.selectedNote = zone.Index
	case mouse.ZoneAction:
		switch zone.Payload {
		case "c":
			return m, m.createQuickNote()
		case "e/Enter":
			return m, m.openEditor()
		case "d":
			return m, m.deleteCurrentNote()
		case "p":
			return m, m.pasteImage()
		case "q":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *AppModel) openEditor() tea.Cmd {
	if len(m.notes) == 0 {
		return nil
	}
	note := m.notes[m.selectedNote]
	editor := m.cfg.Editor
	if editor == "" {
		editor = "micro"
	}

	c := exec.Command(editor, note.Path)
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
}

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

	// 2. Renderizar lista izquierda
	leftView := views.RenderNoteList(m.notes, m.selectedNote, leftWidth, panelHeight, m.activePanel == 0, m.hitTester, 1)

	// 3. Renderizar preview derecho
	var currentNote *storage.Note
	if len(m.notes) > 0 && m.selectedNote < len(m.notes) {
		currentNote = &m.notes[m.selectedNote]
	}
	rightView := views.RenderPreview(currentNote, rightWidth, panelHeight, m.activePanel == 1, m.kitty)

	// Unir paneles horizontalmente
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

	// 4. Renderizar Footer
	footerView := views.RenderFooter(m.width, m.hitTester, m.height-1, m.statusMsg)

	return lipgloss.JoinVertical(lipgloss.Left,
		tabsView,
		mainView,
		footerView,
	)
}
