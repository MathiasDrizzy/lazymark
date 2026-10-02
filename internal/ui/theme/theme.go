package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette define los colores de un tema
type Palette struct {
	Name string
	// Source es la página oficial de la paleta, de donde salen los colores.
	Source   string
	Base     color.Color
	Mantle   color.Color
	Surface0 color.Color
	Surface1 color.Color
	Overlay0 color.Color
	Text     color.Color
	Subtext0 color.Color
	Peach    color.Color
	Mauve    color.Color
	Teal     color.Color
	Green    color.Color
	Red      color.Color
	Blue     color.Color
	Yellow   color.Color
}

// ─── Paletas disponibles ───────────────────────────────────────────

// CatppuccinMocha es el tema oscuro por defecto (dark, warm)
var CatppuccinMocha = Palette{
	Name:     "catppuccin-mocha",
	Source:   "https://catppuccin.com/palette/",
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
	Source:   "https://catppuccin.com/palette/",
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
	Source:   "https://catppuccin.com/palette/",
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
	Source:   "https://catppuccin.com/palette/",
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
	Source:   "https://github.com/folke/tokyonight.nvim",
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
	Source:   "https://github.com/morhetz/gruvbox",
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
	Source:   "https://www.nordtheme.com/docs/colors-and-palettes",
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

// Los colores de las paletas siguientes salen de la fuente oficial de cada tema
// (la URL va en Source). Donde el tema no define un color que la interfaz necesita
// (por ejemplo, dos niveles de superficie), se usa el vecino más cercano de su propia
// paleta y se dice en el comentario.

// Dracula, https://draculatheme.com/contribute y https://spec.draculatheme.com/
// (dark, purple). Mantle es el AnsiBlack de la especificación; Surface0 y Surface1 son
// Selection (#44475a, "Selection" de la especificación) y Overlay0 es Comment (#6272a4), así
// la barra de selección se distingue del borde inactivo. (La página de contribución rotula
// #6272a4 como "Current Line"; la especificación lo llama Comment.) Dracula no tiene un gris
// de texto secundario: Subtext0 repite Foreground.
var Dracula = Palette{
	Name:     "dracula",
	Source:   "https://draculatheme.com/contribute",
	Base:     lipgloss.Color("#282a36"),
	Mantle:   lipgloss.Color("#21222c"),
	Surface0: lipgloss.Color("#44475a"),
	Surface1: lipgloss.Color("#44475a"),
	Overlay0: lipgloss.Color("#6272a4"),
	Text:     lipgloss.Color("#f8f8f2"),
	Subtext0: lipgloss.Color("#f8f8f2"),
	Peach:    lipgloss.Color("#ffb86c"),
	Mauve:    lipgloss.Color("#ff79c6"),
	Teal:     lipgloss.Color("#8be9fd"),
	Green:    lipgloss.Color("#50fa7b"),
	Red:      lipgloss.Color("#ff5555"),
	Blue:     lipgloss.Color("#bd93f9"),
	Yellow:   lipgloss.Color("#f1fa8c"),
}

// OneDark, https://github.com/atom/one-dark-syntax/blob/master/styles/colors.less
// y https://github.com/atom/atom/blob/master/packages/one-dark-ui/styles/ui-variables-custom.less
// (dark, cool). Atom los define en HSL; los hex salen de esos valores. Base es syntax-bg,
// Mantle es darken(base, 3%), Surface0 es el level-1 (lighten 6%) y Surface1 el
// siguiente escalón (lighten 12%).
var OneDark = Palette{
	Name:     "one-dark",
	Source:   "https://github.com/atom/one-dark-syntax",
	Base:     lipgloss.Color("#282c34"),
	Mantle:   lipgloss.Color("#21252b"),
	Surface0: lipgloss.Color("#353b45"),
	Surface1: lipgloss.Color("#434956"),
	Overlay0: lipgloss.Color("#5c6370"),
	Text:     lipgloss.Color("#abb2bf"),
	Subtext0: lipgloss.Color("#828997"),
	Peach:    lipgloss.Color("#d19a66"),
	Mauve:    lipgloss.Color("#c678dd"),
	Teal:     lipgloss.Color("#56b6c2"),
	Green:    lipgloss.Color("#98c379"),
	Red:      lipgloss.Color("#e06c75"),
	Blue:     lipgloss.Color("#61afef"),
	Yellow:   lipgloss.Color("#e5c07b"),
}

// RosePine, https://rosepinetheme.com/palette (valores de
// https://github.com/rose-pine/palette, variante main; dark, warm). Rosé Pine no tiene
// verde ni azul: Green y Blue usan foam, y Teal también.
var RosePine = Palette{
	Name:     "rose-pine",
	Source:   "https://rosepinetheme.com/palette",
	Base:     lipgloss.Color("#191724"),
	Mantle:   lipgloss.Color("#1f1d2e"),
	Surface0: lipgloss.Color("#26233a"),
	Surface1: lipgloss.Color("#403d52"),
	Overlay0: lipgloss.Color("#6e6a86"),
	Text:     lipgloss.Color("#e0def4"),
	Subtext0: lipgloss.Color("#908caa"),
	Peach:    lipgloss.Color("#ebbcba"),
	Mauve:    lipgloss.Color("#c4a7e7"),
	Teal:     lipgloss.Color("#9ccfd8"),
	Green:    lipgloss.Color("#9ccfd8"),
	Red:      lipgloss.Color("#eb6f92"),
	Blue:     lipgloss.Color("#9ccfd8"),
	Yellow:   lipgloss.Color("#f6c177"),
}

// Kanagawa, https://github.com/rebelot/kanagawa.nvim (paleta wave; dark, ink).
var Kanagawa = Palette{
	Name:     "kanagawa",
	Source:   "https://github.com/rebelot/kanagawa.nvim",
	Base:     lipgloss.Color("#1f1f28"),
	Mantle:   lipgloss.Color("#16161d"),
	Surface0: lipgloss.Color("#2a2a37"),
	Surface1: lipgloss.Color("#363646"),
	Overlay0: lipgloss.Color("#727169"),
	Text:     lipgloss.Color("#dcd7ba"),
	Subtext0: lipgloss.Color("#c8c093"),
	Peach:    lipgloss.Color("#ffa066"),
	Mauve:    lipgloss.Color("#957fb8"),
	Teal:     lipgloss.Color("#7aa89f"),
	Green:    lipgloss.Color("#98bb6c"),
	Red:      lipgloss.Color("#e46876"),
	Blue:     lipgloss.Color("#7e9cd8"),
	Yellow:   lipgloss.Color("#e6c384"),
}

// EverforestDark, https://github.com/sainnhe/everforest/blob/master/palette.md
// (dark, contraste medio; green).
var EverforestDark = Palette{
	Name:     "everforest-dark",
	Source:   "https://github.com/sainnhe/everforest",
	Base:     lipgloss.Color("#2d353b"),
	Mantle:   lipgloss.Color("#232a2e"),
	Surface0: lipgloss.Color("#343f44"),
	Surface1: lipgloss.Color("#475258"),
	Overlay0: lipgloss.Color("#7a8478"),
	Text:     lipgloss.Color("#d3c6aa"),
	Subtext0: lipgloss.Color("#9da9a0"),
	Peach:    lipgloss.Color("#e69875"),
	Mauve:    lipgloss.Color("#d699b6"),
	Teal:     lipgloss.Color("#83c092"),
	Green:    lipgloss.Color("#a7c080"),
	Red:      lipgloss.Color("#e67e80"),
	Blue:     lipgloss.Color("#7fbbb3"),
	Yellow:   lipgloss.Color("#dbbc7f"),
}

// SolarizedDark, https://ethanschoonover.com/solarized/ (dark). Solarized no tiene
// un tono más oscuro que base03: Mantle lo repite; Surface1 es base01 y Overlay0 base00,
// para que la selección se distinga del borde inactivo.
var SolarizedDark = Palette{
	Name:     "solarized-dark",
	Source:   "https://ethanschoonover.com/solarized/",
	Base:     lipgloss.Color("#002b36"),
	Mantle:   lipgloss.Color("#002b36"),
	Surface0: lipgloss.Color("#073642"),
	Surface1: lipgloss.Color("#586e75"),
	Overlay0: lipgloss.Color("#657b83"),
	Text:     lipgloss.Color("#839496"),
	Subtext0: lipgloss.Color("#657b83"),
	Peach:    lipgloss.Color("#cb4b16"),
	Mauve:    lipgloss.Color("#6c71c4"),
	Teal:     lipgloss.Color("#2aa198"),
	Green:    lipgloss.Color("#859900"),
	Red:      lipgloss.Color("#dc322f"),
	Blue:     lipgloss.Color("#268bd2"),
	Yellow:   lipgloss.Color("#b58900"),
}

// SolarizedLight, https://ethanschoonover.com/solarized/ (light). Mantle y Surface0
// son base2; Solarized no tiene otro escalón claro-oscuro, así que Surface1 también.
var SolarizedLight = Palette{
	Name:     "solarized-light",
	Source:   "https://ethanschoonover.com/solarized/",
	Base:     lipgloss.Color("#fdf6e3"),
	Mantle:   lipgloss.Color("#eee8d5"),
	Surface0: lipgloss.Color("#eee8d5"),
	Surface1: lipgloss.Color("#eee8d5"),
	Overlay0: lipgloss.Color("#93a1a1"),
	Text:     lipgloss.Color("#657b83"),
	Subtext0: lipgloss.Color("#839496"),
	Peach:    lipgloss.Color("#cb4b16"),
	Mauve:    lipgloss.Color("#6c71c4"),
	Teal:     lipgloss.Color("#2aa198"),
	Green:    lipgloss.Color("#859900"),
	Red:      lipgloss.Color("#dc322f"),
	Blue:     lipgloss.Color("#268bd2"),
	Yellow:   lipgloss.Color("#b58900"),
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
	"dracula":              Dracula,
	"one-dark":             OneDark,
	"rose-pine":            RosePine,
	"kanagawa":             Kanagawa,
	"everforest-dark":      EverforestDark,
	"solarized-dark":       SolarizedDark,
	"solarized-light":      SolarizedLight,
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
		"dracula",
		"one-dark",
		"rose-pine",
		"kanagawa",
		"everforest-dark",
		"solarized-dark",
		"solarized-light",
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
	ActivePanelBorder   lipgloss.Style
	InactivePanelBorder lipgloss.Style
	TabActive           lipgloss.Style
	TabInactive         lipgloss.Style
	SelectedItem        lipgloss.Style
	NormalItem          lipgloss.Style

	// Barra de selección sólida estilo Lazygit a todo lo ancho
	SelectedLineActive   lipgloss.Style
	SelectedLineInactive lipgloss.Style

	TagBadge    lipgloss.Style
	TaskDone    lipgloss.Style
	TaskPending lipgloss.Style
	FooterBar   lipgloss.Style
	FooterKey   lipgloss.Style
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

	SelectedLineActive = lipgloss.NewStyle().
		Background(ColorPeach).
		Foreground(ColorBase).
		Bold(true)

	SelectedLineInactive = lipgloss.NewStyle().
		Background(ColorSurface1).
		Foreground(ColorText)

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
		Background(ColorSurface0).
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
