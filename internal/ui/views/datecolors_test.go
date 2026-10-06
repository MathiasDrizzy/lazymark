package views

import (
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// withDateColors deja la configuración de colores como estaba al terminar la prueba.
func withDateColors(t *testing.T, s DateColorSettings) {
	t.Helper()
	old, oldTheme := DateColors, theme.CurrentThemeName
	t.Cleanup(func() { DateColors = old; theme.ApplyThemeByName(oldTheme) })
	DateColors = s
}

func defaultColors() DateColorSettings {
	return DateColorSettings{Enabled: true, Names: config.DefaultDateColorNames()}
}

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// TestDateStateColors (ORD-020 K1): los estados de una fecha con hoy fijo, en tres temas (oscuro, claro y otro): el color de cada estado es el de su entrada de la paleta del
// tema activo, no un valor fijo.
func TestDateStateColors(t *testing.T) {
	const today = "2026-10-06"
	for _, name := range []string{"catppuccin-mocha", "catppuccin-latte", "dracula"} {
		withDateColors(t, defaultColors())
		if !theme.ApplyThemeByName(name) {
			t.Fatalf("no existe el tema %s", name)
		}
		for _, c := range []struct {
			what string
			f    storage.DateField
			date string
			done bool
			want DateState
			col  color.Color
		}{
			{"vencida", storage.DateDue, "2026-10-05", false, StateOverdue, theme.ColorRed},
			{"vence hoy", storage.DateDue, "2026-10-06", false, StateSoon, theme.ColorYellow},
			{"vence en 1 día (N=0)", storage.DateDue, "2026-10-07", false, StateOnTime, theme.ColorBlue},
			{"vence lejos", storage.DateDue, "2026-12-01", false, StateOnTime, theme.ColorBlue},
			{"ya empezó", storage.DateStart, "2026-10-01", false, StateStarted, theme.ColorTeal},
			{"empieza hoy", storage.DateStart, "2026-10-06", false, StateStarted, theme.ColorTeal},
			{"todavía no empieza", storage.DateStart, "2026-10-09", false, StateNotStarted, theme.ColorSubtext0},
			{"completada", storage.DateDone, "2026-10-05", true, StateDone, theme.ColorGreen},
			{"vencimiento de una tarea hecha", storage.DateDue, "2026-10-01", true, StateNeutral, theme.ColorSubtext0},
			{"programada", storage.DateScheduled, "2026-10-07", false, StateNeutral, theme.ColorSubtext0},
			{"creada", storage.DateCreated, "2026-10-01", false, StateNeutral, theme.ColorSubtext0},
		} {
			if got := StateOf(c.f, c.date, c.done, today); got != c.want {
				t.Errorf("%s / %s: estado %d, se esperaba %d", name, c.what, got, c.want)
			}
			want := c.col
			if c.want != StateNeutral && Contrast(want, theme.ColorBase) < minContrast { // el color por defecto no se lee en este tema: el que dé su cadena
				want = StateColor(c.want)
				if hexOf(want) == hexOf(c.col) {
					t.Errorf("%s / %s: el color de respaldo no puede ser el mismo que no se lee", name, c.what)
				}
			}
			if got := hexOf(DateStyleFor(c.f, c.date, c.done, today).GetForeground()); got != hexOf(want) {
				t.Errorf("%s / %s: color %s, se esperaba %s", name, c.what, got, hexOf(want))
			}
			if name != "catppuccin-latte" && hexOf(want) != hexOf(c.col) {
				t.Errorf("%s / %s: en un tema oscuro gana el color de la paleta (%s), no el de respaldo", name, c.what, hexOf(c.col))
			}
		}
	}
}

// TestDueSoonDays (K3): con due_soon_days = N, vence "por vencer" desde hoy hasta hoy + N; apagados los colores, todas van en el gris claro neutro.
func TestDueSoonDays(t *testing.T) {
	const today = "2026-10-06"
	s := defaultColors()
	s.SoonDays = 3
	withDateColors(t, s)
	for date, want := range map[string]DateState{"2026-10-05": StateOverdue, "2026-10-06": StateSoon, "2026-10-09": StateSoon, "2026-10-10": StateOnTime} {
		if got := StateOf(storage.DateDue, date, false, today); got != want {
			t.Errorf("N=3, vence %s: estado %d, se esperaba %d", date, got, want)
		}
	}
	// otro color elegido para un estado
	s.Names = config.DefaultDateColorNames()
	s.Names["soon"] = "orange"
	DateColors = s
	if hexOf(StateColor(StateSoon)) != hexOf(theme.ColorPeach) {
		t.Error("el color elegido para 'por vencer' (orange) debe ser el naranja de la paleta")
	}
	// apagado: todo neutro, también la vencida
	s.Enabled = false
	DateColors = s
	for _, st := range []DateState{StateOverdue, StateSoon, StateOnTime, StateStarted, StateNotStarted, StateDone} {
		if hexOf(StateColor(st)) != hexOf(theme.ColorSubtext0) {
			t.Errorf("con date_colors apagado el estado %d debe ser neutro", st)
		}
	}
	if DateStyleFor(storage.DateDue, "2026-10-01", false, today).GetBold() {
		t.Error("apagado no hay negrita de vencida")
	}
}

// TestDateColorsAreLegibleInEveryTheme: ningún color de estado por defecto queda ilegible sobre el fondo de su tema (razón de contraste WCAG mínima 2,4:1 en los temas claros
// de paleta pálida y 3:1 casi siempre; la mayoría pasa de 4,5) y todos son más legibles que el gris atenuado (Overlay0) que casi no se veía. Con LAZYMARK_PRINT_CONTRAST=1 imprime
// la tabla tema × estado.
func TestDateColorsAreLegibleInEveryTheme(t *testing.T) {
	withDateColors(t, defaultColors())
	print := os.Getenv("LAZYMARK_PRINT_CONTRAST") != ""
	if print {
		fmt.Printf("%-22s %-7s %-7s %-7s %-8s %-11s %-7s %-9s\n", "tema", "overdue", "soon", "ontime", "started", "notstarted", "done", "overlay0")
	}
	for _, name := range theme.ThemeNames() {
		theme.ApplyThemeByName(name)
		var row []string
		for i, key := range DateStateKeys {
			col := StateColor(DateState(i))
			r := Contrast(col, theme.ColorBase)
			row = append(row, fmt.Sprintf("%.1f", r))
			if r < floorContrast {
				t.Errorf("%s / %s (%s): contraste %.2f < %.1f sobre el fondo del tema", name, key, hexOf(col), r, floorContrast)
			}
		}
		old := Contrast(theme.ColorOverlay0, theme.ColorBase)
		if print {
			fmt.Printf("%-22s %s %.1f\n", name, strings.Join(row, "     "), old)
		}
	}
}

// TestLatteSoonFallsBackToOrange: en Catppuccin Latte el amarillo de la paleta (2,3:1) no se lee; "por vencer" usa el naranja. Y si el usuario elige "warning" para
// otro estado, se respeta aunque sea pálido.
func TestLatteSoonFallsBackToOrange(t *testing.T) {
	withDateColors(t, defaultColors())
	theme.ApplyThemeByName("catppuccin-latte")
	if hexOf(StateColor(StateSoon)) != hexOf(theme.ColorPeach) {
		t.Errorf("latte / por vencer: %s, se esperaba el naranja %s", hexOf(StateColor(StateSoon)), hexOf(theme.ColorPeach))
	}
	s := defaultColors()
	s.Names["ontime"] = "warning"
	DateColors = s
	if hexOf(StateColor(StateOnTime)) != hexOf(theme.ColorYellow) {
		t.Error("un color elegido por el usuario se respeta")
	}
}

// TestDateStateColorsAreDistinct (apoyo): en cada tema, los estados con significado distinto (vencida, por vencer, en fecha, completada) no se resuelven al mismo color
// aunque haya habido respaldo por contraste; "ya empezó" y "sin empezar" tampoco coinciden entre sí.
func TestDateStateColorsAreDistinct(t *testing.T) {
	withDateColors(t, defaultColors())
	for _, name := range theme.ThemeNames() {
		theme.ApplyThemeByName(name)
		seen := map[string]string{}
		for _, st := range []DateState{StateOverdue, StateSoon, StateOnTime, StateDone} {
			h := hexOf(StateColor(st))
			if other, dup := seen[h]; dup {
				t.Errorf("%s: %s y %s tienen el mismo color %s", name, DateStateKeys[st], other, h)
			}
			seen[h] = DateStateKeys[st]
		}
		if hexOf(StateColor(StateStarted)) == hexOf(StateColor(StateNotStarted)) {
			t.Errorf("%s: 'ya empezó' y 'sin empezar' tienen el mismo color", name)
		}
	}
}

// TestDoneMarkHasNoWidth (ORD-022, apoyo): el marcador de tarea hecha no mide nada: no cambia dónde Glamour parte una línea ni el ancho de una celda de tabla.
func TestDoneMarkHasNoWidth(t *testing.T) {
	if w := textwidth.Width(string(DoneMark)); w != 0 {
		t.Errorf("el marcador mide %d celdas, debe medir 0", w)
	}
	line := "- [x] tarea " + DateGlyph(storage.DateDue) + " 2026-10-01"
	if textwidth.Width(MarkDoneDates(line, map[int]bool{1: true})) != textwidth.Width(line) {
		t.Error("marcar una tarea hecha no cambia el ancho de su línea")
	}
}
