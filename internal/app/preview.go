package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

var mdImage = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)

// previewPanel es el panel derecho: markdown renderizado con scroll vertical y
// horizontal. El render se cachea por nota, fecha, ancho y tema.
type previewPanel struct {
	scrollY, scrollX int

	cacheKey   string
	cacheLines []string
}

func (p *previewPanel) reset() { p.scrollY, p.scrollX = 0, 0 }

// lines devuelve el markdown de note renderizado a width columnas.
func (p *previewPanel) lines(note *storage.Note, width int) []string {
	key := fmt.Sprintf("%s|%d|%d|%s", note.Path, note.ModTime.UnixNano(), width, theme.CurrentThemeName)
	if key != p.cacheKey {
		p.cacheKey = key
		p.cacheLines = strings.Split(renderMarkdown(note, width), "\n")
	}
	return p.cacheLines
}

// scroll mueve el desplazamiento vertical; lo acota el render.
func (p *previewPanel) scroll(dy, dx int) {
	p.scrollY = max(0, p.scrollY+dy)
	p.scrollX = max(0, p.scrollX+dx)
}

func (p *previewPanel) view(note *storage.Note, r Rect, active bool) string {
	inner, h := r.W-2, r.H-2
	if note == nil {
		empty := lipgloss.NewStyle().Foreground(theme.ColorOverlay0).Italic(true).Render(" " + i18n.T("Selecciona una nota para ver su contenido", "Select a note to see its content"))
		return theme.RenderPanel(i18n.T("[4]─Vista previa", "[4]─Preview"), "", []string{empty}, r.W, r.H, active)
	}
	all := p.lines(note, inner-1)
	maxY := max(0, len(all)-h)
	p.scrollY = min(p.scrollY, maxY)
	visible := make([]string, 0, h)
	for i := p.scrollY; i < min(len(all), p.scrollY+h); i++ {
		visible = append(visible, " "+textwidth.Cut(all[i], p.scrollX, p.scrollX+inner-1))
	}
	pct := 100
	if maxY > 0 {
		pct = p.scrollY * 100 / maxY
	}
	indicator := fmt.Sprintf("%d%% %d/%d", pct, p.scrollY+1, len(all))
	return theme.RenderPanel("[4]─"+note.Title, indicator, visible, r.W, r.H, active)
}

// renderMarkdown renderiza la nota con Glamour respetando las líneas en blanco
// tal como se escribieron, e intercala las imágenes en su posición.
func renderMarkdown(note *storage.Note, width int) string {
	width = max(10, width)
	style := "dark"
	if theme.CurrentThemeName == "catppuccin-latte" {
		style = "light"
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	render := func(md string) string {
		if err != nil {
			return md
		}
		out, rerr := r.Render(md)
		if rerr != nil {
			return md
		}
		return strings.Trim(out, "\n")
	}
	// Markdown colapsa varias líneas en blanco en una; se renderiza por tramos
	// y se reponen las líneas en blanco tal como se escribieron (H1-4).
	renderKeep := func(md string) string {
		var b strings.Builder
		for i, seg := range splitBlankRuns(md) {
			if i > 0 {
				b.WriteString(strings.Repeat("\n", seg.gap+1))
			}
			if strings.TrimSpace(seg.text) != "" {
				b.WriteString(render(seg.text))
			}
		}
		return b.String()
	}

	content := note.Content
	var sections []string
	last := 0
	for _, m := range mdImage.FindAllStringSubmatchIndex(content, -1) {
		if before := content[last:m[0]]; strings.TrimSpace(before) != "" {
			sections = append(sections, renderKeep(before))
		}
		alt, src := content[m[2]:m[3]], content[m[4]:m[5]]
		last = m[1]
		if !filepath.IsAbs(src) {
			src = filepath.Join(filepath.Dir(note.Path), src)
		}
		caption := lipgloss.NewStyle().Foreground(theme.ColorPeach).Bold(true).Render(fmt.Sprintf("  %s (%s)", alt, filepath.Base(src)))
		if img, ierr := image.RenderInlineToAnsi(src, width-4, 12); ierr == nil && img != "" {
			sections = append(sections, caption+"\n"+img)
		} else {
			sections = append(sections, caption)
		}
	}
	if rest := content[last:]; strings.TrimSpace(rest) != "" {
		sections = append(sections, renderKeep(rest))
	}
	if len(sections) == 0 {
		return content
	}
	return strings.Join(sections, "\n\n")
}

// segment es un tramo de markdown y las líneas en blanco que lo preceden.
type segment struct {
	text string
	gap  int
}

// splitBlankRuns parte md en cada tanda de dos o más líneas en blanco que
// esté fuera de un bloque de código.
func splitBlankRuns(md string) []segment {
	var out []segment
	var cur []string
	blanks, fence := 0, false
	flush := func(gap int) {
		out = append(out, segment{text: strings.Join(cur, "\n"), gap: gap})
		cur = nil
	}
	gap := 0
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fence = !fence
		}
		if !fence && strings.TrimSpace(line) == "" {
			blanks++
			continue
		}
		switch {
		case blanks >= 2 && len(cur) > 0:
			flush(gap)
			gap = blanks
		case blanks > 0:
			cur = append(cur, make([]string, blanks)...)
		}
		blanks = 0
		cur = append(cur, line)
	}
	flush(gap)
	return out
}

// doubleClick detecta un segundo clic en la misma zona en menos de 400 ms.
type doubleClick struct {
	at   time.Time
	zone string
}

func (d *doubleClick) hit(zone string, now time.Time) bool {
	double := zone == d.zone && now.Sub(d.at) < 400*time.Millisecond
	d.at, d.zone = now, zone
	if double {
		d.zone = ""
	}
	return double
}
