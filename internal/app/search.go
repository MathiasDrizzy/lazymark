package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/search"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// La búsqueda de texto completo (`/`): un popup con el campo de texto y los resultados en vivo (nota, línea y contexto, con lo hallado
// resaltado). No hay índice: cada cambio del texto cancela la búsqueda anterior y, pasada una pausa corta, lanza otra en una
// goroutine (internal/search, concurrente y cancelable), así que seguir escribiendo o navegar nunca espera a una búsqueda.
const (
	searchDebounce = 70 * time.Millisecond // pausa tras la última tecla antes de buscar
	searchTimeout  = 10 * time.Second      // una búsqueda no se queda trabajando más que esto
)

// searchTickMsg avisa que pasó la pausa de la generación gen; searchDoneMsg trae sus resultados; searchJumpMsg pide ir a una coincidencia.
type (
	searchTickMsg struct{ gen int }
	searchDoneMsg struct {
		gen   int
		query string
		res   search.Result
		err   error
	}
	searchJumpMsg struct{ m search.Match }
)

type searchPopup struct {
	popupList
	c       *core
	input   textinput.Model
	gen     int // cambia con cada texto: lo que llega de una generación anterior se ignora
	cancel  context.CancelFunc
	state   searchState
	res     search.Result
	matches []search.Match
	errText string
}

type searchState int

const (
	searchIdle searchState = iota // sin texto
	searchWorking
	searchDone
)

func newSearchPopup(c *core) *searchPopup {
	ti := textinput.New()
	ti.SetStyles(themedInputStyles())
	ti.Prompt = ""
	ti.Placeholder = i18n.T("buscar en las notas…", "search the notes…")
	ti.Focus()
	return &searchPopup{c: c, input: ti}
}

func (p *searchPopup) contexts() []Context { return nil }
func (p *searchPopup) bottomRight() bool   { return false }

// close cancela la búsqueda en curso (el popup se cerró).
func (p *searchPopup) close() {
	p.gen++
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
}

func (p *searchPopup) handle(_ Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		return p.jump()
	case "up", "ctrl+p":
		p.move(-1)
		return nil, false
	case "down", "ctrl+n":
		p.move(1)
		return nil, false
	case "pgup":
		p.move(-p.height)
		return nil, false
	case "pgdown":
		p.move(p.height)
		return nil, false
	}
	before := p.input.Value()
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() != before {
		return tea.Batch(cmd, p.schedule()), false
	}
	return cmd, false
}

// paste inserta en el campo el texto pegado.
func (p *searchPopup) paste(msg tea.PasteMsg) tea.Cmd {
	before := p.input.Value()
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() != before {
		return tea.Batch(cmd, p.schedule())
	}
	return cmd
}

func (p *searchPopup) move(d int) {
	if p.n > 0 {
		p.list.set(clamp(p.list.cursor+d, 0, p.n-1), p.n)
	}
}

// schedule corta la búsqueda anterior y, si hay texto, la deja pendiente tras la pausa.
func (p *searchPopup) schedule() tea.Cmd {
	p.close()
	if strings.TrimSpace(p.input.Value()) == "" {
		p.state, p.matches, p.n, p.errText = searchIdle, nil, 0, ""
		p.list = listState{}
		return nil
	}
	p.state = searchWorking
	gen := p.gen
	return tea.Tick(searchDebounce, func(time.Time) tea.Msg { return searchTickMsg{gen} })
}

// run lanza la búsqueda de la generación gen en una goroutine (si sigue siendo la vigente).
func (p *searchPopup) run(gen int) tea.Cmd {
	if gen != p.gen || p.state != searchWorking {
		return nil
	}
	q := strings.TrimSpace(p.input.Value())
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	p.cancel = cancel
	store := p.c.store
	return func() tea.Msg {
		defer cancel()
		res, err := search.Run(ctx, store, q, search.Options{})
		return searchDoneMsg{gen: gen, query: q, res: res, err: err}
	}
}

// done recibe los resultados; los de una generación anterior se tiran.
func (p *searchPopup) done(msg searchDoneMsg) {
	if msg.gen != p.gen {
		return
	}
	p.cancel = nil
	p.state, p.res, p.matches, p.errText = searchDone, msg.res, msg.res.Matches, ""
	if msg.err != nil {
		p.errText = msg.err.Error()
	}
	p.n = len(p.matches)
	p.list = listState{}
}

func (p *searchPopup) jump() (tea.Cmd, bool) {
	if p.state != searchDone || p.list.cursor >= len(p.matches) {
		return nil, false
	}
	m := p.matches[p.list.cursor]
	return func() tea.Msg { return searchJumpMsg{m} }, true
}

// click: el primer clic en un resultado lo selecciona y el segundo salta a él.
func (p *searchPopup) click(_, y int) (tea.Cmd, bool) {
	if _, again, ok := p.clickRow(y); ok && again {
		return p.jump()
	}
	return nil, false
}

// highlight dibuja el texto de una coincidencia con lo hallado resaltado.
func highlight(text string, start, end int) string {
	if start < 0 || end > len(text) || start >= end {
		return lipgloss.NewStyle().Foreground(theme.ColorText).Render(text)
	}
	text0 := lipgloss.NewStyle().Foreground(theme.ColorText)
	return text0.Render(text[:start]) + lipgloss.NewStyle().Foreground(theme.ColorPeach).Bold(true).Render(text[start:end]) + text0.Render(text[end:])
}

func (p *searchPopup) status() string {
	switch {
	case p.errText != "":
		return p.errText
	case p.state == searchIdle:
		return i18n.T("Escribe para buscar en todas las notas", "Type to search all the notes")
	case p.state == searchWorking:
		return i18n.T("Buscando…", "Searching…")
	case p.n == 0:
		return i18n.T("Sin resultados", "No results")
	}
	s := fmt.Sprintf(i18n.T("%d resultado(s) en %d nota(s)", "%d result(s) in %d note(s)"), p.n, p.res.Files)
	if p.res.Truncated {
		s += " · " + i18n.T("hay más: afina la búsqueda", "there are more: narrow the search")
	}
	return s
}

func (p *searchPopup) render(l Layout) string {
	w := popupWidth(l, 96)
	p.input.SetWidth(w - 8)
	field := accent("/ ") + lipgloss.NewStyle().Foreground(theme.ColorText).Render(p.input.View())
	p.top = 3
	p.height = clamp(max(p.n, 1), 1, min(12, max(1, l.H-10)))
	lines := []string{field, dim(p.status())}
	lines = append(lines, p.rows(w-3, func(i int) string {
		m := p.matches[i]
		where := dim(fmt.Sprintf("%s:%d", m.Rel, m.Line))
		room := w - 3 - 4 - textwidth.Width(fmt.Sprintf("%s:%d", m.Rel, m.Line))
		text, s, e := m.Text, m.Start, m.End
		if textwidth.Width(text) > room { // el contexto se corta por la derecha; lo hallado se ve si cabe
			text = textwidth.Truncate(text, max(8, room), "…")
			if e > len(text) {
				s, e = -1, -1
			}
		}
		return where + "  " + highlight(text, s, e)
	})...)
	return theme.RenderPopup(i18n.T("Buscar", "Search"), "[Enter] "+i18n.T("ir", "go")+" · "+escHint, lines, w)
}

// jumpTo lleva la vista a una coincidencia: selecciona la nota en el panel Notas (abriendo sus carpetas y quitando el filtro si la
// escondía), pone el foco en la vista previa y la deja posicionada en la línea.
func (m *AppModel) jumpTo(hit search.Match) {
	note := m.noteByPath(hit.Path)
	if note == nil {
		m.c.reload()
		if note = m.noteByPath(hit.Path); note == nil {
			m.c.setStatus("%s", i18n.T("La nota ya no existe", "The note no longer exists"))
			return
		}
	}
	m.kanbanOn = false
	m.zoom = false
	base := m.c.store.BaseDir
	if rel, err := filepath.Rel(base, filepath.Dir(hit.Path)); err == nil && rel != "." { // abre las carpetas del camino
		dir := base
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			dir = filepath.Join(dir, part)
			m.notes.expanded[dir] = true
		}
	}
	m.notes.reload()
	if !m.notes.selectPath(note.Path) {
		m.notes.tagFilter = ""
		m.notes.list = listState{}
		m.notes.reload()
		m.notes.selectPath(note.Path)
	}
	m.relayout()
	m.setFocus(panelNotes)
	m.setFocus(panelPreview)
	inner, h := m.layout.Preview.W-2, m.layout.Preview.H-2
	if inner >= 8 && h >= 1 {
		lines := m.preview.lines(note, inner-1)
		m.preview.scrollX = 0
		m.preview.scrollY = max(0, renderedLineOf(lines, note, hit.Line)-h/3)
	}
	m.c.setStatus("%s:%d", hit.Rel, hit.Line)
}

// lineKey es la huella de una línea de la nota para buscarla en el markdown renderizado: sus primeras palabras sin marcas.
func lineKey(raw string) string {
	clean := strings.NewReplacer("*", "", "_", "", "`", "", "[", "", "]", "", "#", "", ">", "").Replace(raw)
	words := strings.Fields(clean)
	if len(words) > 0 && (words[0] == "-" || words[0] == "x" || words[0] == "X") {
		words = words[1:]
	}
	return strings.Join(words[:min(3, len(words))], " ")
}

// renderedLineOf devuelve la fila del markdown renderizado donde está la línea line (desde 1) de la nota: la ocurrencia que
// corresponde por orden entre las líneas con el mismo comienzo; si no la encuentra, estima la posición proporcional.
func renderedLineOf(lines []string, note *storage.Note, line int) int {
	raws := strings.Split(note.Content, "\n")
	if line < 1 || line > len(raws) {
		return 0
	}
	if key := lineKey(raws[line-1]); key != "" {
		nth := 0
		for _, other := range raws[:line-1] {
			if lineKey(other) == key {
				nth++
			}
		}
		for i, l := range lines {
			if strings.Contains(textwidth.Strip(l), key) {
				if nth == 0 {
					return i
				}
				nth--
			}
		}
	}
	return (line - 1) * len(lines) / max(1, len(raws))
}
