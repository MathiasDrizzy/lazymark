package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ops"
)

// parser separa los flags de los argumentos posicionales, vengan en el orden que vengan
// (`task move <id> doing --json` y `task move --json <id> doing` son lo mismo).
type parser struct {
	fs      *flag.FlagSet
	json    bool
	dir     string
	help    bool
	posArgs []string
}

func newParser(name, defaultDir string) *parser {
	p := &parser{fs: flag.NewFlagSet(name, flag.ContinueOnError)}
	p.fs.SetOutput(io.Discard)
	p.fs.BoolVar(&p.json, "json", false, "")
	p.fs.StringVar(&p.dir, "dir", defaultDir, "")
	p.fs.BoolVar(&p.help, "h", false, "")
	p.fs.BoolVar(&p.help, "help", false, "")
	return p
}

// parse lee args; un flag desconocido, sin valor o con un valor inválido es un error de uso (código 2).
func (p *parser) parse(args []string) error {
	for len(args) > 0 {
		if err := p.fs.Parse(args); err != nil {
			return &ops.Error{Code: ExitUsage, Err: err}
		}
		args = p.fs.Args()
		if len(args) > 0 {
			p.posArgs = append(p.posArgs, args[0])
			args = args[1:]
		}
	}
	return nil
}

// need exige exactamente n argumentos posicionales.
func (p *parser) need(n int, usage string) error {
	if len(p.posArgs) != n {
		return &ops.Error{Code: ExitUsage, Err: fmt.Errorf("uso: %s", usage)}
	}
	return nil
}

func (p *parser) service() (*ops.Service, error) {
	dir := p.dir
	if dir == "" {
		dir = config.DefaultNotesDir()
	}
	return ops.New(dir)
}

// plain quita de un texto los caracteres de control (ESC, BEL, C1, \r…) menos el salto de línea y el tabulador: el
// contenido de una nota no es confiable y una secuencia como OSC 52 escribiría en el portapapeles de la terminal.
// Es solo para la salida de texto; `--json` ya los escapa y lleva el contenido exacto.
func plain(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// printJSON escribe v con sangría (el esquema de cada comando está en docs/cli.md).
func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func usageText(es, en string) string { return i18n.T(es, en) }

const taskUsageES = `uso: lazymark task list   [--json] [--pending] [--column <id>] [--note <ruta>] [--dir <carpeta>]
     lazymark task toggle <id> [--json] [--dir <carpeta>]
     lazymark task move   <id> <columna> [--json] [--dir <carpeta>]
     lazymark task due    <id> <AAAA-MM-DD|none> [--json] [--dir <carpeta>]   (vencimiento 📅)
     lazymark task start  <id> <AAAA-MM-DD|none> [--json] [--dir <carpeta>]   (inicio 🛫)
códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos (no se toca nada) · 3 no existe · 4 la nota cambió (no se escribe)
`

const taskUsageEN = `usage: lazymark task list   [--json] [--pending] [--column <id>] [--note <path>] [--dir <folder>]
       lazymark task toggle <id> [--json] [--dir <folder>]
       lazymark task move   <id> <column> [--json] [--dir <folder>]
       lazymark task due    <id> <YYYY-MM-DD|none> [--json] [--dir <folder>]   (due date 📅)
       lazymark task start  <id> <YYYY-MM-DD|none> [--json] [--dir <folder>]   (start date 🛫)
exit codes: 0 ok · 1 failed · 2 invalid arguments (nothing touched) · 3 not found · 4 the note changed (nothing written)
`

const noteUsageES = `uso: lazymark note list [--json] [--dir <carpeta>]
     lazymark note show <ruta> [--json] [--dir <carpeta>]
     lazymark note new  <título> [--folder <subcarpeta>] [--empty] [--template <nombre>] [--json] [--dir <carpeta>]
códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos o ruta fuera de la carpeta de notas · 3 no existe
`

const noteUsageEN = `usage: lazymark note list [--json] [--dir <folder>]
       lazymark note show <path> [--json] [--dir <folder>]
       lazymark note new  <title> [--folder <subfolder>] [--empty] [--template <name>] [--json] [--dir <folder>]
exit codes: 0 ok · 1 failed · 2 invalid arguments or a path outside the notes folder · 3 not found
`

// RunTask ejecuta `lazymark task …` (list, toggle, move).
func RunTask(args []string, defaultNotesDir string) error {
	return RunTaskWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunTaskWithWriter ejecuta los subcomandos de tareas escribiendo la salida en w.
func RunTaskWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(taskUsageES, taskUsageEN)
	if len(args) == 0 {
		return &ops.Error{Code: ExitUsage, Err: errors.New("falta el subcomando (list, toggle, move)\n" + help)}
	}
	action := args[0]
	if action == "-h" || action == "--help" {
		fmt.Fprint(w, help)
		return nil
	}
	p := newParser("task "+action, defaultNotesDir)
	var (
		pending      bool
		column, note string
		path         string
		line         int
	)
	switch action {
	case "list":
		p.fs.BoolVar(&pending, "pending", false, "")
		p.fs.StringVar(&column, "column", "", "")
		p.fs.StringVar(&note, "note", "", "")
	case "toggle":
		p.fs.StringVar(&path, "path", "", "") // forma anterior: --path <nota> --line <n>
		p.fs.IntVar(&line, "line", 0, "")
	case "move", "due", "start":
	default:
		return &ops.Error{Code: ExitUsage, Err: fmt.Errorf("subcomando de tarea desconocido: %q (list, toggle, move, due, start)", action)}
	}
	if err := p.parse(args[1:]); err != nil {
		return err
	}
	if p.help {
		fmt.Fprint(w, help)
		return nil
	}

	switch action {
	case "list":
		if err := p.need(0, "lazymark task list [--json] [--pending] [--column <id>] [--note <ruta>]"); err != nil {
			return err
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		tasks, err := svc.ListTasks(ops.TaskFilter{PendingOnly: pending, Column: column, Note: note})
		if err != nil {
			return err
		}
		if p.json {
			return printJSON(w, tasks)
		}
		for _, t := range tasks {
			mark := "[ ]"
			if t.Done {
				mark = "[x]"
			}
			fmt.Fprint(w, plain(fmt.Sprintf("%s %s  %s  (%s)%s\n", mark, t.ID, t.Text, t.Column, datesSuffix(t))))
		}
		return nil

	case "toggle":
		var id string
		if path != "" || line != 0 { // forma anterior
			if path == "" || line <= 0 || len(p.posArgs) != 0 {
				return &ops.Error{Code: ExitUsage, Err: errors.New("uso: lazymark task toggle --path <nota> --line <n>")}
			}
			svc, err := p.service()
			if err != nil {
				return err
			}
			if id, err = svc.IDByLine(path, line); err != nil {
				return err
			}
			return finishMove(w, p, svc, func() (ops.TaskDTO, error) { return svc.ToggleTask(id) })
		}
		if err := p.need(1, "lazymark task toggle <id>"); err != nil {
			return err
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		return finishMove(w, p, svc, func() (ops.TaskDTO, error) { return svc.ToggleTask(p.posArgs[0]) })

	case "due", "start":
		if err := p.need(2, "lazymark task "+action+" <id> <AAAA-MM-DD|none>"); err != nil {
			return err
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		return finishMove(w, p, svc, func() (ops.TaskDTO, error) { return svc.SetDate(p.posArgs[0], action, p.posArgs[1]) })

	default: // move
		if err := p.need(2, "lazymark task move <id> <columna>"); err != nil {
			return err
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		return finishMove(w, p, svc, func() (ops.TaskDTO, error) { return svc.MoveTask(p.posArgs[0], p.posArgs[1]) })
	}
}

func finishMove(w io.Writer, p *parser, _ *ops.Service, do func() (ops.TaskDTO, error)) error {
	t, err := do()
	if err != nil {
		return err
	}
	if p.json {
		return printJSON(w, t)
	}
	fmt.Fprint(w, plain(fmt.Sprintf("%s → %s%s\n", t.ID, t.Column, datesSuffix(t))))
	return nil
}

// datesSuffix muestra las fechas de una tarea en la salida de texto con símbolos (▸ inicio, ◷ vencimiento, ✓ completada), no con
// los emojis del archivo: "  ▸ 2026-05-01 ◷ 2026-05-10 (vencida)". `--json` lleva las fechas en sus campos.
func datesSuffix(t ops.TaskDTO) string {
	var parts []string
	for _, d := range [][2]string{{"▸", t.Start}, {"◷", t.Due}, {"✓", t.Completed}} {
		if d[1] != "" {
			parts = append(parts, d[0]+" "+d[1])
		}
	}
	if len(parts) == 0 {
		return ""
	}
	out := "  " + strings.Join(parts, " ")
	if t.Overdue {
		out += " (" + i18n.T("vencida", "overdue") + ")"
	}
	return out
}

// RunNote ejecuta `lazymark note …` (list, show, new).
func RunNote(args []string, defaultNotesDir string) error {
	return RunNoteWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunNoteWithWriter ejecuta los subcomandos de notas escribiendo la salida en w.
func RunNoteWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(noteUsageES, noteUsageEN)
	if len(args) == 0 {
		return &ops.Error{Code: ExitUsage, Err: errors.New("falta el subcomando (list, show, new)\n" + help)}
	}
	action := args[0]
	if action == "-h" || action == "--help" {
		fmt.Fprint(w, help)
		return nil
	}
	if action == "get" { // nombre anterior de show
		action = "show"
	}
	p := newParser("note "+action, defaultNotesDir)
	var (
		folder   string
		template string
		empty    bool
	)
	switch action {
	case "list", "show":
	case "new":
		p.fs.StringVar(&folder, "folder", "", "")
		p.fs.BoolVar(&empty, "empty", false, "")
		p.fs.StringVar(&template, "template", "", "")
	default:
		return &ops.Error{Code: ExitUsage, Err: fmt.Errorf("subcomando de nota desconocido: %q (list, show, new)", action)}
	}
	if err := p.parse(args[1:]); err != nil {
		return err
	}
	if p.help {
		fmt.Fprint(w, help)
		return nil
	}

	switch action {
	case "list":
		if err := p.need(0, "lazymark note list [--json]"); err != nil {
			return err
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		notes, err := svc.ListNotes()
		if err != nil {
			return err
		}
		if p.json {
			return printJSON(w, notes)
		}
		for _, n := range notes {
			fmt.Fprintf(w, "%s (%s)\n", plain(n.Title), plain(n.Path))
		}
		return nil

	case "show":
		if err := p.need(1, "lazymark note show <ruta>"); err != nil {
			return err
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		n, err := svc.ShowNote(p.posArgs[0])
		if err != nil {
			return err
		}
		if p.json {
			return printJSON(w, n)
		}
		_, err = io.WriteString(w, plain(n.Content))
		return err

	default: // new
		if len(p.posArgs) == 0 {
			return &ops.Error{Code: ExitUsage, Err: errors.New("uso: lazymark note new <título> [--folder <subcarpeta>] [--empty] [--template <nombre>]")}
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		n, err := svc.NewNoteFromTemplate(strings.Join(p.posArgs, " "), folder, template, empty)
		if err != nil {
			return err
		}
		if p.json {
			return printJSON(w, n)
		}
		fmt.Fprintf(w, "%s\n", n.Path)
		return nil
	}
}

const searchUsageES = `uso: lazymark search <texto> [--regex] [--case] [--limit <n>] [--json] [--dir <carpeta>]
Busca <texto> en todas las notas (sin distinguir mayúsculas; con --case, distinguiéndolas; con --regex, una expresión regular).
Imprime "nota:línea: texto". Sin coincidencias no es un error (sale con 0 y no imprime nada).
códigos de salida: 0 ok · 1 falló · 2 búsqueda vacía o expresión inválida (no se toca nada)
`

const searchUsageEN = `usage: lazymark search <text> [--regex] [--case] [--limit <n>] [--json] [--dir <folder>]
Searches <text> in all the notes (case-insensitive; with --case, case-sensitive; with --regex, a regular expression).
Prints "note:line: text". No matches is not an error (exits with 0 and prints nothing).
exit codes: 0 ok · 1 failed · 2 empty search or invalid expression (nothing touched)
`

// RunSearch ejecuta `lazymark search …`.
func RunSearch(args []string, defaultNotesDir string) error {
	return RunSearchWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunSearchWithWriter ejecuta la búsqueda escribiendo la salida en w.
func RunSearchWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(searchUsageES, searchUsageEN)
	p := newParser("search", defaultNotesDir)
	var (
		regex, caseSensitive bool
		limit                int
	)
	p.fs.BoolVar(&regex, "regex", false, "")
	p.fs.BoolVar(&caseSensitive, "case", false, "")
	p.fs.IntVar(&limit, "limit", 0, "")
	if err := p.parse(args); err != nil {
		return err
	}
	if p.help {
		fmt.Fprint(w, help)
		return nil
	}
	if len(p.posArgs) == 0 {
		return &ops.Error{Code: ExitUsage, Err: errors.New("uso: lazymark search <texto> [--regex] [--case] [--limit <n>] [--json]")}
	}
	svc, err := p.service()
	if err != nil {
		return err
	}
	res, err := svc.Search(strings.Join(p.posArgs, " "), regex, caseSensitive, limit)
	if err != nil {
		return err
	}
	if p.json {
		return printJSON(w, res)
	}
	for _, m := range res.Matches {
		fmt.Fprint(w, plain(fmt.Sprintf("%s:%d: %s\n", m.Note, m.Line, m.Text)))
	}
	return nil
}

const dailyUsageES = `uso: lazymark daily [--json] [--dir <carpeta>]
Crea (o abre, si ya existe) la nota de hoy, journal/AAAA-MM-DD.md, con la plantilla templates/daily.md ({{date}}, {{time}} y {{title}}
se reemplazan); sin plantilla, con el título de la fecha. Imprime la ruta de la nota; no la modifica si ya existía.
códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos o una carpeta fuera de la carpeta de notas
`

const dailyUsageEN = `usage: lazymark daily [--json] [--dir <folder>]
Creates (or opens, if it already exists) today's note, journal/YYYY-MM-DD.md, from the template templates/daily.md ({{date}}, {{time}}
and {{title}} are replaced); without a template, with the date as its title. Prints the note's path; it does not modify an existing one.
exit codes: 0 ok · 1 failed · 2 invalid arguments or a folder outside the notes folder
`

// RunDaily ejecuta `lazymark daily`.
func RunDaily(args []string, defaultNotesDir string) error {
	return RunDailyWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunDailyWithWriter ejecuta `lazymark daily` escribiendo la salida en w.
func RunDailyWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(dailyUsageES, dailyUsageEN)
	p := newParser("daily", defaultNotesDir)
	if err := p.parse(args); err != nil {
		return err
	}
	if p.help {
		fmt.Fprint(w, help)
		return nil
	}
	if err := p.need(0, "lazymark daily [--json]"); err != nil {
		return err
	}
	svc, err := p.service()
	if err != nil {
		return err
	}
	d, err := svc.Daily()
	if err != nil {
		return err
	}
	if p.json {
		return printJSON(w, d)
	}
	fmt.Fprintf(w, "%s\n", d.Path)
	return nil
}
