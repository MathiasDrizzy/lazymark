package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
	"github.com/charmbracelet/x/term"
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
		return &ops.Error{Code: ExitUsage, Err: i18n.Errorf("uso: %s", "usage: %s", usage)}
	}
	return nil
}

func (p *parser) service() (*ops.Service, error) {
	dir := p.dir
	if dir == "" {
		dir = config.DefaultNotesDir()
	}
	svc, err := ops.New(dir)
	if err != nil {
		return nil, err
	}
	views.DateGlyphs = svc.UserConfig().DateGlyphArray() // los glifos propios de las fechas (date_glyphs) también valen en la salida de texto
	svc.Notice = func(msg string) { fmt.Fprintln(Stderr, plain(i18n.T("aviso: ", "note: ")+msg)) }
	return svc, nil
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
códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos, ruta fuera de la carpeta de notas o plantilla no válida · 3 no existe
`

const noteUsageEN = `usage: lazymark note list [--json] [--dir <folder>]
       lazymark note show <path> [--json] [--dir <folder>]
       lazymark note new  <title> [--folder <subfolder>] [--empty] [--template <name>] [--json] [--dir <folder>]
exit codes: 0 ok · 1 failed · 2 invalid arguments, a path outside the notes folder or an invalid template · 3 not found
`

// RunTask ejecuta `lazymark task …` (list, toggle, move).
func RunTask(args []string, defaultNotesDir string) error {
	return RunTaskWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunTaskWithWriter ejecuta los subcomandos de tareas escribiendo la salida en w.
func RunTaskWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(taskUsageES, taskUsageEN)
	if len(args) == 0 {
		return &ops.Error{Code: ExitUsage, Err: errors.New(i18n.E("falta el subcomando (list, toggle, move)", "missing subcommand (list, toggle, move)") + "\n" + help)}
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
		return &ops.Error{Code: ExitUsage, Err: i18n.Errorf("subcomando de tarea desconocido: %q (list, toggle, move, due, start)", "unknown task subcommand: %q (list, toggle, move, due, start)", action)}
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
		colored := colorEnabled(w, svc)
		for _, t := range tasks {
			mark := "[ ]"
			if t.Done {
				mark = "[x]"
			}
			head := plain(fmt.Sprintf("%s %s  %s  (%s)", mark, t.ID, t.Text, t.Column))
			if colored {
				fmt.Fprintln(w, head+datesSuffixColored(t, storage.Today()))
			} else {
				fmt.Fprintln(w, head+plain(datesSuffix(t)))
			}
		}
		return nil

	case "toggle":
		var id string
		if path != "" || line != 0 { // forma anterior
			if path == "" || line <= 0 || len(p.posArgs) != 0 {
				return &ops.Error{Code: ExitUsage, Err: i18n.NewError("uso: lazymark task toggle --path <nota> --line <n>", "usage: lazymark task toggle --path <note> --line <n>")}
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

func finishMove(w io.Writer, p *parser, svc *ops.Service, do func() (ops.TaskDTO, error)) error {
	t, err := do()
	if err != nil {
		return err
	}
	if svc.UserConfig().DateWarnings && t.Start != "" && t.Due != "" && t.Start > t.Due { // (con date_warnings activo) el inicio después del vencimiento: se avisa por stderr, no bloquea
		fmt.Fprintln(Stderr, plain(fmt.Sprintf(i18n.T("aviso: el inicio (%s) es posterior al vencimiento (%s)", "warning: the start (%s) is after the due date (%s)"), t.Start, t.Due)))
	}
	if p.json {
		return printJSON(w, t)
	}
	head := plain(fmt.Sprintf("%s → %s", t.ID, t.Column))
	if colorEnabled(w, svc) { // en una terminal las fechas llevan el color de su estado, igual que en `task list`
		fmt.Fprintln(w, head+datesSuffixColored(t, storage.Today()))
		return nil
	}
	fmt.Fprintln(w, head+plain(datesSuffix(t)))
	return nil
}

// datesSuffix muestra las fechas de una tarea en la salida de texto con símbolos (▸ inicio, ◷ vencimiento, ✓ completada), no con
// los emojis del archivo: "  ▸ 2026-05-01 ◷ 2026-05-10 (vencida)". `--json` lleva las fechas en sus campos.
func datesSuffix(t ops.TaskDTO) string {
	var parts []string
	for _, d := range []struct {
		g    string
		f    storage.DateField
		date string
	}{{"▸", storage.DateStart, t.Start}, {"◑", storage.DateScheduled, t.Scheduled}, {"◷", storage.DateDue, t.Due}, {"✓", storage.DateDone, t.Completed}, {"+", storage.DateCreated, t.Created}} {
		if d.date != "" {
			parts = append(parts, cliGlyph(d.f, d.g)+" "+d.date)
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
		return &ops.Error{Code: ExitUsage, Err: errors.New(i18n.E("falta el subcomando (list, show, new)", "missing subcommand (list, show, new)") + "\n" + help)}
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
		return &ops.Error{Code: ExitUsage, Err: i18n.Errorf("subcomando de nota desconocido: %q (list, show, new)", "unknown note subcommand: %q (list, show, new)", action)}
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
			return &ops.Error{Code: ExitUsage, Err: i18n.NewError("uso: lazymark note new <título> [--folder <subcarpeta>] [--empty] [--template <nombre>]", "usage: lazymark note new <title> [--folder <subfolder>] [--empty] [--template <name>]")}
		}
		svc, err := p.service()
		if err != nil {
			return err
		}
		n, err := svc.NewNoteFromTemplate(strings.Join(p.posArgs, " "), folder, template, empty)
		if err != nil {
			return err
		}
		warn(n.Warnings)
		if p.json {
			return printJSON(w, n)
		}
		fmt.Fprintf(w, "%s\n", n.Path)
		return nil
	}
}

// Stderr es donde van los avisos de la CLI (variables desconocidas de una plantilla); los tests lo reemplazan.
var Stderr io.Writer = os.Stderr

// warn escribe un aviso por cada variable desconocida de la plantilla; la nota se creó igual, con la variable tal cual.
func warn(unknown []string) {
	for _, v := range unknown {
		fmt.Fprintf(Stderr, "%s\n", plain(i18n.T("aviso: variable desconocida: ", "warning: unknown variable: ")+v))
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
		return &ops.Error{Code: ExitUsage, Err: i18n.NewError("uso: lazymark search <texto> [--regex] [--case] [--limit <n>] [--json]", "usage: lazymark search <text> [--regex] [--case] [--limit <n>] [--json]")}
	}
	svc, err := p.service()
	if err != nil {
		return err
	}
	res, err := svc.Search(strings.Join(p.posArgs, " "), regex, caseSensitive, limit)
	if err != nil {
		return err
	}
	if res.TimedOut {
		fmt.Fprintln(Stderr, i18n.T("aviso: búsqueda cortada: tardó más de 10 s (puede haber más coincidencias)", "warning: search cut short: it took more than 10 s (there may be more matches)"))
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
códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos, una carpeta fuera de la carpeta de notas o plantilla no válida
`

const dailyUsageEN = `usage: lazymark daily [--json] [--dir <folder>]
Creates (or opens, if it already exists) today's note, journal/YYYY-MM-DD.md, from the template templates/daily.md ({{date}}, {{time}}
and {{title}} are replaced); without a template, with the date as its title. Prints the note's path; it does not modify an existing one.
exit codes: 0 ok · 1 failed · 2 invalid arguments, a folder outside the notes folder or an invalid template
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
	warn(d.Warnings)
	if p.json {
		return printJSON(w, d)
	}
	fmt.Fprintf(w, "%s\n", d.Path)
	return nil
}

const datesUsageES = `uso: lazymark dates migrate --to dataview|emoji [--dry-run] [--json] [--dir <carpeta>]
Pasa las fechas de todas las tareas al formato elegido. Los dos son formatos de Obsidian Tasks: elige el de Ajustes → Tasks → Task format.
  --to dataview   [start:: 2026-05-01] [due:: 2026-05-10]   (Task format: Dataview)
  --to emoji      🛫 2026-05-01 📅 2026-05-10               (Task format: Tasks, el de Obsidian por defecto)
Sin --to dice cuántas tareas hay en cada formato y qué comando probar. Nunca es automático.
Solo toca líneas de tarea (no párrafos ni bloques de código), conserva el resto de la línea y escribe cada nota de forma atómica.
Con --dry-run muestra el cambio sin escribir nada. Es idempotente: una segunda corrida no cambia nada.
códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos · 4 una nota cambió mientras se migraba (esa no se escribió)
`

const datesUsageEN = `usage: lazymark dates migrate --to dataview|emoji [--dry-run] [--json] [--dir <folder>]
Moves the dates of every task to the chosen format. Both are Obsidian Tasks formats: pick the one in Settings → Tasks → Task format.
  --to dataview   [start:: 2026-05-01] [due:: 2026-05-10]   (Task format: Dataview)
  --to emoji      🛫 2026-05-01 📅 2026-05-10               (Task format: Tasks, Obsidian's default)
Without --to it says how many tasks use each format and which command to try. It is never automatic.
It only touches task lines (not paragraphs or code blocks), keeps the rest of the line and writes each note atomically.
With --dry-run it shows the change without writing anything. It is idempotent: a second run changes nothing.
exit codes: 0 ok · 1 failed · 2 invalid arguments · 4 a note changed while migrating (that one was not written)
`

// RunDates ejecuta `lazymark dates …` (migrate).
func RunDates(args []string, defaultNotesDir string) error {
	return RunDatesWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunDatesWithWriter ejecuta `lazymark dates migrate` escribiendo la salida en w.
func RunDatesWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(datesUsageES, datesUsageEN)
	if len(args) == 0 {
		return &ops.Error{Code: ExitUsage, Err: errors.New(i18n.T("falta el subcomando (migrate)", "missing subcommand (migrate)") + "\n" + help)}
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(w, help)
		return nil
	}
	if args[0] != "migrate" {
		return &ops.Error{Code: ExitUsage, Err: fmt.Errorf(i18n.T("subcomando desconocido: %q (migrate)", "unknown subcommand: %q (migrate)"), args[0])}
	}
	p := newParser("dates migrate", defaultNotesDir)
	var to string
	var dry bool
	p.fs.StringVar(&to, "to", "", "")
	p.fs.BoolVar(&dry, "dry-run", false, "")
	if err := p.parse(args[1:]); err != nil {
		return err
	}
	if p.help {
		fmt.Fprint(w, help)
		return nil
	}
	if err := p.need(0, "lazymark dates migrate --to dataview|emoji [--dry-run]"); err != nil {
		return err
	}
	if to != "" && to != "dataview" && to != "emoji" {
		return &ops.Error{Code: ExitUsage, Err: fmt.Errorf(i18n.T("--to debe ser dataview o emoji (no %q)", "--to must be dataview or emoji (not %q)"), to)}
	}
	svc, err := p.service()
	if err != nil {
		return err
	}
	if to == "" { // sin --to: se dice qué formato usa hoy el vault y qué comando probar
		return &ops.Error{Code: ExitUsage, Err: errors.New(missingToMessage(svc.DateFormatCounts()))}
	}
	res, err := svc.MigrateDates(to, dry)
	if err != nil {
		if res.Lines > 0 { // se cortó a mitad (si falló la primera nota no se migró nada: solo el error, sin un "0 línea(s)" que parezca éxito): se informa qué notas ya se migraron (y el código de salida no es 0)
			if p.json {
				_ = printJSON(w, res)
			} else {
				fmt.Fprintln(w, plain(fmt.Sprintf(i18n.T("Migración interrumpida: ya se escribieron %d línea(s) en %d nota(s):", "Migration interrupted: %d line(s) in %d note(s) were already written:"), res.Lines, res.Notes)))
				seen := map[string]bool{}
				for _, c := range res.Changes {
					if !seen[c.Note] {
						seen[c.Note] = true
						fmt.Fprintln(w, "  "+plain(c.Note))
					}
				}
			}
		}
		return err
	}
	if p.json {
		return printJSON(w, res)
	}
	for _, c := range res.Changes {
		fmt.Fprintf(w, "%s\n", plain(fmt.Sprintf("%s:%d", c.Note, c.Line)))
		fmt.Fprintf(w, "- %s\n+ %s\n", plain(c.Before), plain(c.After))
	}
	msg := i18n.T("%d línea(s) en %d nota(s) pasan a %s", "%d line(s) in %d note(s) move to %s")
	if dry {
		msg += " (" + i18n.T("simulación: no se escribió nada", "dry run: nothing was written") + ")"
	}
	fmt.Fprintf(w, msg+"\n", res.Lines, res.Notes, to)
	return nil
}

// missingToMessage es el error de `dates migrate` sin --to: dice qué formato de fechas usa hoy el vault (cuántas tareas en emoji y cuántas en Dataview) y sugiere
// el comando para pasar al otro formato con --dry-run, que no escribe nada.
func missingToMessage(emoji, dataview int) string {
	head := i18n.T("falta --to (dataview o emoji).", "missing --to (dataview or emoji).")
	counts := fmt.Sprintf(i18n.T("Hoy el vault tiene %d tarea(s) con emojis y %d tarea(s) con Dataview.", "This vault has %d task(s) with emoji and %d task(s) with Dataview."), emoji, dataview)
	var hint string
	switch {
	case emoji == 0 && dataview == 0:
		return head + "\n" + i18n.T("Ninguna tarea tiene fechas: no hay nada que migrar.", "No task has dates: there is nothing to migrate.")
	case emoji > 0 && dataview == 0:
		hint = i18n.T("Si en Obsidian Tasks tu Task format es Dataview, prueba:", "If your Obsidian Tasks Task format is Dataview, try:") + "\n  lazymark dates migrate --to dataview --dry-run"
	case dataview > 0 && emoji == 0:
		hint = i18n.T("Si en Obsidian Tasks tu Task format es Tasks (emojis), prueba:", "If your Obsidian Tasks Task format is Tasks (emoji), try:") + "\n  lazymark dates migrate --to emoji --dry-run"
	default:
		hint = i18n.T("Están mezclados. Elige el que usa tu Task format de Obsidian Tasks:", "They are mixed. Pick the one your Obsidian Tasks Task format uses:") + "\n  lazymark dates migrate --to dataview --dry-run   (Task format: Dataview)\n  lazymark dates migrate --to emoji --dry-run      (Task format: Tasks)"
	}
	return head + "\n" + counts + "\n" + hint + "\n" + i18n.T("(--dry-run solo muestra el cambio; quítalo para escribir)", "(--dry-run only shows the change; drop it to write)")
}

// isTerminal dice si el descriptor es una terminal (una variable para que las pruebas la simulen).
var isTerminal = term.IsTerminal

// colorEnabled dice si la salida de texto lleva color: solo en una terminal (no en una tubería ni redirigida a un archivo), sin NO_COLOR y con date_colors
// activado en la configuración; además aplica el tema configurado para que los colores sean los de su paleta. `--json` nunca llega aquí.
func colorEnabled(w io.Writer, svc *ops.Service) bool {
	f, ok := w.(*os.File)
	if !ok || !isTerminal(f.Fd()) || os.Getenv("NO_COLOR") != "" {
		return false
	}
	cfg := svc.UserConfig()
	if cfg == nil || !cfg.DateColors {
		return false
	}
	theme.ApplyThemeByName(cfg.Theme)
	views.DateColors = views.DateColorSettings{Enabled: true, SoonDays: cfg.DueSoonDays, Names: cfg.DateColorNames}
	return true
}

// datesSuffixColored es datesSuffix con cada fecha en el color de su estado (vencida, por vencer, en fecha…), el mismo de la TUI. Solo para una terminal.
func datesSuffixColored(t ops.TaskDTO, today string) string {
	var parts []string
	for _, d := range []struct {
		g    string
		f    storage.DateField
		date string
	}{{"▸", storage.DateStart, t.Start}, {"◑", storage.DateScheduled, t.Scheduled}, {"◷", storage.DateDue, t.Due}, {"✓", storage.DateDone, t.Completed}, {"+", storage.DateCreated, t.Created}} {
		if d.date != "" {
			parts = append(parts, views.DateStyleFor(d.f, d.date, t.Done, today).Render(cliGlyph(d.f, d.g)+" "+d.date))
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

const kanbanUsageES = `uso: lazymark kanban retag --from <prefijo> --to <prefijo> [--dry-run] [--json] [--dir <carpeta>]
Cambia el prefijo de la etiqueta de columna del tablero en todas las tareas: #kb/doing pasa a #<nuevo>/doing. Úsalo después de cambiar kanban_tag en la configuración, porque cambiar el ajuste NO migra las notas que ya tienen otro prefijo. Solo toca líneas de tarea (no párrafos, código en línea ni enlaces), escribe cada nota de forma
atómica y es idempotente: una segunda corrida no cambia nada. Con --dry-run muestra el cambio sin escribir nada. códigos de salida: 0 ok · 1 falló · 2 argumentos inválidos · 4 una nota cambió mientras se cambiaba (esa no se escribió)

`

const kanbanUsageEN = `usage: lazymark kanban retag --from <prefix> --to <prefix> [--dry-run] [--json] [--dir <folder>]
Changes the prefix of the board's column tag in every task: #kb/doing becomes #<new>/doing. Use it after changing kanban_tag in the configuration, because changing the setting does NOT migrate the notes that already have another prefix. It only touches task lines (not paragraphs, inline code or links), writes each note atomically and is
idempotent: a second run changes nothing. With --dry-run it shows the change without writing anything. exit codes: 0 ok · 1 failed · 2 invalid arguments · 4 a note changed while it was being changed (that one was not written)

`

// RunKanban ejecuta `lazymark kanban …` (retag).
func RunKanban(args []string, defaultNotesDir string) error {
	return RunKanbanWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunKanbanWithWriter ejecuta `lazymark kanban retag` escribiendo la salida en w.
func RunKanbanWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	help := usageText(kanbanUsageES, kanbanUsageEN)
	if len(args) == 0 {
		return &ops.Error{Code: ExitUsage, Err: errors.New(i18n.E("falta el subcomando (retag)", "missing subcommand (retag)") + "\n" + help)}
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(w, help)
		return nil
	}
	if args[0] != "retag" {
		return &ops.Error{Code: ExitUsage, Err: fmt.Errorf(i18n.E("subcomando desconocido: %q (retag)", "unknown subcommand: %q (retag)"), args[0])}
	}
	p := newParser("kanban retag", defaultNotesDir)
	var from, to string
	var dry bool
	p.fs.StringVar(&from, "from", "", "")
	p.fs.StringVar(&to, "to", "", "")
	p.fs.BoolVar(&dry, "dry-run", false, "")
	if err := p.parse(args[1:]); err != nil {
		return err
	}
	if p.help {
		fmt.Fprint(w, help)
		return nil
	}
	if err := p.need(0, "lazymark kanban retag --from <prefix> --to <prefix> [--dry-run]"); err != nil {
		return err
	}
	svc, err := p.service()
	if err != nil {
		return err
	}
	res, err := svc.RetagKanban(from, to, dry)
	if err != nil {
		if res.Lines > 0 {
			if p.json {
				_ = printJSON(w, res)
			} else {
				fmt.Fprintln(w, plain(fmt.Sprintf(i18n.E("Cambio interrumpido: ya se escribieron %d línea(s) en %d nota(s):", "Change interrupted: %d line(s) in %d note(s) were already written:"), res.Lines, res.Notes)))
				seen := map[string]bool{}
				for _, c := range res.Changes {
					if !seen[c.Note] {
						seen[c.Note] = true
						fmt.Fprintln(w, "  "+plain(c.Note))
					}
				}
			}
		}
		return err
	}
	if p.json {
		return printJSON(w, res)
	}
	for _, c := range res.Changes {
		fmt.Fprintf(w, "%s\n", plain(fmt.Sprintf("%s:%d", c.Note, c.Line)))
		fmt.Fprintf(w, "- %s\n+ %s\n", plain(c.Before), plain(c.After))
	}
	msg := i18n.E("%d línea(s) en %d nota(s): #%s/ pasa a #%s/", "%d line(s) in %d note(s): #%s/ becomes #%s/")
	if dry {
		msg += " (" + i18n.E("simulación: no se escribió nada", "dry run: nothing was written") + ")"
	}
	fmt.Fprintf(w, msg+"\n", res.Lines, res.Notes, from, to)
	if !dry && res.Lines > 0 {
		fmt.Fprintln(w, i18n.E("Recuerda poner kanban_tag = \""+to+"\" en la configuración (Ajustes) para que el tablero lea el nuevo prefijo.", "Remember to set kanban_tag = \""+to+"\" in the configuration (Settings) so the board reads the new prefix."))
	}
	return nil
}

// cliGlyph es el glifo de un campo de fecha en la salida de texto: el propio de la configuración (date_glyphs) o el símbolo de texto de siempre.
func cliGlyph(f storage.DateField, def string) string {
	if g := views.DateGlyphs[f]; g != "" {
		return g
	}
	return def
}
