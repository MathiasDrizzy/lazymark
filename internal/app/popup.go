package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// popup es un modelo de la pila de popups. Recibe teclas ya resueltas a
// acciones (más el mensaje original) y clics relativos a su esquina superior
// izquierda. done=true lo saca de la pila.
type popup interface {
	contexts() []Context
	handle(a Action, msg tea.KeyPressMsg) (cmd tea.Cmd, done bool)
	click(x, y int) (cmd tea.Cmd, done bool)
	render(l Layout) string
	bottomRight() bool
}

// popupRect ubica un popup ya renderizado: centrado, o abajo a la derecha
// dejando a la vista el borde de los paneles de fondo.
func popupRect(l Layout, p popup, out string) Rect {
	w, h := lipgloss.Width(out), lipgloss.Height(out)
	bodyH := l.Footer.Y
	if p.bottomRight() {
		return Rect{max(0, l.W-w-1), max(0, bodyH-h-1), w, h}
	}
	return Rect{max(0, (l.W-w)/2), max(0, (bodyH-h)/2), w, h}
}

func popupWidth(l Layout, want int) int { return min(want, l.W-4) }

var escHint = "[Esc]"

func dim(s string) string { return lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(s) }

func accent(s string) string {
	return lipgloss.NewStyle().Foreground(theme.ColorPeach).Bold(true).Render(s)
}

// popupList es la parte común de los popups con una lista navegable: el primer
// clic sobre una fila mueve el cursor y el segundo la activa (H1-7).
type popupList struct {
	list listState
	n    int
	// top es la fila del popup donde empieza la lista (debajo del borde y los encabezados).
	top    int
	height int
}

func (pl *popupList) nav(a Action) bool {
	switch a {
	case actUp:
		pl.list.set((pl.list.cursor-1+pl.n)%max(1, pl.n), pl.n)
	case actDown:
		pl.list.set((pl.list.cursor+1)%max(1, pl.n), pl.n)
	case actTop:
		pl.list.set(0, pl.n)
	case actBottom:
		pl.list.set(pl.n-1, pl.n)
	default:
		return false
	}
	return true
}

// clickRow devuelve el índice clicado y si ya estaba bajo el cursor.
func (pl *popupList) clickRow(y int) (idx int, again bool, ok bool) {
	i := pl.list.offset + y - pl.top
	if y < pl.top || y >= pl.top+pl.height || i < 0 || i >= pl.n {
		return 0, false, false
	}
	again = i == pl.list.cursor
	pl.list.set(i, pl.n)
	return i, again, true
}

func (pl *popupList) rows(w int, render func(i int) string) []string {
	from, to := pl.list.visible(pl.height, pl.n)
	var out []string
	for i := from; i < to; i++ {
		cursor := "  "
		if i == pl.list.cursor {
			cursor = accent("> ")
		}
		out = append(out, cursor+render(i))
	}
	for len(out) < pl.height {
		out = append(out, "")
	}
	return out
}

// ─── Confirmación ──────────────────────────────────────────────────

type confirmPopup struct {
	title, msg string
	onYes      func() tea.Cmd
	buttonsRow int
}

func newConfirmPopup(title, msg string, onYes func() tea.Cmd) *confirmPopup {
	return &confirmPopup{title: title, msg: msg, onYes: onYes}
}

func (p *confirmPopup) contexts() []Context { return []Context{ctxConfirm} }
func (p *confirmPopup) bottomRight() bool   { return false }

func (p *confirmPopup) handle(a Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case a == actConfirm:
		return p.onYes(), true
	case msg.String() == "n":
		return nil, true
	}
	return nil, false
}

func (p *confirmPopup) click(x, y int) (tea.Cmd, bool) {
	if y != p.buttonsRow {
		return nil, false
	}
	if x < 16 {
		return p.onYes(), true
	}
	return nil, true
}

func (p *confirmPopup) render(l Layout) string {
	w := popupWidth(l, 56)
	var lines []string
	for _, para := range strings.Split(p.msg, "\n") {
		wrapped := lipgloss.NewStyle().Width(w - 4).Render(para)
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	lines = append(lines, "")
	p.buttonsRow = len(lines) + 1
	lines = append(lines, accent("[y] "+i18n.T("Sí", "Yes"))+"         "+dim("[n] No"))
	return theme.RenderPopup(p.title, escHint, lines, w)
}

// ─── Nombre (crear / renombrar) ────────────────────────────────────

type inputPopup struct {
	title    string
	input    textinput.Model
	onSubmit func(string) tea.Cmd
}

func newInputPopup(title, initial string, onSubmit func(string) tea.Cmd) *inputPopup {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetValue(initial)
	ti.CursorEnd()
	ti.Focus()
	return &inputPopup{title: title, input: ti, onSubmit: onSubmit}
}

func (p *inputPopup) contexts() []Context { return nil }
func (p *inputPopup) bottomRight() bool   { return false }

func (p *inputPopup) handle(_ Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if msg.String() == "enter" {
		name := strings.TrimSpace(p.input.Value())
		if name == "" {
			return nil, false
		}
		return p.onSubmit(name), true
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd, false
}

func (p *inputPopup) click(int, int) (tea.Cmd, bool) { return nil, false }

func (p *inputPopup) render(l Layout) string {
	w := popupWidth(l, 52)
	p.input.SetWidth(w - 6)
	field := lipgloss.NewStyle().Foreground(theme.ColorText).Render(p.input.View())
	label := dim(i18n.T("Nombre:", "Name:"))
	return theme.RenderPopup(p.title, "[Enter] OK · "+escHint, []string{label, field, ""}, w)
}

// ─── Mover a carpeta ───────────────────────────────────────────────

type movePopup struct {
	popupList
	base    string
	folders []string
	count   int
	onPick  func(string) tea.Cmd
}

func newMovePopup(base string, folders []string, count int, onPick func(string) tea.Cmd) *movePopup {
	p := &movePopup{base: base, folders: folders, count: count, onPick: onPick}
	p.n = len(folders)
	return p
}

func (p *movePopup) contexts() []Context { return []Context{ctxPopup, ctxNav} }
func (p *movePopup) bottomRight() bool   { return false }

func (p *movePopup) handle(a Action, _ tea.KeyPressMsg) (tea.Cmd, bool) {
	if p.nav(a) {
		return nil, false
	}
	if a == actConfirm && p.list.cursor < p.n {
		return p.onPick(p.folders[p.list.cursor]), true
	}
	return nil, false
}

// click solo selecciona la carpeta; mover se confirma con Enter (decisión G).
func (p *movePopup) click(_, y int) (tea.Cmd, bool) {
	p.clickRow(y)
	return nil, false
}

func (p *movePopup) render(l Layout) string {
	w := popupWidth(l, 54)
	p.top = 3
	p.height = clamp(p.n, 1, max(1, l.H-10))
	lines := []string{dim(fmt.Sprintf(i18n.T("Destino para %d elemento(s):", "Destination for %d item(s):"), p.count)), ""}
	lines = append(lines, p.rows(w-4, func(i int) string {
		rel, _ := filepath.Rel(p.base, p.folders[i])
		if rel == "." {
			rel = "/"
		}
		return iconFolderClosed + " " + rel
	})...)
	return theme.RenderPopup(i18n.T("Mover", "Move"), "[Enter] OK · "+escHint, lines, w)
}

// ─── Cheatsheet ────────────────────────────────────────────────────

// cheatsheetPopup lista los atajos del contexto actual tal como están en el
// keymap: no puede mostrar un atajo que no exista (X9).
type cheatsheetPopup struct {
	keys Keymap
	ctx  Context
}

func (p *cheatsheetPopup) contexts() []Context { return nil }
func (p *cheatsheetPopup) bottomRight() bool   { return true }

func (p *cheatsheetPopup) handle(a Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	return nil, p.keys.Lookup(msg.String(), ctxGlobal) == actCheatsheet
}

func (p *cheatsheetPopup) click(int, int) (tea.Cmd, bool) { return nil, false }

// sections devuelve los atajos del cheatsheet: los del contexto, la
// navegación y los globales.
func (p *cheatsheetPopup) sections() [][]Binding {
	out := [][]Binding{p.keys.In(p.ctx)}
	if p.ctx != ctxKanban {
		out = append(out, p.keys.In(ctxNav))
	}
	return append(out, p.keys.In(ctxGlobal))
}

func (p *cheatsheetPopup) render(l Layout) string {
	const keyW = 14
	colW := 0
	var items []string
	for i, sec := range p.sections() {
		if i > 0 {
			items = append(items, "")
		}
		for _, b := range sec {
			item := accent(textwidth.Pad(b.KeyLabel(), keyW)) + lipgloss.NewStyle().Foreground(theme.ColorText).Render(b.Desc())
			colW = max(colW, textwidth.Width(item)+2)
			items = append(items, item)
		}
	}
	// Si no entra en una columna, se reparte en varias para no esconder
	// ningún atajo.
	rows := max(3, l.Footer.Y-4)
	cols := (len(items) + rows - 1) / rows
	rows = (len(items) + cols - 1) / cols
	w := popupWidth(l, cols*colW+4)
	cw := (w - 4) / cols
	lines := make([]string, rows)
	for i, it := range items {
		r, c := i%rows, i/rows
		cell := textwidth.Fit(it, cw-1)
		if c < cols-1 {
			cell += " "
		}
		lines[r] += textwidth.Pad("", c*cw-textwidth.Width(lines[r])) + cell
	}
	return theme.RenderPopup(i18n.T("Atajos", "Keybindings"), escHint, lines, w)
}
