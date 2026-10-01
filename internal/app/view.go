package app

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// View declara pantalla alternativa y mouse; Bubble Tea v2 los reaplica en cada
// frame, también al volver del editor externo.
func (m *AppModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	if m.c.cfg.MouseClick {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

// render compone la pantalla: paneles según el Layout y, encima, los popups
// como capas del compositor de Lip Gloss v2.
func (m *AppModel) render() string {
	if m.quitting || m.w == 0 {
		return ""
	}
	l := m.layout
	if l.TooSmall {
		return m.renderTooSmall()
	}
	m.ht.Clear()

	var body string
	switch {
	case m.kanbanOn:
		body = m.kanban.view(l.Kanban, m.ht)
	case m.zoom && m.focus == panelPreview:
		body = m.preview.view(m.previewNote(), l.Preview, true)
	default:
		left := []string{m.notes.view(l.Notes, m.focus == panelNotes)}
		if !l.Tasks.Empty() {
			left = append(left, m.tasks.view(l.Tasks, m.focus == panelTasks))
		}
		if !l.Tags.Empty() {
			left = append(left, m.tags.view(l.Tags, m.focus == panelTags, m.notes.tagFilter))
		}
		col := strings.Join(left, "\n")
		body = lipgloss.JoinHorizontal(lipgloss.Top, col, m.renderPreview())
	}
	base := body + "\n" + m.renderFooter()

	if len(m.c.popups) == 0 {
		return base
	}
	layers := []*lipgloss.Layer{lipgloss.NewLayer(base)}
	for i, p := range m.c.popups {
		out := p.render(l)
		r := popupRect(l, p, out)
		layers = append(layers, lipgloss.NewLayer(out).X(r.X).Y(r.Y).Z(i+1).ID(fmt.Sprintf("popup-%d", i)))
	}
	return lipgloss.NewCompositor(layers...).Render()
}

// renderPreview elige qué mostrar a la derecha según el último panel izquierdo.
func (m *AppModel) renderPreview() string {
	r, active := m.layout.Preview, m.focus == panelPreview
	switch m.lastLeft {
	case panelTasks:
		// la nota de la tarea seleccionada, posicionada en su línea
		return m.preview.view(m.previewNote(), r, active)
	case panelTags:
		return views.RenderTagPreview(m.c.notes, m.tags.current(), r.W, r.H, active)
	}
	if e := m.notes.current(); e != nil && e.Type == storage.EntryFolder {
		return m.renderFolderPreview(e, r, active)
	}
	return m.preview.view(m.notes.currentNote(), r, active)
}

// renderFolderPreview lista las notas que hay dentro de la carpeta bajo el cursor.
func (m *AppModel) renderFolderPreview(e *storage.NoteEntry, r Rect, active bool) string {
	prefix := e.Path + string(filepath.Separator)
	var lines []string
	for _, n := range m.c.notes {
		if rel, ok := strings.CutPrefix(n.Path, prefix); ok {
			lines = append(lines, " "+lipgloss.NewStyle().Foreground(theme.ColorTeal).Render(iconNote)+" "+
				lipgloss.NewStyle().Foreground(theme.ColorText).Render(rel)+"  "+dim(n.ModTime.Format("02 Jan 2006")))
		}
	}
	footer := fmt.Sprintf(i18n.T("%d nota(s)", "%d note(s)"), len(lines))
	if len(lines) == 0 {
		lines = []string{" " + dim(i18n.T("Carpeta vacía", "Empty folder"))}
	}
	return theme.RenderPanel("[4]─"+iconFolderOpen+" "+e.Name, footer, lines, r.W, r.H, active)
}

// renderFooter dibuja la barra inferior estilo lazygit: la papelera abajo a la
// izquierda, los atajos del contexto ("Acción: tecla | …") y el estado a la
// derecha. Cada atajo es un botón clicable.
func (m *AppModel) renderFooter() string {
	f := m.layout.Footer
	keyStyle := lipgloss.NewStyle().Foreground(theme.ColorBlue)
	sep := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(" | ")

	trash := fmt.Sprintf("%s %d", "\U000f0a7a", m.c.trashCount)
	out := lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(trash) + " "
	x := textwidth.Width(trash) + 1
	m.ht.Register("footer-trash", mouse.ZoneAction, f.X, f.Y, f.X+x-2, f.Y, int(actTrash), "")

	status := lipgloss.NewStyle().Foreground(theme.ColorGreen).Render(m.c.status)
	room := f.W - textwidth.Width(m.c.status) - 2
	// Los atajos globales (Atajos: ?, Salir: q) siempre quedan visibles, como
	// en lazygit: si no hay lugar, se recortan los del panel.
	ctx := m.contexts()[0]
	global := m.c.keys.Footer(ctxGlobal)
	panel := m.c.keys.Footer(ctx)
	width := func(bs []Binding) int {
		w := 0
		for i, b := range bs {
			if i > 0 {
				w += 3
			}
			w += textwidth.Width(b.Desc() + ": " + keyLabel(b.Keys[0]))
		}
		return w
	}
	for len(panel) > 0 && x+width(append(append([]Binding{}, panel...), global...)) > room {
		panel = panel[:len(panel)-1]
	}
	for i, b := range append(panel, global...) {
		label := b.Desc() + ": " + keyLabel(b.Keys[0])
		w := textwidth.Width(label)
		if i > 0 {
			out += sep
			x += 3
		}
		if x+w > room {
			break
		}
		m.ht.Register(fmt.Sprintf("footer-%d", i), mouse.ZoneAction, f.X+x, f.Y, f.X+x+w-1, f.Y, int(b.Action), "")
		out += keyStyle.Render(label)
		x += w
	}
	if room < 0 {
		status = ""
	}
	return textwidth.Pad(out, f.W-textwidth.Width(status)) + status
}

// renderTooSmall muestra el aviso de tamaño mínimo en vez de una UI rota (X6).
func (m *AppModel) renderTooSmall() string {
	msg := []string{
		i18n.T("Terminal muy pequeña", "Terminal too small"),
		fmt.Sprintf("%dx%d", m.w, m.h),
		fmt.Sprintf(i18n.T("Mínimo %dx%d", "Minimum %dx%d"), MinWidth, MinHeight),
	}
	lines := make([]string, m.h)
	top := max(0, (m.h-len(msg))/2)
	for i := range lines {
		txt := ""
		if j := i - top; j >= 0 && j < len(msg) {
			txt = msg[j]
		}
		pad := max(0, (m.w-textwidth.Width(txt))/2)
		lines[i] = textwidth.Fit(textwidth.Repeat(" ", pad)+txt, m.w)
	}
	return strings.Join(lines, "\n")
}
