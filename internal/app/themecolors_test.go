package app

import (
	"fmt"
	"image/color"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

var sgrRe = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// colorsIn devuelve todos los colores (fg y bg) que usa una salida con ANSI, como
// "#rrggbb" para truecolor y "256:n" para los índices de la paleta de 256 colores.
func colorsIn(out string) map[string]bool {
	seen := map[string]bool{}
	for _, m := range sgrRe.FindAllStringSubmatch(out, -1) {
		parts := strings.FieldsFunc(m[1], func(r rune) bool { return r == ';' || r == ':' })
		for i := 0; i < len(parts); i++ {
			switch parts[i] {
			case "38", "48", "58":
				if i+2 < len(parts) && parts[i+1] == "5" {
					seen["256:"+parts[i+2]] = true
					i += 2
				} else if i+4 < len(parts) && parts[i+1] == "2" {
					r, _ := strconv.Atoi(parts[i+2])
					g, _ := strconv.Atoi(parts[i+3])
					b, _ := strconv.Atoi(parts[i+4])
					seen[fmt.Sprintf("#%02x%02x%02x", r, g, b)] = true
					i += 4
				}
			}
		}
	}
	return seen
}

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// paletteColors son los 14 colores de una paleta, en "#rrggbb".
func paletteColors(p theme.Palette) map[string]bool {
	out := map[string]bool{}
	for _, c := range []color.Color{p.Base, p.Mantle, p.Surface0, p.Surface1, p.Overlay0, p.Text, p.Subtext0,
		p.Peach, p.Mauve, p.Teal, p.Green, p.Red, p.Blue, p.Yellow} {
		out[hexOf(c)] = true
	}
	return out
}

// TestThemeColorsComeFromPalette (T3): en cada tema, TODO lo que se pinta (notas con la vista
// previa de Glamour, Kanban, Ajustes y el cheatsheet) usa solo colores de la paleta del tema.
// Antes la vista previa usaba los índices fijos de 256 colores del estilo "dark" de Glamour
// (228, 63, 252…), así que no cambiaba con el tema.
func TestThemeColorsComeFromPalette(t *testing.T) {
	defer theme.ApplyThemeByName("catppuccin-mocha")
	states := []struct {
		name string
		keys []string
	}{
		{"notas", nil}, {"kanban", []string{"W"}}, {"ajustes", []string{","}}, {"atajos", []string{"?"}},
	}
	for _, name := range theme.ThemeNames() {
		p := theme.AvailableThemes[name]
		allowed := paletteColors(p)
		for _, st := range states {
			m := newTestModel(t, 120, 35) // New aplica el tema de la config: el de la prueba va después
			theme.ApplyThemeByName(name)
			press(m, "down", "down") // una nota con Markdown a la vista
			press(m, st.keys...)
			var extra []string
			for c := range colorsIn(m.View().Content) {
				if !allowed[c] {
					extra = append(extra, c)
				}
			}
			if len(extra) > 0 {
				sort.Strings(extra)
				t.Errorf("%s / %s: colores fuera de la paleta: %v", name, st.name, extra)
			}
		}
	}
}

// TestPreviewFollowsTheme (T3): dos temas distintos dan colores distintos en la vista previa.
func TestPreviewFollowsTheme(t *testing.T) {
	defer theme.ApplyThemeByName("catppuccin-mocha")
	seen := map[string]string{}
	for _, name := range theme.ThemeNames() {
		m := newTestModel(t, 120, 35)
		theme.ApplyThemeByName(name)
		n := &m.c.notes[0]
		var cs []string
		for c := range colorsIn(strings.Join(m.preview.lines(n, 60), "\n")) {
			cs = append(cs, c)
		}
		sort.Strings(cs)
		key := strings.Join(cs, ",")
		if other, dup := seen[key]; dup {
			t.Errorf("%s y %s dan los mismos colores en la vista previa: %s", name, other, key)
		}
		seen[key] = name
	}
}
