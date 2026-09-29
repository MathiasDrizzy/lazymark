package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Paleta Catppuccin Mocha Oficial
var (
	ColorBase     = lipgloss.Color("#1e1e2e")
	ColorMantle   = lipgloss.Color("#181825")
	ColorSurface0 = lipgloss.Color("#313244")
	ColorSurface1 = lipgloss.Color("#45475a")
	ColorOverlay0 = lipgloss.Color("#6c7086")
	ColorText     = lipgloss.Color("#cdd6f4")
	ColorSubtext0 = lipgloss.Color("#a6adc8")
	ColorPeach    = lipgloss.Color("#fab387")
	ColorMauve    = lipgloss.Color("#cba6f7")
	ColorTeal     = lipgloss.Color("#94e2d5")
	ColorGreen    = lipgloss.Color("#a6e3a1")
	ColorRed      = lipgloss.Color("#f38ba8")
	ColorBlue     = lipgloss.Color("#89b4fa")
	ColorYellow   = lipgloss.Color("#f9e2af")
)

// Estilos de Paneles y Bordes
var (
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
		Background(ColorMantle).
		Padding(0, 1)

	FooterKey = lipgloss.NewStyle().
		Foreground(ColorMauve).
		Bold(true)
)
