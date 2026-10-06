package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// El perezoso dormido (variante c2 de la marca) aparece en los estados de reposo de la vista previa:
// la carpeta de notas vacía, una carpeta sin notas, una nota vacía y nada seleccionado. Es chico y va
// abajo a la derecha del panel, sin tapar el texto (ver mascot.go); el ajuste "Mascota" lo apaga.

// Estados de reposo.
const (
	restNone      = "none"      // nada seleccionado
	restFolder    = "folder"    // una carpeta sin notas
	restEmptyNote = "emptynote" // una nota sin contenido
	restWelcome   = "welcome"   // la carpeta de notas está vacía
)

// restKind dice qué estado de reposo muestra la vista previa ("" si muestra otra cosa). Sigue
// la misma elección que renderPreview.
func (m *AppModel) restKind() string {
	switch m.lastLeft {
	case panelTags:
		return ""
	case panelTasks:
		if n := m.previewNote(); n == nil {
			return restNone
		} else if noteIsEmpty(n.Content) {
			return restEmptyNote
		}
		return ""
	}
	if e := m.notes.current(); e != nil && e.Type == storage.EntryFolder {
		prefix := e.Path + "/"
		for _, n := range m.c.notes {
			if strings.HasPrefix(filepathSlash(n.Path), filepathSlash(prefix)) {
				return ""
			}
		}
		return restFolder
	}
	if n := m.notes.currentNote(); n != nil {
		if noteIsEmpty(n.Content) {
			return restEmptyNote
		}
		return ""
	}
	if len(m.notes.entries) == 0 && m.notes.tagFilter == "" {
		return restWelcome
	}
	return restNone
}

func filepathSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

// restMessage devuelve las líneas de texto que acompañan al perezoso.
func (m *AppModel) restMessage(kind string) []string {
	dim := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Italic(true)
	key := lipgloss.NewStyle().Foreground(theme.ColorBlue).Bold(true)
	switch kind {
	case restWelcome:
		c := key.Render(m.c.keys.Key(actNewNote, ctxNotes))
		q := key.Render(m.c.keys.Key(actCheatsheet, ctxGlobal))
		return []string{
			lipgloss.NewStyle().Foreground(theme.ColorText).Render(i18n.T("Tu carpeta de notas está vacía", "Your notes folder is empty")),
			fillKeys(i18n.T("Pulsa %s para crear una nota y %s para ver los atajos", "Press %s to create a note and %s for the keys"), dim, c, q),
		}
	case restFolder:
		return []string{dim.Render(i18n.T("Carpeta vacía", "Empty folder"))}
	case restEmptyNote:
		return []string{dim.Render(i18n.T("Nota vacía", "Empty note"))}
	}
	return []string{dim.Render(i18n.T("Selecciona una nota para ver su contenido", "Select a note to see its content"))}
}

// renderRest dibuja el panel de la vista previa en un estado de reposo: el texto centrado y, si hay
// sitio y la mascota está activada, el perezoso dormido abajo a la derecha.
func (m *AppModel) renderRest(kind string, title, footer string, r Rect, active bool) string {
	msg := m.restMessage(kind)
	inner, h := r.W-2, r.H-2
	lines := make([]string, h)
	top := max(0, (h-len(msg))/2)
	for i, l := range msg {
		if top+i < h {
			lines[top+i] = textwidth.Repeat(" ", max(0, (inner-textwidth.Width(l))/2)) + l
		}
	}
	if m.mascotShows(kind, r) {
		block, headRow, headCol := m.mascotBlock()
		left := inner - mascotCols - 1 // columna (del interior del panel) donde empieza el bloque de la mascota
		hintRow, hintCol := m.hintPlacement(h, len(block), headRow, headCol, left, inner)
		hint := m.hintShowing(kind, r)
		base := append([]string(nil), lines...) // las filas sin la mascota (el "click me!" reemplaza las celdas transparentes de una fila del bloque)
		for i, l := range block {
			y := h - len(block) + i
			lines[y] = textwidth.Pad(lines[y], left) + l
		}
		if hint && hintRow >= 0 && hintRow < len(lines) && inner >= len([]rune(hintText)) { // justo encima de la cabeza (la primera fila visible del sprite), centrado sobre ella; sin borde ni fondo
			lines[hintRow] = textwidth.Pad(base[hintRow], hintCol) + lipgloss.NewStyle().Foreground(theme.ColorSubtext0).Italic(true).Render(hintText)
		}
	}
	return theme.RenderPanel(title, footer, lines, r.W, r.H, active)
}

// fillKeys arma una frase con teclas resaltadas: cada %s del texto (un solo texto traducible, así el orden de las palabras
// lo decide cada idioma) se reemplaza por una de keys, y el resto va con el estilo dim.
func fillKeys(text string, dim lipgloss.Style, keys ...string) string {
	parts := strings.Split(text, "%s")
	var b strings.Builder
	for i, part := range parts {
		if part != "" {
			b.WriteString(dim.Render(part))
		}
		if i < len(parts)-1 && i < len(keys) {
			b.WriteString(keys[i])
		}
	}
	return b.String()
}

// hintPlacement es dónde va el "click me!": la fila justo encima de la primera fila visible del sprite (headRow dentro del bloque de blockRows filas que acaba en
// el borde de abajo del panel de alto h; si la cabeza llega al borde del bloque, la fila libre de encima) y la columna que lo centra sobre la cabeza (headCol es la
// columna central de lo visible, left donde empieza el bloque), sin salirse del panel de ancho inner.
func (m *AppModel) hintPlacement(h, blockRows, headRow, headCol, left, inner int) (row, col int) {
	row = max(0, h-blockRows+headRow-1)
	col = left + headCol - len([]rune(hintText))/2
	col = max(0, min(col, inner-len([]rune(hintText))))
	return row, col
}

// noteIsEmpty dice si una nota cuenta como vacía a efectos del estado de reposo (y de la mascota): sin contenido, o solo con su título (una línea `# …`). Una nota con
// cualquier otra cosa (texto, tareas, otro encabezado) no lo está.
func noteIsEmpty(content string) bool {
	titles := 0
	for _, raw := range strings.Split(strings.TrimPrefix(content, "\ufeff"), "\n") {
		l := strings.TrimRight(raw, " \t\r")
		indent := len(l) - len(strings.TrimLeft(l, " \t"))
		switch {
		case l == "":
		case indent < 4 && !strings.HasPrefix(l[indent:], "\t") && strings.HasPrefix(strings.TrimLeft(l, " "), "# ") && titles == 0: // un `# …` de hasta 3 espacios es el título; con 4 o con tabulación es código
			titles++
		default:
			return false
		}
	}
	return true
}
