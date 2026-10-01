package app

import (
	"fmt"
	"path/filepath"
	"strconv"
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
//
// El ancho se reparte con un presupuesto: primero se acorta el estado para que
// siempre quepan los atajos globales (Atajos: ?, Salir: q) y después se quitan
// de a uno los atajos del panel, del último al primero. La barra nunca se
// vacía mientras quepa un atajo y mide siempre exactamente el ancho.
func (m *AppModel) renderFooter() string {
	f := m.layout.Footer
	keyStyle := lipgloss.NewStyle().Foreground(theme.ColorBlue)
	sep := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(" | ")
	const sepW, gap = 3, 2

	trash := "\U000f0a7a " + strconv.Itoa(m.c.trashCount)
	trashW := textwidth.Width(trash) + 1 // y un espacio
	label := func(b Binding) string { return b.Desc() + ": " + keyLabel(b.Keys[0]) }
	span := func(bs []Binding) int {
		w := 0
		for i, b := range bs {
			if i > 0 {
				w += sepW
			}
			w += textwidth.Width(label(b))
		}
		return w
	}
	join := func(a, b []Binding) []Binding { return append(append([]Binding{}, a...), b...) }

	panel := m.c.keys.Footer(m.contexts()[0])
	global := m.c.keys.Footer(ctxGlobal)

	// 1) el estado se acorta para dejar sitio a los atajos globales (o desaparece)
	status := m.c.status
	if maxStatus := f.W - trashW - span(global) - gap; maxStatus <= 0 {
		status = ""
	} else if textwidth.Width(status) > maxStatus {
		status = textwidth.Truncate(status, maxStatus, textwidth.Ellipsis)
	}
	statusW := textwidth.Width(status)
	avail := f.W - trashW - statusW
	if statusW > 0 {
		avail -= gap
	}

	// 2) atajos: se quitan de a uno los del panel; si ni los globales caben, el último de ellos
	var hints []Binding
	for k := len(panel); k >= 0; k-- {
		if cand := join(panel[:k], global); span(cand) <= avail {
			hints = cand
			break
		}
	}
	for hints == nil && len(global) > 0 {
		global = global[:len(global)-1] // prioridad: se conserva el primero
		if span(global) <= avail {
			hints = global
		}
		if len(global) == 0 {
			break
		}
	}

	out := lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(trash) + " "
	m.ht.Register("footer-trash", mouse.ZoneAction, f.X, f.Y, f.X+trashW-2, f.Y, int(actTrash), "")
	x := trashW
	for i, b := range hints {
		if i > 0 {
			out += sep
			x += sepW
		}
		l := label(b)
		w := textwidth.Width(l)
		m.ht.Register("footer-"+strconv.Itoa(i), mouse.ZoneAction, f.X+x, f.Y, f.X+x+w-1, f.Y, int(b.Action), "")
		out += keyStyle.Render(l)
		x += w
	}
	right := lipgloss.NewStyle().Foreground(theme.ColorGreen).Render(status)
	return textwidth.Fit(textwidth.Pad(out, f.W-statusW)+right, f.W)
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
