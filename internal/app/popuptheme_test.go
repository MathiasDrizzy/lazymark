package app

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

func rgb(c color.Color) [3]uint32 {
	r, g, b, _ := c.RGBA()
	return [3]uint32{r >> 8, g >> 8, b >> 8}
}

func paletteSet(p theme.Palette) map[[3]uint32]string {
	return map[[3]uint32]string{
		rgb(p.Base): "Base", rgb(p.Mantle): "Mantle", rgb(p.Surface0): "Surface0", rgb(p.Surface1): "Surface1",
		rgb(p.Overlay0): "Overlay0", rgb(p.Text): "Text", rgb(p.Subtext0): "Subtext0", rgb(p.Peach): "Peach",
		rgb(p.Mauve): "Mauve", rgb(p.Teal): "Teal", rgb(p.Green): "Green", rgb(p.Red): "Red",
		rgb(p.Blue): "Blue", rgb(p.Yellow): "Yellow",
	}
}

// popupKinds abre cada uno de los popups de la aplicación.
var popupKinds = []struct {
	name string
	open func(t *testing.T, m *AppModel)
}{
	{"atajos", func(t *testing.T, m *AppModel) { press(m, "?") }},
	{"ajustes", func(t *testing.T, m *AppModel) { press(m, ",") }},
	{"papelera", func(t *testing.T, m *AppModel) { press(m, "x") }},
	{"mover", func(t *testing.T, m *AppModel) {
		m.notes.selectPath(filepath.Join(m.c.store.BaseDir, "compras.md"))
		press(m, "m")
	}},
	{"confirmar", func(t *testing.T, m *AppModel) {
		m.notes.selectPath(filepath.Join(m.c.store.BaseDir, "compras.md"))
		press(m, "d")
	}},
	{"nombre", func(t *testing.T, m *AppModel) { press(m, "c") }},
}

func setPopupBackground(m *AppModel, mode string) {
	// "Fondo de popups" se prueba con "Fondo de pantalla" = terminal: con "tema" la pantalla
	// entera se pinta con el Base y ningún popup "sin fondo" deja ver el de la terminal (T4).
	m.c.cfg.ScreenBackground = config.ScreenBackgroundTerminal
	m.c.cfg.PopupBackground = mode
	m.onSettingsChange()
}

// TestPopupsFollowThemeAndBackground (H1-11, C12): con los 7 temas y los 2 modos
// de "Fondo de popups", todos los popups usan solo colores de la paleta del tema,
// cubren todas sus celdas (nada del fondo se ve a través) y, según el modo, no
// pintan fondo ("none", respeta la transparencia de la terminal) o pintan el
// color base del tema ("theme").
func TestPopupsFollowThemeAndBackground(t *testing.T) {
	t.Cleanup(func() { theme.ApplyThemeByName("catppuccin-mocha") })
	for _, name := range theme.ThemeNames() {
		pal := theme.AvailableThemes[name]
		allowed := paletteSet(pal)
		for _, mode := range []string{config.PopupBackgroundTerminal, config.PopupBackgroundTheme} {
			for _, kind := range popupKinds {
				t.Run(fmt.Sprintf("%s/%s/%s", name, mode, kind.name), func(t *testing.T) {
					m := newTestModel(t, 120, 35)
					m.c.cfg.Theme = name
					theme.ApplyThemeByName(name)
					setPopupBackground(m, mode)
					kind.open(t, m)
					p := m.c.top()
					if p == nil {
						t.Fatal("no se abrió el popup")
					}
					out := p.render(m.layout)
					r := popupRect(m.layout, p, out)

					// el popup solo, sobre un lienzo vacío
					alone := lipgloss.NewCanvas(r.W, r.H)
					alone.Compose(lipgloss.NewLayer(out))
					// la pantalla completa, con el popup encima de los paneles
					full := lipgloss.NewCanvas(m.w, m.h)
					full.Compose(lipgloss.NewLayer(m.View().Content))

					sawSelection := false
					for y := 0; y < r.H; y++ {
						for x := 0; x < r.W; x++ {
							a, b := alone.CellAt(x, y), full.CellAt(r.X+x, r.Y+y)
							// 1) sin huecos: lo que se ve es el popup, no lo que hay debajo
							if (a == nil) != (b == nil) || (a != nil && (a.Content != b.Content || !a.Style.Equal(&b.Style))) {
								t.Fatalf("hueco en (%d,%d): el popup solo tiene %+v y en pantalla se ve %+v", x, y, a, b)
							}
							if a == nil || a.Width == 0 {
								continue
							}
							if a.Content == "" {
								t.Fatalf("celda (%d,%d) sin pintar", x, y)
							}
							// 2) solo colores de la paleta del tema
							for what, c := range map[string]color.Color{"fg": a.Style.Fg, "bg": a.Style.Bg} {
								if c == nil {
									continue
								}
								if _, ok := allowed[rgb(c)]; !ok {
									t.Fatalf("celda (%d,%d) %q: %s %v no es de la paleta de %s", x, y, a.Content, what, rgb(c), name)
								}
							}
							// 3) el fondo según el modo
							bg := a.Style.Bg
							isSel := bg != nil && rgb(bg) == rgb(pal.Surface1)
							sawSelection = sawSelection || isSel
							switch mode {
							case config.PopupBackgroundTerminal:
								if bg != nil && !isSel {
									t.Fatalf("modo terminal: la celda (%d,%d) pinta fondo %v", x, y, allowed[rgb(bg)])
								}
							case config.PopupBackgroundTheme:
								if bg == nil || (!isSel && rgb(bg) != rgb(pal.Base)) {
									t.Fatalf("modo theme: la celda (%d,%d) debe llevar el color base, tiene %v", x, y, bg)
								}
							}
						}
					}
					// 4) borde y título con el acento del tema
					if c := alone.CellAt(0, 0); c == nil || c.Style.Fg == nil || rgb(c.Style.Fg) != rgb(pal.Peach) {
						t.Errorf("la esquina del borde no usa el acento del tema: %+v", c)
					}
					// 5) las listas muestran la selección con el color de selección del tema
					if (kind.name == "ajustes" || kind.name == "mover") && !sawSelection {
						t.Error("la fila seleccionada no usa el color de selección del tema")
					}
				})
			}
		}
	}
}

// TestPopupBackgroundSettingToggles (H1-11, C.4): el ajuste "Fondo de popups" (tema | terminal, como el de pantalla)
// se recorre desde Ajustes con ← y →, se aplica al momento y se guarda.
func TestPopupBackgroundSettingToggles(t *testing.T) {
	t.Cleanup(func() { theme.PopupSolid = false })
	m := newTestModel(t, 120, 35)
	if m.c.cfg.PopupBackground != config.PopupBackgroundTheme || !theme.PopupSolid {
		t.Fatal("por defecto los popups llevan el fondo del tema, como la pantalla")
	}
	press(m, ",")
	sp := m.c.top().(*settingsPopup)
	sp.list.set(int(setPopupBg), sp.n)
	if got := plain(m); !strings.Contains(got, "Fondo de popups") || !strings.Contains(got, "tema") {
		t.Errorf("el ajuste no aparece con su valor:\n%s", got)
	}
	press(m, "right")
	if m.c.cfg.PopupBackground != config.PopupBackgroundTerminal || theme.PopupSolid {
		t.Fatalf("tras → debería ser terminal (cfg=%q solid=%v)", m.c.cfg.PopupBackground, theme.PopupSolid)
	}
	if got := plain(m); !strings.Contains(got, "terminal") {
		t.Errorf("el valor nuevo no se ve:\n%s", got)
	}
	data, err := os.ReadFile(m.c.cfg.Path())
	if err != nil || !strings.Contains(string(data), `"popup_background": "terminal"`) {
		t.Errorf("el ajuste no se guardó (%v):\n%s", err, data)
	}
	press(m, "left")
	if m.c.cfg.PopupBackground != config.PopupBackgroundTheme || !theme.PopupSolid {
		t.Errorf("tras ← debería volver a theme")
	}
}
