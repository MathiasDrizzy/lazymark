package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MathiasDrizzy/lazymark/internal/app"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var (
		customDir   string
		showVersion bool
		noMouse     bool
		themeName   string
	)

	flag.StringVar(&customDir, "dir", "", "Ruta al directorio de notas Markdown (por defecto: $HOME/Documents/notes)")
	flag.BoolVar(&showVersion, "version", false, "Muestra la versión de lazymark y sale")
	flag.BoolVar(&showVersion, "v", false, "Alias para --version")
	flag.BoolVar(&noMouse, "no-mouse", false, "Desactiva la interacción con ratón y clics")
	flag.StringVar(&themeName, "theme", "", "Tema de colores (catppuccin-mocha, catppuccin-latte, catppuccin-frappe, catppuccin-macchiato, tokyo-night, gruvbox-dark, nord)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s [opciones]\n\n", config.AppName)
		fmt.Fprintf(os.Stderr, "lazymark — TUI para notas en Markdown, tareas e imágenes estilo Lazygit\n\n")
		fmt.Fprintf(os.Stderr, "Opciones disponibles:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nAtajos Principales:\n")
		fmt.Fprintf(os.Stderr, "  [1..4]      Cambiar de pestaña (Notas, Categorías, Tareas, Adjuntos)\n")
		fmt.Fprintf(os.Stderr, "  [↑/↓ ó j/k] Navegar entre ítems\n")
		fmt.Fprintf(os.Stderr, "  [←/→ ó h/l] Alternar entre lista y vista previa\n")
		fmt.Fprintf(os.Stderr, "  [Tab]       Alternar entre lista y vista previa\n")
		fmt.Fprintf(os.Stderr, "  [g / G]     Ir al primer / último ítem\n")
		fmt.Fprintf(os.Stderr, "  [Enter / e] Abrir nota en el editor externo ($EDITOR o micro)\n")
		fmt.Fprintf(os.Stderr, "  [c]         Crear nueva nota\n")
		fmt.Fprintf(os.Stderr, "  [p]         Pegar imagen desde el portapapeles\n")
		fmt.Fprintf(os.Stderr, "  [d]         Borrar nota seleccionada\n")
		fmt.Fprintf(os.Stderr, "  [f]         Filtrar tareas (pestaña Tareas)\n")
		fmt.Fprintf(os.Stderr, "  [t]         Ciclar tema de colores\n")
		fmt.Fprintf(os.Stderr, "  [q / Esc]   Salir de la aplicación\n")
		fmt.Fprintf(os.Stderr, "  [Clic]      Seleccionar notas, cambiar pestañas y presionar botones con el mouse\n")
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("%s v%s\n", config.AppName, config.Version)
		os.Exit(0)
	}

	cfg, err := config.Load(customDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al inicializar la configuración: %v\n", err)
		os.Exit(1)
	}

	if noMouse {
		cfg.MouseClick = false
	}

	if themeName != "" {
		cfg.Theme = themeName
	}

	appModel, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al inicializar lazymark: %v\n", err)
		os.Exit(1)
	}

	var opts []tea.ProgramOption
	opts = append(opts, tea.WithAltScreen())

	if cfg.MouseClick {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(appModel, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error durante la ejecución: %v\n", err)
		os.Exit(1)
	}
}
