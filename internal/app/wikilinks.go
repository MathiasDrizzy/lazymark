package app

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/links"
	"github.com/MathiasDrizzy/lazymark/internal/search"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// Wikilinks en la vista previa (H5-2). Los `[[nota]]` del texto se cambian, ANTES de pasárselo a Glamour, por su texto visible entre dos
// marcadores de uso privado; después de que Glamour ajustó las líneas, cada marcador se cambia por el estilo del enlace (azul subrayado si
// la nota existe, durazno si no; resaltado el seleccionado) y se anotan sus posiciones en pantalla para el clic. Los marcadores no se
// ven nunca: las demás funciones (tareas, búsqueda) leen las líneas sin ellos. Al final de la vista previa va la lista de backlinks.
const (
	linkOpen  = ''
	linkClose = ''
)

// previewLink es un enlace de la vista previa: un wikilink del texto o una entrada de backlinks.
type previewLink struct {
	Display string
	Target  string // lo que está escrito, sin alias ni anchor ("" si apunta a un encabezado de la misma nota)
	Anchor  string
	Path    string // la nota a la que resuelve, "" si no existe
	Line    int    // línea a la que se salta en la nota de destino (los backlinks, la del enlace)
	Back    bool
	Dir     string // carpeta de la nota que lo contiene: ahí se crea la nota si falta
}

func (l previewLink) missing() bool { return l.Path == "" }

// markLinks cambia los wikilinks de content (fuera de código) por su texto entre marcadores y devuelve los enlaces en el mismo orden.
func markLinks(content string, ix *links.Index, from string) (string, []previewLink) {
	ls := links.Parse(content)
	if len(ls) == 0 {
		return content, nil
	}
	lines := strings.Split(content, "\n")
	out := make([]previewLink, len(ls))
	for i := len(ls) - 1; i >= 0; i-- { // de atrás hacia adelante: los desplazamientos anteriores siguen valiendo
		l := ls[i]
		pl := previewLink{Display: l.Display(), Target: l.Target, Anchor: l.Anchor, Line: 1, Dir: filepath.Dir(from)}
		if n, ok := ix.Resolve(l, from); ok {
			pl.Path = n.Path
		}
		out[i] = pl
		line := lines[l.Line-1]
		lines[l.Line-1] = line[:l.Start] + string(linkOpen) + pl.Display + string(linkClose) + line[l.End:]
	}
	return strings.Join(lines, "\n"), out
}

// backlinkLines arma la sección de backlinks que va al final de la vista previa: un título y una línea por cada enlace, con el nombre de
// la nota enlazante como enlace.
func backlinkLines(back []links.Backlink, width int) ([]string, []previewLink) {
	if len(back) == 0 {
		return nil, nil
	}
	head := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("── "+i18n.T("Enlaces a esta nota (%d)", "Links to this note (%d)")+" ", len(back)) + strings.Repeat("─", max(2, width/3)))
	out := []string{"", head}
	var pls []previewLink
	dim := lipgloss.NewStyle().Foreground(theme.ColorOverlay0)
	for _, b := range back {
		title := textwidth.NoControl(b.Note.Title) // el nombre de la nota y su texto no son de fiar: sin caracteres de control hacia la terminal
		pls = append(pls, previewLink{Display: title, Path: b.Note.Path, Line: b.Line, Back: true})
		ctx := textwidth.Truncate(textwidth.NoControl(links.Plain(b.Text)), max(8, width-textwidth.Width(title)-12), "…")
		out = append(out, "  "+string(linkOpen)+title+string(linkClose)+dim.Render(fmt.Sprintf(":%d  %s", b.Line, ctx)))
	}
	return out, pls
}

// stripLinkMarks quita los marcadores de una línea.
func stripLinkMarks(s string) string {
	if !strings.ContainsAny(s, string(linkOpen)+string(linkClose)) {
		return s
	}
	return strings.NewReplacer(string(linkOpen), "", string(linkClose), "").Replace(s)
}

// linkHit es una zona de la pantalla (relativa al borde del panel) que pertenece al enlace k.
type linkHit struct{ y, x0, x1, k int }

// sgr devuelve las secuencias que abren y cierran el estilo (el truco: se pinta un marcador y se parte lo que lo rodea).
func sgr(st lipgloss.Style) (on, off string) {
	parts := strings.SplitN(st.Render("\x00"), "\x00", 2)
	return parts[0], parts[1]
}

// decorateLink cambia los marcadores de la línea i por el estilo de cada enlace y anota sus zonas. open es el enlace que ya estaba abierto al
// empezar la línea (-1 si ninguno) y next el índice del próximo que se abre. Devuelve la línea.
func (p *previewPanel) decorateLink(line string, open, next, row int, hits *[]linkHit) string {
	style := func(k int) (string, string) {
		st := lipgloss.NewStyle().Underline(true)
		switch l := p.links[k]; {
		case p.sel == k+1:
			st = st.Bold(true).Foreground(theme.ColorBase).Background(theme.ColorBlue)
		case l.missing():
			st = st.Foreground(theme.ColorPeach)
		default:
			st = st.Foreground(theme.ColorBlue)
		}
		on, _ := sgr(st)
		textOn, _ := sgr(lipgloss.NewStyle().Foreground(theme.ColorText))
		return on, "\x1b[22;24;39;49m" + textOn // al cerrar: sin negrita, subrayado ni colores, y el color de texto del tema
	}
	var b strings.Builder
	col := 0
	cur := open
	zone := func(k, from, to int) {
		if to > from {
			*hits = append(*hits, linkHit{y: row, x0: from, x1: to, k: k})
		}
	}
	start := 0
	if cur >= 0 {
		on, _ := style(cur)
		b.WriteString(on)
	}
	for i := 0; i < len(line); {
		if line[i] == 0x1b { // secuencia ANSI: se copia tal cual
			j := i + 1
			if j < len(line) && line[j] == '[' {
				j++
				for j < len(line) && !(line[j] >= 0x40 && line[j] <= 0x7e) {
					j++
				}
				j++
			}
			b.WriteString(line[i:min(j, len(line))])
			i = min(j, len(line))
			continue
		}
		r, size := rune(0), 0
		for _, c := range line[i:] {
			r, size = c, len(string(c))
			break
		}
		switch r {
		case linkOpen:
			cur = next
			next++
			start = col
			if cur < len(p.links) {
				on, _ := style(cur)
				b.WriteString(on)
			}
		case linkClose:
			if cur >= 0 && cur < len(p.links) {
				_, off := style(cur)
				b.WriteString(off)
				zone(cur, start, col)
			}
			cur = -1
		default:
			b.WriteString(line[i : i+size])
			col += textwidth.Width(line[i : i+size])
		}
		i += size
	}
	if cur >= 0 && cur < len(p.links) { // el enlace sigue abierto en la línea de abajo: se cierra aquí y se reabre allá
		_, off := style(cur)
		b.WriteString(off)
		zone(cur, start, col)
	}
	return b.String()
}

// ─── seguir un enlace ──────────────────────────────────────────────

// followLink va a la nota del enlace k de la vista previa; si no existe, ofrece crearla.
func (m *AppModel) followLink(k int) tea.Cmd {
	if k < 0 || k >= len(m.preview.links) {
		return nil
	}
	l := m.preview.links[k]
	if !l.missing() {
		line := l.Line
		if n := m.noteByPath(l.Path); n != nil && l.Anchor != "" && !l.Back {
			line = headingLine(n, l.Anchor)
		}
		m.jumpToPath(l.Path, line)
		return nil
	}
	name := strings.TrimSpace(l.Target)
	if name == "" {
		m.c.setStatus("%s", i18n.T("El encabezado no existe en esta nota", "The heading does not exist in this note"))
		return nil
	}
	m.c.push(newConfirmPopup(i18n.T("Crear nota", "Create note"),
		fmt.Sprintf(i18n.T("La nota «%s» no existe.\n¿Quieres crearla?", "The note \"%s\" does not exist.\nDo you want to create it?"), name),
		func() tea.Cmd {
			dir := l.Dir
			if strings.Contains(name, "/") {
				// un enlace de ruta ([[carpeta/nota]]) crea la nota en esa carpeta, y la carpeta si no existe, siempre dentro de la carpeta de
				// notas (EnsureFolder rechaza ".." y rutas absolutas y no sigue enlaces simbólicos hacia fuera: `[[../../x]]` no escribe
				// fuera); si la ruta no vale, la crea junto a la nota que contiene el enlace
				if real, err := m.c.store.EnsureFolder(path.Dir(name)); err == nil {
					dir = real
				}
				name = filepath.Base(filepath.FromSlash(name))
			}
			n, err := m.c.store.CreateNoteInDir(dir, name)
			if err != nil {
				m.c.errStatus("No se pudo crear la nota", "Could not create the note", err)
				return nil
			}
			m.c.reload()
			m.afterChange()
			m.jumpToPath(n.Path, 1)
			m.c.setStatus(i18n.T("Nota creada: %s", "Note created: %s"), filepath.Base(n.Path))
			return nil
		}))
	return nil
}

// headingLine devuelve la línea de la nota donde está el encabezado (o el bloque ^id) del anchor; 1 si no lo encuentra.
func headingLine(n *storage.Note, anchor string) int {
	want := strings.ToLower(strings.TrimSpace(anchor))
	fence := "" // lo que está en un bloque de código no es un encabezado ni un bloque de la nota
	for i, l := range strings.Split(n.Content, "\n") {
		t := strings.TrimSpace(l)
		if f := storage.FenceMarker(t); f != "" {
			switch {
			case fence == "":
				fence = f
			case f[0] == fence[0] && len(f) >= len(fence) && strings.Trim(t, f[:1]) == "":
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if strings.HasPrefix(want, "^") {
			if strings.HasSuffix(t, " "+anchor) || t == anchor {
				return i + 1
			}
			continue
		}
		if strings.HasPrefix(t, "#") && strings.ToLower(strings.TrimSpace(strings.TrimLeft(t, "#"))) == want {
			return i + 1
		}
	}
	return 1
}

// jumpToPath lleva la vista a la nota path y a su línea (como la búsqueda).
func (m *AppModel) jumpToPath(path string, line int) {
	rel, _ := filepath.Rel(m.c.store.BaseDir, path)
	m.jumpTo(search.Match{Path: path, Rel: filepath.ToSlash(rel), Line: line})
}

// moveLink selecciona el enlace siguiente (d=1) o anterior (d=-1) de la vista previa y lo deja a la vista.
func (m *AppModel) moveLink(d int) {
	p := &m.preview
	n := len(p.links)
	if n == 0 {
		m.c.setStatus("%s", i18n.T("Esta nota no tiene enlaces", "This note has no links"))
		return
	}
	if p.sel == 0 {
		if d > 0 {
			p.sel = 1
		} else {
			p.sel = n
		}
	} else {
		p.sel = (p.sel-1+d+n)%n + 1
	}
	h := m.layout.Preview.H - 2
	if line := p.linkLine[p.sel-1]; line < p.scrollY || line >= p.scrollY+h {
		p.scrollY = max(0, line-h/3)
	}
}

// ─── renombrar con enlaces ─────────────────────────────────────────

// applyLinkEdits escribe las ediciones (línea por línea, solo si la línea sigue siendo la que se vio) después de renombrar la nota old a
// renamed; las de la propia nota renombrada van a su ruta nueva. Devuelve cuántas se aplicaron y cuántas no.
func (c *core) applyLinkEdits(edits []links.Edit, old, renamed string) (done, failed int) {
	for _, e := range edits {
		path := e.Path
		if path == old {
			path = renamed
		}
		if err := c.store.ReplaceLineIf(path, e.Line, e.Before, e.After, time.Time{}); err != nil {
			failed++
			continue
		}
		done++
	}
	return done, failed
}

// renameLinksPopup muestra qué enlaces cambian al renombrar una nota y deja elegir: actualizarlos (y), renombrar sin tocarlos (n) o
// cancelar (Esc).
type renameLinksPopup struct {
	base   string
	edits  []links.Edit
	choose func(update bool) tea.Cmd
}

func newRenameLinksPopup(base string, edits []links.Edit, choose func(bool) tea.Cmd) *renameLinksPopup {
	return &renameLinksPopup{base: base, edits: edits, choose: choose}
}

func (p *renameLinksPopup) contexts() []Context            { return nil }
func (p *renameLinksPopup) bottomRight() bool              { return false }
func (p *renameLinksPopup) click(int, int) (tea.Cmd, bool) { return nil, false }

func (p *renameLinksPopup) handle(_ Action, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "y", "enter":
		return p.choose(true), true
	case "n":
		return p.choose(false), true
	}
	return nil, false
}

func (p *renameLinksPopup) render(l Layout) string {
	w := popupWidth(l, 90)
	files := map[string]bool{}
	for _, e := range p.edits {
		files[e.Path] = true
	}
	lines := []string{lipgloss.NewStyle().Foreground(theme.ColorText).Render(fmt.Sprintf(i18n.T("%d línea(s) de %d nota(s) enlazan a esta nota:", "%d line(s) in %d note(s) link to this note:"), len(p.edits), len(files))), ""}
	room := max(4, l.H-12)
	shown := 0
	red, green := lipgloss.NewStyle().Foreground(theme.ColorRed), lipgloss.NewStyle().Foreground(theme.ColorGreen)
	for _, e := range p.edits {
		if shown+3 > room {
			lines = append(lines, dim(fmt.Sprintf("… "+i18n.T("y %d más", "and %d more"), len(p.edits)-shown/3)))
			break
		}
		rel, _ := filepath.Rel(p.base, e.Path)
		lines = append(lines,
			dim(fmt.Sprintf("%s:%d", filepath.ToSlash(rel), e.Line)),
			red.Render(textwidth.Truncate("- "+strings.TrimSpace(strings.TrimSuffix(e.Before, "\r")), w-4, "…")),
			green.Render(textwidth.Truncate("+ "+strings.TrimSpace(strings.TrimSuffix(e.After, "\r")), w-4, "…")))
		shown += 3
	}
	lines = append(lines, "", accent("[y] "+i18n.T("Actualizar los enlaces", "Update the links"))+"   "+dim("[n] "+i18n.T("Solo renombrar", "Just rename")))
	return theme.RenderPopup(i18n.T("Renombrar", "Rename"), escHint, lines, w)
}
