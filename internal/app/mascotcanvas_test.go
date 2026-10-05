package app

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/ui/sprite"
	"github.com/charmbracelet/x/ansi"
)

// touchedEdges devuelve qué bordes de la grilla toca algún píxel visible (el de abajo es el suelo y no cuenta).
func touchedEdges(g sprite.Grid) (left, right, top bool) {
	w, h := g.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !g[y][x].Set {
				continue
			}
			left = left || x == 0
			right = right || x == w-1
			top = top || y == 0
		}
	}
	return
}

// TestMascotFramesDoNotTouchTheCanvasEdge (ORD-015 C.1): ningún píxel visible de ningún cuadro de las 5 animaciones toca el borde del
// lienzo (salvo el suelo): ni en el sprite de Kitty ni en el de cuadrantes. Antes el salto perdía 3 px arriba y el baile 1 px a la derecha.
func TestMascotFramesDoNotTouchTheCanvasEdge(t *testing.T) {
	for kind, fr := range map[string]*sprite.Frames{"kitty": frames36(), "cuadrantes": frames()} {
		for _, a := range fr.Anims {
			for _, name := range a.Frames {
				if l, r, top := touchedEdges(fr.Grids[name]); l || r || top {
					t.Errorf("%s/%s/%s: toca el borde del lienzo (izq=%v der=%v arriba=%v)", kind, a.Name, name, l, r, top)
				}
			}
		}
	}
}

// TestMascotImageNeverClips (ORD-015 C.1): con cualquier tamaño de celda razonable, el cuadro escalado en el lienzo de mascotCols x mascotRows
// celdas deja margen a izquierda, derecha y arriba: la imagen que ve la terminal no corta nada.
func TestMascotImageNeverClips(t *testing.T) {
	for _, cell := range [][2]int{{7, 14}, {8, 16}, {9, 18}, {10, 20}, {12, 24}, {18, 36}, {20, 40}} {
		cw, ch := cell[0], cell[1]
		for _, a := range frames36().Anims {
			for _, name := range a.Frames {
				img, k := frames36().Grids[name].Image(mascotCols*cw, mascotRows*ch)
				if img.Bounds().Dx() != mascotCols*cw || img.Bounds().Dy() != mascotRows*ch {
					t.Fatalf("celda %dx%d: la imagen mide %v y el lienzo pide %dx%d", cw, ch, img.Bounds(), mascotCols*cw, mascotRows*ch)
				}
				b := img.Bounds()
				for y := b.Min.Y; y < b.Max.Y; y++ {
					for x := b.Min.X; x < b.Max.X; x++ {
						if img.NRGBAAt(x, y).A == 0 {
							continue
						}
						if x == b.Min.X || x == b.Max.X-1 || y == b.Min.Y {
							t.Fatalf("celda %dx%d (k=%d) %s/%s: el píxel (%d,%d) toca el borde de %v", cw, ch, k, a.Name, name, x, y, b)
						}
					}
				}
			}
		}
	}
}

// TestMascotFitsAt80x24 (ORD-015 C.1): a 80x24 el lienzo cabe en el estado vacío sin pisar el texto; si en algún tamaño no cabe, no se muestra
// (nunca se muestra cortada).
func TestMascotFitsAt80x24(t *testing.T) {
	r := newEmptyRig(t, 80, 24, false)
	if !r.mascotVisible() {
		t.Fatalf("a 80x24 la mascota debe verse:\n%s", plain(r.AppModel))
	}
	rect := r.mascotRect()
	if !r.layout.Preview.Contains(rect.X, rect.Y) || !r.layout.Preview.Contains(rect.X+rect.W-1, rect.Y+rect.H-1) {
		t.Errorf("el lienzo %+v se sale de la vista previa %+v", rect, r.layout.Preview)
	}
	rows := screen(r.AppModel)
	for _, kind := range []string{restWelcome, restFolder, restEmptyNote, restNone} {
		for _, l := range r.restMessage(kind) {
			text := strings.TrimSpace(ansi.Strip(l))
			for y := rect.Y; y < rect.Y+rect.H && y < len(rows); y++ {
				if text != "" && strings.Contains(ansi.Strip(rows[y]), text) && kind == r.restKind() {
					t.Errorf("la mascota (filas %d-%d) pisa el texto %q en la fila %d", rect.Y, rect.Y+rect.H-1, text, y)
				}
			}
		}
	}
	// en un panel demasiado bajo no se muestra
	for h := 8; h < 24; h++ {
		rr := newEmptyRig(t, 80, h, false)
		if rr.mascotVisible() {
			rect := rr.mascotRect()
			if rect.Y < rr.layout.Preview.Y || rect.Y+rect.H > rr.layout.Preview.Y+rr.layout.Preview.H {
				t.Errorf("80x%d: se muestra y se sale del panel: %+v en %+v", h, rect, rr.layout.Preview)
			}
		}
	}
}
