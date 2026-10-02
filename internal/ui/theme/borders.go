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

// PopupSolid decide el fondo de los popups: false (por defecto) los deja con el
// fondo de la terminal, así respetan su transparencia; true los pinta con el
// color base del tema. Lo fija la app según el ajuste "Fondo de popups".
var PopupSolid bool

// PopupBackground devuelve el color de fondo de los popups, o nil si no pintan fondo.
func PopupBackground() color.Color {
	if PopupSolid {
		return ColorBase
	}
	return nil
}

// RenderPopup dibuja un popup con esquinas redondeadas y los colores del tema
// (acento en el borde y el título, texto y apagado de la paleta). Pinta todas
// sus celdas, con o sin fondo según PopupSolid, así nada de lo que hay debajo se
// ve a través. hint va alineado a la derecha del borde inferior.
func RenderPopup(title, hint string, lines []string, width int) string {
	bg := PopupBackground()
	style := func(fg color.Color, bold bool) lipgloss.Style {
		s := lipgloss.NewStyle().Foreground(fg).Bold(bold)
		if bg != nil {
			s = s.Background(bg)
		}
		return s
	}
	border, titleStyle, hintStyle := style(ColorPeach, false), style(ColorPeach, true), style(ColorOverlay0, false)
	inner := width - 2

	rows := make([]string, 0, len(lines)+2)
	rows = append(rows, border.Render("╭")+edge(border, titleStyle, title, inner, false)+border.Render("╮"))
	for _, l := range lines {
		rows = append(rows, border.Render("│")+Paint(textwidth.Fit(" "+l, inner), bg)+border.Render("│"))
	}
	rows = append(rows, border.Render("╰")+edge(border, hintStyle, hint, inner, true)+border.Render("╯"))
	return strings.Join(rows, "\n")
}

// Selected pinta la fila seleccionada de una lista con el color de selección del tema.
func Selected(line string) string { return Paint(line, ColorSurface1) }

// bgReset encuentra las secuencias SGR que apagan el color de fondo.
var bgReset = regexp.MustCompile(`\x1b\[(0?|49)m`)

// Paint pinta el fondo bg (nil = ninguno) en toda la línea y lo reaplica después de cada reset
// interno, así ningún segmento con estilo propio deja un hueco sin fondo.
func Paint(line string, bg color.Color) string {
	if bg == nil {
		return line + "\x1b[0m" // sin fondo: se respeta el de la terminal
	}
	seq := bgSeq(bg)
	return seq + bgReset.ReplaceAllString(line, "${0}"+seq) + "\x1b[0m"
}

// bgSeq devuelve la secuencia SGR que activa el color de fondo c.
func bgSeq(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return "\x1b[48;2;" + strconv.Itoa(int(r>>8)) + ";" + strconv.Itoa(int(g>>8)) + ";" + strconv.Itoa(int(b>>8)) + "m"
}
