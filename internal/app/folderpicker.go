package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// folderPickerPopup deja elegir una carpeta recorriendo el disco: Enter entra en
// la carpeta marcada, ".." sube y `s` elige la carpeta en la que se está.
type folderPickerPopup struct {
	popupList
	dir     string
	subdirs []string
	onPick  func(path string) tea.Cmd
	err     string
	// fila del popup donde está el botón "Usar esta carpeta"
	buttonRow, buttonEnd int
}

func newFolderPicker(start string, onPick func(path string) tea.Cmd) *folderPickerPopup {
	p := &folderPickerPopup{onPick: onPick}
	p.open(start)
	return p
}

// open lista las subcarpetas visibles de dir (sin las ocultas).
func (p *folderPickerPopup) open(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		p.err = err.Error()
		return
	}
	p.err = ""
	p.dir = dir
	p.subdirs = nil
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if isDir(filepath.Join(dir, e.Name())) {
			p.subdirs = append(p.subdirs, e.Name())
		}
	}
	sort.Slice(p.subdirs, func(i, j int) bool { return strings.ToLower(p.subdirs[i]) < strings.ToLower(p.subdirs[j]) })
	p.n = len(p.subdirs)
	if p.hasParent() {
		p.n++ // la fila ".."
	}
	p.list = listState{}
}

func isDir(path string) bool {
	fi, err := os.Stat(path) // sigue los enlaces simbólicos
	return err == nil && fi.IsDir()
}

func (p *folderPickerPopup) hasParent() bool { return filepath.Dir(p.dir) != p.dir }

// name devuelve el nombre de la fila i ("..", o una subcarpeta).
func (p *folderPickerPopup) name(i int) string {
	if p.hasParent() {
		if i == 0 {
			return ".."
		}
		i--
	}
	return p.subdirs[i]
}

// enter entra en la carpeta de la fila i (o sube si es "..").
func (p *folderPickerPopup) enter(i int) {
	if i < 0 || i >= p.n {
		return
	}
	n := p.name(i)
	if n == ".." {
		prev := filepath.Base(p.dir)
		p.open(filepath.Dir(p.dir))
		for j := 0; j < p.n; j++ { // deja el cursor en la carpeta de la que se vino
			if p.name(j) == prev {
				p.list.set(j, p.n)
			}
		}
		return
	}
	p.open(filepath.Join(p.dir, n))
}

func (p *folderPickerPopup) contexts() []Context { return []Context{ctxPopup, ctxNav} }
func (p *folderPickerPopup) bottomRight() bool   { return false }

func (p *folderPickerPopup) handle(a Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if p.nav(a) {
		return nil, false
	}
	switch {
	case a == actConfirm || a == actRight:
		p.enter(p.list.cursor)
	case a == actLeft && p.hasParent():
		p.enter(0)
	case msg.String() == "s":
		return p.onPick(p.dir), true
	}
	return nil, false
}

// click: el primer clic en una fila la marca y el segundo entra; el botón elige.
func (p *folderPickerPopup) click(x, y int) (tea.Cmd, bool) {
	if y == p.buttonRow && x <= p.buttonEnd {
		return p.onPick(p.dir), true
	}
	if i, again, ok := p.clickRow(y); ok && again {
		p.enter(i)
	}
	return nil, false
}

func (p *folderPickerPopup) render(l Layout) string {
	w := popupWidth(l, 60)
	p.top = 3
	p.height = clamp(p.n, 1, max(1, l.H-13))
	lines := []string{dim(leftTruncate(p.dir, w-4)), ""}
	if p.err != "" {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(theme.ColorRed).Render(textwidth.Truncate(p.err, w-6, textwidth.Ellipsis)))
	} else if p.n == 0 {
		lines = append(lines, "  "+dim(i18n.T("(sin subcarpetas)", "(no subfolders)")))
	} else {
		lines = append(lines, p.rows(w-3, func(i int) string {
			if p.name(i) == ".." {
				return lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(".. " + i18n.T("(subir)", "(up)"))
			}
			return lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(iconFolderClosed) + " " + lipgloss.NewStyle().Foreground(theme.ColorText).Render(p.name(i))
		})...)
	}
	lines = append(lines, "")
	btn := "[s] " + i18n.T("Usar esta carpeta", "Use this folder")
	p.buttonRow, p.buttonEnd = len(lines)+1, textwidth.Width(btn)+2
	lines = append(lines, accent(btn))
	return theme.RenderPopup(i18n.T("Carpeta de notas", "Notes folder"), "[Enter] "+i18n.T("abrir", "open")+" · "+escHint, lines, w)
}

// leftTruncate acorta una ruta por la izquierda para que quepa en w celdas.
func leftTruncate(s string, w int) string {
	if textwidth.Width(s) <= w || w < 2 {
		return s
	}
	total := textwidth.Width(s)
	return textwidth.Ellipsis + textwidth.Cut(s, total-(w-1), total)
}

// shortPath muestra la ruta con "~" en vez de la carpeta personal.
func shortPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if rel == "." {
				return "~"
			}
			return "~" + string(filepath.Separator) + rel
		}
	}
	return path
}

// applyNotesDir cambia la carpeta de notas sin reiniciar: cambia el almacén,
// reinicia el estado de los paneles, recarga el árbol y guarda la ruta.
func (m *AppModel) applyNotesDir(path string) tea.Cmd {
	abs, err := filepath.Abs(path)
	if err != nil {
		m.c.errStatus("Carpeta no válida", "Invalid folder", err)
		return nil
	}
	if !isDir(abs) {
		m.c.errStatus("Carpeta no válida", "Invalid folder", i18n.Errorf("%s no es una carpeta", "%s is not a folder", abs))
		return nil
	}
	store := newStore(abs)
	if _, err := store.ListNotes(); err != nil {
		m.c.errStatus("No se pudo leer la carpeta", "Could not read the folder", err)
		return nil
	}
	m.c.cfg.SetNotesDir(abs) // elección explícita: pasa a ser la carpeta por defecto
	m.c.store = store
	m.c.save()

	m.notes = newNotesPanel(m.c)
	m.tasks.list, m.tags.list = listState{}, listState{}
	m.kanban.selected = nil
	m.preview.reset()
	m.imgNote = ""
	m.c.kitty.Reset()
	m.relayout()
	m.c.setStatus(i18n.T("Carpeta de notas: %s (%d notas)", "Notes folder: %s (%d notes)"), shortPath(abs), len(m.c.notes))
	return nil
}

// openNotesPicker abre el selector de carpeta partiendo de la carpeta actual.
func (m *AppModel) openNotesPicker() {
	m.c.push(newFolderPicker(m.c.store.BaseDir, m.applyNotesDir))
}
