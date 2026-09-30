package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette define los colores de un tema
type Palette struct {
	Name     string
	Base     lipgloss.Color
	Mantle   lipgloss.Color
	Surface0 lipgloss.Color
	Surface1 lipgloss.Color
	Overlay0 lipgloss.Color
	Text     lipgloss.Color
	Subtext0 lipgloss.Color
	Peach    lipgloss.Color
	Mauve    lipgloss.Color
	Teal     lipgloss.Color
	Green    lipgloss.Color
	Red      lipgloss.Color
	Blue     lipgloss.Color
	Yellow   lipgloss.Color
}

// ─── Paletas disponibles ───────────────────────────────────────────

// CatppuccinMocha es el tema oscuro por defecto (dark, warm)
var CatppuccinMocha = Palette{
	Name:     "catppuccin-mocha",
	Base:     lipgloss.Color("#1e1e2e"),
	Mantle:   lipgloss.Color("#181825"),
	Surface0: lipgloss.Color("#313244"),
	Surface1: lipgloss.Color("#45475a"),
	Overlay0: lipgloss.Color("#6c7086"),
	Text:     lipgloss.Color("#cdd6f4"),
	Subtext0: lipgloss.Color("#a6adc8"),
	Peach:    lipgloss.Color("#fab387"),
	Mauve:    lipgloss.Color("#cba6f7"),
	Teal:     lipgloss.Color("#94e2d5"),
	Green:    lipgloss.Color("#a6e3a1"),
	Red:      lipgloss.Color("#f38ba8"),
	Blue:     lipgloss.Color("#89b4fa"),
	Yellow:   lipgloss.Color("#f9e2af"),
}

// CatppuccinLatte es el tema claro (light, soft)
var CatppuccinLatte = Palette{
	Name:     "catppuccin-latte",
	Base:     lipgloss.Color("#eff1f5"),
	Mantle:   lipgloss.Color("#e6e9ef"),
	Surface0: lipgloss.Color("#ccd0da"),
	Surface1: lipgloss.Color("#bcc0cc"),
	Overlay0: lipgloss.Color("#9ca0b0"),
	Text:     lipgloss.Color("#4c4f69"),
	Subtext0: lipgloss.Color("#6c6f85"),
	Peach:    lipgloss.Color("#fe640b"),
	Mauve:    lipgloss.Color("#8839ef"),
	Teal:     lipgloss.Color("#179299"),
	Green:    lipgloss.Color("#40a02b"),
	Red:      lipgloss.Color("#d20f39"),
	Blue:     lipgloss.Color("#1e66f5"),
	Yellow:   lipgloss.Color("#df8e1d"),
}

// CatppuccinFrappe es el tema intermedio oscuro (dark, cool)
var CatppuccinFrappe = Palette{
	Name:     "catppuccin-frappe",
	Base:     lipgloss.Color("#303446"),
	Mantle:   lipgloss.Color("#292c3c"),
	Surface0: lipgloss.Color("#414559"),
	Surface1: lipgloss.Color("#51576d"),
	Overlay0: lipgloss.Color("#737994"),
	Text:     lipgloss.Color("#c6d0f5"),
	Subtext0: lipgloss.Color("#a5adce"),
	Peach:    lipgloss.Color("#ef9f76"),
	Mauve:    lipgloss.Color("#ca9ee6"),
	Teal:     lipgloss.Color("#81c8be"),
	Green:    lipgloss.Color("#a6d189"),
	Red:      lipgloss.Color("#e78284"),
	Blue:     lipgloss.Color("#8caaee"),
	Yellow:   lipgloss.Color("#e5c890"),
}

// CatppuccinMacchiato es el tema intermedio cálido (dark, warm)
var CatppuccinMacchiato = Palette{
	Name:     "catppuccin-macchiato",
	Base:     lipgloss.Color("#24273a"),
	Mantle:   lipgloss.Color("#1e2030"),
	Surface0: lipgloss.Color("#363a4f"),
	Surface1: lipgloss.Color("#494d64"),
	Overlay0: lipgloss.Color("#6e738d"),
	Text:     lipgloss.Color("#cad3f5"),
	Subtext0: lipgloss.Color("#a5adcb"),
	Peach:    lipgloss.Color("#f5a97f"),
	Mauve:    lipgloss.Color("#c6a0f6"),
	Teal:     lipgloss.Color("#8bd5ca"),
	Green:    lipgloss.Color("#a6da95"),
	Red:      lipgloss.Color("#ed8796"),
	Blue:     lipgloss.Color("#8aadf4"),
	Yellow:   lipgloss.Color("#eed49f"),
}

// TokyoNight es un tema popular para terminales (dark, blue)
var TokyoNight = Palette{
	Name:     "tokyo-night",
	Base:     lipgloss.Color("#1a1b26"),
	Mantle:   lipgloss.Color("#16161e"),
	Surface0: lipgloss.Color("#292e42"),
	Surface1: lipgloss.Color("#3b4261"),
	Overlay0: lipgloss.Color("#545c7e"),
	Text:     lipgloss.Color("#c0caf5"),
	Subtext0: lipgloss.Color("#a9b1d6"),
	Peach:    lipgloss.Color("#ff9e64"),
	Mauve:    lipgloss.Color("#bb9af7"),
	Teal:     lipgloss.Color("#73daca"),
	Green:    lipgloss.Color("#9ece6a"),
	Red:      lipgloss.Color("#f7768e"),
	Blue:     lipgloss.Color("#7aa2f7"),
	Yellow:   lipgloss.Color("#e0af68"),
}

// GruvboxDark es un tema retro cálido (dark, retro)
var GruvboxDark = Palette{
	Name:     "gruvbox-dark",
	Base:     lipgloss.Color("#282828"),
	Mantle:   lipgloss.Color("#1d2021"),
	Surface0: lipgloss.Color("#3c3836"),
	Surface1: lipgloss.Color("#504945"),
	Overlay0: lipgloss.Color("#928374"),
	Text:     lipgloss.Color("#ebdbb2"),
	Subtext0: lipgloss.Color("#d5c4a1"),
	Peach:    lipgloss.Color("#fe8019"),
	Mauve:    lipgloss.Color("#d3869b"),
	Teal:     lipgloss.Color("#8ec07c"),
	Green:    lipgloss.Color("#b8bb26"),
	Red:      lipgloss.Color("#fb4934"),
	Blue:     lipgloss.Color("#83a598"),
	Yellow:   lipgloss.Color("#fabd2f"),
}

// Nord es un tema frío nórdico (dark, cold)
var Nord = Palette{
	Name:     "nord",
	Base:     lipgloss.Color("#2e3440"),
	Mantle:   lipgloss.Color("#242933"),
	Surface0: lipgloss.Color("#3b4252"),
	Surface1: lipgloss.Color("#434c5e"),
	Overlay0: lipgloss.Color("#4c566a"),
	Text:     lipgloss.Color("#eceff4"),
	Subtext0: lipgloss.Color("#d8dee9"),
	Peach:    lipgloss.Color("#d08770"),
	Mauve:    lipgloss.Color("#b48ead"),
	Teal:     lipgloss.Color("#8fbcbb"),
	Green:    lipgloss.Color("#a3be8c"),
	Red:      lipgloss.Color("#bf616a"),
	Blue:     lipgloss.Color("#81a1c1"),
	Yellow:   lipgloss.Color("#ebcb8b"),
}

// AvailableThemes contiene todos los temas disponibles indexados por nombre
var AvailableThemes = map[string]Palette{
	"catppuccin-mocha":     CatppuccinMocha,
	"catppuccin-latte":     CatppuccinLatte,
	"catppuccin-frappe":    CatppuccinFrappe,
	"catppuccin-macchiato": CatppuccinMacchiato,
	"tokyo-night":          TokyoNight,
	"gruvbox-dark":         GruvboxDark,
	"nord":                 Nord,
}

// ThemeNames devuelve los nombres de temas en orden para la UI de selección
func ThemeNames() []string {
	return []string{
		"catppuccin-mocha",
		"catppuccin-latte",
		"catppuccin-frappe",
		"catppuccin-macchiato",
		"tokyo-night",
		"gruvbox-dark",
		"nord",
	}
}

// ─── Colores activos (variables globales usadas por todos los renders) ──

var (
	ColorBase     = CatppuccinMocha.Base
	ColorMantle   = CatppuccinMocha.Mantle
	ColorSurface0 = CatppuccinMocha.Surface0
	ColorSurface1 = CatppuccinMocha.Surface1
	ColorOverlay0 = CatppuccinMocha.Overlay0
	ColorText     = CatppuccinMocha.Text
	ColorSubtext0 = CatppuccinMocha.Subtext0
	ColorPeach    = CatppuccinMocha.Peach
	ColorMauve    = CatppuccinMocha.Mauve
	ColorTeal     = CatppuccinMocha.Teal
	ColorGreen    = CatppuccinMocha.Green
	ColorRed      = CatppuccinMocha.Red
	ColorBlue     = CatppuccinMocha.Blue
	ColorYellow   = CatppuccinMocha.Yellow
)

// ─── Estilos activos (reconstruidos al cambiar de tema) ────────────

var (
	ActivePanelBorder  lipgloss.Style
	InactivePanelBorder lipgloss.Style
	TabActive          lipgloss.Style
	TabInactive        lipgloss.Style
	SelectedItem       lipgloss.Style
	NormalItem         lipgloss.Style
	TagBadge           lipgloss.Style
	TaskDone           lipgloss.Style
	TaskPending        lipgloss.Style
	FooterBar          lipgloss.Style
	FooterKey          lipgloss.Style
)

// CurrentThemeName almacena el nombre del tema actualmente aplicado
var CurrentThemeName = "catppuccin-mocha"

func init() {
	ApplyPalette(CatppuccinMocha)
}

// ApplyPalette aplica una paleta de colores y reconstruye todos los estilos
func ApplyPalette(p Palette) {
	CurrentThemeName = p.Name

	// Actualizar colores globales
	ColorBase = p.Base
	ColorMantle = p.Mantle
	ColorSurface0 = p.Surface0
	ColorSurface1 = p.Surface1
	ColorOverlay0 = p.Overlay0
	ColorText = p.Text
	ColorSubtext0 = p.Subtext0
	ColorPeach = p.Peach
	ColorMauve = p.Mauve
	ColorTeal = p.Teal
	ColorGreen = p.Green
	ColorRed = p.Red
	ColorBlue = p.Blue
	ColorYellow = p.Yellow

	// Reconstruir estilos
	ActivePanelBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPeach).
		Padding(0, 1)

	InactivePanelBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSurface0).
		Padding(0, 1)

	TabActive = lipgloss.NewStyle().
		Foreground(ColorBase).
		Background(ColorPeach).
		Bold(true).
		Padding(0, 2)

	TabInactive = lipgloss.NewStyle().
		Foreground(ColorSubtext0).
		Background(ColorSurface0).
		Padding(0, 2)

	SelectedItem = lipgloss.NewStyle().
		Foreground(ColorPeach).
		Bold(true)

	NormalItem = lipgloss.NewStyle().
		Foreground(ColorText)

	TagBadge = lipgloss.NewStyle().
		Foreground(ColorTeal).
		Background(ColorSurface0).
		Padding(0, 1).
		MarginRight(1)

	TaskDone = lipgloss.NewStyle().
		Foreground(ColorOverlay0).
		Strikethrough(true)

	TaskPending = lipgloss.NewStyle().
		Foreground(ColorYellow)

	FooterBar = lipgloss.NewStyle().
		Foreground(ColorSubtext0).
		Padding(0, 1)

	FooterKey = lipgloss.NewStyle().
		Foreground(ColorMauve).
		Bold(true)
}

// ApplyThemeByName aplica un tema por nombre. Devuelve false si no existe.
func ApplyThemeByName(name string) bool {
	p, ok := AvailableThemes[name]
	if !ok {
		return false
	}
	ApplyPalette(p)
	return true
}

// NextTheme cicla al siguiente tema disponible y lo aplica
func NextTheme() string {
	names := ThemeNames()
	for i, n := range names {
		if n == CurrentThemeName {
			next := names[(i+1)%len(names)]
			ApplyThemeByName(next)
			return next
		}
	}
	// Fallback: aplicar el primero
	ApplyThemeByName(names[0])
	return names[0]
}

// PrevTheme cicla al tema anterior disponible y lo aplica
func PrevTheme() string {
	names := ThemeNames()
	for i, n := range names {
		if n == CurrentThemeName {
			prev := names[(i-1+len(names))%len(names)]
			ApplyThemeByName(prev)
			return prev
		}
	}
	// Fallback: aplicar el primero
	ApplyThemeByName(names[0])
	return names[0]
}

