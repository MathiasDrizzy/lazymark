package app

import (
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/assets/brand"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/sprite"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// El perezoso dormido (variante c2 de la marca) aparece en los estados de reposo de la vista previa:
// la carpeta de notas vacía, una carpeta sin notas y nada seleccionado. Mide 32x16 celdas: con
// Kitty es la imagen embebida (placeholders, con la misma carga asíncrona que las imágenes de
// las notas) y sin Kitty, medios bloques (▀) con los colores exactos del SVG. Tiene colores fijos:
// no depende del tema. Si el panel no tiene sitio para él, no se muestra.
const (
	restCols = 32
	restRows = 16
)

// Estados de reposo.
const (
	restNone    = "none"    // nada seleccionado
	restFolder  = "folder"  // una carpeta sin notas
	restWelcome = "welcome" // la carpeta de notas está vacía
)

var (
	sleeperOnce  sync.Once
	sleeperLines []string
)

// sleeperHalfBlocks devuelve el perezoso en medios bloques, leído del SVG embebido.
func sleeperHalfBlocks() []string {
	sleeperOnce.Do(func() {
		if g, err := sprite.ParseSVG(brand.SleepingSVG); err == nil {
			sleeperLines = sprite.HalfBlocks(g)
		}
	})
	return sleeperLines
}

// restKind dice qué estado de reposo muestra la vista previa ("" si muestra otra cosa). Sigue
// la misma elección que renderPreview.
func (m *AppModel) restKind() string {
	switch m.lastLeft {
	case panelTags:
		return ""
	case panelTasks:
		if m.previewNote() == nil {
			return restNone
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
	if m.notes.currentNote() != nil {
		return ""
	}
	if len(m.notes.entries) == 0 && m.notes.tagFilter == "" {
		return restWelcome
	}
	return restNone
}

func filepathSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

// restFits indica si el panel de la vista previa tiene sitio para el perezoso y dos líneas de texto.
func (m *AppModel) restFits() bool {
	r := m.layout.Preview
	return r.W-2 >= restCols+2 && r.H-2 >= restRows+4
}

// sleeper devuelve las 16 líneas del perezoso: la imagen de Kitty si ya está lista y, mientras
// carga o sin Kitty, los medios bloques.
func (m *AppModel) sleeper() []string {
	if lines, ok := m.c.kitty.BlockData("sleeping", brand.SleepingPNG, restCols, restRows); ok {
		return lines
	}
	return sleeperHalfBlocks()
}

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
			dim.Render(i18n.T("Pulsa ", "Press ")) + c + dim.Render(i18n.T(" para crear una nota y ", " to create a note and ")) + q + dim.Render(i18n.T(" para ver los atajos", " for the keys")),
		}
	case restFolder:
		return []string{dim.Render(i18n.T("Carpeta vacía", "Empty folder"))}
	}
	return []string{dim.Render(i18n.T("Selecciona una nota para ver su contenido", "Select a note to see its content"))}
}

// renderRest dibuja el panel de la vista previa en un estado de reposo: el perezoso centrado con su
// texto debajo, o solo el texto si no cabe.
func (m *AppModel) renderRest(kind string, title, footer string, r Rect, active bool) string {
	msg := m.restMessage(kind)
	inner, h := r.W-2, r.H-2
	center := func(s string) string {
		return textwidth.Repeat(" ", max(0, (inner-textwidth.Width(s))/2)) + s
	}
	var lines []string
	if m.restFits() {
		block := m.sleeper()
		total := len(block) + 1 + len(msg)
		for i := 0; i < max(0, (h-total)/2); i++ {
			lines = append(lines, "")
		}
		for _, l := range block {
			lines = append(lines, center(l))
		}
		lines = append(lines, "")
		for _, l := range msg {
			lines = append(lines, center(l))
		}
	} else {
		for _, l := range msg {
			lines = append(lines, " "+l)
		}
	}
	return theme.RenderPanel(title, footer, lines, r.W, r.H, active)
}

// wantsSleeper indica si se está mostrando el perezoso con espacio para él (prepareImages lo usa
// para pedir su imagen antes del render, como con las imágenes de las notas).
func (m *AppModel) wantsSleeper() bool {
	return !m.kanbanOn && !m.layout.TooSmall && !(m.zoom && m.focus == panelPreview) && m.restKind() != "" && m.restFits()
}
