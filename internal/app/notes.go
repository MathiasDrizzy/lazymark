package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/links"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// Iconos Nerd Font del árbol (todos de una celda).
const (
	iconFolderOpen   = ""
	iconFolderClosed = ""
	iconNote         = ""
	iconChecked      = "✓"
)

// notesPanel es el panel [1] Notas: árbol de carpetas y notas en un solo panel.
type notesPanel struct {
	c         *core
	entries   []storage.NoteEntry
	list      listState
	expanded  map[string]bool
	selected  map[string]bool
	tagFilter string
}

func newNotesPanel(c *core) *notesPanel {
	p := &notesPanel{c: c, expanded: map[string]bool{}, selected: map[string]bool{}}
	p.reload()
	return p
}

// reload relee el árbol (o las notas del tag filtrado) conservando el cursor
// sobre la misma ruta cuando sigue existiendo.
func (p *notesPanel) reload() {
	keep := ""
	if e := p.current(); e != nil {
		keep = e.Path
	}
	p.c.reload()
	if p.tagFilter != "" {
		p.entries = nil
		for _, n := range views.NotesForTag(p.c.notes, p.tagFilter) {
			note := n
			p.entries = append(p.entries, storage.NoteEntry{Type: storage.EntryNote, Name: n.ID, Path: n.Path, Note: &note, ModTime: n.ModTime})
		}
	} else if entries, err := p.c.store.ListTreeEntries(p.expanded); err == nil {
		p.entries = entries
	}
	if keep == "" || !p.selectPath(keep) {
		p.list.set(p.list.cursor, len(p.entries))
	}
}

func (p *notesPanel) current() *storage.NoteEntry {
	if p.list.cursor < len(p.entries) {
		return &p.entries[p.list.cursor]
	}
	return nil
}

// currentNote devuelve la nota bajo el cursor (nil si es una carpeta).
func (p *notesPanel) currentNote() *storage.Note {
	if e := p.current(); e != nil && e.Type == storage.EntryNote {
		return e.Note
	}
	return nil
}

// selectPath pone el cursor sobre path; devuelve false si no está en la lista.
func (p *notesPanel) selectPath(path string) bool {
	for i, e := range p.entries {
		if e.Path == path {
			p.list.set(i, len(p.entries))
			return true
		}
	}
	return false
}

// targetDir es la carpeta donde se crea algo nuevo: la carpeta bajo el cursor
// o la carpeta de la nota bajo el cursor.
func (p *notesPanel) targetDir() string {
	e := p.current()
	switch {
	case e == nil:
		return p.c.store.BaseDir
	case e.Type == storage.EntryFolder:
		p.expanded[e.Path] = true
		return e.Path
	default:
		return filepath.Dir(e.Path)
	}
}

func (p *notesPanel) setFilter(tag string) {
	if p.tagFilter == tag {
		tag = ""
	}
	p.tagFilter = tag
	p.list = listState{}
	p.reload()
	if tag == "" {
		p.c.setStatus("%s", i18n.T("Filtro de categoría quitado", "Category filter cleared"))
		return
	}
	p.c.setStatus(i18n.T("Filtrando por #%s (%d notas)", "Filtering by #%s (%d notes)"), tag, len(p.entries))
}

func (p *notesPanel) toggleFolder(path string) {
	// Una carpeta que no está en el mapa se muestra abierta.
	open, ok := p.expanded[path]
	p.expanded[path] = ok && !open
	p.reload()
	p.selectPath(path)
}

func (p *notesPanel) key(a Action) tea.Cmd {
	n := len(p.entries)
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
		e := p.current()
		if e == nil {
			return p.promptCreate(false)
		}
		if e.Type == storage.EntryFolder {
			p.toggleFolder(e.Path)
			return nil
		}
		return p.c.openEditor(e.Path, 1)
	case actEdit:
		if note := p.currentNote(); note != nil {
			return p.c.openEditor(note.Path, 1)
		}
	case actNewNote:
		return p.promptCreate(false)
	case actNewFromTemplate:
		return p.promptFromTemplate()
	case actNewFolder:
		return p.promptCreate(true)
	case actRename:
		return p.promptRename()
	case actMove:
		return p.promptMove()
	case actDelete:
		return p.promptDelete()
	case actSelect:
		if e := p.current(); e != nil && e.Type == storage.EntryNote {
			if p.selected[e.Path] {
				delete(p.selected, e.Path)
			} else {
				p.selected[e.Path] = true
			}
			p.list.move(1, n)
		}
	case actSelectAll:
		p.selectAll()
	case actEscape:
		if len(p.selected) > 0 {
			p.selected = map[string]bool{}
			return nil
		}
		if p.tagFilter != "" {
			p.setFilter("")
		}
	}
	return nil
}

func (p *notesPanel) selectAll() {
	all := true
	for _, e := range p.entries {
		if e.Type == storage.EntryNote && !p.selected[e.Path] {
			all = false
		}
	}
	p.selected = map[string]bool{}
	if !all {
		for _, e := range p.entries {
			if e.Type == storage.EntryNote {
				p.selected[e.Path] = true
			}
		}
	}
}

// click maneja un clic en la fila row del contenido (0 = primera fila visible).
func (p *notesPanel) click(row int, double bool) tea.Cmd {
	i := p.list.offset + row
	if row < 0 || i >= len(p.entries) {
		return nil
	}
	again := i == p.list.cursor
	p.list.set(i, len(p.entries))
	e := p.entries[i]
	if e.Type == storage.EntryFolder {
		if again || double {
			p.toggleFolder(e.Path)
		}
		return nil
	}
	if double {
		return p.c.openEditor(e.Path, 1)
	}
	return nil
}

func (p *notesPanel) promptCreate(folder bool) tea.Cmd {
	dir := p.targetDir()
	if folder {
		p.c.push(newInputPopup(i18n.T("Nueva carpeta", "New folder"), i18n.T("carpeta", "folder"), func(name string) tea.Cmd {
			path, err := p.c.store.CreateFolderInDir(dir, name)
			if err != nil {
				p.c.errStatus("No se pudo crear la carpeta", "Could not create folder", err)
				return nil
			}
			p.expanded[path] = true
			p.reload()
			p.selectPath(path)
			p.c.setStatus(i18n.T("Carpeta creada: %s", "Folder created: %s"), filepath.Base(path))
			return nil
		}))
		return nil
	}
	suggest := fmt.Sprintf("%s %d", i18n.T("Nueva nota", "New note"), len(p.c.notes)+1)
	p.c.push(newInputPopup(i18n.T("Nueva nota", "New note"), suggest, func(name string) tea.Cmd {
		note, err := p.c.store.CreateNoteInDir(dir, name)
		if err != nil {
			p.c.errStatus("No se pudo crear la nota", "Could not create note", err)
			return nil
		}
		p.reload()
		p.selectPath(note.Path)
		p.c.setStatus(i18n.T("Nota creada: %s", "Note created: %s"), note.ID)
		return nil
	}))
	return nil
}

func (p *notesPanel) promptRename() tea.Cmd {
	e := p.current()
	if e == nil {
		return nil
	}
	old := strings.TrimSuffix(e.Name, ".md")
	path := e.Path
	isNote := e.Type == storage.EntryNote
	p.c.push(newInputPopup(i18n.T("Renombrar", "Rename"), old, func(name string) tea.Cmd {
		rename := func(edits []links.Edit) {
			newPath, err := p.c.store.Rename(path, name)
			if err != nil {
				p.c.errStatus("No se pudo renombrar", "Could not rename", err)
				return
			}
			if p.expanded[path] {
				p.expanded[newPath] = true
			}
			delete(p.selected, path)
			status := fmt.Sprintf(i18n.T("Renombrado a %s", "Renamed to %s"), filepath.Base(newPath))
			if len(edits) > 0 {
				done, failed := p.c.applyLinkEdits(edits, path, newPath)
				status += fmt.Sprintf(" · "+i18n.T("%d línea(s) con enlaces actualizada(s)", "%d line(s) with links updated"), done)
				if failed > 0 {
					status += fmt.Sprintf(" · "+i18n.T("%d no se pudo(ieron)", "%d could not be updated"), failed)
				}
			}
			p.c.reload()
			p.reload()
			p.selectPath(newPath)
			p.c.setStatus("%s", status)
		}
		// si otras notas enlazan a esta con [[wikilinks]], se ofrece actualizarlos (con el detalle de lo que cambia)
		if isNote && p.c.links != nil {
			if edits := p.c.links.RenameEdits(path, storage.Slug(name)); len(edits) > 0 && storage.Slug(name) != strings.ToLower(old) {
				p.c.push(newRenameLinksPopup(p.c.store.BaseDir, edits, func(update bool) tea.Cmd {
					if update {
						rename(edits)
					} else {
						rename(nil)
					}
					return nil
				}))
				return nil
			}
		}
		rename(nil)
		return nil
	}))
	return nil
}

// targets devuelve las notas a operar: la selección múltiple o la entrada actual.
func (p *notesPanel) targets() []string {
	if len(p.selected) > 0 {
		var out []string
		for _, e := range p.entries {
			if p.selected[e.Path] {
				out = append(out, e.Path)
			}
		}
		for path := range p.selected {
			if !contains(out, path) {
				out = append(out, path)
			}
		}
		return out
	}
	if e := p.current(); e != nil {
		return []string{e.Path}
	}
	return nil
}

func (p *notesPanel) promptMove() tea.Cmd {
	paths := p.targets()
	if len(paths) == 0 {
		return nil
	}
	folders, err := p.c.store.ListFolders()
	if err != nil {
		p.c.errStatus("No se pudieron listar las carpetas", "Could not list folders", err)
		return nil
	}
	p.c.push(newMovePopup(p.c.store.BaseDir, folders, len(paths), func(dest string) tea.Cmd {
		moved := 0
		var lastErr error
		for _, path := range paths {
			if err := p.c.store.MoveNote(path, dest); err != nil {
				lastErr = err
				continue
			}
			moved++
		}
		p.selected = map[string]bool{}
		p.expanded[dest] = true
		p.reload()
		if len(paths) == 1 && moved == 1 {
			p.selectPath(filepath.Join(dest, filepath.Base(paths[0])))
		}
		if lastErr != nil {
			p.c.errStatus("Algunas notas no se movieron", "Some notes were not moved", lastErr)
			return nil
		}
		p.c.setStatus(i18n.T("%d elemento(s) movido(s)", "%d item(s) moved"), moved)
		return nil
	}))
	return nil
}

func (p *notesPanel) promptDelete() tea.Cmd {
	paths := p.targets()
	if len(paths) == 0 {
		return nil
	}
	trash := func() tea.Cmd {
		for _, path := range paths {
			if _, err := p.c.store.MoveToTrash(path); err != nil {
				p.c.errStatus("No se pudo borrar", "Could not delete", err)
			}
			delete(p.selected, path)
			delete(p.expanded, path)
		}
		p.reload()
		p.c.setStatus(i18n.T("%d elemento(s) a la papelera", "%d item(s) moved to trash"), len(paths))
		return nil
	}
	title := i18n.T("Borrar", "Delete")
	if len(paths) > 1 {
		return p.c.confirm(title, fmt.Sprintf(i18n.T("¿Mover %d notas a la papelera?", "Move %d notes to trash?"), len(paths)), false, trash)
	}
	e := p.current()
	if e != nil && e.Type == storage.EntryFolder {
		items := p.c.store.CountFolderItems(e.Path)
		if items > 0 {
			// Borrar una carpeta con contenido siempre pide confirmación (H1-3).
			msg := fmt.Sprintf(i18n.T("La carpeta '%s' tiene %d elemento(s). ¿Moverla a la papelera con todo su contenido?", "Folder '%s' has %d item(s). Move it to trash with all its content?"), e.Name, items)
			return p.c.confirm(title, msg, true, trash)
		}
		return trash()
	}
	return p.c.confirm(title, fmt.Sprintf(i18n.T("¿Mover '%s' a la papelera?", "Move '%s' to trash?"), filepath.Base(paths[0])), false, trash)
}

func (p *notesPanel) title() string {
	t := i18n.T("[1]─Notas", "[1]─Notes")
	if p.tagFilter != "" {
		t += " #" + p.tagFilter
	}
	return t
}

func (p *notesPanel) view(r Rect, active bool) string {
	h := r.H - 2
	from, to := p.list.visible(h, len(p.entries))
	var lines []string
	if len(p.entries) == 0 {
		empty := i18n.T("Sin notas. 'c': nueva nota, 'F': carpeta", "No notes. 'c': new note, 'F': folder")
		if p.tagFilter != "" {
			empty = fmt.Sprintf(i18n.T("Sin notas con #%s. Esc: quitar filtro", "No notes with #%s. Esc: clear filter"), p.tagFilter)
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Italic(true).Render(empty))
	}
	for i := from; i < to; i++ {
		lines = append(lines, p.row(p.entries[i], r.W-2, i == p.list.cursor, active))
	}
	footer := counter(p.list.cursor, len(p.entries))
	if len(p.selected) > 0 {
		footer = fmt.Sprintf("%d sel · %s", len(p.selected), footer)
	}
	return theme.RenderPanel(p.title(), footer, lines, r.W, r.H, active)
}

// row compone una fila del árbol con ancho exacto w: el nombre se corta con
// "…" y la fecha (o el conteo de la carpeta) queda alineada a la derecha.
func (p *notesPanel) row(e storage.NoteEntry, w int, cursor, active bool) string {
	indent := textwidth.Repeat("  ", e.Depth)
	var icon, right string
	iconStyle := lipgloss.NewStyle().Foreground(theme.ColorTeal)
	if e.Type == storage.EntryFolder {
		icon = iconFolderClosed
		if e.Expanded {
			icon = iconFolderOpen
		}
		iconStyle = lipgloss.NewStyle().Foreground(theme.ColorPeach)
		right = fmt.Sprintf("%d", e.Children)
	} else {
		icon = iconNote
		right = e.ModTime.Format("02 Jan")
	}
	mark := " "
	if p.selected[e.Path] {
		mark = lipgloss.NewStyle().Foreground(theme.ColorGreen).Bold(true).Render(iconChecked)
	}
	left := mark + indent + iconStyle.Render(icon) + " " + lipgloss.NewStyle().Foreground(theme.ColorText).Render(textwidth.NoControl(e.Name)) // un nombre de archivo no es de fiar
	return listRow(left, right, w, cursor, active)
}

// listRow une una parte izquierda (cortada con "…") y una columna derecha fija
// en exactamente w celdas. La fila bajo el cursor lleva la barra de selección
// sólida de ancho completo.
func listRow(left, right string, w int, cursor, active bool) string {
	if textwidth.Width(right) > w/3 {
		right = textwidth.Truncate(right, w/3, textwidth.Ellipsis)
	}
	rw := textwidth.Width(right)
	if right == "" || w-rw-1 < 12 {
		right, rw = "", 0
	}
	leftW := w - rw
	if rw > 0 {
		leftW--
	}
	line := textwidth.Fit(left, leftW)
	if rw > 0 {
		line += " " + lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(right)
	}
	if !cursor {
		return line
	}
	bg := theme.ColorSurface0
	if active {
		bg = theme.ColorSurface1
		line = lipgloss.NewStyle().Bold(true).Render(line)
	}
	return theme.Paint(line, bg)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// promptFromTemplate elige una plantilla de templates/ y pide el nombre de la nota nueva, que se crea con su contenido.
func (p *notesPanel) promptFromTemplate() tea.Cmd {
	names := p.c.store.Templates()
	if len(names) == 0 {
		p.c.setStatus("%s", i18n.T("No hay plantillas: crea notas en templates/", "No templates: create notes in templates/"))
		return nil
	}
	dir := p.targetDir()
	if p.c.store.InTemplates(dir) { // una nota nueva dentro de templates/ sería una plantilla, no una nota: va a la raíz
		dir = p.c.store.BaseDir
	}
	p.c.push(newTemplatePopup(names, func(tpl string) tea.Cmd {
		suggest := fmt.Sprintf("%s %d", i18n.T("Nueva nota", "New note"), len(p.c.notes)+1)
		p.c.push(newInputPopup(i18n.T("Nueva nota", "New note")+" · "+tpl, suggest, func(name string) tea.Cmd {
			note, err := p.c.store.CreateNoteFromTemplate(dir, name, tpl, time.Now())
			if err != nil {
				p.c.errStatus("No se pudo crear la nota", "Could not create note", err)
				return nil
			}
			p.c.reload()
			p.reload()
			p.selectPath(note.Path)
			if !p.c.warnUnknown(note.Warnings) {
				p.c.setStatus(i18n.T("Nota creada: %s", "Note created: %s"), note.ID)
			}
			return nil
		}))
		return nil
	}))
	return nil
}

// openDaily abre la nota de hoy (journal/AAAA-MM-DD.md), creándola con la plantilla daily si no existe.
func (m *AppModel) openDaily() tea.Cmd {
	note, created, err := m.c.store.DailyNote(time.Now())
	if err != nil {
		m.c.errStatus("No se pudo abrir la nota diaria", "Could not open the daily note", err)
		return nil
	}
	m.c.reload()
	m.jumpToPath(note.Path, 1)
	if m.c.warnUnknown(note.Warnings) {
		// la nota se creó igual; el aviso de la variable desconocida pasa por delante del de creada
	} else if created {
		m.c.setStatus(i18n.T("Nota diaria creada: %s", "Daily note created: %s"), note.ID)
	} else {
		m.c.setStatus(i18n.T("Nota diaria: %s", "Daily note: %s"), note.ID)
	}
	return nil
}

// warnUnknown avisa en la barra de las {{variables}} desconocidas de la plantilla con que se creó una nota (la nota se creó con ellas tal
// cual). Devuelve si avisó.
func (c *core) warnUnknown(unknown []string) bool {
	if len(unknown) == 0 {
		return false
	}
	c.setStatus(i18n.T("Variable desconocida: %s", "Unknown variable: %s"), textwidth.NoControl(strings.Join(unknown, ", ")))
	return true
}
