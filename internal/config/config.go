package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	Version = "0.1.0"
	AppName = "lazymark"
)

// KeybindingsConfig almacena los atajos de teclado configurables
type KeybindingsConfig struct {
	NewNote     string `json:"new_note"`
	NewFolder   string `json:"new_folder"`
	Edit        string `json:"edit"`
	Delete      string `json:"delete"`
	Move        string `json:"move"`
	PasteImage  string `json:"paste_image"`
	TogglePanel string `json:"toggle_panel"`
	Settings    string `json:"settings"`
	Cheatsheet  string `json:"cheatsheet"`
	Quit        string `json:"quit"`
}

// DefaultKeybindings devuelve los atajos predeterminados del sistema
func DefaultKeybindings() KeybindingsConfig {
	return KeybindingsConfig{
		NewNote:     "c",
		NewFolder:   "F",
		Edit:        "e",
		Delete:      "d",
		Move:        "m",
		PasteImage:  "ctrl+v",
		TogglePanel: "tab",
		Settings:    "?",
		Cheatsheet:  "h",
		Quit:        "q",
	}
}

// Config almacena las preferencias de ejecución de la aplicación.
type Config struct {
	NotesDir       string            `json:"notes_dir"`
	Editor         string            `json:"editor"`
	MouseClick     bool              `json:"mouse_click"`
	Theme          string            `json:"theme"`
	Language       string            `json:"language"`
	ShowTagsTab    bool              `json:"show_tags_tab"`
	ShowTasksTab   bool              `json:"show_tasks_tab"`
	ShowGalleryTab bool              `json:"show_gallery_tab"`
	ConfirmDelete      bool              `json:"confirm_delete"`
	HideCompletedTasks bool              `json:"hide_completed_tasks"`
	SidebarRatio       float64           `json:"sidebar_ratio"`
	KeybindingMode     string            `json:"keybinding_mode"`
	Keybindings        KeybindingsConfig `json:"keybindings"`
	configPath         string            `json:"-"`
}

func configFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, AppName, "config.json")
}

// DefaultConfig devuelve la configuración inicial por defecto
func DefaultConfig(notesDir string) *Config {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if path, err := exec.LookPath("micro"); err == nil {
			editor = path
		} else if _, err := os.Stat("/opt/homebrew/bin/micro"); err == nil {
			editor = "/opt/homebrew/bin/micro"
		} else if _, err := os.Stat("/usr/local/bin/micro"); err == nil {
			editor = "/usr/local/bin/micro"
		} else if path, err := exec.LookPath("vim"); err == nil {
			editor = path
		} else if path, err := exec.LookPath("nano"); err == nil {
			editor = path
		} else {
			editor = "micro"
		}
	}

	return &Config{
		NotesDir:           notesDir,
		Editor:             editor,
		MouseClick:         true,
		Theme:              "catppuccin-mocha",
		Language:           "auto",
		ShowTagsTab:        true,
		ShowTasksTab:       true,
		ShowGalleryTab:     false,
		ConfirmDelete:      true,
		HideCompletedTasks: false,
		SidebarRatio:       0.33,
		KeybindingMode:     "dual",
		Keybindings:        DefaultKeybindings(),
	}
}

// Load carga la configuración desde disco o crea una con valores por defecto.
func Load(customDir string) (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	notesDir := customDir
	if notesDir == "" {
		notesDir = filepath.Join(home, "Documents", "notes")
	}

	// Asegurar que el directorio de notas exista
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, err
	}

	// Asegurar subdirectorio de assets / imágenes
	assetsDir := filepath.Join(notesDir, "assets")
	_ = os.MkdirAll(assetsDir, 0755)

	cfg := DefaultConfig(notesDir)
	cfgPath := configFilePath()
	cfg.configPath = cfgPath

	// Intentar leer archivo de configuración existente
	if data, err := os.ReadFile(cfgPath); err == nil {
		var diskCfg Config
		if err := json.Unmarshal(data, &diskCfg); err == nil {
			if diskCfg.Editor != "" {
				cfg.Editor = diskCfg.Editor
			}
			if diskCfg.Theme != "" {
				cfg.Theme = diskCfg.Theme
			}
			if diskCfg.Language != "" {
				cfg.Language = diskCfg.Language
			}
			if diskCfg.KeybindingMode != "" {
				cfg.KeybindingMode = diskCfg.KeybindingMode
			}
			if diskCfg.Keybindings.NewNote != "" {
				cfg.Keybindings.NewNote = diskCfg.Keybindings.NewNote
			}
			if diskCfg.Keybindings.NewFolder != "" {
				cfg.Keybindings.NewFolder = diskCfg.Keybindings.NewFolder
			}
			if diskCfg.Keybindings.Edit != "" {
				cfg.Keybindings.Edit = diskCfg.Keybindings.Edit
			}
			if diskCfg.Keybindings.Delete != "" {
				cfg.Keybindings.Delete = diskCfg.Keybindings.Delete
			}
			if diskCfg.Keybindings.Move != "" {
				cfg.Keybindings.Move = diskCfg.Keybindings.Move
			}
			if diskCfg.Keybindings.PasteImage != "" {
				cfg.Keybindings.PasteImage = diskCfg.Keybindings.PasteImage
			}
			if diskCfg.Keybindings.TogglePanel != "" {
				cfg.Keybindings.TogglePanel = diskCfg.Keybindings.TogglePanel
			}
			if diskCfg.Keybindings.Settings != "" {
				cfg.Keybindings.Settings = diskCfg.Keybindings.Settings
			}
			if diskCfg.Keybindings.Cheatsheet != "" {
				cfg.Keybindings.Cheatsheet = diskCfg.Keybindings.Cheatsheet
			}
			if diskCfg.Keybindings.Quit != "" {
				cfg.Keybindings.Quit = diskCfg.Keybindings.Quit
			}
			cfg.MouseClick = diskCfg.MouseClick
			cfg.ShowTagsTab = diskCfg.ShowTagsTab
			cfg.ShowTasksTab = diskCfg.ShowTasksTab
			cfg.ShowGalleryTab = diskCfg.ShowGalleryTab
			if diskCfg.SidebarRatio >= 0.15 && diskCfg.SidebarRatio <= 0.75 {
				cfg.SidebarRatio = diskCfg.SidebarRatio
			}
			if customDir == "" && diskCfg.NotesDir != "" {
				cfg.NotesDir = diskCfg.NotesDir
			}
		}
	}

	// Validar que el editor configurado realmente exista; si no, hacer fallback a uno instalado
	installed := DetectInstalledEditors()
	editorValid := false
	for _, ed := range installed {
		if strings.Contains(strings.ToLower(cfg.Editor), ed) {
			editorValid = true
			break
		}
	}
	if !editorValid && len(installed) > 0 {
		cfg.Editor = installed[0]
		_ = cfg.Save()
	}

	return cfg, nil
}

// DetectInstalledEditors devuelve la lista de editores presentes en el sistema
func DetectInstalledEditors() []string {
	var list []string
	candidates := []string{"micro", "vim", "nano", "nvim"}
	for _, c := range candidates {
		bin := ResolveEditorBin(c)
		if _, err := os.Stat(bin); err == nil {
			list = append(list, c)
		} else if _, err := exec.LookPath(c); err == nil {
			list = append(list, c)
		}
	}
	if len(list) == 0 {
		list = append(list, "micro")
	}
	return list
}

// Save persiste la configuración actual en ~/.config/lazymark/config.json
func (c *Config) Save() error {
	if c.configPath == "" {
		c.configPath = configFilePath()
	}
	if err := os.MkdirAll(filepath.Dir(c.configPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configPath, data, 0644)
}

// ResolveEditorBin busca la ruta absoluta ejecutable para el editor
func ResolveEditorBin(name string) string {
	if name == "" {
		name = "micro"
	}

	// Si es una ruta absoluta o relativa existente
	if strings.Contains(name, string(filepath.Separator)) {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	// Buscar en PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	// Rutas conocidas en macOS / Linux
	knownPaths := []string{
		"/opt/homebrew/bin/" + name,
		"/usr/local/bin/" + name,
		"/usr/bin/" + name,
		"/bin/" + name,
	}
	for _, kp := range knownPaths {
		if _, err := os.Stat(kp); err == nil {
			return kp
		}
	}

	return name
}
