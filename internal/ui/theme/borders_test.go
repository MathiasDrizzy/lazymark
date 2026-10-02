package theme

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderPanelGeometry(t *testing.T) {
	ApplyPalette(CatppuccinMocha)
	cases := []struct {
		w, h          int
		title, footer string
		active        bool
	}{
		{20, 5, "[1]─Notas", "1 of 10", true},
		{20, 5, "[1]─Notas", "1 of 10", false},
		{40, 10, "[2]─Tareas", "3 of 12", true},
		{15, 4, "[3]─UnTítuloMuyLargoQueHayQueCortar", "99 of 99", true},
		{12, 3, "", "", false},
	}
	body := []string{"línea con ⚠️ y ❤️ larga larga larga", " nota.md"}
	for _, c := range cases {
		lines := strings.Split(RenderPanel(c.title, c.footer, body, c.w, c.h, c.active), "\n")
		if len(lines) != c.h {
			t.Fatalf("%dx%d: %d líneas", c.w, c.h, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != c.w {
				t.Fatalf("%dx%d %q: línea %d mide %d: %q", c.w, c.h, c.title, i, w, ansi.Strip(l))
			}
		}
		top, bottom := ansi.Strip(lines[0]), ansi.Strip(lines[c.h-1])
		if !strings.HasPrefix(top, "╭") || !strings.HasSuffix(top, "╮") {
			t.Errorf("borde superior sin esquinas redondeadas: %q", top)
		}
		if !strings.HasPrefix(bottom, "╰") || !strings.HasSuffix(bottom, "╯") {
			t.Errorf("borde inferior sin esquinas redondeadas: %q", bottom)
		}
		if c.footer != "" && c.w >= 20 && !strings.HasSuffix(bottom, c.footer+"─╯") {
			t.Errorf("el contador no está abajo a la derecha: %q", bottom)
		}
		if c.title != "" && c.w >= 20 && !strings.HasPrefix(top, "╭─"+c.title) {
			t.Errorf("el título no está embebido arriba a la izquierda: %q", top)
		}
	}
}

// TestRenderPopupPaintsEveryCell comprueba celda por celda que el popup cubre
// todo su rectángulo aunque el contenido traiga resets de estilo. Con PopupSolid
// cada celda lleva el color base del tema; sin él, ninguna lleva fondo (así se
// respeta la transparencia de la terminal) pero todas están pintadas.
func TestRenderPopupPaintsEveryCell(t *testing.T) {
	t.Cleanup(func() { PopupSolid = false })
	for _, solid := range []bool{false, true} {
		PopupSolid = solid
		ApplyPalette(CatppuccinMocha)
		styled := lipgloss.NewStyle().Foreground(ColorRed).Render("rojo") + " normal " + lipgloss.NewStyle().Bold(true).Render("negrita")
		out := RenderPopup("Título", "[Esc]", []string{styled, "", "texto ⚠️ ancho"}, 30)
		lines := strings.Split(out, "\n")
		canvas := lipgloss.NewCanvas(30, len(lines))
		canvas.Compose(lipgloss.NewLayer(out))
		for y := 0; y < len(lines); y++ {
			for x := 0; x < 30; x++ {
				cell := canvas.CellAt(x, y)
				if cell == nil {
					t.Fatalf("solid=%v: celda (%d,%d) sin pintar", solid, x, y)
				}
				if cell.Width == 0 {
					continue // continuación de un glifo ancho: la cubre la celda anterior
				}
				if solid && (cell.Style.Bg == nil || rgb(cell.Style.Bg) != rgb(ColorBase)) {
					t.Fatalf("solid: celda (%d,%d) %q debe llevar el color base: %v", x, y, cell.Content, cell.Style.Bg)
				}
				if !solid && cell.Style.Bg != nil {
					t.Fatalf("sin fondo: celda (%d,%d) %q pinta fondo %v", x, y, cell.Content, cell.Style.Bg)
				}
			}
		}
		if bottom := ansi.Strip(lines[len(lines)-1]); !strings.HasSuffix(bottom, "[Esc]─╯") {
			t.Errorf("[Esc] no está alineado dentro del borde inferior: %q", bottom)
		}
	}
}

func rgb(c color.Color) [3]uint32 {
	r, g, b, _ := c.RGBA()
	return [3]uint32{r >> 8, g >> 8, b >> 8}
}
