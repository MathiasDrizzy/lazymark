package sprite

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/assets/brand"
	"github.com/charmbracelet/x/ansi"
)

func TestParseEmbeddedSVG(t *testing.T) {
	g, err := ParseSVG(brand.SleepingSVG)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := g.Size(); w != 32 || h != 32 {
		t.Fatalf("la grilla es %dx%d, se esperaba 32x32", w, h)
	}
	set := 0
	for _, row := range g {
		for _, p := range row {
			if p.Set {
				set++
			}
		}
	}
	if set < 300 {
		t.Errorf("solo %d píxeles con color: el SVG no se leyó bien", set)
	}
}

// TestHalfBlocksComeFromSVG (Z2): el render de medios bloques sale del SVG: cambiar un píxel del
// SVG cambia exactamente la celda que lo contiene (y solo esa), con el color exacto.
func TestHalfBlocksComeFromSVG(t *testing.T) {
	base, _ := ParseSVG(brand.SleepingSVG)
	baseCells := HalfBlockCells(base)
	if len(baseCells) != 16 {
		t.Fatalf("32 píxeles de alto son 16 filas, hay %d", len(baseCells))
	}
	for _, l := range HalfBlocks(base) {
		if w := ansi.StringWidth(l); w != 32 {
			t.Fatalf("cada fila mide 32 celdas, mide %d", w)
		}
	}
	// el píxel (9,2) es el contorno de la cabeza (#11111b); pintarlo de rojo puro cambia la celda (9, 1)
	svg := strings.Replace(brand.SleepingSVG, `<rect x="9" y="2" width="1" height="1" fill="#11111b"/>`, `<rect x="9" y="2" width="1" height="1" fill="#ff0000"/>`, 1)
	if svg == brand.SleepingSVG {
		t.Fatal("el píxel de prueba no está en el SVG")
	}
	g, _ := ParseSVG(svg)
	cells := HalfBlockCells(g)
	changed := 0
	for y := range cells {
		for x := 0; x < 32; x++ {
			if cells[y][x] != baseCells[y][x] {
				changed++
				if x != 9 || y != 1 {
					t.Errorf("cambió la celda (%d,%d), que no es la del píxel", x, y)
				}
			}
		}
	}
	if changed != 1 {
		t.Fatalf("debía cambiar 1 celda y cambiaron %d", changed)
	}
	if got := cells[1][9]; !strings.Contains(got, "\x1b[38;2;255;0;0m") {
		t.Errorf("la celda no lleva el rojo exacto del SVG: %q", got)
	}
}

func TestTransparentPixels(t *testing.T) {
	g := Grid{{{255, 0, 0, true}, {}}, {{}, {0, 0, 255, true}}}
	l := HalfBlocks(g)
	if len(l) != 1 || !strings.Contains(l[0], "▀") || !strings.Contains(l[0], "▄") {
		t.Errorf("rojo arriba a la izquierda y azul abajo a la derecha: %q", l)
	}
	if strings.Contains(l[0], "\x1b[48;") {
		t.Error("un píxel transparente no debe pintar fondo")
	}
}

// TestImageIsIntegerNearestNeighbor (M2): el cuadro se escala en un múltiplo entero de su grilla, con vecino más
// cercano (solo los colores del sprite y el transparente, sin mezclas), centrado en un lienzo del tamaño exacto pedido.
func TestImageIsIntegerNearestNeighbor(t *testing.T) {
	f, err := ParseFrames(brand.Sprite16)
	if err != nil {
		t.Fatal(err)
	}
	g := f.Grids["sleep"]
	img, k := g.Image(144, 144)
	if k != 9 || img.Bounds().Dx() != 144 || img.Bounds().Dy() != 144 {
		t.Fatalf("144x144 debía dar factor 9: k=%d %v", k, img.Bounds())
	}
	pal := map[[4]uint8]bool{{0, 0, 0, 0}: true}
	for _, row := range g {
		for _, p := range row {
			if p.Set {
				pal[[4]uint8{p.R, p.G, p.B, 255}] = true
			}
		}
	}
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			c := img.NRGBAAt(x, y)
			if !pal[[4]uint8{c.R, c.G, c.B, c.A}] {
				t.Fatalf("(%d,%d) tiene un color mezclado %v: el escalado no es de vecino más cercano", x, y, c)
			}
			// cada píxel de la grilla es un bloque k x k uniforme
			if c != img.NRGBAAt(x/9*9, y/9*9) {
				t.Fatalf("(%d,%d) no es del mismo bloque que su esquina", x, y)
			}
		}
	}
	// un lienzo mayor que el sprite escalado lo deja centrado; uno menor da factor 1 mínimo
	img2, k2 := g.Image(160, 168)
	if k2 != 10 || img2.Bounds().Dx() != 160 || img2.Bounds().Dy() != 168 {
		t.Errorf("160x168: k=%d %v", k2, img2.Bounds())
	}
	if corner := img2.NRGBAAt(0, 0); corner.A != 0 {
		t.Error("el borde del lienzo debe ser transparente")
	}
}

func TestParseFramesErrors(t *testing.T) {
	for name, text := range map[string]string{
		"letra sin paleta":   "K 000000\n@frame a\nKX\nKK\n",
		"no cuadrado":        "K 000000\n@frame a\nKK\n",
		"animación inválida": "K 000000\n@frame a\nKK\nKK\n@anim x b\n",
	} {
		if _, err := ParseFrames(text); err == nil {
			t.Errorf("%s: debía fallar", name)
		}
	}
	f, _ := ParseFrames(brand.Sprite16)
	if len(f.Anims) < 4 {
		t.Errorf("faltan animaciones: %d", len(f.Anims))
	}
}
