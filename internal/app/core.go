package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
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

	notes      []storage.Note
	tags       []views.TagInfo
	tasks      []views.FlatTask
	taskFilter views.TaskFilter
	board      views.KanbanBoard
	trashCount int

	status string
	popups []popup
}

// reload relee las notas del disco y recalcula todo lo derivado.
func (c *core) reload() {
	if notes, err := c.store.ListNotes(); err == nil {
		c.notes = notes
	}
	c.tags = views.CollectTags(c.notes)
	c.tasks = views.CollectTasks(c.scopedNotes(), c.taskFilter)
	c.board = views.CollectKanban(c.notes)
	c.trashCount = c.store.CountTrash()
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
		c.popups = c.popups[:n-1]
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
	if path == "" {
		return nil
	}
	// El editor puede traer argumentos ("code --wait").
	fields := strings.Fields(c.cfg.Editor)
	if len(fields) == 0 {
		fields = []string{"micro"}
	}
	bin := config.ResolveEditorBin(fields[0])
	args := append([]string{}, fields[1:]...)
	lower := strings.ToLower(filepath.Base(bin))
	if line > 1 && (strings.Contains(lower, "micro") || strings.Contains(lower, "vim") || strings.Contains(lower, "nano")) {
		args = append(args, fmt.Sprintf("+%d", line))
	}
	args = append(args, path)
	return tea.ExecProcess(exec.Command(bin, args...), func(err error) tea.Msg {
		return EditorFinishedMsg{Path: path, Err: err}
	})
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
