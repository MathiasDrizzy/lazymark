package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version la inyecta GoReleaser con -ldflags "-X …/config.Version=…".
var Version = "0.1.0"

const (
	// Valores de Config.KeybindingMode: "lazy" son solo las flechas y las teclas
	// propias; "dual" suma los atajos Vim (h j k l g G). Hasta v0.1.0 "lazy" se
	// llamaba "lazygit": se sigue leyendo.
	KeybindingModeLazy = "lazy"
	KeybindingModeDual = "dual"

	// Valores de Config.ScreenBackground: "theme" (por defecto) pinta toda la pantalla
	// con el color base del tema; "terminal" deja el fondo de la terminal (y su transparencia).
	ScreenBackgroundTheme    = "theme"
	ScreenBackgroundTerminal = "terminal"

	// Valores de Config.PopupBackground.
	PopupBackgroundNone  = "none"
	PopupBackgroundTheme = "theme"

	AppName = "lazymark"

	// KeymapVersion sube cuando cambian los atajos por defecto; un config.json
	// de una versión anterior se migra a los atajos nuevos.
	KeymapVersion = 2
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
		Settings:    ",",
		Cheatsheet:  "?",
		Quit:        "q",
	}
}

// Config almacena las preferencias de ejecución de la aplicación.
type Config struct {
	NotesDir           string            `json:"notes_dir"`
	Editor             string            `json:"editor"`
	MouseClick         bool              `json:"mouse_click"`
	Theme              string            `json:"theme"`
	Language           string            `json:"language"`
	ShowTagsTab        bool              `json:"show_tags_tab"`
	ShowTasksTab       bool              `json:"show_tasks_tab"`
	ConfirmDelete      bool              `json:"confirm_delete"`
	HideCompletedTasks bool              `json:"hide_completed_tasks"`
	SidebarRatio       float64           `json:"sidebar_ratio"`
	KeybindingMode     string            `json:"keybinding_mode"`
	Keybindings        KeybindingsConfig `json:"keybindings"`
	KeymapVersion      int               `json:"keymap_version"`
	TaskScope          string            `json:"task_scope"`
	// PopupBackground: "none" (por defecto) deja el fondo de la terminal en los
	// popups, respetando su transparencia; "theme" pinta el color base del tema.
	PopupBackground string `json:"popup_background"`
	// ScreenBackground: "theme" (por defecto) o "terminal". Ver ScreenBackgroundTheme.
	ScreenBackground string `json:"screen_background"`
	configPath       string `json:"-"`

	// notesDirFromFlag indica que NotesDir viene de --dir y vale solo para esta
	// ejecución: Save conserva en el archivo savedNotesDir (la carpeta guardada
	// antes, vacía si no había) hasta que SetNotesDir la elija de forma explícita.
	notesDirFromFlag bool   `json:"-"`
	savedNotesDir    string `json:"-"`
}

// configFilePath es la ruta de config.json según el sistema (os.UserConfigDir).
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
		ConfirmDelete:      true,
		HideCompletedTasks: false,
		SidebarRatio:       0.33,
		KeybindingMode:     KeybindingModeDual,
		Keybindings:        DefaultKeybindings(),
		KeymapVersion:      KeymapVersion,
		TaskScope:          "all",
		PopupBackground:    PopupBackgroundNone,
		ScreenBackground:   ScreenBackgroundTheme,
	}
}

// DefaultNotesDir devuelve la ruta por defecto del directorio de notas ($HOME/Documents/notes)
func DefaultNotesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Documents", "notes")
}

// Load carga la configuración desde disco o crea una con valores por defecto.
func Load(customDir string) (*Config, error) {
	notesDir := customDir
	if notesDir == "" {
		notesDir = DefaultNotesDir()
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
	cfg.notesDirFromFlag = customDir != "" // sin archivo previo, --dir tampoco se guarda

	// Leer el archivo encima de los defaults: un campo ausente conserva su
	// valor por defecto en vez de quedar en cero.
	if data, err := os.ReadFile(cfgPath); err == nil {
		disk := *cfg
		disk.KeymapVersion = 0
		disk.NotesDir = "" // para distinguir "no está en el archivo" del valor por defecto
		if err := json.Unmarshal(data, &disk); err == nil {
			if disk.KeymapVersion < KeymapVersion {
				// Atajos de una versión anterior: se reemplazan por los nuevos.
				disk.Keybindings = DefaultKeybindings()
			} else {
				disk.Keybindings = mergeKeybindings(DefaultKeybindings(), disk.Keybindings)
			}
			disk.KeymapVersion = KeymapVersion
			if disk.SidebarRatio < 0.15 || disk.SidebarRatio > 0.75 {
				disk.SidebarRatio = cfg.SidebarRatio
			}
			if disk.TaskScope == "" {
				disk.TaskScope = "all"
			}
			disk.KeybindingMode = normalizeKeybindingMode(disk.KeybindingMode)
			if disk.ScreenBackground != ScreenBackgroundTerminal {
				disk.ScreenBackground = ScreenBackgroundTheme // valor ausente o desconocido
			}
			if disk.PopupBackground != PopupBackgroundTheme {
				disk.PopupBackground = PopupBackgroundNone // valor ausente o desconocido
			}
			disk.savedNotesDir = disk.NotesDir
			disk.notesDirFromFlag = customDir != ""
			if customDir != "" || disk.NotesDir == "" {
				disk.NotesDir = notesDir
			}
			disk.configPath = cfgPath
			*cfg = disk
		}
	}

	// El editor configurado se respeta tal cual (H1-5). Solo se reemplaza si su
	// ejecutable no existe en ningún lado.
	if !EditorExists(cfg.Editor) {
		if installed := DetectInstalledEditors(); len(installed) > 0 {
			cfg.Editor = installed[0]
		}
	}

	return cfg, nil
}

// normalizeKeybindingMode migra el valor antiguo "lazygit" a "lazy"; todo lo que no
// sea "lazy" queda en "dual".
func normalizeKeybindingMode(mode string) string {
	switch strings.ToLower(mode) {
	case "lazy", "lazygit":
		return KeybindingModeLazy
	}
	return KeybindingModeDual
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

// SplitEditor separa el comando del editor en ejecutable y argumentos. Acepta
// una ruta con espacios sin comillas ("C:\Program Files\Editor\ed.exe"), una
// ruta entre comillas seguida de argumentos, y "nombre arg1 arg2".
func SplitEditor(editor string) (bin string, args []string) {
	editor = strings.TrimSpace(editor)
	if editor == "" {
		return "", nil
	}
	if isFile(editor) {
		return editor, nil
	}
	if q := editor[0]; q == '"' || q == '\'' {
		if end := strings.IndexByte(editor[1:], q); end >= 0 {
			return editor[1 : 1+end], strings.Fields(editor[2+end:])
		}
	}
	fields := strings.Fields(editor)
	for k := len(fields); k > 1; k-- {
		if cand := strings.Join(fields[:k], " "); isFile(cand) {
			return cand, fields[k:]
		}
	}
	return fields[0], fields[1:]
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// EditorExists indica si el ejecutable del editor (sin sus argumentos) existe
// como ruta o en el PATH.
func EditorExists(editor string) bool {
	bin, _ := SplitEditor(editor)
	if bin == "" {
		return false
	}
	resolved := ResolveEditorBin(bin)
	if _, err := os.Stat(resolved); err == nil {
		return true
	}
	_, err := exec.LookPath(resolved)
	return err == nil
}

// SetNotesDir elige la carpeta de notas de forma explícita (Ajustes): desde aquí
// se guarda como la carpeta por defecto, incluso si la ejecución empezó con --dir.
func (c *Config) SetNotesDir(dir string) {
	c.NotesDir = dir
	c.notesDirFromFlag = false
}

// Path devuelve la ruta del archivo de configuración.
func (c *Config) Path() string {
	if c.configPath == "" {
		c.configPath = configFilePath()
	}
	return c.configPath
}

// Save persiste la configuración actual en el archivo de configuración del
// usuario (Config.Path): ~/Library/Application Support/lazymark/config.json en
// macOS, $XDG_CONFIG_HOME o ~/.config/lazymark/config.json en Linux y
// %AppData%\lazymark\config.json en Windows.
func (c *Config) Save() error {
	if c.configPath == "" {
		c.configPath = configFilePath()
	}
	if err := os.MkdirAll(filepath.Dir(c.configPath), 0755); err != nil {
		return err
	}
	out := *c
	if c.notesDirFromFlag {
		out.NotesDir = c.savedNotesDir // --dir vale solo para esta ejecución
	}
	data, err := json.MarshalIndent(&out, "", "  ")
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

// mergeKeybindings completa con los defaults los atajos que vienen vacíos.
func mergeKeybindings(def, user KeybindingsConfig) KeybindingsConfig {
	pick := func(u, d string) string {
		if u != "" {
			return u
		}
		return d
	}
	return KeybindingsConfig{
		NewNote:     pick(user.NewNote, def.NewNote),
		NewFolder:   pick(user.NewFolder, def.NewFolder),
		Edit:        pick(user.Edit, def.Edit),
		Delete:      pick(user.Delete, def.Delete),
		Move:        pick(user.Move, def.Move),
		PasteImage:  pick(user.PasteImage, def.PasteImage),
		TogglePanel: pick(user.TogglePanel, def.TogglePanel),
		Settings:    pick(user.Settings, def.Settings),
		Cheatsheet:  pick(user.Cheatsheet, def.Cheatsheet),
		Quit:        pick(user.Quit, def.Quit),
	}
}
