package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
)

// handleClick resuelve un clic izquierdo. Con un popup abierto, solo cuenta lo
// que cae dentro de su rectángulo: un clic fuera se ignora y no cambia el foco.
func (m *AppModel) handleClick(msg tea.MouseClickMsg) tea.Cmd {
	if !m.c.cfg.MouseClick || msg.Button != tea.MouseLeft || m.layout.TooSmall {
		return nil
	}
	x, y := msg.X, msg.Y
	if p := m.c.top(); p != nil {
		r := popupRect(m.layout, p, p.render(m.layout))
		if !r.Contains(x, y) {
			return nil
		}
		cmd, done := p.click(x-r.X, y-r.Y)
		if done {
			m.removePopup(p)
		}
		m.relayout()
		return cmd
	}

	l := m.layout
	if l.Footer.Contains(x, y) {
		if z, ok := m.ht.Check(x, y); ok {
			return m.do(Action(z.Index))
		}
		return nil
	}
	if m.kanbanOn {
		z, ok := m.ht.Check(x, y)
		switch {
		case !ok:
		case z.Type == mouse.ZoneAction: // el botón "← Notas (Esc)"
			return m.do(Action(z.Index))
		default:
			return m.kanban.click(z, m.clicks.hit(z.ID, time.Now()))
		}
		return nil
	}
	if cmd, ok := m.mascotClick(x, y); ok {
		return cmd
	}
	if l.Divider.Contains(x, y) {
		m.dragging = true
		return nil
	}
	switch {
	case l.Notes.Contains(x, y):
		m.setFocus(panelNotes)
		row := y - l.Notes.Y - 1
		before := m.notes.list.cursor
		cmd := m.notes.click(row, m.clicks.hit(rowZone("n", m.notes.list.offset+row), time.Now()))
		if m.notes.list.cursor != before {
			m.preview.reset()
		}
		return m.afterPanel(cmd)
	case l.Tasks.Contains(x, y):
		m.setFocus(panelTasks)
		row := y - l.Tasks.Y - 1
		return m.afterPanel(m.tasks.click(x-l.Tasks.X, row, m.clicks.hit(rowZone("t", m.tasks.list.offset+row), time.Now()), m.afterChange))
	case l.Tags.Contains(x, y):
		m.setFocus(panelTags)
		m.tags.click(y-l.Tags.Y-1, m.filterTag)
		m.relayout()
	case l.Preview.Contains(x, y):
		m.setFocus(panelPreview)
		for _, h := range m.preview.hits { // un clic en un enlace lo sigue
			if h.y == y-l.Preview.Y && x-l.Preview.X >= h.x0 && x-l.Preview.X < h.x1 {
				m.preview.sel = h.k + 1
				return m.followLink(h.k)
			}
		}
	}
	return nil
}

func rowZone(prefix string, i int) string {
	return prefix + string(rune('0'+i%10)) + string(rune('a'+i/10%26))
}

// handleMotion arrastra el divisor entre columnas (SPEC §4).
func (m *AppModel) handleMotion(msg tea.MouseMotionMsg) {
	if m.kanbanOn && m.kanban.press != nil { // arrastre de una tarjeta del Kanban
		m.kanban.motion(msg.X, m.layout.Kanban.W)
		return
	}
	if !m.dragging || m.w == 0 {
		return
	}
	m.ratio = min(0.75, max(0.15, float64(msg.X+1)/float64(m.w)))
	m.relayout()
}

func (m *AppModel) handleRelease() {
	if m.kanbanOn && m.kanban.press != nil {
		m.kanban.release()
		return
	}
	if m.dragging {
		m.dragging = false
		m.c.cfg.SidebarRatio = m.ratio
		m.c.save()
	}
}

// handleWheel desplaza el panel que está bajo el puntero.
func (m *AppModel) handleWheel(msg tea.MouseWheelMsg) {
	if !m.c.cfg.MouseClick || m.c.top() != nil || m.kanbanOn {
		return
	}
	dir := 0
	switch msg.Button {
	case tea.MouseWheelUp:
		dir = -1
	case tea.MouseWheelDown:
		dir = 1
	default:
		return
	}
	l := m.layout
	switch {
	case l.Preview.Contains(msg.X, msg.Y):
		m.preview.scroll(3*dir, 0)
	case l.Notes.Contains(msg.X, msg.Y):
		m.notes.list.move(dir, len(m.notes.entries))
		m.preview.reset()
	case l.Tasks.Contains(msg.X, msg.Y):
		m.tasks.list.move(dir, len(m.c.tasks))
	case l.Tags.Contains(msg.X, msg.Y):
		m.tags.list.move(dir, len(m.c.tags))
	}
}
