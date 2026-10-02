package app

import (
	"image/color"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func hexOrEmpty(c color.Color) string {
	if c == nil {
		return ""
	}
	return hexOf(c)
}

// bgCells devuelve el fondo "#rrggbb" de cada celda de la pantalla ("" = fondo de la terminal).
func bgCells(m *AppModel) [][]string {
	buf := uv.NewScreenBuffer(m.w, m.h)
	buf.Method = ansi.GraphemeWidth // como la terminal: ⚠️ y ❤️ ocupan 2 celdas
	uv.NewStyledString(m.View().Content).Draw(buf, buf.Bounds())
	out := make([][]string, m.h)
	for y := range out {
		out[y] = make([]string, m.w)
		for x := range out[y] {
			c := buf.CellAt(x, y)
			switch {
			case c == nil:
			case c.Width == 0: // la segunda mitad de un glifo ancho comparte el estilo de la primera
				out[y][x] = "-"
			default:
				out[y][x] = hexOrEmpty(c.Style.Bg)
			}
		}
	}
	return out
}

// TestScreenBackgroundTheme (T4): con "theme" ninguna celda queda con el fondo de la terminal: en
// notas, Kanban, Ajustes y el cheatsheet (con popups del tema), y todas las celdas llevan
// el color Base del tema o un color de la paleta; los huecos y los espacios, exactamente Base.
// Con "terminal" quedan celdas sin fondo (la transparencia se respeta).
func TestScreenBackgroundTheme(t *testing.T) {
	defer theme.ApplyThemeByName("catppuccin-mocha")
	defer func() { theme.PopupSolid = false }()
	states := []struct {
		name string
		keys []string
	}{{"notas", nil}, {"kanban", []string{"W"}}, {"ajustes", []string{","}}, {"atajos", []string{"?"}}, {"papelera", []string{"x"}}}
	for _, name := range []string{"solarized-light", "dracula", "catppuccin-mocha"} {
		p := theme.AvailableThemes[name]
		base := hexOf(p.Base)
		allowed := paletteColors(p)
		for _, st := range states {
			m := newTestModel(t, 120, 35)
			theme.ApplyThemeByName(name)
			m.c.cfg.PopupBackground = config.PopupBackgroundTheme // con popups "terminal" sus celdas dejan el fondo de la terminal a propósito (TestPopupTerminalOnThemedScreen)
			m.c.cfg.ScreenBackground = config.ScreenBackgroundTheme
			m.onSettingsChange()
			press(m, st.keys...)
			holes, spaces := 0, 0
			for y, row := range bgCells(m) {
				for x, bg := range row {
					switch {
					case bg == "":
						holes++
						if holes <= 3 {
							t.Errorf("%s/%s: la celda (%d,%d) tiene el fondo de la terminal", name, st.name, x, y)
						}
					case bg == "-":
					case !allowed[bg]:
						t.Errorf("%s/%s: la celda (%d,%d) tiene un fondo fuera de la paleta: %s", name, st.name, x, y, bg)
					case bg == base:
						spaces++
					}
				}
			}
			if spaces < 500 {
				t.Errorf("%s/%s: solo %d celdas con el Base del tema; la pantalla no está pintada", name, st.name, spaces)
			}

			m.c.cfg.ScreenBackground = config.ScreenBackgroundTerminal
			def := 0
			for _, row := range bgCells(m) {
				for _, bg := range row {
					if bg == "" {
						def++
					}
				}
			}
			if def == 0 {
				t.Errorf("%s/%s: con terminal debe quedar el fondo de la terminal", name, st.name)
			}
		}
	}
}

// TestScreenBackgroundSetting (T4): el ajuste "Fondo de pantalla" está en Ajustes, alterna entre
// tema y terminal, se guarda y se ve en vivo.
func TestScreenBackgroundSettingInSettings(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, ",")
	p := m.c.top().(*settingsPopup)
	p.list.cursor = int(setScreenBg)
	if got := p.value(setScreenBg); got != "tema" {
		t.Errorf("por defecto = %q, se esperaba tema", got)
	}
	if p.label(setScreenBg) != "Fondo de pantalla" {
		t.Errorf("etiqueta = %q", p.label(setScreenBg))
	}
	press(m, "right")
	if m.c.cfg.ScreenBackground != config.ScreenBackgroundTerminal || p.value(setScreenBg) != "terminal" {
		t.Errorf("tras cambiar: %q / %q", m.c.cfg.ScreenBackground, p.value(setScreenBg))
	}
	press(m, "right")
	if m.c.cfg.ScreenBackground != config.ScreenBackgroundTheme {
		t.Error("el segundo cambio debe volver a tema")
	}
}

// TestPopupTerminalOnThemedScreen (C.4): con la pantalla pintada con el tema y el fondo de popups en "terminal", las
// celdas del popup quedan con el fondo por defecto de la terminal (respeta su transparencia) y las de fuera siguen con
// el Base del tema; con "theme" el popup lleva el Base. En los dos casos el popup tapa todas sus celdas.
func TestPopupTerminalOnThemedScreen(t *testing.T) {
	defer theme.ApplyThemeByName("catppuccin-mocha")
	defer func() { theme.PopupSolid = false }()
	for _, name := range []string{"dracula", "solarized-light"} {
		p := theme.AvailableThemes[name]
		base := hexOf(p.Base)
		for _, mode := range []string{config.PopupBackgroundTerminal, config.PopupBackgroundTheme} {
			m := newTestModel(t, 120, 35)
			theme.ApplyThemeByName(name)
			m.c.cfg.PopupBackground = mode
			m.c.cfg.ScreenBackground = config.ScreenBackgroundTheme
			m.onSettingsChange()
			press(m, "?")
			top := m.c.top()
			r := popupRect(m.layout, top, top.render(m.layout))
			cells := bgCells(m)
			inside, outside := map[string]int{}, map[string]int{}
			for y := r.Y; y < r.Y+r.H; y++ {
				for x := r.X; x < r.X+r.W; x++ {
					inside[cells[y][x]]++
				}
			}
			for x := 0; x < r.X; x++ { // la franja de la izquierda del popup, en las filas del popup
				for y := r.Y; y < r.Y+r.H; y++ {
					outside[cells[y][x]]++
				}
			}
			if outside[base] == 0 || outside[""] != 0 {
				t.Errorf("%s/%s: fuera del popup la pantalla debe estar pintada con el Base: %v", name, mode, outside)
			}
			switch mode {
			case config.PopupBackgroundTerminal:
				if inside[""] < r.W*r.H*3/4 || inside[base] != 0 {
					t.Errorf("%s/terminal: las celdas del popup deben quedar con el fondo de la terminal: %v", name, inside)
				}
			case config.PopupBackgroundTheme:
				if inside[""] != 0 || inside[base] < r.W*r.H/2 {
					t.Errorf("%s/theme: las celdas del popup deben llevar el Base: %v", name, inside)
				}
			}
		}
	}
}
