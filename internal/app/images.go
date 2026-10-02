package app

import (
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// imageDebounce es lo que se espera, tras un cambio de selección, antes de decodificar sus
// imágenes: con teclas repetidas se salta la espera de las selecciones intermedias, y al navegar
// despacio no se nota.
const imageDebounce = 40 * time.Millisecond

// imageTickMsg llega imageDebounce después de que una selección pidió imágenes.
type imageTickMsg struct{ sel int }

// imageLoadedMsg es el resultado de decodificar y codificar una imagen fuera de Update.
type imageLoadedMsg struct {
	sel  int
	job  image.Job
	tmpl string
	err  error
}

// startImageJobs lanza, cada una en su goroutine (tea.Cmd), las imágenes que la selección actual
// pidió. Si la selección ya cambió, el tick no hace nada.
func (m *AppModel) startImageJobs(t imageTickMsg) tea.Cmd {
	k := m.c.kitty
	if t.sel != k.Selection() {
		return nil
	}
	var cmds []tea.Cmd
	for _, job := range k.TakeJobs() {
		cmds = append(cmds, func() tea.Msg {
			tmpl, err := image.Encode(job)
			return imageLoadedMsg{sel: t.sel, job: job, tmpl: tmpl, err: err}
		})
	}
	return tea.Batch(cmds...)
}

// displayedNote es la nota que el panel derecho está mostrando (nil si es una
// carpeta, un tag o no hay selección).
func (m *AppModel) displayedNote() *storage.Note {
	if m.zoom && m.focus == panelPreview {
		return m.previewNote() // el render maximizado muestra siempre la nota
	}
	switch m.lastLeft {
	case panelTags:
		return nil
	case panelTasks:
		return m.previewNote()
	}
	return m.notes.currentNote()
}

// prepareImages deja los gráficos coherentes con lo que se va a dibujar y
// devuelve los comandos que los mandan a la terminal. Cambiar de nota borra las
// imágenes (a=d); al volver se transmiten de nuevo. Un popup NO las borra: con
// placeholders Unicode la imagen es texto (https://sw.kovidgoyal.net/kitty/graphics-protocol/#graphics-unicode-placeholders),
// así que el popup se pinta encima y solo tapa las celdas que ocupa. Se
// ejecuta dentro de Update para que el render de View ya encuentre el
// markdown en caché y no tenga que emitir nada.
func (m *AppModel) prepareImages() []tea.Cmd {
	k := m.c.kitty
	visible := !m.kanbanOn && !m.layout.TooSmall && !m.quitting && !m.c.editing
	k.SetVisible(visible)
	note := m.displayedNote()
	path := ""
	if note != nil {
		path = note.Path
	}
	if path != m.imgNote {
		m.imgNote = path
		k.Reset()
		k.NewSelection() // lo que se pidió o se está cargando de la nota anterior ya no importa
	}
	if visible && note != nil && !m.quitting {
		m.preview.lines(note, m.layout.Preview.W-3)
	}
	var cmds []tea.Cmd
	if k.HasWanted() && m.imgTickSel != k.Selection() {
		// hay imágenes por cargar: se espera un instante a que la selección se asiente
		// (con teclas repetidas, solo la última selección llega a cargar su imagen)
		m.imgTickSel = k.Selection()
		sel := k.Selection()
		cmds = append(cmds, tea.Tick(imageDebounce, func(time.Time) tea.Msg { return imageTickMsg{sel: sel} }))
	}
	for _, seq := range k.TakePending() {
		cmds = append(cmds, m.emit(seq))
	}
	return slices.DeleteFunc(cmds, func(c tea.Cmd) bool { return c == nil })
}
