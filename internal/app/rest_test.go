package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

// newEmptyRig crea el modelo con la carpeta de notas vacía.
func newEmptyRig(t *testing.T, w, h int, kittySupport bool) *imageRig {
	t.Helper()
	r := newImageRig(t, w, h, kittySupport)
	entries, _ := os.ReadDir(r.c.store.BaseDir)
	for _, e := range entries {
		os.RemoveAll(filepath.Join(r.c.store.BaseDir, e.Name()))
	}
	r.afterChange()
	r.Update(tea.WindowSizeMsg{Width: w, Height: h})
	r.take()
	return r
}

func gridOf(content string, w, h int) *uv.ScreenBuffer {
	buf := uv.NewScreenBuffer(w, h)
	buf.Method = ansi.GraphemeWidth
	uv.NewStyledString(content).Draw(buf, buf.Bounds())
	return &buf
}

// TestSleepingSlothHalfBlocks (Z1): con la carpeta vacía y sin Kitty, a 120x35 se ven la bienvenida y
// el perezoso en 32x16 celdas de medios bloques con los colores exactos del SVG (celda por celda, en la
// posición donde está centrado); a 60x20 no cabe, no se muestra y el layout no se rompe.
func TestSleepingSlothHalfBlocks(t *testing.T) {
	r := newEmptyRig(t, 120, 35, false)
	out := plain(r.AppModel)
	for _, want := range []string{"Tu carpeta de notas está vacía", "Pulsa c para crear una nota y ? para ver los atajos"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	want := sleeperHalfBlocks()
	if len(want) != restRows {
		t.Fatalf("el perezoso tiene %d filas, se esperaban %d", len(want), restRows)
	}
	// ubicación: la primera fila no vacía del perezoso, buscada en la pantalla
	lines := screen(r.AppModel)
	y0, x0 := -1, -1
	for i, l := range want {
		pl := ansi.Strip(l)
		if strings.TrimSpace(pl) == "" {
			continue
		}
		lead := len(pl) - len(strings.TrimLeft(pl, " "))
		for y, sl := range lines {
			if idx := strings.Index(ansi.Strip(sl), strings.TrimSpace(pl)); idx >= 0 {
				y0, x0 = y-i, ansi.StringWidth(ansi.Strip(sl)[:idx])-lead
				break
			}
		}
		break
	}
	if y0 < 0 {
		t.Fatalf("no se encontró el perezoso en la pantalla:\n%s", out)
	}
	got := gridOf(r.View().Content, r.w, r.h)
	exp := gridOf(strings.Join(want, "\n"), restCols, restRows)
	for y := 0; y < restRows; y++ {
		for x := 0; x < restCols; x++ {
			e, g := exp.CellAt(x, y), got.CellAt(x0+x, y0+y)
			if e.Content != g.Content || hexOrEmpty(e.Style.Fg) != hexOrEmpty(g.Style.Fg) || (e.Style.Bg != nil && hexOrEmpty(e.Style.Bg) != hexOrEmpty(g.Style.Bg)) {
				t.Fatalf("la celda (%d,%d) del perezoso difiere: se esperaba %q fg=%s bg=%s y hay %q fg=%s bg=%s",
					x, y, e.Content, hexOrEmpty(e.Style.Fg), hexOrEmpty(e.Style.Bg), g.Content, hexOrEmpty(g.Style.Fg), hexOrEmpty(g.Style.Bg))
			}
		}
	}

	// a 60x20 no cabe: sin perezoso, con el texto, y todo mide lo que debe
	small := newEmptyRig(t, 60, 20, false)
	if strings.ContainsAny(plain(small.AppModel), "▀▄") {
		t.Error("a 60x20 no debe mostrarse el perezoso")
	}
	if !strings.Contains(plain(small.AppModel), "carpeta de notas está vacía") {
		t.Errorf("a 60x20 debe verse el texto de bienvenida:\n%s", plain(small.AppModel))
	}
	for i, l := range screen(small.AppModel) {
		if w := ansi.StringWidth(l); w != 60 {
			t.Errorf("a 60x20 la línea %d mide %d", i, w)
		}
	}
}

// TestSleepingSlothKitty (Z1): con Kitty, la imagen embebida se transmite una vez, por la ruta
// asíncrona, y se ve como 32x16 celdas de placeholders; mientras llega se ven los medios bloques; a
// 60x20 no hay imagen.
func TestSleepingSlothKitty(t *testing.T) {
	r := newEmptyRig(t, 120, 35, true)
	if placeholderCells(r.AppModel) != 0 || !strings.ContainsAny(plain(r.AppModel), "▀▄") {
		t.Error("mientras carga la imagen debe verse el perezoso en medios bloques")
	}
	r.settle()
	if n := placeholderCells(r.AppModel); n != restCols*restRows {
		t.Errorf("celdas de imagen = %d, se esperaban %d", n, restCols*restRows)
	}
	if strings.ContainsAny(plain(r.AppModel), "▀▄") {
		t.Error("con la imagen lista ya no se ven los medios bloques")
	}
	if got := r.take(); transmits(got) != 1 {
		t.Errorf("debía transmitirse 1 imagen embebida: %d", transmits(got))
	}
	// no se retransmite al repintar ni al abrir un popup
	press(r.AppModel, "?", "esc")
	if got := r.take(); len(got) != 0 {
		t.Errorf("repintar no debe emitir nada: %q", got)
	}
	small := newEmptyRig(t, 60, 20, true)
	small.settle()
	if strings.Count(small.View().Content, string(kitty.Placeholder)) != 0 || len(small.c.kitty.TakeJobs()) != 0 {
		t.Error("a 60x20 no debe pedirse ni mostrarse la imagen")
	}
	for i, l := range screen(small.AppModel) {
		if w := ansi.StringWidth(l); w != 60 {
			t.Errorf("a 60x20 la línea %d mide %d", i, w)
		}
	}
}

// TestRestStates: el perezoso también aparece sin selección y en una carpeta sin notas, con su texto.
func TestRestStates(t *testing.T) {
	r := newEmptyRig(t, 120, 35, false)
	if err := os.Mkdir(filepath.Join(r.c.store.BaseDir, "vacia"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.afterChange()
	r.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	out := plain(r.AppModel)
	if !strings.Contains(out, "Carpeta vacía") || !strings.ContainsAny(out, "▀▄") {
		t.Errorf("una carpeta sin notas debe mostrar el perezoso y 'Carpeta vacía':\n%s", out)
	}
	// con notas, no hay perezoso
	n := newTestModel(t, 120, 35)
	if strings.ContainsAny(plain(n), "▀▄") {
		t.Error("con notas no debe verse el perezoso")
	}
}
