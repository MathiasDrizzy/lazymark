package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// TaskJSON representa una tarea en formato JSON estructurado para CLI / MCP
type TaskJSON struct {
	Line      int    `json:"line"`
	Text      string `json:"text"`
	CleanText string `json:"clean_text"`
	Done      bool   `json:"done"`
	Stage     string `json:"stage"` // "todo", "doing", "done"
	NoteTitle string `json:"note_title"`
	NotePath  string `json:"note_path"`
}

// NoteJSON representa una nota en formato JSON estructurado
type NoteJSON struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Tags       []string `json:"tags"`
	TasksCount int      `json:"tasks_count"`
	ModTime    string   `json:"mod_time"`
}

func stageToString(s storage.TaskStage) string {
	switch s {
	case storage.StageDoing:
		return "doing"
	case storage.StageDone:
		return "done"
	default:
		return "todo"
	}
}

// RunTask ejecuta subcomandos relacionados con tareas (list, toggle)
func RunTask(args []string, defaultNotesDir string) error {
	return RunTaskWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunTaskWithWriter ejecuta subcomandos de tareas escribiendo la salida en el writer indicado
func RunTaskWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	if len(args) == 0 {
		return fmt.Errorf("subcomando de tarea requerido: list, toggle")
	}

	action := args[0]
	fs := flag.NewFlagSet("task "+action, flag.ContinueOnError)
	fs.SetOutput(w)

	var (
		jsonOutput  bool
		pendingOnly bool
		notesDir    string
		notePath    string
		lineNum     int
	)

	fs.BoolVar(&jsonOutput, "json", false, "Salida en formato JSON estructurado")
	fs.StringVar(&notesDir, "dir", defaultNotesDir, "Directorio de notas Markdown")

	switch action {
	case "list":
		fs.BoolVar(&pendingOnly, "pending", false, "Solo mostrar tareas pendientes")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		if notesDir == "" {
			notesDir = config.DefaultNotesDir()
		}

		st := storage.New(notesDir)
		notes, err := st.ListNotes()
		if err != nil {
			return fmt.Errorf("error al listar notas: %w", err)
		}

		filter := views.TaskFilterAll
		if pendingOnly {
			filter = views.TaskFilterPending
		}

		flatTasks := views.CollectTasks(notes, filter)

		if jsonOutput {
			var out []TaskJSON
			for _, t := range flatTasks {
				stage := storage.GetTaskStage(t.Task)
				out = append(out, TaskJSON{
					Line:      t.Line,
					Text:      t.Text,
					CleanText: storage.CleanTaskText(t.Text),
					Done:      t.Done,
					Stage:     stageToString(stage),
					NoteTitle: t.NoteTitle,
					NotePath:  t.NotePath,
				})
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		for _, t := range flatTasks {
			check := "[ ]"
			if t.Done {
				check = "[x]"
			}
			fmt.Fprintf(w, "%s %s:%d - %s\n", check, t.NotePath, t.Line, t.Text)
		}
		return nil

	case "toggle":
		fs.StringVar(&notePath, "path", "", "Ruta de la nota")
		fs.IntVar(&lineNum, "line", 0, "Número de línea de la tarea")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		if notePath == "" || lineNum <= 0 {
			return fmt.Errorf("parámetros --path <ruta> y --line <línea> son obligatorios")
		}

		if notesDir == "" {
			notesDir = config.DefaultNotesDir()
		}

		st := storage.New(notesDir)
		newStatus, err := st.ToggleTask(notePath, lineNum)
		if err != nil {
			return fmt.Errorf("error al alternar tarea: %w", err)
		}

		if jsonOutput {
			res := map[string]interface{}{
				"success": true,
				"path":    notePath,
				"line":    lineNum,
				"done":    newStatus,
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}

		statusStr := "pendiente"
		if newStatus {
			statusStr = "completada"
		}
		fmt.Fprintf(w, "Tarea en %s:%d alternada a %s\n", notePath, lineNum, statusStr)
		return nil

	default:
		return fmt.Errorf("subcomando de tarea desconocido: '%s'. Usa 'list' o 'toggle'", action)
	}
}

// RunNote ejecuta subcomandos relacionados con notas (list, get)
func RunNote(args []string, defaultNotesDir string) error {
	return RunNoteWithWriter(os.Stdout, args, defaultNotesDir)
}

// RunNoteWithWriter ejecuta subcomandos de notas escribiendo en el writer indicado
func RunNoteWithWriter(w io.Writer, args []string, defaultNotesDir string) error {
	if len(args) == 0 {
		return fmt.Errorf("subcomando de nota requerido: list, get")
	}

	action := args[0]
	fs := flag.NewFlagSet("note "+action, flag.ContinueOnError)
	fs.SetOutput(w)

	var (
		jsonOutput bool
		notesDir   string
	)

	fs.BoolVar(&jsonOutput, "json", false, "Salida en formato JSON estructurado")
	fs.StringVar(&notesDir, "dir", defaultNotesDir, "Directorio de notas Markdown")

	switch action {
	case "list":
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		if notesDir == "" {
			notesDir = config.DefaultNotesDir()
		}

		st := storage.New(notesDir)
		notes, err := st.ListNotes()
		if err != nil {
			return fmt.Errorf("error al listar notas: %w", err)
		}

		if jsonOutput {
			var out []NoteJSON
			for _, n := range notes {
				out = append(out, NoteJSON{
					ID:         n.ID,
					Title:      n.Title,
					Path:       n.Path,
					Tags:       n.Tags,
					TasksCount: len(n.Tasks),
					ModTime:    n.ModTime.Format(time.RFC3339),
				})
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		for _, n := range notes {
			fmt.Fprintf(w, "%s (%s)\n", n.Title, n.Path)
		}
		return nil

	case "get":
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		remaining := fs.Args()
		if len(remaining) == 0 {
			return fmt.Errorf("se requiere la ruta de la nota: lazymark note get <path>")
		}

		notePath := remaining[0]
		data, err := os.ReadFile(notePath)
		if err != nil {
			return fmt.Errorf("error al leer nota en '%s': %w", notePath, err)
		}

		if jsonOutput {
			res := map[string]interface{}{
				"path":    notePath,
				"content": string(data),
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}

		_, err = w.Write(data)
		return err

	default:
		return fmt.Errorf("subcomando de nota desconocido: '%s'. Usa 'list' o 'get'", action)
	}
}

// Helper para parsear enteros seguros
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
