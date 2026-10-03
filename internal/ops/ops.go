// Package ops son las operaciones de lazymark sobre las notas y el tablero Kanban sin interfaz: las comparten la
// línea de comandos (`lazymark note|task`) y el servidor MCP, así las dos hacen lo mismo con las mismas reglas de
// validación (rutas dentro de la carpeta de notas, ids estables, columnas válidas) y los mismos errores.
package ops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// Códigos de salida (los usa la CLI como código del proceso y se documentan en docs/cli.md).
const (
	ExitOK       = 0
	ExitFailure  = 1 // el comando era válido pero falló (lectura, escritura…)
	ExitUsage    = 2 // argumentos inválidos o una ruta que no es una nota de la carpeta: no se tocó nada
	ExitNotFound = 3 // la nota, la tarea o la columna pedida no existe
	ExitConflict = 4 // la nota cambió en disco mientras se escribía: no se escribió nada
)

// Error es un error con su código de salida.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func usage(format string, a ...any) error {
	return &Error{ExitUsage, fmt.Errorf(format, a...)}
}

// Code devuelve el código de salida de err: el de un *Error, 2 para una ruta fuera de la carpeta de notas, 3 para
// algo que no existe, 4 si la nota cambió y 1 para el resto. nil es 0.
func Code(err error) int {
	var e *Error
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &e):
		return e.Code
	case errors.Is(err, storage.ErrOutsideNotes):
		return ExitUsage
	case errors.Is(err, storage.ErrNoteChanged):
		return ExitConflict
	case errors.Is(err, storage.ErrTaskNotFound), errors.Is(err, os.ErrNotExist):
		return ExitNotFound
	}
	return ExitFailure
}

// Service agrupa la carpeta de notas y las columnas del tablero.
type Service struct {
	Store  *storage.Storage
	Cols   storage.Columns
	Titles []string
	Config []config.KanbanColumn
}

// New crea el servicio para una carpeta de notas ("" es la de por defecto), con las columnas de la config del
// usuario. No crea nada en disco.
func New(notesDir string) (*Service, error) {
	cfg, err := config.LoadReadOnly(notesDir)
	if err != nil {
		return nil, err
	}
	lang := string(i18n.CurrentLanguage())
	titles := make([]string, len(cfg.KanbanColumns))
	for i, c := range cfg.KanbanColumns {
		titles[i] = c.DisplayTitle(lang)
	}
	return &Service{Store: storage.New(cfg.NotesDir), Cols: storage.Columns(cfg.KanbanIDs()), Titles: titles, Config: cfg.KanbanColumns}, nil
}

// NoteDTO es una nota en la salida JSON (esquema estable, docs/cli.md).
type NoteDTO struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Tags       []string `json:"tags"`
	TasksCount int      `json:"tasks_count"`
	ModTime    string   `json:"mod_time"`
}

// NoteContentDTO es una nota con su contenido.
type NoteContentDTO struct {
	NoteDTO
	Content string `json:"content"`
}

func noteDTO(n storage.Note) NoteDTO {
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	return NoteDTO{ID: n.ID, Title: n.Title, Path: n.Path, Tags: tags, TasksCount: len(n.Tasks), ModTime: n.ModTime.Format(time.RFC3339)}
}

// TaskDTO es una tarea en la salida JSON (esquema estable, docs/cli.md).
type TaskDTO struct {
	ID        string `json:"id"`
	Text      string `json:"text"`   // el texto sin las etiquetas del tablero
	Column    string `json:"column"` // el id de su columna
	Done      bool   `json:"done"`
	Start     string `json:"start"`     // 🛫 AAAA-MM-DD, o ""
	Due       string `json:"due"`       // 📅 AAAA-MM-DD, o ""
	Completed string `json:"completed"` // ✅ AAAA-MM-DD, o ""
	Overdue   bool   `json:"overdue"`   // tiene vencimiento anterior a hoy y no está hecha
	Line      int    `json:"line"`
	Note      string `json:"note"` // la ruta de la nota, relativa a la carpeta de notas
	NoteTitle string `json:"note_title"`
	Path      string `json:"path"` // la ruta absoluta
}

func (s *Service) taskDTO(n storage.Note, id string, t storage.Task) TaskDTO {
	return TaskDTO{
		ID: id, Text: storage.CleanTaskText(t.Text), Column: s.Cols[s.Cols.Of(t)], Done: t.Done,
		Start: t.Dates.Start, Due: t.Dates.Due, Completed: t.Dates.Done, Overdue: storage.Overdue(t.Done, t.Dates.Due, storage.Today()),
		Line: t.Line, Note: strings.SplitN(id, "#", 2)[0], NoteTitle: n.Title, Path: n.Path,
	}
}

// ListNotes lista las notas.
func (s *Service) ListNotes() ([]NoteDTO, error) {
	notes, err := s.notes()
	if err != nil {
		return nil, err
	}
	out := []NoteDTO{}
	for _, n := range notes {
		out = append(out, noteDTO(n))
	}
	return out, nil
}

// notes lista las notas ordenadas por ruta: la salida de la CLI y del MCP es determinista (la de la TUI va por fecha).
func (s *Service) notes() ([]storage.Note, error) {
	notes, err := s.Store.ListNotes()
	if err != nil {
		return nil, err
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Path < notes[j].Path })
	return notes, nil
}

// ShowNote devuelve una nota (una ruta absoluta o relativa a la carpeta de notas, dentro de ella) con su contenido.
func (s *Service) ShowNote(path string) (NoteContentDTO, error) {
	real, err := s.Store.ResolveNote(path)
	if err != nil {
		return NoteContentDTO{}, err
	}
	data, err := os.ReadFile(real)
	if err != nil {
		return NoteContentDTO{}, err
	}
	notes, _ := s.Store.ListNotes()
	for _, n := range notes {
		if same(n.Path, real) {
			return NoteContentDTO{NoteDTO: noteDTO(n), Content: string(data)}, nil
		}
	}
	return NoteContentDTO{NoteDTO: NoteDTO{Path: real, Tags: []string{}}, Content: string(data)}, nil
}

func same(a, b string) bool {
	ra, err1 := resolve(a)
	rb, err2 := resolve(b)
	return err1 == nil && err2 == nil && ra == rb
}

// NewNote crea una nota con ese título en una subcarpeta (relativa a la carpeta de notas; "" es la raíz). Con empty
// solo lleva su título, sin la plantilla con fecha y primera tarea.
func (s *Service) NewNote(title, folder string, empty bool) (NoteDTO, error) {
	if strings.TrimSpace(title) == "" {
		return NoteDTO{}, usage("falta el título de la nota")
	}
	if strings.ContainsFunc(title, unicode.IsControl) { // un salto de línea en el título escribiría contenido arbitrario
		return NoteDTO{}, usage("el título no puede llevar saltos de línea ni caracteres de control")
	}
	if len([]rune(title)) > 200 {
		return NoteDTO{}, usage("el título es demasiado largo (máximo 200 caracteres)")
	}
	dir, err := s.Store.ResolveFolder(folder)
	if err != nil {
		return NoteDTO{}, err
	}
	body := ""
	if empty {
		body = "# " + title + "\n"
	}
	n, err := s.Store.CreateNoteInDirWithBody(dir, title, body)
	if err != nil {
		if errors.Is(err, storage.ErrNoteExists) || strings.Contains(err.Error(), "vacío") {
			return NoteDTO{}, usage("%v", err)
		}
		return NoteDTO{}, err
	}
	full, err := s.ShowNote(n.Path) // con sus etiquetas y su cantidad de tareas ya leídas del archivo
	if err != nil {
		return NoteDTO{}, err
	}
	return full.NoteDTO, nil
}

// TaskFilter filtra ListTasks.
type TaskFilter struct {
	PendingOnly bool   // sin las hechas
	Column      string // solo esa columna (id)
	Note        string // solo esa nota (ruta absoluta o relativa)
}

// ListTasks lista las tareas con su id y su columna.
func (s *Service) ListTasks(f TaskFilter) ([]TaskDTO, error) {
	col := -1
	if f.Column != "" {
		var err error
		if col, err = s.ColumnIndex(f.Column); err != nil {
			return nil, err
		}
	}
	notePath := ""
	if f.Note != "" {
		var err error
		if notePath, err = s.Store.ResolveNote(f.Note); err != nil {
			return nil, err
		}
	}
	notes, err := s.notes()
	if err != nil {
		return nil, err
	}
	out := []TaskDTO{}
	for _, n := range notes {
		if notePath != "" && !same(n.Path, notePath) {
			continue
		}
		ids := s.Store.TaskIDs(n)
		for i, t := range n.Tasks {
			if (f.PendingOnly && t.Done) || (col >= 0 && s.Cols.Of(t) != col) {
				continue
			}
			out = append(out, s.taskDTO(n, ids[i], t))
		}
	}
	return out, nil
}

// ColumnIndex busca una columna por su id o por su título visible (sin distinguir mayúsculas). Si no existe
// es un error de uso que dice cuáles hay.
func (s *Service) ColumnIndex(name string) (int, error) {
	if i := s.Cols.Index(name); i >= 0 {
		return i, nil
	}
	for i, c := range s.Config { // también por título, en cualquier idioma
		if strings.EqualFold(s.Titles[i], name) {
			return i, nil
		}
		for _, l := range i18n.Languages { // el título en cualquier idioma
			if strings.EqualFold(c.DisplayTitle(string(l)), name) {
				return i, nil
			}
		}
	}
	return 0, usage("la columna %q no existe (hay: %s)", name, strings.Join(s.Cols, ", "))
}

// MoveTask lleva la tarea con ese id a una columna. Escribe solo la línea de la tarea y no pisa una nota que cambió
// en disco desde que se leyó (ErrNoteChanged, código 4).
func (s *Service) MoveTask(id, column string) (TaskDTO, error) {
	target, err := s.ColumnIndex(column)
	if err != nil {
		return TaskDTO{}, err
	}
	return s.move(id, func(storage.Task) int { return target })
}

// ToggleTask marca la tarea como hecha (la lleva a la columna de hecho) o, si ya lo estaba, la devuelve a la primera
// columna: lo mismo que Espacio en el tablero.
func (s *Service) ToggleTask(id string) (TaskDTO, error) {
	return s.move(id, func(t storage.Task) int {
		if s.Cols.Of(t) == s.Cols.DoneIndex() {
			return 0
		}
		return s.Cols.DoneIndex()
	})
}

func (s *Service) move(id string, target func(storage.Task) int) (TaskDTO, error) {
	n, t, err := s.Store.FindTask(id)
	if err != nil {
		return TaskDTO{}, err
	}
	if err := s.Store.MoveTask(n.Path, t.Line, s.Cols, target(t), n.ModTime); err != nil {
		return TaskDTO{}, err
	}
	return s.afterWrite(n.Path, t.Line)
}

// SetDate pone la fecha de inicio (field "start") o de vencimiento ("due") de la tarea con ese id, o la quita (date ""
// o "none"). La fecha debe ser AAAA-MM-DD y existir en el calendario; si no, es un error de uso y no se toca nada. La
// fecha de completada no se pone a mano: la agrega y la quita el movimiento a la columna de hecho. Escribe solo la línea
// de la tarea y no pisa una nota que cambió en disco (código 4).
func (s *Service) SetDate(id, field, date string) (TaskDTO, error) {
	var f storage.DateField
	switch strings.ToLower(field) {
	case "start":
		f = storage.DateStart
	case "due":
		f = storage.DateDue
	default:
		return TaskDTO{}, usage("el campo de fecha %q no existe (start o due)", field)
	}
	if strings.EqualFold(date, "none") {
		date = ""
	}
	if date != "" && !storage.ValidDate(date) {
		return TaskDTO{}, usage("%q no es una fecha válida: se espera AAAA-MM-DD (o none para quitarla)", date)
	}
	n, t, err := s.Store.FindTask(id)
	if err != nil {
		return TaskDTO{}, err
	}
	if err := s.Store.SetTaskDate(n.Path, t.Line, f, date, n.ModTime); err != nil {
		return TaskDTO{}, err
	}
	return s.afterWrite(n.Path, t.Line)
}

// afterWrite devuelve la tarea de la línea line de la nota path, con su id actual, ya escrita. El id sale de leer la nota de nuevo
// (no es el de antes): si el cambio alteró el texto de la tarea (por ejemplo quitó un marcador que contaba para su huella) el id
// es otro, y buscar el viejo daría "no existe" aunque la escritura ya se hizo.
func (s *Service) afterWrite(path string, line int) (TaskDTO, error) {
	notes, err := s.Store.ListNotes()
	if err != nil {
		return TaskDTO{}, err
	}
	for _, n := range notes {
		if !same(n.Path, path) {
			continue
		}
		ids := s.Store.TaskIDs(n)
		for i, t := range n.Tasks {
			if t.Line == line {
				return s.taskDTO(n, ids[i], t), nil
			}
		}
	}
	return TaskDTO{}, fmt.Errorf("%w: la tarea de la línea %d de %s ya no está tras escribirla", storage.ErrTaskNotFound, line, path)
}

// IDByLine devuelve el id de la tarea que está en esa línea de la nota (para el formato anterior --path --line).
func (s *Service) IDByLine(path string, line int) (string, error) {
	real, err := s.Store.ResolveNote(path)
	if err != nil {
		return "", err
	}
	notes, err := s.Store.ListNotes()
	if err != nil {
		return "", err
	}
	for _, n := range notes {
		if !same(n.Path, real) {
			continue
		}
		for i, t := range n.Tasks {
			if t.Line == line {
				return s.Store.TaskIDs(n)[i], nil
			}
		}
	}
	return "", fmt.Errorf("%w: no hay una tarea en la línea %d de %s", storage.ErrTaskNotFound, line, path)
}

// ColumnDTO y BoardDTO son el tablero en la salida JSON.
type ColumnDTO struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	Cards []TaskDTO `json:"cards"`
}

type BoardDTO struct {
	Columns []ColumnDTO `json:"columns"`
}

// Board devuelve el tablero: las columnas con sus tarjetas, en el orden de cada nota.
func (s *Service) Board() (BoardDTO, error) {
	tasks, err := s.ListTasks(TaskFilter{})
	if err != nil {
		return BoardDTO{}, err
	}
	b := BoardDTO{Columns: make([]ColumnDTO, len(s.Cols))}
	for i, id := range s.Cols {
		b.Columns[i] = ColumnDTO{ID: id, Title: s.Titles[i], Cards: []TaskDTO{}}
	}
	for _, t := range tasks {
		i := s.Cols.Index(t.Column)
		b.Columns[i].Cards = append(b.Columns[i].Cards, t)
	}
	return b, nil
}

// ColumnIDs devuelve los ids de las columnas, para las descripciones de ayuda.
func (s *Service) ColumnIDs() []string { return append([]string(nil), s.Cols...) }

// resolve devuelve la ruta canónica de un archivo existente (con los symlinks resueltos).
func resolve(p string) (string, error) { return filepath.EvalSymlinks(p) }
