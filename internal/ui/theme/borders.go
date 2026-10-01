package theme

import (
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
)

// RenderPanel dibuja un panel estilo lazygit de exactamente width x height celdas:
//
//	╭─[1]─Notas──────────────╮
//	│contenido               │
//	╰──────────────── 1 of 3─╯
//
// El título va embebido en el borde superior y footer (el contador) en el borde
// inferior derecho. Cada línea se corta con "…" o se rellena al ancho interior.
// El panel activo resalta el borde y el título con el color de acento.
func RenderPanel(title, footer string, lines []string, width, height int, active bool) string {
	if width < 4 {
		width = 4
	}
	if height < 2 {
		height = 2
	}
	borderCol, titleCol := ColorOverlay0, ColorSubtext0
	if active {
		borderCol, titleCol = ColorGreen, ColorGreen
	}
	border := lipgloss.NewStyle().Foreground(borderCol)
	titleStyle := lipgloss.NewStyle().Foreground(titleCol).Bold(active)
	inner := width - 2

	rows := make([]string, 0, height)
	rows = append(rows, border.Render("╭")+edge(border, titleStyle, title, inner, false)+border.Render("╮"))
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rows = append(rows, border.Render("│")+textwidth.Fit(line, inner)+"\x1b[0m"+border.Render("│"))
	}
	rows = append(rows, border.Render("╰")+edge(border, titleStyle, footer, inner, true)+border.Render("╯"))
	return strings.Join(rows, "\n")
}

// edge construye un borde horizontal de n celdas con label embebido: a la
// izquierda ("─label───") o a la derecha ("───label─").
func edge(border, label lipgloss.Style, text string, n int, right bool) string {
	if text == "" || n < 4 {
		return border.Render(textwidth.Repeat("─", n))
	}
	text = textwidth.Truncate(text, n-2, textwidth.Ellipsis)
	dashes := border.Render(textwidth.Repeat("─", n-1-textwidth.Width(text)))
	styled := label.Render(text)
	if right {
		return dashes + styled + border.Render("─")
	}
	return border.Render("─") + styled + dashes
}

// RenderBoxWithTitle es RenderPanel con un margen de una columna a cada lado
// del contenido (ancho útil width-4). Lo usan Kanban y las vistas previas.
func RenderBoxWithTitle(title, badge, content string, width, height int, active bool) string {
	inner := width - 4
	src := strings.Split(content, "\n")
	lines := make([]string, len(src))
	for i, l := range src {
		lines[i] = " " + textwidth.Fit(l, inner) + " "
	}
	return RenderPanel(title, badge, lines, width, height, active)
}

// RenderPopup dibuja un popup con esquinas redondeadas que pinta todas sus
// celdas con el fondo del tema: cada segmento lleva su propio fondo, así un
// reset interno no deja huecos. hint va alineado a la derecha del borde inferior.
func RenderPopup(title, hint string, lines []string, width int) string {
	bg := ColorMantle
	border := lipgloss.NewStyle().Foreground(ColorPeach).Background(bg)
	titleStyle := lipgloss.NewStyle().Foreground(ColorPeach).Background(bg).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(ColorOverlay0).Background(bg)
	inner := width - 2

	rows := make([]string, 0, len(lines)+2)
	rows = append(rows, border.Render("╭")+edge(border, titleStyle, title, inner, false)+border.Render("╮"))
	for _, l := range lines {
		rows = append(rows, border.Render("│")+Paint(textwidth.Fit(" "+l, inner), bg)+border.Render("│"))
	}
	rows = append(rows, border.Render("╰")+edge(border, hintStyle, hint, inner, true)+border.Render("╯"))
	return strings.Join(rows, "\n")
}

// bgReset encuentra las secuencias SGR que apagan el color de fondo.
var bgReset = regexp.MustCompile(`\x1b\[(0?|49)m`)

// Paint pinta el fondo bg en toda la línea y lo reaplica después de cada reset
// interno, así ningún segmento con estilo propio deja un hueco sin fondo.
func Paint(line string, bg color.Color) string {
	seq := bgSeq(bg)
	return seq + bgReset.ReplaceAllString(line, "${0}"+seq) + "\x1b[0m"
}

// bgSeq devuelve la secuencia SGR que activa el color de fondo c.
func bgSeq(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return "\x1b[48;2;" + strconv.Itoa(int(r>>8)) + ";" + strconv.Itoa(int(g>>8)) + ";" + strconv.Itoa(int(b>>8)) + "m"
}
