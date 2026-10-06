package config

import (
	"encoding/json"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/safeio"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// Version es la versión del programa. El release la inyecta con -ldflags
// "-X …/config.Version=…"; si no (go install), sale del módulo que registra Go; y si nada de eso
// existe (compilado desde el árbol), vale "dev".
var Version = resolveVersion(injectedVersion, buildInfo())

// injectedVersion es lo que GoReleaser pone con -ldflags "-X …/config.injectedVersion=…".
var injectedVersion = ""

func buildInfo() *debug.BuildInfo {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return bi
	}
	return nil
}

// resolveVersion elige la versión: ldflags, luego la versión del módulo principal (sin la "v"),
// y "dev" si no hay ninguna ("(devel)" es lo que registra Go al compilar desde el árbol).
func resolveVersion(ldflags string, info *debug.BuildInfo) string {
	if ldflags != "" {
		return ldflags
	}
	if info != nil {
		if v := strings.TrimPrefix(info.Main.Version, "v"); v != "" && info.Main.Version != "(devel)" {
			return v
		}
	}
	return "dev"
}

const (
	// Valores de Config.KeybindingMode: "lazy" son solo las flechas y las teclas
	// propias; "dual" suma los atajos Vim (h j k l g G). Hasta v0.1.0 "lazy" se
	// llamaba "lazygit": se sigue leyendo.
	KeybindingModeLazy = "lazy"
	KeybindingModeDual = "dual"

	// Valores de Config.ScreenBackground: "theme" (por defecto) pinta toda la pantalla
	// con el color base del tema; "terminal" deja el fondo de la terminal (y su transparencia).
	// Valores de Config.KanbanCards: "cards" (por defecto) dibuja cada tarea como una tarjeta con borde, su nota y sus
	// fechas; "compact" es la vista de una fila por tarea.
	KanbanCardsRects   = "cards"
	KanbanCardsCompact = "compact"

	ScreenBackgroundTheme    = "theme"
	ScreenBackgroundTerminal = "terminal"

	// Valores de Config.PopupBackground: los mismos que los de ScreenBackground. "terminal" (por defecto) deja el fondo
	// de la terminal en los popups, aunque la pantalla esté pintada con el del tema; "theme" los pinta con él.
	PopupBackgroundTheme    = "theme"
	PopupBackgroundTerminal = "terminal"

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
	// PopupBackground: "theme" (por defecto, como la pantalla) pinta el color base del tema; "terminal" deja el
	// fondo de la terminal en los popups, respetando su transparencia, aunque la pantalla esté pintada con el tema.
	PopupBackground string `json:"popup_background"`
	// NerdFont: si las fechas de las tareas se dibujan con glifos de Nerd Font (por defecto) o con símbolos de texto.
	NerdFont bool `json:"nerd_font"`
	// KanbanCards: "cards" o "compact". Ver KanbanCardsRects.
	KanbanCards string `json:"kanban_cards"`
	// KanbanColumns son las columnas del tablero (por defecto todo, doing y done). La columna de una tarea
	// se guarda como un tag al final de su línea: `- [ ] tarea #kb/doing`.
	KanbanColumns []KanbanColumn `json:"kanban_columns"`
	// Mascot: si el perezoso dormido aparece en los estados de reposo (carpeta o nota vacía). Por defecto sí.
	Mascot bool `json:"mascot"`
	// ClickHint: si, tras un rato sin tocar nada, aparece muy tenue el texto "click me!" sobre la mascota. Por defecto sí; no hace falta apagar la mascota para quitarlo.
	ClickHint bool `json:"click_hint"`
	// MaxNoteMB es el tamaño máximo, en MB, de una nota que lazymark lee (listado, vista previa, CLI y MCP). Una más grande se muestra como "demasiado grande" y no se carga. Por defecto 10.
	MaxNoteMB int `json:"max_note_mb"`
	// DateFormat es el formato en que se escriben las fechas de las tareas en el archivo: "dataview" (`[due:: 2026-05-10]`), "emoji" (el formato por
	// defecto de Obsidian Tasks) o "" (por defecto: dataview, salvo en un vault que ya solo tiene fechas con emojis). Se leen siempre los dos.
	DateFormat string `json:"date_format"`
	// DateFormatNoticeShown: ya se avisó (una sola vez) que el vault tiene fechas con emojis y se escriben así.
	DateFormatNoticeShown bool `json:"date_format_notice_shown"`
	// DateColors: si las fechas de las tareas se dibujan con un color según su estado (vencida, por vencer, en fecha, inicio, completada). Por defecto sí; apagado,
	// todas van en un gris claro neutro.
	DateColors bool `json:"date_colors"`
	// DueSoonDays: cuántos días antes del vencimiento (0 a 30) una fecha cuenta como "por vencer". 0 (por defecto): solo el mismo día.
	DueSoonDays int `json:"due_soon_days"`
	// DateColorNames: el color de la paleta del tema que usa cada estado: overdue, soon, ontime, started, notstarted y done, con un valor de DateColorChoices.
	DateColorNames map[string]string `json:"date_color_names"`
	// DateWarnings: el aviso de que el inicio de una tarea es posterior a su vencimiento (popup de Fechas y CLI). Por defecto sí.
	DateWarnings bool `json:"date_warnings"`
	// DateFormatNotice: el aviso único que explica que las fechas se escriben con emojis porque el vault ya las tiene así. Por defecto sí; con no, nunca aparece.
	DateFormatNotice bool `json:"date_format_notice"`
	// ClickHintIdleSeconds, ClickHintShowSeconds y ClickHintEverySeconds: la cadencia del "click me!" sobre la mascota: segundos de quietud antes de que aparezca (20),
	// cuánto se ve (15) y cada cuánto vuelve mientras sigas quieto (60). Cada uno de 5 a 600; el segundo va siempre por debajo del tercero.
	ClickHintIdleSeconds  int `json:"click_hint_idle_seconds"`
	ClickHintShowSeconds  int `json:"click_hint_show_seconds"`
	ClickHintEverySeconds int `json:"click_hint_every_seconds"`
	// TrashDays: cuántos días se guarda lo que se borra en la papelera (0 a 365; 20 por defecto). 0 = sin papelera: borrar elimina de inmediato y pide siempre confirmación.
	TrashDays int `json:"trash_days"`
	// DailyFolder, DailyName y TemplatesFolder: dónde viven las notas diarias (`journal`), cómo se llaman (`YYYY-MM-DD`: lleva YYYY o AAAA, MM y DD) y dónde están las
	// plantillas (`templates`). Rutas relativas a la carpeta de notas, sin ".." ni rutas absolutas.
	DailyFolder     string `json:"daily_folder"`
	DailyName       string `json:"daily_name"`
	TemplatesFolder string `json:"templates_folder"`
	// KanbanTag: el prefijo de la etiqueta de columna del tablero (`kb`: `#kb/doing`). Cambiarlo no migra las notas que ya tienen otro prefijo: `lazymark kanban retag`.
	KanbanTag string `json:"kanban_tag"`
	// NotesSort ("name" por defecto, como siempre, o "modified": las notas más recientes primero, las carpetas siguen por nombre) y TasksSort ("note" por defecto: en el orden de
	// las notas, o "due": primero las pendientes con vencimiento, de la más vencida a la más lejana; después las sin fecha y al final las hechas) ordenan los paneles Notas y Tareas.
	NotesSort string `json:"notes_sort"`
	TasksSort string `json:"tasks_sort"`
	// DateGlyphs reemplaza el glifo de cada campo de fecha (claves start, due, done, scheduled, created): un solo carácter de ancho 1 o 2. Lo que falta o no vale usa el de Nerd Font o de texto.
	DateGlyphs map[string]string `json:"date_glyphs"`
	// ScreenBackground: "theme" (por defecto) o "terminal". Ver ScreenBackgroundTheme.
	ScreenBackground string `json:"screen_background"`
	configPath       string `json:"-"`
	// warnings son los avisos de la lectura de config.json (un campo con el tipo equivocado, una clave desconocida, un archivo que no se pudo leer); unknown guarda las claves que
	// lazymark no conoce para volver a escribirlas tal cual; unreadable dice que el archivo no se pudo leer entero: antes de guardar encima se hace una copia (.bak).
	warnings   []string
	unknownKey []string // el aviso de cada clave desconocida (la CLI no lo repite en cada comando; la TUI sí lo muestra)
	unknown    map[string]json.RawMessage
	unreadable bool

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
		NotesDir:              notesDir,
		Editor:                editor,
		MouseClick:            true,
		Theme:                 "catppuccin-mocha",
		Language:              "auto",
		ShowTagsTab:           true,
		ShowTasksTab:          true,
		ConfirmDelete:         true,
		HideCompletedTasks:    false,
		SidebarRatio:          0.33,
		KeybindingMode:        KeybindingModeDual,
		Keybindings:           DefaultKeybindings(),
		KeymapVersion:         KeymapVersion,
		TaskScope:             "all",
		PopupBackground:       PopupBackgroundTheme,
		KanbanCards:           KanbanCardsRects,
		ScreenBackground:      ScreenBackgroundTheme,
		Mascot:                true,
		ClickHint:             true,
		MaxNoteMB:             10,
		DateColors:            true,
		DateWarnings:          true,
		DateFormatNotice:      true,
		ClickHintIdleSeconds:  20,
		ClickHintShowSeconds:  15,
		ClickHintEverySeconds: 60,
		TrashDays:             20,
		DailyFolder:           "journal",
		DailyName:             "YYYY-MM-DD",
		TemplatesFolder:       "templates",
		KanbanTag:             "kb",
		NotesSort:             "name",
		TasksSort:             "note",
		DateColorNames:        DefaultDateColorNames(),
		NerdFont:              true,
		KanbanColumns:         DefaultKanbanColumns(),
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
	return load(customDir, true)
}

// LoadReadOnly es Load sin efectos: no crea la carpeta de notas ni su assets/. La usan los comandos de la
// línea de comandos, que no deben tocar nada si los argumentos son inválidos.
func LoadReadOnly(customDir string) (*Config, error) {
	return load(customDir, false)
}

func load(customDir string, create bool) (*Config, error) {
	notesDir := customDir
	if notesDir == "" {
		notesDir = DefaultNotesDir()
	}

	if create {
		// Asegurar que el directorio de notas exista
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			return nil, err
		}

		// Asegurar subdirectorio de assets / imágenes
		assetsDir := filepath.Join(notesDir, "assets")
		_ = os.MkdirAll(assetsDir, 0755)
	}

	cfg := DefaultConfig(notesDir)
	cfgPath := configFilePath()
	cfg.configPath = cfgPath
	cfg.notesDirFromFlag = customDir != "" // sin archivo previo, --dir tampoco se guarda

	// Leer el archivo encima de los defaults: un campo ausente conserva su valor por defecto en vez de quedar en cero. La lectura es TOLERANTE: cada clave se aplica por
	// separado; una con el tipo equivocado vuelve a su defecto con un aviso y el resto del archivo se respeta; una clave desconocida se conserva (y se avisa); y un archivo
	// que no es JSON no se pierde: se usan los defectos y, antes de guardar encima, se hace una copia.
	data, readErr := os.ReadFile(cfgPath)
	if readErr != nil && !os.IsNotExist(readErr) { // existe pero no se puede leer (permisos, no es un archivo…): tampoco se pisa sin guardar antes lo que haya
		cfg.unreadable = true
		cfg.warnings = append(cfg.warnings, i18n.Errorf("config.json no se pudo leer (%v): se usan los valores por defecto; al guardar, el original queda en config.json.bak", "config.json could not be read (%v): the defaults are used; when saving, the original is kept as config.json.bak", readErr).Error())
	}
	if readErr == nil {
		disk := *cfg
		disk.KeymapVersion = 0
		disk.KanbanColumns = nil // Unmarshal mezclaría los elementos con los de por defecto (sus títulos)
		disk.NotesDir = ""       // para distinguir "no está en el archivo" del valor por defecto
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			cfg.unreadable = true
			cfg.warnings = append(cfg.warnings, i18n.Errorf("config.json no se pudo leer (%v): se usan los valores por defecto; al guardar, el original queda en config.json.bak", "config.json could not be read (%v): the defaults are used; when saving, the original is kept as config.json.bak", err).Error())
		} else {
			disk.applyRaw(raw)
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
			if !validKanbanColumns(disk.KanbanColumns) {
				disk.KanbanColumns = DefaultKanbanColumns() // ausente o inválida
			}
			if disk.ScreenBackground != ScreenBackgroundTerminal {
				disk.ScreenBackground = ScreenBackgroundTheme // valor ausente o desconocido
			}
			disk.Language = normalizeLanguage(disk.Language)
			if disk.DateFormat != "dataview" && disk.DateFormat != "emoji" {
				disk.DateFormat = "" // ausente o desconocido: el de por defecto
			}
			disk.DueSoonDays = min(max(disk.DueSoonDays, 0), 30)
			disk.normalizeOptions()
			disk.DateColorNames = normalizeDateColorNames(disk.DateColorNames)
			if disk.KanbanCards != KanbanCardsCompact {
				disk.KanbanCards = KanbanCardsRects // ausente o desconocido
			}
			switch disk.PopupBackground {
			case PopupBackgroundTheme, PopupBackgroundTerminal:
			case "none": // el valor de antes (sin fondo): es el fondo de la terminal
				disk.PopupBackground = PopupBackgroundTerminal
			default: // ausente o desconocido: el de por defecto, como la pantalla
				disk.PopupBackground = PopupBackgroundTheme
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
// MaxNoteBytes es MaxNoteMB en bytes (10 MB si no es un valor válido).
func (c *Config) MaxNoteBytes() int64 {
	if c.MaxNoteMB <= 0 || c.MaxNoteMB > 4096 {
		return 10 << 20
	}
	return int64(c.MaxNoteMB) << 20
}

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
	if len(c.unknown) > 0 { // las claves que lazymark no conoce se vuelven a escribir tal cual
		var known map[string]json.RawMessage
		if json.Unmarshal(data, &known) == nil {
			for k, v := range c.unknown {
				if _, ok := known[k]; !ok {
					known[k] = v
				}
			}
			if merged, err := json.MarshalIndent(known, "", "  "); err == nil {
				data = merged
			}
		}
	}
	if c.unreadable { // el archivo no se pudo leer entero: lo que había queda guardado antes de escribir encima
		dest, err := backupUnreadable(c.configPath)
		if err != nil {
			return err // sin copia no se pisa el original
		}
		if dest != "" {
			c.warnings = append(c.warnings, i18n.Errorf("el config.json que no se pudo leer quedó guardado en %s", "the config.json that could not be read was kept as %s", dest).Error())
		}
		c.unreadable = false
	}
	return safeio.WriteFileAtomic(c.configPath, data, 0o644)
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

// normalizeLanguage lleva el idioma de la config a "auto" o al código de uno soportado (en, es, pt, fr, de, it, ja, zh):
// "pt-BR" y "zh_CN" valen como pt y zh; un valor vacío o desconocido, como auto.
func normalizeLanguage(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), "auto") {
		return "auto"
	}
	if l, ok := i18n.Parse(v); ok {
		return string(l)
	}
	return "auto"
}

// DateColorChoices son los colores de la paleta del tema que se pueden elegir para el estado de una fecha (los nombres que entiende views.PaletteColor).
var DateColorChoices = []string{"error", "warning", "orange", "success", "accent", "info", "special", "text", "muted"}

// DefaultDateColorNames son los colores por defecto de cada estado de fecha (la tabla razonada está en ui/views/datecolors.go).
func DefaultDateColorNames() map[string]string {
	return map[string]string{"overdue": "error", "soon": "warning", "ontime": "accent", "started": "info", "notstarted": "muted", "done": "success"}
}

// normalizeDateColorNames completa con los de por defecto los estados que faltan y reemplaza los nombres que no son un color de la paleta.
func normalizeDateColorNames(in map[string]string) map[string]string {
	out := DefaultDateColorNames()
	for k := range out {
		for _, c := range DateColorChoices {
			if in[k] == c {
				out[k] = c
			}
		}
	}
	return out
}

// Los rangos y los valores por defecto de las opciones de ORD-025.
const (
	minHintSeconds, maxHintSeconds = 5, 600
	maxTrashDays                   = 365
)

var KanbanTagRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,19}$`)

// normalizeOptions lleva cada opción a un valor válido: uno inválido o fuera de rango vuelve a su valor por defecto (nunca rompe la app).
func (c *Config) normalizeOptions() {
	d := DefaultConfig("")
	c.ClickHintIdleSeconds = hintSecondsOr(c.ClickHintIdleSeconds, d.ClickHintIdleSeconds)
	c.ClickHintShowSeconds = hintSecondsOr(c.ClickHintShowSeconds, d.ClickHintShowSeconds)
	c.ClickHintEverySeconds = hintSecondsOr(c.ClickHintEverySeconds, d.ClickHintEverySeconds)
	if c.ClickHintShowSeconds >= c.ClickHintEverySeconds { // se tiene que ir antes de volver a salir
		c.ClickHintShowSeconds, c.ClickHintEverySeconds = d.ClickHintShowSeconds, d.ClickHintEverySeconds
		if c.ClickHintShowSeconds >= c.ClickHintEverySeconds {
			c.ClickHintEverySeconds = c.ClickHintShowSeconds + 1
		}
	}
	if c.TrashDays < 0 || c.TrashDays > maxTrashDays {
		c.TrashDays = d.TrashDays
	}
	c.DailyFolder = folderOr(c.DailyFolder, d.DailyFolder)
	c.TemplatesFolder = folderOr(c.TemplatesFolder, d.TemplatesFolder)
	if !ValidDailyName(c.DailyName) {
		c.DailyName = d.DailyName
	}
	if !KanbanTagRe.MatchString(c.KanbanTag) {
		c.KanbanTag = d.KanbanTag
	}
	if c.NotesSort != "modified" {
		c.NotesSort = d.NotesSort
	}
	if c.TasksSort != "due" {
		c.TasksSort = d.TasksSort
	}
	glyphs := map[string]string{}
	for _, k := range []string{"start", "due", "done", "scheduled", "created"} {
		if g := c.DateGlyphs[k]; ValidDateGlyph(g) {
			glyphs[k] = g
		}
	}
	c.DateGlyphs = glyphs
}

func hintSecondsOr(v, def int) int {
	if v < minHintSeconds || v > maxHintSeconds {
		return def
	}
	return v
}

// folderOr devuelve la carpeta normalizada (con "/") si es válida, o def.
func folderOr(v, def string) string {
	if f, ok := CleanRelFolder(v); ok {
		return f
	}
	return def
}

// CleanRelFolder valida una carpeta relativa a la carpeta de notas: sin ruta absoluta, sin "..", sin segmentos vacíos ni ocultos (que empiezan con ".") y sin caracteres de
// control ni los que no valen en un nombre de archivo. Devuelve la ruta con "/".
func CleanRelFolder(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 100 || strings.HasPrefix(v, "/") || strings.HasPrefix(v, `\`) || filepath.IsAbs(v) || regexp.MustCompile(`^[A-Za-z]:`).MatchString(v) {
		return "", false
	}
	parts := strings.Split(strings.ReplaceAll(v, `\`, "/"), "/")
	for i, p := range parts {
		if p == "" && i == len(parts)-1 { // una barra final se tolera
			parts = parts[:i]
			break
		}
		if p == "" || p == "." || p == ".." || strings.HasPrefix(p, ".") || strings.EqualFold(p, "assets") || strings.ContainsAny(p, "<>:\"|?*") || strings.IndexFunc(p, unicode.IsControl) >= 0 || safeio.ReservedWindowsName(p) { // CON, NUL, COM1… son dispositivos en Windows: se rechazan en todos los sistemas (el vault viaja)
			return "", false
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "/"), true
}

// ValidDailyName dice si v sirve de formato del nombre de la nota diaria: lleva el año (YYYY o AAAA), el mes (MM) y el día (DD), un nombre de archivo sin separadores,
// sin ".." ni caracteres de control, y no más de 60 caracteres.
func ValidDailyName(v string) bool {
	if v == "" || len(v) > 60 || strings.ContainsAny(v, "/\\:*?\"<>|") || strings.Contains(v, "..") || strings.HasPrefix(v, ".") || strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return false
	}
	return (strings.Contains(v, "YYYY") || strings.Contains(v, "AAAA")) && strings.Contains(v, "MM") && strings.Contains(v, "DD")
}

// FormatDailyName es el nombre (sin .md) de la nota diaria de t con el formato v (YYYY o AAAA, MM, DD).
func FormatDailyName(v string, t time.Time) string {
	r := strings.NewReplacer("YYYY", t.Format("2006"), "AAAA", t.Format("2006"), "MM", t.Format("01"), "DD", t.Format("02"))
	return r.Replace(v)
}

// ValidDateGlyph dice si g sirve de glifo de fecha: un solo carácter (sin control ni espacio) de ancho 1 o 2 celdas. El vacío (sin glifo propio) es válido solo para quien
// llama (se trata como "no está").
func ValidDateGlyph(g string) bool {
	rs := []rune(g)
	if len(rs) != 1 || unicode.IsControl(rs[0]) || unicode.IsSpace(rs[0]) || !utf8.ValidString(g) {
		return false
	}
	w := textwidth.Width(g)
	return w == 1 || w == 2
}

// DateGlyphArray son los glifos propios de las fechas (date_glyphs) en el orden de los campos: inicio, vencimiento, completada, programada y creada; un vacío es "sin glifo propio".
func (c *Config) DateGlyphArray() [5]string {
	var out [5]string
	for f, k := range []string{"start", "due", "done", "scheduled", "created"} {
		out[f] = c.DateGlyphs[k]
	}
	return out
}

var (
	knownKeysOnce sync.Once
	knownKeys     map[string]bool
)

// jsonKeys son las claves de config.json que lazymark conoce (las etiquetas json de Config).
func jsonKeys() map[string]bool {
	knownKeysOnce.Do(func() {
		knownKeys = map[string]bool{}
		t := reflect.TypeOf(Config{})
		for i := 0; i < t.NumField(); i++ {
			if tag := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]; tag != "" && tag != "-" {
				knownKeys[tag] = true
			}
		}
	})
	return knownKeys
}

// applyRaw aplica las claves de config.json a c una por una: una con el tipo equivocado se salta con un aviso (queda el defecto) y el resto se respeta; una clave que lazymark no
// conoce se guarda (c.unknown) para volver a escribirla tal cual y se avisa de ella (un error de tecleo, por ejemplo).
func (c *Config) applyRaw(raw map[string]json.RawMessage) {
	known := jsonKeys()
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !known[k] {
			if c.unknown == nil {
				c.unknown = map[string]json.RawMessage{}
			}
			c.unknown[k] = raw[k]
			c.unknownKey = append(c.unknownKey, i18n.Errorf("config.json: la clave %q no existe en lazymark (se conserva tal cual)", "config.json: the key %q does not exist in lazymark (it is kept as it is)", k).Error())
			continue
		}
		one, _ := json.Marshal(map[string]json.RawMessage{k: raw[k]})
		tmp := *c
		if err := json.Unmarshal(one, &tmp); err != nil {
			c.warnings = append(c.warnings, i18n.Errorf("config.json: el valor de %q no es válido (%v): se usa el valor por defecto", "config.json: the value of %q is not valid (%v): the default is used", k, err).Error())
			continue
		}
		*c = tmp
	}
}

// Warnings son los avisos de la lectura de la configuración (campos inválidos, claves desconocidas, archivo ilegible), listos para mostrar; vacío si todo estaba bien.
func (c *Config) Warnings() []string {
	return append(append([]string(nil), c.warnings...), c.unknownKey...)
}

// Problems son los avisos que importan en cada comando (un valor inválido que se reemplazó, un archivo ilegible): sin los de claves desconocidas, que solo informan.
func (c *Config) Problems() []string { return append([]string(nil), c.warnings...) }

// backupUnreadable copia el config.json que no se pudo leer a config.json.bak (o .bak.1, .bak.2… si ya hay uno) antes de que Save lo reemplace; devuelve la ruta de la copia.
func backupUnreadable(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil // ya no está: no hay nada que guardar
	}
	if err != nil {
		return "", err // existe y no se puede leer: sin copia no se pisa
	}
	dest := path + ".bak"
	for i := 1; ; i++ {
		if _, err := os.Lstat(dest); os.IsNotExist(err) || i > 9 {
			break
		}
		dest = fmt.Sprintf("%s.bak.%d", path, i)
	}
	return dest, safeio.WriteFileAtomic(dest, data, 0o600)
}
