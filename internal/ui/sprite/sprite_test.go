package sprite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/assets/brand"
	"github.com/charmbracelet/x/ansi"
)

// logoSVG lee el logo de reposo (el SVG ya no va dentro del binario: solo lo usan los tests).
func logoSVG(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "assets", "brand", "reposo", "lazymark.svg"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseEmbeddedSVG(t *testing.T) {
	g, err := ParseSVG(logoSVG(t))
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

func px(r, g, b uint8) Pixel { return Pixel{r, g, b, true} }

// TestQuadrants: cada celda muestra 2x2 píxeles con a lo sumo dos colores exactos; un píxel transparente deja el
// fondo de la pantalla; y cambiar un píxel cambia solo su celda.
func TestQuadrants(t *testing.T) {
	red, blue := px(255, 0, 0), px(0, 0, 255)
	// rojo arriba, azul abajo → "▀" con fg rojo y bg azul
	l := Quadrants(Grid{{red, red}, {blue, blue}})
	if len(l) != 1 || !strings.Contains(l[0], "▀") || !strings.Contains(l[0], "\x1b[38;2;255;0;0m") || !strings.Contains(l[0], "\x1b[48;2;0;0;255m") {
		t.Errorf("mitad rojo y mitad azul: %q", l)
	}
	// un solo color: bloque lleno
	if l := Quadrants(Grid{{red, red}, {red, red}}); !strings.Contains(l[0], "█") {
		t.Errorf("un color: %q", l)
	}
	// diagonal: rojo arriba a la izquierda y azul abajo a la derecha, el resto transparente: sin fondo
	l = Quadrants(Grid{{red, {}}, {{}, blue}})
	if strings.Contains(l[0], "\x1b[48;") {
		t.Errorf("un píxel transparente no debe pintar fondo: %q", l)
	}
	// todo transparente: un espacio sin color
	if l := Quadrants(Grid{{{}, {}}, {{}, {}}}); l[0] != " " {
		t.Errorf("transparente: %q", l)
	}
	// 16x8 píxeles son 8 celdas x 4 filas
	g := make(Grid, 8)
	for y := range g {
		g[y] = make([]Pixel, 16)
		for x := range g[y] {
			g[y][x] = px(uint8(x*10), uint8(y*10), 40)
		}
	}
	lines := Quadrants(g)
	if len(lines) != 4 {
		t.Fatalf("8 píxeles de alto son 4 filas: %d", len(lines))
	}
	for _, line := range lines {
		if w := ansi.StringWidth(line); w != 8 {
			t.Errorf("cada fila mide 8 celdas, mide %d", w)
		}
	}
	base := QuadrantCells(g)
	g[5][9] = px(255, 255, 255) // pertenece a la celda (4, 2)
	changed := 0
	for y, row := range QuadrantCells(g) {
		for x, cell := range row {
			if cell != base[y][x] {
				changed++
				if x != 4 || y != 2 {
					t.Errorf("cambió la celda (%d,%d), que no es la del píxel", x, y)
				}
			}
		}
	}
	if changed != 1 {
		t.Errorf("debía cambiar 1 celda y cambiaron %d", changed)
	}
}

func loadBoth(t *testing.T) (f8, f36 *Frames) {
	t.Helper()
	var err error
	if f8, err = ParseFrames(brand.Sprite8); err != nil {
		t.Fatal(err)
	}
	if f36, err = ParseFrames(brand.Sprite36); err != nil {
		t.Fatal(err)
	}
	return
}

// TestImageIsIntegerNearestNeighbor (M2): el cuadro se escala en un múltiplo entero de su grilla, con vecino más
// cercano (solo los colores del sprite y el transparente, sin mezclas), centrado a los lados y apoyado en el borde
// de abajo del lienzo del tamaño exacto pedido.
func TestImageIsIntegerNearestNeighbor(t *testing.T) {
	_, f36 := loadBoth(t)
	g := f36.Grids["sleep"]
	pal := map[[4]uint8]bool{{0, 0, 0, 0}: true}
	for _, row := range g {
		for _, p := range row {
			if p.Set {
				pal[[4]uint8{p.R, p.G, p.B, 255}] = true
			}
		}
	}
	for _, c := range []struct{ w, h, k int }{{144, 144, 4}, {72, 72, 2}, {80, 84, 2}, {160, 168, 4}} {
		img, k := g.Image(c.w, c.h)
		if k != c.k || img.Bounds().Dx() != c.w || img.Bounds().Dy() != c.h {
			t.Fatalf("%dx%d debía dar factor %d: k=%d %v", c.w, c.h, c.k, k, img.Bounds())
		}
		ox, oy := (c.w-36*k)/2, c.h-36*k
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				p := img.NRGBAAt(x, y)
				if !pal[[4]uint8{p.R, p.G, p.B, p.A}] {
					t.Fatalf("%dx%d (%d,%d) tiene un color mezclado %v", c.w, c.h, x, y, p)
				}
				if y >= oy && x >= ox && x < ox+36*k { // cada píxel de la grilla es un bloque k x k uniforme
					bx, by := ox+(x-ox)/k*k, oy+(y-oy)/k*k
					if p != img.NRGBAAt(bx, by) {
						t.Fatalf("%dx%d (%d,%d) no es del mismo bloque que su esquina", c.w, c.h, x, y)
					}
				} else if p.A != 0 {
					t.Fatalf("%dx%d (%d,%d): el margen del lienzo debe ser transparente", c.w, c.h, x, y)
				}
			}
		}
	}
	// el sprite descansa abajo: la última fila del lienzo es la última de la grilla (aquí, transparente si el
	// margen de abajo del cuadro lo es); un lienzo más alto deja el espacio de más arriba
	img, _ := g.Image(144, 200)
	for x := 0; x < 144; x++ {
		if img.NRGBAAt(x, 0).A != 0 {
			t.Fatal("con un lienzo más alto el espacio sobrante queda arriba")
		}
	}
}

func TestParseFramesErrors(t *testing.T) {
	for name, text := range map[string]string{
		"letra sin paleta":   "K 000000\n@frame a\nKX\nKK\n",
		"filas desiguales":   "K 000000\n@frame a\nKK\nK\n",
		"tamaños distintos":  "K 000000\n@frame a\nKK\nKK\n@frame b\nKKK\nKKK\n",
		"animación inválida": "K 000000\n@frame a\nKK\nKK\n@anim x b\n",
		"vacío":              "K 000000\n@frame a\n",
	} {
		if _, err := ParseFrames(text); err == nil {
			t.Errorf("%s: debía fallar", name)
		}
	}
	if f, err := ParseFrames("K 000000\n@frame a\nKKKK\nKKKK\n@anim x a a\n"); err != nil || f.Grids["a"] == nil {
		t.Errorf("un cuadro que no es cuadrado es válido: %v", err)
	}
}

// TestSprites (C.2): los dos sprites (36x36 para Kitty y 16x8 para cuadrantes) tienen los mismos cuadros, animaciones y
// secuencias; el cuadro dormido de Kitty es el logo (reposo/lazymark.svg) píxel por píxel, corrido (2,4); y las
// animaciones se mueven: cada una cambia el cuadro casi en cada paso y todas terminan dormidas.
func TestSprites(t *testing.T) {
	f8, f36 := loadBoth(t)
	for name, g := range f36.Grids {
		if w, h := g.Size(); w != 36 || h != 36 {
			t.Errorf("%s mide %dx%d", name, w, h)
		}
		if _, ok := f8.Grids[name]; !ok {
			t.Errorf("el cuadro %s está en 36x36 y no en 16x8", name)
		}
	}
	for name, g := range f8.Grids {
		if w, h := g.Size(); w != 16 || h != 8 {
			t.Errorf("%s mide %dx%d", name, w, h)
		}
		if _, ok := f36.Grids[name]; !ok {
			t.Errorf("el cuadro %s está en 16x8 y no en 36x36", name)
		}
	}
	if len(f36.Anims) != 5 || len(f8.Anims) != 5 {
		t.Fatalf("animaciones: %d y %d, se esperaban 5 (wake, wave, dance, jump, spin)", len(f36.Anims), len(f8.Anims))
	}
	for i, a := range f36.Anims {
		if a.Name != f8.Anims[i].Name || strings.Join(a.Frames, " ") != strings.Join(f8.Anims[i].Frames, " ") {
			t.Errorf("la animación %s difiere entre los dos sprites", a.Name)
		}
		if a.Frames[len(a.Frames)-1] != "sleep" {
			t.Errorf("%s no termina dormida", a.Name)
		}
		moves := 0
		for j := 1; j < len(a.Frames); j++ {
			if a.Frames[j] != a.Frames[j-1] {
				moves++
			}
		}
		if moves*100 < (len(a.Frames)-1)*70 {
			t.Errorf("%s: solo cambia el cuadro en %d de %d pasos (se pide >= 70%%): no es fluida", a.Name, moves, len(a.Frames)-1)
		}
		if len(a.Frames) < 16 {
			t.Errorf("%s tiene %d cuadros: faltan cuadros intermedios", a.Name, len(a.Frames))
		}
	}
	logo, _ := ParseSVG(logoSVG(t))
	sleep := f36.Grids["sleep"]
	for y := range sleep {
		for x := range sleep[y] {
			var want Pixel
			if lx, ly := x-2, y-4; lx >= 0 && ly >= 0 && lx < 32 && ly < 32 {
				want = logo[ly][lx]
			}
			if sleep[y][x] != want {
				t.Fatalf("el cuadro dormido difiere del logo en (%d,%d): %v vs %v", x, y, sleep[y][x], want)
			}
		}
	}
	if _, k := sleep.Image(8*18, 4*36); k != 4 {
		t.Errorf("a 144x144 px el factor debe ser 4: %d", k)
	}
	if _, k := sleep.Image(8*9, 4*18); k != 2 {
		t.Errorf("a 72x72 px el factor debe ser 2: %d", k)
	}
}

// TestRobustAgainstBadGrids: una grilla vacía o con filas desiguales no hace entrar en pánico a Image ni a
// QuadrantCells, y un lienzo absurdo no reserva memoria (hallazgos de la segunda opinión del agente de apoyo).
func TestRobustAgainstBadGrids(t *testing.T) {
	p := px(1, 2, 3)
	var empty Grid
	if img, k := empty.Image(10, 10); k != 1 || img.Bounds().Dx() != 1 {
		t.Errorf("grilla vacía: %v %d", img.Bounds(), k)
	}
	uneven := Grid{{p, p, p, p}, {p}}
	if img, k := uneven.Image(10, 10); k != 1 || img.Bounds().Dx() != 1 {
		t.Errorf("filas desiguales: %v %d", img.Bounds(), k)
	}
	if QuadrantCells(uneven) != nil || QuadrantCells(empty) != nil && len(QuadrantCells(empty)) != 0 {
		t.Error("QuadrantCells no debe dibujar una grilla mal formada")
	}
	ok := Grid{{p, p}, {p, p}}
	if img, _ := ok.Image(1<<30, 1<<30); img.Bounds().Dx() != 1 {
		t.Errorf("un lienzo de 1<<30 px no debe reservarse: %v", img.Bounds())
	}
}
