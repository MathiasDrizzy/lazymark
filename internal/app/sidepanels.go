package app

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// tasksPanel es el panel [2] Tareas: los checkboxes del alcance configurado.
type tasksPanel struct {
	c    *core
	list listState
}

func (p *tasksPanel) current() *views.FlatTask {
	if p.list.cursor < len(p.c.tasks) {
		return &p.c.tasks[p.list.cursor]
	}
	return nil
}

func (p *tasksPanel) key(a Action, reload func()) tea.Cmd {
	n := len(p.c.tasks)
	switch a {
	case actUp:
		p.list.move(-1, n)
	case actDown:
		p.list.move(1, n)
	case actTop:
		p.list.set(0, n)
	case actBottom:
		p.list.set(n-1, n)
	case actPageUp:
		p.list.move(-10, n)
	case actPageDown:
		p.list.move(10, n)
	case actEnter:
		if t := p.current(); t != nil {
			return p.c.openEditor(t.NotePath, t.Line)
		}
	case actToggleTask:
		t := p.current()
		if t == nil {
			return nil
		}
		done, err := p.c.store.ToggleTask(t.NotePath, t.Line)
		if err != nil {
			p.c.errStatus("No se pudo actualizar la tarea", "Could not update task", err)
			return nil
		}
		text := t.Text
		reload()
		p.list.set(p.list.cursor, len(p.c.tasks))
		if done {
			p.c.setStatus(i18n.T("Tarea completada: %s", "Task completed: %s"), text)
		} else {
			p.c.setStatus(i18n.T("Tarea reabierta: %s", "Task reopened: %s"), text)
		}
	case actHideDone:
		p.c.cfg.HideCompletedTasks = !p.c.cfg.HideCompletedTasks
		p.c.save()
		p.c.taskFilter = views.TaskFilterAll
		if p.c.cfg.HideCompletedTasks {
			p.c.taskFilter = views.TaskFilterPending
		}
		p.c.tasks = views.CollectTasks(p.c.scopedNotes(), p.c.taskFilter)
		p.list.set(0, len(p.c.tasks))
		p.c.setStatus("%s: %s", i18n.T("Tareas", "Tasks"), views.TaskFilterLabel(p.c.taskFilter))
	case actTaskFilter:
		p.c.taskFilter = (p.c.taskFilter + 1) % 3
		p.c.tasks = views.CollectTasks(p.c.scopedNotes(), p.c.taskFilter)
		p.list.set(0, len(p.c.tasks))
		p.c.setStatus("%s: %s", i18n.T("Filtro", "Filter"), views.TaskFilterLabel(p.c.taskFilter))
	}
	return nil
}

func (p *tasksPanel) click(row int, double bool) tea.Cmd {
	i := p.list.offset + row
	if row < 0 || i >= len(p.c.tasks) {
		return nil
	}
	p.list.set(i, len(p.c.tasks))
	if double {
		return p.key(actEnter, nil)
	}
	return nil
}

func (p *tasksPanel) view(r Rect, active bool) string {
	from, to := p.list.visible(r.H-2, len(p.c.tasks))
	var lines []string
	for i := from; i < to; i++ {
		t := p.c.tasks[i]
		box := lipgloss.NewStyle().Foreground(theme.ColorYellow).Render("☐")
		text := lipgloss.NewStyle().Foreground(theme.ColorText).Render(storage.CleanTaskText(t.Text))
		if t.Done {
			box = lipgloss.NewStyle().Foreground(theme.ColorGreen).Render("☑")
			// El tachado se abre y se cierra en la misma línea (H2-5).
			text = lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Strikethrough(true).Render(storage.CleanTaskText(t.Text))
		}
		note := strings.TrimSuffix(filepath.Base(t.NotePath), ".md")
		lines = append(lines, listRow(" "+box+" "+text, note, r.W-2, i == p.list.cursor, active))
	}
	title := i18n.T("[2]─Tareas", "[2]─Tasks")
	if p.c.taskFilter != views.TaskFilterAll {
		title += " · " + views.TaskFilterLabel(p.c.taskFilter)
	}
	return theme.RenderPanel(title, counter(p.list.cursor, len(p.c.tasks)), lines, r.W, r.H, active)
}

// tagsPanel es el panel [3] Categorías: compacto, filtra el árbol por tag.
type tagsPanel struct {
	c    *core
	list listState
}

func (p *tagsPanel) current() string {
	if p.list.cursor < len(p.c.tags) {
		return p.c.tags[p.list.cursor].Name
	}
	return ""
}

func (p *tagsPanel) key(a Action, filter func(tag string)) tea.Cmd {
	n := len(p.c.tags)
	switch a {
	case actUp:
		p.list.move(-1, n)
	case actDown:
		p.list.move(1, n)
	case actTop:
		p.list.set(0, n)
	case actBottom:
		p.list.set(n-1, n)
	case actEnter:
		if tag := p.current(); tag != "" {
			filter(tag)
		}
	}
	return nil
}

// click selecciona el tag y filtra el árbol con un solo clic (H1-9).
func (p *tagsPanel) click(row int, filter func(tag string)) {
	i := p.list.offset + row
	if row < 0 || i >= len(p.c.tags) {
		return
	}
	p.list.set(i, len(p.c.tags))
	filter(p.c.tags[i].Name)
}

func (p *tagsPanel) view(r Rect, active bool, activeTag string) string {
	from, to := p.list.visible(r.H-2, len(p.c.tags))
	var lines []string
	for i := from; i < to; i++ {
		t := p.c.tags[i]
		style := lipgloss.NewStyle().Foreground(theme.ColorBlue)
		mark := " "
		if t.Name == activeTag {
			style = style.Bold(true).Foreground(theme.ColorGreen)
			mark = lipgloss.NewStyle().Foreground(theme.ColorGreen).Render("●")
		}
		lines = append(lines, listRow(mark+style.Render("#"+t.Name), fmt.Sprintf("%d", t.NoteCount), r.W-2, i == p.list.cursor, active))
	}
	return theme.RenderPanel(i18n.T("[3]─Categorías", "[3]─Categories"), counter(p.list.cursor, len(p.c.tags)), lines, r.W, r.H, active)
}
