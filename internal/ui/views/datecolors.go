package views

import (
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"image/color"
	"math"
	"regexp"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// Colores de las fechas según su estado. (Los colores por defecto de cada estado y los que se pueden elegir están en config: DefaultDateColorNames y DateColorChoices.)
// Salen de la paleta del tema activo (nunca valores fijos), así se ven bien en los temas claros y oscuros y con el fondo de la
// terminal. Cada estado usa por defecto un color de la paleta que se puede cambiar en Ajustes (date_color_names); `date_colors` los apaga todos y `due_soon_days`
// decide cuántos días antes del vencimiento cuenta como "por vencer".

// DateState es el estado de una fecha de tarea a efectos del color.
type DateState int

const (
	StateOverdue    DateState = iota // vencimiento anterior a hoy, tarea sin hacer
	StateSoon                        // vencimiento hoy o en los próximos N días
	StateOnTime                      // vencimiento más allá de hoy + N
	StateStarted                     // inicio igual o anterior a hoy
	StateNotStarted                  // inicio posterior a hoy
	StateDone                        // fecha de completada
	StateNeutral                     // programada, creada, las de una tarea ya hecha, o los colores apagados
)

// DateStateKeys son las claves de la configuración de cada estado con color propio, en el orden de Ajustes.
var DateStateKeys = [...]string{"overdue", "soon", "ontime", "started", "notstarted", "done"}

// DateColorSettings es la configuración de los colores de las fechas (la fija la app desde la config).
type DateColorSettings struct {
	Enabled  bool
	SoonDays int
	Names    map[string]string
}

// DateColors es la configuración vigente.
var DateColors = DateColorSettings{Enabled: true, Names: config.DefaultDateColorNames()}

// PaletteColor devuelve el color de la paleta del tema activo con ese nombre; un nombre desconocido es el gris claro.
func PaletteColor(name string) color.Color {
	switch name {
	case "error":
		return theme.ColorRed
	case "warning":
		return theme.ColorYellow
	case "orange":
		return theme.ColorPeach
	case "success":
		return theme.ColorGreen
	case "accent":
		return theme.ColorBlue
	case "info":
		return theme.ColorTeal
	case "special":
		return theme.ColorMauve
	case "text":
		return theme.ColorText
	}
	return theme.ColorSubtext0
}

// StateOf dice en qué estado está la fecha date del campo f de una tarea (done: la tarea está hecha) con today (AAAA-MM-DD) como hoy.
func StateOf(f storage.DateField, date string, done bool, today string) DateState {
	switch f {
	case storage.DateDone:
		return StateDone
	case storage.DateDue:
		if done {
			return StateNeutral
		}
		if storage.Overdue(done, date, today) {
			return StateOverdue
		}
		if t, err := time.Parse("2006-01-02", today); err == nil {
			if limit := t.AddDate(0, 0, DateColors.SoonDays).Format("2006-01-02"); date <= limit {
				return StateSoon
			}
		}
		return StateOnTime
	case storage.DateStart:
		if done {
			return StateNeutral
		}
		if date <= today {
			return StateStarted
		}
		return StateNotStarted
	}
	return StateNeutral
}

// minContrast es la razón de contraste WCAG mínima de un color por defecto sobre el fondo del tema (3:1: la de texto grande y de interfaz).
const minContrast = 2.9

// floorContrast es el contraste mínimo con el que un color de respaldo se sigue usando si ninguno de la cadena llega a minContrast.
const floorContrast = 2.4

// defaultChains son, por estado, los colores de la paleta que se prueban para el color POR DEFECTO, en orden: el primero que se lee sobre el fondo del tema
// (contraste ≥ 2,9:1, el 3:1 de WCAG con margen de redondeo) gana, y el texto del tema es el último recurso. En los temas oscuros gana siempre el primero; en los claros el amarillo y el verde de la paleta
// quedan muy pálidos sobre el fondo (el amarillo de Latte da 2,3:1) y se cambian por el siguiente.
var defaultChains = map[string][]string{
	"overdue":    {"error", "orange", "text"},
	"soon":       {"warning", "orange", "text"},
	"ontime":     {"accent", "info", "special", "text"},
	"started":    {"info", "accent", "special", "text"},
	"notstarted": {"muted", "text"},
	"done":       {"success", "info", "text"},
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		x := float64(v>>8) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// Contrast es la razón de contraste WCAG (1 a 21) entre dos colores.
func Contrast(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// defaultOrder es el orden en que se resuelven los colores por defecto: los estados que más importan eligen primero, y un estado no toma un color que ya tiene otro
// (en Rosé Pine el verde y el azul de la paleta son el mismo).
var defaultOrder = []DateState{StateOverdue, StateSoon, StateDone, StateOnTime, StateStarted, StateNotStarted}

// defaultColor es el color por defecto del estado s en el tema activo: de su cadena, el primero que se lee (≥ 3:1) y que no tiene ya otro estado; si ninguno se lee,
// el de mayor contraste mientras se distinga del fondo (≥ 2,4:1), para que el estado conserve su color en los temas claros de paleta pálida; el texto del tema es el último
// recurso.
func defaultColor(s DateState) color.Color {
	taken := map[string]bool{}
	var chosen color.Color
	for _, st := range defaultOrder {
		key := DateStateKeys[st]
		var pick color.Color
		best, bestRatio := color.Color(nil), 0.0
		for _, name := range defaultChains[key] {
			if name == "text" {
				continue // el texto del tema es el último recurso, no compite con los colores de la cadena
			}
			c := PaletteColor(name)
			if taken[hexKey(c)] {
				continue
			}
			r := Contrast(c, theme.ColorBase)
			if r >= minContrast {
				pick = c
				break
			}
			if r > bestRatio {
				best, bestRatio = c, r
			}
		}
		if pick == nil && best != nil && bestRatio >= floorContrast {
			pick = best
		}
		if pick == nil {
			pick = theme.ColorText
		}
		taken[hexKey(pick)] = true
		if st == s {
			chosen = pick
		}
	}
	return chosen
}

func hexKey(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("%04x%04x%04x", r, g, b)
}

// StateColor es el color del estado s con la configuración vigente: el que el usuario eligió para ese estado o, si dejó el de por defecto, el de defaultColor. Con los
// colores apagados (o un estado neutro) es el gris claro del tema.
func StateColor(s DateState) color.Color {
	if !DateColors.Enabled || s == StateNeutral {
		return theme.ColorSubtext0
	}
	key := DateStateKeys[s]
	def := config.DefaultDateColorNames()[key]
	if n := DateColors.Names[key]; n != "" && n != def {
		return PaletteColor(n) // elegido por el usuario: se respeta
	}
	return defaultColor(s)
}

// DateStyleFor es el estilo de la fecha date del campo f: el color de su estado, y negrita las vencidas y las por vencer.
func DateStyleFor(f storage.DateField, date string, done bool, today string) lipgloss.Style {
	s := StateOf(f, date, done, today)
	st := lipgloss.NewStyle().Foreground(StateColor(s))
	if DateColors.Enabled && (s == StateOverdue || s == StateSoon) {
		st = st.Bold(true)
	}
	return st
}

// DateColorsKey resume la configuración de colores vigente (y el día): una vista que se guarda en caché con fechas coloreadas (la vista previa) la incluye en su clave.
func DateColorsKey(today string) string {
	k := fmt.Sprintf("%v|%d|%s", DateColors.Enabled, DateColors.SoonDays, today)
	for _, key := range DateStateKeys {
		k += "|" + DateColors.Names[key]
	}
	return k
}

var (
	// dateOnLine reconoce, en una línea ya dibujada (con sus códigos ANSI), un glifo de fecha, un espacio y una fecha AAAA-MM-DD.
	dateOnLine = regexp.MustCompile(`([\x{f135}\x{f073}\x{f00c}\x{f252}\x{f067}▸◷✓◑+])( )(\d{4}-\d{2}-\d{2})`)
	ansiSeq    = regexp.MustCompile(`\x1b\[[0-9;:]*m`)
)

// ColorDateLines pinta, en las líneas de la vista previa (markdown ya dibujado), cada "glifo fecha" con el color de su estado. El glifo dice qué campo es; si la
// línea es una tarea hecha ([✓]) su inicio y vencimiento van neutros. Cambia solo el color del primer plano (no deshace el resto de los estilos de la línea).
func ColorDateLines(lines []string, today string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l
		if !strings.ContainsAny(l, "▸◷✓◑+") {
			continue
		}
		done := strings.Contains(ansiSeq.ReplaceAllString(l, ""), "[✓]")
		out[i] = dateOnLine.ReplaceAllStringFunc(l, func(m string) string {
			sm := dateOnLine.FindStringSubmatch(m)
			f, ok := fieldOfGlyph(sm[1])
			if !ok {
				return m
			}
			st := StateOf(f, sm[3], done, today)
			col := StateColor(st)
			r, g, b, _ := col.RGBA()
			set := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r>>8, g>>8, b>>8)
			reset := "\x1b[39m"
			if DateColors.Enabled && (st == StateOverdue || st == StateSoon) {
				set, reset = "\x1b[1m"+set, "\x1b[22m"+reset
			}
			return set + m + reset
		})
	}
	return out
}

func fieldOfGlyph(g string) (storage.DateField, bool) {
	for f := storage.DateStart; f <= storage.DateCreated; f++ {
		if DateGlyph(f) == g {
			return f, true
		}
	}
	return 0, false
}
