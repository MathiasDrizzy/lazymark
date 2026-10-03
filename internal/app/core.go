package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/links"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// EditorFinishedMsg se emite cuando el editor externo termina.
type EditorFinishedMsg struct {
	Path string
	Err  error
}

// core es el estado compartido por el modelo raíz, los paneles y los popups:
// datos derivados de las notas, configuración, mensaje de estado y la pila de
// popups. Los paneles guardan solo su propio estado de interfaz.
type core struct {
	cfg   *config.Config
	store *storage.Storage
	kitty *image.Client
	clip  *clipboard.Saver
	keys  Keymap

	notes       []storage.Note
	links       *links.Index // para resolver wikilinks y calcular backlinks (se rehace en cada reload)
	gen         int          // cambia en cada reload: invalida lo que dependa del conjunto de notas
	tags        []views.TagInfo
	tasks       []views.FlatTask
	taskFilter  views.TaskFilter
	board       views.KanbanBoard
	trashCount  int
	trashWarned int // cuántas entradas inválidas de la papelera ya se avisaron

	status string
	popups []popup

	// editing es true mientras el editor externo tiene la terminal: las
	// imágenes no se transmiten hasta que vuelva.
	editing bool
}

// newStore crea el almacén de notas de una carpeta.
func newStore(dir string) *storage.Storage { return storage.New(dir) }

// reload relee las notas del disco y recalcula todo lo derivado.
func (c *core) reload() {
	if notes, err := c.store.ListNotes(); err == nil {
		c.notes = notes
	}
	c.links = links.NewIndex(c.store.BaseDir, c.notes)
	c.gen++
	c.tags = views.CollectTags(c.notes)
	c.tasks = views.CollectTasks(c.scopedNotes(), c.taskFilter)
	c.board = views.CollectKanban(c.notes, c.cols(), c.columnTitles())
	c.trashCount = c.store.CountTrash()
	if issues := c.store.TrashIssues(); len(issues) > 0 && len(issues) != c.trashWarned {
		c.trashWarned = len(issues) // se avisa una vez por cambio, no en cada recarga
		c.setStatus("%s", i18n.T("Papelera: se ignoraron entradas inválidas de trash.json", "Trash: invalid trash.json entries were ignored"))
	}
}

// scopedNotes aplica el alcance de tareas de la config: "all", "tag:<tag>" o
// "folder:<ruta relativa>".
func (c *core) scopedNotes() []storage.Note {
	scope := c.cfg.TaskScope
	switch {
	case strings.HasPrefix(scope, "tag:"):
		return views.NotesForTag(c.notes, strings.TrimPrefix(scope, "tag:"))
	case strings.HasPrefix(scope, "folder:"):
		dir := filepath.Join(c.store.BaseDir, strings.TrimPrefix(scope, "folder:")) + string(filepath.Separator)
		var out []storage.Note
		for _, n := range c.notes {
			if strings.HasPrefix(n.Path, dir) {
				out = append(out, n)
			}
		}
		return out
	}
	return c.notes
}

func (c *core) setStatus(format string, a ...any) {
	c.status = fmt.Sprintf(format, a...)
}

func (c *core) save() { _ = c.cfg.Save() }

func (c *core) push(p popup) { c.popups = append(c.popups, p) }

func (c *core) pop() {
	if n := len(c.popups); n > 0 {
		closePopup(c.popups[n-1])
		c.popups = c.popups[:n-1]
	}
}

// closePopup avisa al popup que se va (si le importa: la búsqueda cancela su trabajo).
func closePopup(p popup) {
	if cl, ok := p.(interface{ close() }); ok {
		cl.close()
	}
}

func (c *core) top() popup {
	if n := len(c.popups); n > 0 {
		return c.popups[n-1]
	}
	return nil
}

// confirm abre una confirmación, salvo que el setting de confirmación esté
// desactivado: entonces ejecuta la acción directamente.
func (c *core) confirm(title, msg string, always bool, onYes func() tea.Cmd) tea.Cmd {
	if !always && !c.cfg.ConfirmDelete {
		return onYes()
	}
	c.push(newConfirmPopup(title, msg, onYes))
	return nil
}

// openEditor suspende la TUI y abre path en el editor configurado.
func (c *core) openEditor(path string, line int) tea.Cmd {
	cmd := c.editorCommand(path, line)
	if cmd == nil {
		return nil
	}
	c.editing = true
	c.kitty.Reset() // el editor se abre sin imágenes en la terminal
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return EditorFinishedMsg{Path: path, Err: err}
	})
}

// editorCommand arma el comando del editor para path. El editor corre con
// LAZYMARK_NOTE (la nota que abre) y con la carpeta de este ejecutable en el PATH, para
// que sus plugins puedan llamar a `lazymark paste` sin más configuración.
func (c *core) editorCommand(path string, line int) *exec.Cmd {
	if path == "" {
		return nil
	}
	// El editor puede traer argumentos ("code --wait") y una ruta con espacios.
	name, extra := config.SplitEditor(c.cfg.Editor)
	if name == "" {
		name = "micro"
	}
	bin := config.ResolveEditorBin(name)
	args := append([]string{}, extra...)
	lower := strings.ToLower(filepath.Base(bin))
	if line > 1 && (strings.Contains(lower, "micro") || strings.Contains(lower, "vim") || strings.Contains(lower, "nano")) {
		args = append(args, fmt.Sprintf("+%d", line))
	}
	args = append(args, path)
	cmd := exec.Command(bin, args...)
	exe, _ := os.Executable()
	cmd.Env = editorEnv(os.Environ(), path, exe)
	return cmd
}

// editorEnv agrega a env LAZYMARK_NOTE=note y, si hace falta, la carpeta de exe al
// principio del PATH. No modifica env.
func editorEnv(env []string, note, exe string) []string {
	out := make([]string, 0, len(env)+2)
	pathIdx := -1
	for _, kv := range env {
		if strings.HasPrefix(kv, "LAZYMARK_NOTE=") {
			continue
		}
		if strings.HasPrefix(kv, "PATH=") {
			pathIdx = len(out)
		}
		out = append(out, kv)
	}
	out = append(out, "LAZYMARK_NOTE="+note)
	if exe != "" {
		dir := filepath.Dir(exe)
		if pathIdx < 0 {
			out = append(out, "PATH="+dir)
		} else if cur := strings.TrimPrefix(out[pathIdx], "PATH="); !slices.Contains(filepath.SplitList(cur), dir) {
			out[pathIdx] = "PATH=" + dir + string(os.PathListSeparator) + cur
		}
	}
	return out
}

// appendLine agrega text al final de la nota sin reescribir el resto (X10).
func appendLine(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (c *core) errStatus(es, en string, err error) {
	c.setStatus("%s: %v", i18n.T(es, en), err)
}

// listState es el cursor y el desplazamiento de una lista con scroll.
type listState struct {
	cursor, offset int
}

// move desplaza el cursor delta posiciones dentro de n elementos.
func (s *listState) move(delta, n int) {
	s.set(s.cursor+delta, n)
}

// set coloca el cursor en i, acotado a [0, n).
func (s *listState) set(i, n int) {
	if n <= 0 {
		s.cursor, s.offset = 0, 0
		return
	}
	s.cursor = clamp(i, 0, n-1)
}

// visible ajusta el desplazamiento para que el cursor entre en h filas y
// devuelve el rango [desde, hasta) de elementos visibles.
func (s *listState) visible(h, n int) (int, int) {
	if h < 1 {
		h = 1
	}
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+h {
		s.offset = s.cursor - h + 1
	}
	s.offset = clamp(s.offset, 0, max(0, n-h))
	return s.offset, min(n, s.offset+h)
}

// counter es el texto "n of m" del borde inferior de un panel.
func counter(cursor, n int) string {
	if n == 0 {
		return "0 of 0"
	}
	return fmt.Sprintf("%d of %d", cursor+1, n)
}

// cols son los ids de las columnas del tablero según la config.
func (c *core) cols() storage.Columns { return storage.Columns(c.cfg.KanbanIDs()) }

// columnTitles son los títulos visibles de las columnas en el idioma activo.
func (c *core) columnTitles() []string {
	lang := string(i18n.CurrentLanguage())
	titles := make([]string, len(c.cfg.KanbanColumns))
	for i, col := range c.cfg.KanbanColumns {
		titles[i] = col.DisplayTitle(lang)
	}
	return titles
}
