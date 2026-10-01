package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/app"
	"github.com/MathiasDrizzy/lazymark/internal/cli"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/mcp"
	tea "github.com/charmbracelet/bubbletea"
)

func extractDirArg(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], "--dir=") {
			return strings.TrimPrefix(args[i], "--dir=")
		}
	}
	return ""
}

func main() {
	// 1. Interceptar subcomandos headless CLI e inter-agente (task, note, mcp)
	for idx, arg := range os.Args[1:] {
		if arg == "task" {
			dir := extractDirArg(os.Args[1:])
			if err := cli.RunTask(os.Args[idx+2:], dir); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if arg == "note" {
			dir := extractDirArg(os.Args[1:])
			if err := cli.RunNote(os.Args[idx+2:], dir); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if arg == "mcp" {
			dir := extractDirArg(os.Args[1:])
			if err := mcp.RunServer(dir); err != nil {
				fmt.Fprintf(os.Stderr, "Error en servidor MCP: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if !strings.HasPrefix(arg, "-") {
			break
		}
	}

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
		fmt.Fprintf(os.Stderr, "Uso: %s [opciones] [subcomando]\n\n", config.AppName)
		fmt.Fprintf(os.Stderr, "lazymark — TUI para notas en Markdown, tareas e imágenes estilo Lazygit\n\n")
		fmt.Fprintf(os.Stderr, "Subcomandos Headless e Inter-Agente:\n")
		fmt.Fprintf(os.Stderr, "  task list [--json] [--pending]           Lista tareas con filtros\n")
		fmt.Fprintf(os.Stderr, "  task toggle --path <nota> --line <num>   Alterna el estado de una tarea\n")
		fmt.Fprintf(os.Stderr, "  note list [--json]                       Lista notas Markdown\n")
		fmt.Fprintf(os.Stderr, "  note get <ruta>                          Obtiene el contenido de una nota\n")
		fmt.Fprintf(os.Stderr, "  mcp                                      Inicia el servidor MCP nativo (JSON-RPC 2.0)\n\n")
		fmt.Fprintf(os.Stderr, "Opciones disponibles:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nAtajos Principales (Hoja 1: Vista Lazygit):\n")
		fmt.Fprintf(os.Stderr, "  [1..4]      Navegar paneles modulares (Notas, Tareas, Tags, Preview)\n")
		fmt.Fprintf(os.Stderr, "  [W / K]     Alternar entre Hoja 1 (Lazygit) y Hoja 2 (Tablero Kanban)\n")
		fmt.Fprintf(os.Stderr, "  [↑/↓ ó j/k] Navegar entre ítems\n")
		fmt.Fprintf(os.Stderr, "  [←/→ ó h/l] Alternar foco de panel\n")
		fmt.Fprintf(os.Stderr, "  [Tab]       Alternar panel activo\n")
		fmt.Fprintf(os.Stderr, "  [Enter / e] Abrir nota en el editor externo ($EDITOR o micro)\n")
		fmt.Fprintf(os.Stderr, "  [c]         Crear nueva nota\n")
		fmt.Fprintf(os.Stderr, "  [p]         Pegar imagen desde el portapapeles\n")
		fmt.Fprintf(os.Stderr, "  [d]         Borrar nota seleccionada\n")
		fmt.Fprintf(os.Stderr, "  [Espacio/x] Completar tarea atómicamente\n")
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
