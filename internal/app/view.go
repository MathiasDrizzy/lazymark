package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
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
	content := m.render()
	if m.c.cfg.ScreenBackground != config.ScreenBackgroundTerminal {
		content = theme.PaintBackground(content, theme.ColorBase)
	}
	v := tea.NewView(content)
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
		body = m.renderKanbanBar() + "\n" + m.kanban.view(l.Kanban, m.ht)
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
	if kind := m.restKind(); kind != "" {
		title := i18n.T("[4]─Vista previa", "[4]─Preview")
		if e := m.notes.current(); kind == restFolder && e != nil {
			title = "[4]─" + iconFolderOpen + " " + e.Name
			return m.renderRest(kind, title, fmt.Sprintf(i18n.T("%d nota(s)", "%d note(s)"), 0), r, active)
		}
		if n := m.displayedNote(); kind == restEmptyNote && n != nil {
			title = "[4]─" + n.Title
		}
		return m.renderRest(kind, title, "", r, active)
	}
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

// renderKanbanBar dibuja la fila de arriba del Kanban: el botón "← Notas (Esc)"
// que vuelve a la vista de notas, clicable.
func (m *AppModel) renderKanbanBar() string {
	r := m.layout.KanbanBar
	label := " ← " + i18n.T("Notas", "Notes") + " (" + buttonKey(m.c.keys.Key(actKanban, ctxKanban)) + ") "
	w := textwidth.Width(label)
	m.ht.Register("kanban-back", mouse.ZoneAction, r.X, r.Y, r.X+w-1, r.Y, int(actKanban), "")
	btn := lipgloss.NewStyle().Foreground(theme.ColorBase).Background(theme.ColorBlue).Bold(true).Render(label)
	// la franja lleva fondo en todo el ancho: así el compositor no recorta los espacios finales
	bar := lipgloss.NewStyle().Background(theme.ColorMantle)
	title := bar.Foreground(theme.ColorOverlay0).Render("  " + i18n.T("Tablero Kanban", "Kanban board"))
	pad := bar.Render(strings.Repeat(" ", max(0, r.W-w-textwidth.Width("  "+i18n.T("Tablero Kanban", "Kanban board")))))
	return textwidth.Fit(btn+title+pad, r.W)
}

// hint es un atajo de la barra inferior: un botón clicable con su acción. long es
// la etiqueta completa y short la forma corta que se usa si la barra no alcanza.
// Un hint "fijo" (los botones del Kanban) nunca se quita: antes se acorta.
type hint struct {
	long, short string
	action      Action
	button      bool // etiqueta "Acción (tecla)", se pinta con el estilo de botón
}

// bindingHint convierte un atajo del keymap en un hint "Acción: tecla".
func bindingHint(b Binding) hint {
	l := b.Desc() + ": " + keyLabel(b.Keys[0])
	return hint{long: l, short: l, action: b.Action}
}

// buttonKey es el nombre de una tecla dentro de un botón: "Espacio", "Enter", "Esc".
func buttonKey(k string) string {
	switch k {
	case "space":
		return i18n.T("Espacio", "Space")
	case "enter":
		return "Enter"
	case "esc":
		return "Esc"
	}
	return keyLabel(k)
}

// kanbanHints son las acciones de la tarjeta seleccionada, siempre a la vista.
func (m *AppModel) kanbanHints() []hint {
	key := func(a Action) string { return buttonKey(m.c.keys.Key(a, ctxKanban)) }
	return hintList([]hint{
		{i18n.T("Mover ← (%s)", "Move ← (%s)"), "← (%s)", actMoveCardLeft, true},
		{i18n.T("Mover → (%s)", "Move → (%s)"), "→ (%s)", actMoveCardRight, true},
		{i18n.T("Listo/Por hacer (%s)", "Done/To do (%s)"), i18n.T("Listo (%s)", "Done (%s)"), actToggleTask, true},
		{i18n.T("Editar (%s)", "Edit (%s)"), i18n.T("Editar (%s)", "Edit (%s)"), actEdit, true},
	}).withKeys(key)
}

// withKeys rellena el %s de cada etiqueta con la tecla de su acción.
func (hs hintList) withKeys(key func(Action) string) []hint {
	out := make([]hint, len(hs))
	for i, h := range hs {
		h.long = fmt.Sprintf(h.long, key(h.action))
		h.short = fmt.Sprintf(h.short, key(h.action))
		out[i] = h
	}
	return out
}

type hintList []hint

// footerHints devuelve los hints del panel (o de las tarjetas del Kanban) y los
// globales. En la vista de notas, el botón "Kanban (W)" encabeza los globales:
// es lo último que se quita.
func (m *AppModel) footerHints() (panel, global []hint, pinned bool) {
	for _, b := range m.c.keys.Footer(ctxGlobal) {
		global = append(global, bindingHint(b))
	}
	if m.kanbanOn {
		// "Notas (W)" es el botón "Kanban (W)" de la vista de notas, en el mismo lugar de la barra
		if k := m.c.keys.Key(actKanban, ctxGlobal); k != "" {
			nb := i18n.T("Notas", "Notes") + " (" + buttonKey(k) + ")"
			global = append([]hint{{nb, nb, actKanban, true}}, global...)
		}
		return m.kanbanHints(), global, true
	}
	if k := m.c.keys.Key(actKanban, ctxGlobal); k != "" {
		kb := fmt.Sprintf("Kanban (%s)", buttonKey(k))
		global = append([]hint{{kb, kb, actKanban, true}}, global...)
	}
	for _, b := range m.c.keys.Footer(m.contexts()[0]) {
		panel = append(panel, bindingHint(b))
	}
	return panel, global, false
}

// renderFooter dibuja la barra inferior estilo lazygit: la papelera abajo a la
// izquierda, los atajos del contexto y el estado a la derecha. Cada atajo es un
// botón clicable.
//
// El ancho se reparte con un presupuesto: primero se acorta el estado para que
// siempre quepan los atajos globales (Atajos: ?, Salir: q) y después se quitan
// de a uno los atajos del panel, del último al primero. Los botones del Kanban
// no se quitan: si no caben, se usan sus etiquetas cortas y el estado cede.
// La barra nunca se vacía mientras quepa un atajo y mide siempre exactamente el ancho.
func (m *AppModel) renderFooter() string {
	f := m.layout.Footer
	keyStyle := lipgloss.NewStyle().Foreground(theme.ColorBlue)
	sep := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(" | ")
	const sepW, gap = 3, 2

	trash := "\U000f0a7a " + strconv.Itoa(m.c.trashCount)
	trashW := textwidth.Width(trash) + 1 // y un espacio
	panel, global, pinned := m.footerHints()

	type shown struct {
		hint
		short bool
	}
	text := func(h shown) string {
		if h.short {
			return h.hint.short
		}
		return h.hint.long
	}
	span := func(hs []shown) int {
		w := 0
		for i, h := range hs {
			if i > 0 {
				w += sepW
			}
			w += textwidth.Width(text(h))
		}
		return w
	}
	mk := func(hs []hint, short bool) []shown {
		out := make([]shown, len(hs))
		for i, h := range hs {
			out[i] = shown{h, short}
		}
		return out
	}
	join := func(a, b []shown) []shown { return append(append([]shown{}, a...), b...) }
	globalShown := mk(global, false)

	// 1) el estado se acorta para dejar sitio a los atajos globales (o desaparece)
	status := m.c.status
	if maxStatus := f.W - trashW - span(globalShown) - gap; maxStatus <= 0 {
		status = ""
	} else if textwidth.Width(status) > maxStatus {
		status = textwidth.Truncate(status, maxStatus, textwidth.Ellipsis)
	}
	statusW := textwidth.Width(status)
	avail := f.W - trashW - statusW
	if statusW > 0 {
		avail -= gap
	}

	// 2) atajos. Primero los del panel completos, luego con etiquetas cortas y, si
	// aun así no caben, se quitan de a uno del último al primero (salvo los fijos).
	var hints []shown
	candidates := [][]shown{join(mk(panel, false), globalShown), join(mk(panel, true), globalShown)}
	if pinned {
		// el estado cede ante los botones del Kanban
		for _, c := range candidates {
			if span(c)+trashW <= f.W {
				hints = c
				avail = f.W - trashW
				break
			}
		}
	}
	for _, c := range candidates {
		if hints == nil && span(c) <= avail {
			hints = c
		}
	}
	if pinned && hints == nil {
		// antes que quitar un botón del Kanban se quitan los globales, del último al primero
		// (el botón "Notas (W)" encabeza los globales: es lo último que se va)
		for k := len(global) - 1; hints == nil && k >= 0; k-- {
			if c := join(mk(panel, true), mk(global[:k], false)); span(c)+trashW <= f.W {
				hints = c
			}
		}
		if hints == nil {
			hints = mk(panel, true)
			for len(hints) > 0 && span(hints) > f.W-trashW {
				hints = hints[:len(hints)-1]
			}
		}
	}
	for k := len(panel) - 1; hints == nil && k >= 0; k-- {
		if cand := join(mk(panel[:k], true), globalShown); span(cand) <= avail {
			hints = cand
		}
	}
	gl := globalShown
	for hints == nil && len(gl) > 0 {
		gl = gl[:len(gl)-1] // prioridad: se conserva el primero
		if span(gl) <= avail {
			hints = gl
		}
		if len(gl) == 0 {
			break
		}
	}
	if pinned { // el estado ocupa lo que dejan los botones; un aviso corto (→ En progreso) cabe siempre
		need := min(textwidth.Width(m.c.status), 18) + gap
		for _, drop := range []Action{actQuit, actCheatsheet} {
			if f.W-trashW-span(hints) >= need {
				break
			}
			for i, h := range hints {
				if h.action == drop {
					hints = append(append([]shown{}, hints[:i]...), hints[i+1:]...)
					break
				}
			}
		}
		left := f.W - trashW - span(hints)
		if left <= gap {
			status = ""
		} else if textwidth.Width(m.c.status) > left-gap {
			status = textwidth.Truncate(m.c.status, left-gap, textwidth.Ellipsis)
		} else {
			status = m.c.status
		}
		statusW = textwidth.Width(status)
	}

	out := lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(trash) + " "
	m.ht.Register("footer-trash", mouse.ZoneAction, f.X, f.Y, f.X+trashW-2, f.Y, int(actTrash), "")
	x := trashW
	for i, h := range hints {
		if i > 0 {
			out += sep
			x += sepW
		}
		l := text(h)
		w := textwidth.Width(l)
		m.ht.Register("footer-"+strconv.Itoa(i), mouse.ZoneAction, f.X+x, f.Y, f.X+x+w-1, f.Y, int(h.action), "")
		if h.button {
			out += theme.FooterKey.Render(l)
		} else {
			out += keyStyle.Render(l)
		}
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
