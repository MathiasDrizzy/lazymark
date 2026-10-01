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
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

var mdImage = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)

// previewPanel es el panel derecho: markdown renderizado con scroll vertical y
// horizontal. El render se cachea por nota, fecha, ancho y tema.
type previewPanel struct {
	imgs             *image.Client
	scrollY, scrollX int

	cacheKey   string
	cacheLines []string
}

func (p *previewPanel) reset() { p.scrollY, p.scrollX = 0, 0 }

// lines devuelve el markdown de note renderizado a width columnas.
func (p *previewPanel) lines(note *storage.Note, width int) []string {
	key := fmt.Sprintf("%s|%d|%d|%s|%d", note.Path, note.ModTime.UnixNano(), width, theme.CurrentThemeName, p.imgs.Generation())
	if key != p.cacheKey {
		p.cacheKey = key
		p.cacheLines = strings.Split(renderMarkdown(note, width, p.imgs), "\n")
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
func renderMarkdown(note *storage.Note, width int, imgs *image.Client) string {
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
	indent := 2 // sangría de Glamour; las imágenes se alinean con el texto
	measured := false
	render := func(md string) string {
		if err != nil {
			return md
		}
		out, rerr := r.Render(md)
		if rerr != nil {
			return md
		}
		out = strings.Trim(out, "\n")
		if !measured {
			// la sangría del texto es la menor de sus líneas: el título lleva un espacio de más
			least := -1
			for _, l := range strings.Split(textwidth.Strip(out), "\n") {
				if strings.TrimSpace(l) != "" {
					if n := len(l) - len(strings.TrimLeft(l, " ")); least < 0 || n < least {
						least = n
					}
				}
			}
			if least >= 0 {
				indent, measured = least, true
			}
		}
		return out
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
		src := content[m[4]:m[5]]
		last = m[1]
		if !filepath.IsAbs(src) {
			src = filepath.Join(filepath.Dir(note.Path), src)
		}
		sections = append(sections, imageBlock(imgs, src, width, indent))
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

// maxImageRows es el alto máximo, en filas, con el que se dibuja una imagen.
const maxImageRows = 18

// imageBlock devuelve el bloque que reemplaza a `![](ruta)` en el preview: la
// imagen (placeholders de Kitty) alineada con el texto, o su texto de
// reemplazo si la terminal no soporta gráficos o no se puede mostrar.
func imageBlock(imgs *image.Client, path string, width, indent int) string {
	pad := strings.Repeat(" ", indent)
	if lines, ok := imgs.Block(path, max(2, width-indent-1), maxImageRows); ok {
		for i := range lines {
			lines[i] = pad + lines[i]
		}
		return strings.Join(lines, "\n")
	}
	return pad + lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(image.Label(path))
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

// taskKey es la huella de una tarea para buscarla en el markdown renderizado:
// sus primeras palabras sin marcas de énfasis ni de enlace.
func taskKey(text string) string {
	clean := strings.NewReplacer("*", "", "_", "", "`", "", "[", "", "]", "").Replace(storage.CleanTaskText(text))
	words := strings.Fields(clean)
	return strings.Join(words[:min(3, len(words))], " ")
}

// taskRenderedLine devuelve la fila del markdown ya renderizado donde está la
// tarea t. Si el texto se repite en la nota, usa la ocurrencia que corresponde
// por orden de línea; si no la encuentra, estima la posición proporcional.
func taskRenderedLine(lines []string, note *storage.Note, t views.FlatTask) int {
	if key := taskKey(t.Text); key != "" {
		nth := 0
		for _, other := range note.Tasks {
			if other.Line < t.Line && taskKey(other.Text) == key {
				nth++
			}
		}
		for i, l := range lines {
			if strings.Contains(textwidth.Strip(l), key) {
				if nth == 0 {
					return i
				}
				nth--
			}
		}
	}
	total := max(1, strings.Count(note.Content, "\n")+1)
	return t.Line * len(lines) / total
}
