package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/app"
	"github.com/MathiasDrizzy/lazymark/internal/cli"
	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/editors"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/mcp"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
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

// usageHeader es la ayuda de `lazymark --help` antes de la lista de opciones.
func usageHeader() string {
	return i18n.T("Uso: ", "Usage: ") + config.AppName + i18n.T(" [opciones] [subcomando]\n\n", " [options] [command]\n\n") +
		config.AppName + i18n.T(" — notas Markdown, tareas y tablero Kanban en la terminal, sin esfuerzo (lazy)\n\n", ": markdown notes, tasks and a Kanban board in the terminal, the lazy way\n\n") +
		i18n.T("Subcomandos (sin interfaz, para scripts y agentes):\n", "Commands (headless, for scripts and agents):\n") +
		"  task list [--json] [--pending] [--dir <dir>]\n" +
		"  task toggle --path <note> --line <n> [--dir <dir>]\n" +
		"  note list [--json] [--dir <dir>]\n" +
		"  note get <path> [--dir <dir>]\n" +
		"  paste [--no-newline] [<note.md>]\n" +
		"  editor-plugins install|uninstall [micro|vim|nano]\n" +
		"  mcp [--dir <dir>]\n\n" +
		i18n.T("Opciones:\n", "Options:\n")
}

// usageFooter apunta a los atajos reales: la lista completa sale del keymap,
// así que la ayuda no repite teclas que podrían desfasarse.
func usageFooter() string {
	return "\n" + i18n.T("Dentro de la aplicación, ? muestra los atajos del panel actual y , abre los ajustes.\n", "Inside the app, ? shows the keys of the current panel and , opens the settings.\n")
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
		if arg == "paste" {
			os.Exit(cli.RunPaste(os.Args[idx+2:], os.Getenv, clipboard.New(), os.Stdout, os.Stderr))
		}
		if arg == "editor-plugins" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(cli.RunEditorPlugins(os.Args[idx+2:], editors.Env{Home: home, Getenv: os.Getenv}, os.Stdout, os.Stderr))
		}
		if arg == "mcp" {
			for _, a := range os.Args[idx+2:] {
				if a == "-h" || a == "--help" {
					fmt.Println(i18n.T("uso: lazymark mcp [--dir <carpeta>]  (servidor MCP por stdio)", "usage: lazymark mcp [--dir <folder>]  (MCP server over stdio)"))
					return
				}
			}
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

	flag.StringVar(&customDir, "dir", "", i18n.T("Carpeta de notas (por defecto: ~/Documents/notes)", "Notes folder (default: ~/Documents/notes)"))
	flag.BoolVar(&showVersion, "version", false, i18n.T("Muestra la versión y sale", "Print the version and exit"))
	flag.BoolVar(&showVersion, "v", false, i18n.T("Alias de --version", "Alias for --version"))
	flag.BoolVar(&noMouse, "no-mouse", false, i18n.T("Desactiva el mouse", "Disable mouse input"))
	flag.StringVar(&themeName, "theme", "", i18n.T("Tema de colores: ", "Color theme: ")+strings.Join(theme.ThemeNames(), ", "))

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageHeader())
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, usageFooter())
	}

	flag.Parse()

	if flag.NArg() > 0 { // una palabra suelta que no es un subcomando: no se abre la interfaz por error
		fmt.Fprintf(os.Stderr, i18n.T("lazymark: subcomando desconocido %q\n\n", "lazymark: unknown command %q\n\n"), flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}

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

	// Pantalla alternativa y mouse se declaran en AppModel.View (Bubble Tea v2)
	p := tea.NewProgram(appModel)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error durante la ejecución: %v\n", err)
		os.Exit(1)
	}
}
