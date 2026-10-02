package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/ui/sprite"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
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

// mascotMatches comprueba, celda por celda, que la mascota de la pantalla es el cuadro name en medios bloques.
func mascotMatches(t *testing.T, r *imageRig, name string) bool {
	t.Helper()
	want := sprite.HalfBlocks(frames().Grids[name])
	rect := r.mascotRect()
	got := gridOf(r.View().Content, r.w, r.h)
	exp := gridOf(strings.Join(want, "\n"), mascotCols, mascotRows)
	for y := 0; y < mascotRows; y++ {
		for x := 0; x < mascotCols; x++ {
			e, g := exp.CellAt(x, y), got.CellAt(rect.X+x, rect.Y+y)
			if e.Content != g.Content || hexOrEmpty(e.Style.Fg) != hexOrEmpty(g.Style.Fg) || (e.Style.Bg != nil && hexOrEmpty(e.Style.Bg) != hexOrEmpty(g.Style.Bg)) {
				return false
			}
		}
	}
	return true
}

// TestMascotBottomRight (M1): a 120x35 y 80x24, con la carpeta vacía, una carpeta sin notas, una nota vacía
// y sin selección, la mascota (16x8 celdas, sin Kitty en medios bloques con los colores exactos del
// cuadro) está abajo a la derecha del panel de la vista previa y el texto queda arriba, sin tapar.
func TestMascotBottomRight(t *testing.T) {
	for _, sz := range []struct{ w, h int }{{120, 35}, {80, 24}} {
		r := newEmptyRig(t, sz.w, sz.h, false)
		check := func(state, wantText string) {
			t.Helper()
			out := plain(r.AppModel)
			if !strings.Contains(out, wantText) {
				t.Fatalf("%dx%d %s: falta %q:\n%s", sz.w, sz.h, state, wantText, out)
			}
			rect, pv := r.mascotRect(), r.layout.Preview
			if rect.X+rect.W != pv.X+pv.W-2 || rect.Y+rect.H != pv.Y+pv.H-1 || rect.W != 16 || rect.H != 8 {
				t.Errorf("%dx%d %s: la mascota %+v no está abajo a la derecha de %+v", sz.w, sz.h, state, rect, pv)
			}
			if !mascotMatches(t, r, "sleep") {
				t.Errorf("%dx%d %s: la mascota no es el cuadro dormido en esa posición:\n%s", sz.w, sz.h, state, out)
			}
			// el texto está en filas que la mascota no ocupa
			for y, l := range strings.Split(out, "\n") {
				if strings.Contains(l, wantText) && y >= rect.Y {
					t.Errorf("%dx%d %s: el texto %q (fila %d) cae donde empieza la mascota (fila %d)", sz.w, sz.h, state, wantText, y, rect.Y)
				}
			}
		}
		check("carpeta vacía", "Tu carpeta de notas está vacía")

		// una carpeta sin notas
		os.Mkdir(filepath.Join(r.c.store.BaseDir, "vacia"), 0o755)
		r.afterChange()
		r.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		check("carpeta sin notas", "Carpeta vacía")

		// una nota vacía
		os.Remove(filepath.Join(r.c.store.BaseDir, "vacia"))
		os.WriteFile(filepath.Join(r.c.store.BaseDir, "vacia.md"), nil, 0o644)
		r.afterChange()
		r.notes.selectPath(filepath.Join(r.c.store.BaseDir, "vacia.md"))
		r.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		check("nota vacía", "Nota vacía")

		// con una nota con texto no hay mascota
		os.WriteFile(filepath.Join(r.c.store.BaseDir, "vacia.md"), []byte("# Hola\n\ntexto\n"), 0o644)
		r.afterChange()
		r.notes.selectPath(filepath.Join(r.c.store.BaseDir, "vacia.md"))
		if strings.ContainsAny(plain(r.AppModel), "▀▄") {
			t.Errorf("%dx%d: con una nota con contenido no debe verse la mascota", sz.w, sz.h)
		}
	}
	// a 60x20 no cabe: sin mascota y sin romper el layout
	small := newEmptyRig(t, 60, 20, false)
	if strings.ContainsAny(plain(small.AppModel), "▀▄") || !strings.Contains(plain(small.AppModel), "carpeta de notas está vacía") {
		t.Errorf("a 60x20 no cabe: sin mascota y con el texto:\n%s", plain(small.AppModel))
	}
	for i, l := range screen(small.AppModel) {
		if w := ansi.StringWidth(l); w != 60 {
			t.Errorf("a 60x20 la línea %d mide %d", i, w)
		}
	}
}

// TestMascotKitty (M1, M2): con Kitty la mascota son 16x8 celdas de placeholders de una imagen cuyo tamaño
// en píxeles es el de esas celdas (con el tamaño de celda que contestó la terminal), una sola transmisión por cuadro.
func TestMascotKitty(t *testing.T) {
	r := newEmptyRig(t, 120, 35, true)
	r.Update(uv.CellSizeEvent{Width: 10, Height: 21})
	r.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	if r.cellW != 10 || r.cellH != 21 {
		t.Fatalf("el tamaño de celda no se guardó: %dx%d", r.cellW, r.cellH)
	}
	if n := placeholderCells(r.AppModel); n != 16*8 {
		t.Errorf("celdas de la mascota = %d, se esperaban 128", n)
	}
	got := r.take()
	if transmits(got) != 1 {
		t.Fatalf("debía transmitirse 1 imagen: %d", transmits(got))
	}
	// no se retransmite al repintar
	_ = r.View()
	if again := r.take(); len(again) != 0 {
		t.Errorf("repintar no debe emitir nada: %q", again)
	}
	// la imagen tiene el tamaño exacto de las celdas, con el sprite en múltiplos enteros (nitidez)
	img, k := frames32().Grids["sleep"].Image(16*10, 8*21)
	if b := img.Bounds(); b.Dx() != 160 || b.Dy() != 168 || k != 5 {
		t.Errorf("imagen %dx%d con factor %d, se esperaba 160x168 con 5 (el cuadro de 32x32)", b.Dx(), b.Dy(), k)
	}
}

// TestMascotSetting (M3): con Mascota = no no aparece en ningún lado (ni medios bloques ni placeholders), se
// puede cambiar en vivo desde Ajustes y un clic donde estaba no hace nada.
func TestMascotSetting(t *testing.T) {
	r := newEmptyRig(t, 120, 35, true)
	r.settle()
	if !mascotVisibleOnScreen(r) {
		t.Fatal("por defecto debe verse")
	}
	rect := r.mascotRect()
	press(r.AppModel, ",")
	p := r.c.top().(*settingsPopup)
	p.list.cursor = int(setMascot)
	if p.label(setMascot) != "Mascota" || p.value(setMascot) != "sí" {
		t.Errorf("fila: %q = %q", p.label(setMascot), p.value(setMascot))
	}
	press(r.AppModel, "right")
	press(r.AppModel, "esc")
	if r.c.cfg.Mascot || p.value(setMascot) != "no" {
		t.Fatal("el cambio no se aplicó")
	}
	if mascotVisibleOnScreen(r) {
		t.Errorf("con Mascota = no no debe quedar nada:\n%s", plain(r.AppModel))
	}
	click(r.AppModel, rect.X+3, rect.Y+3)
	if r.mascot.playing {
		t.Error("un clic donde estaba no debe animar nada")
	}
	r2 := newEmptyRig(t, 120, 35, false)
	r2.c.cfg.Mascot = false
	if mascotVisibleOnScreen(r2) {
		t.Error("sin Kitty y con Mascota = no tampoco debe verse")
	}
}

func mascotVisibleOnScreen(r *imageRig) bool {
	return strings.ContainsAny(plain(r.AppModel), "▀▄") || placeholderCells(r.AppModel) > 0
}

// TestMascotAnimates (M4): un clic en la mascota lanza una animación por ticks (cada clic la siguiente: abrir los
// ojos, saludar, bailar, dar una vuelta), cada cuadro se ve en pantalla, al terminar vuelve a dormir; con un popup
// abierto no se anima y abrir uno corta la que suena.
func TestMascotAnimates(t *testing.T) {
	r := newEmptyRig(t, 120, 35, false)
	rect := r.mascotRect()
	for i, anim := range frames().Anims {
		if cmd, ok := r.mascotClick(rect.X+5, rect.Y+4); !ok || cmd == nil {
			t.Fatalf("clic %d: no lanzó la animación", i)
		}
		if !r.mascot.playing || frames().Anims[r.mascot.anim].Name != anim.Name {
			t.Fatalf("clic %d: debía sonar %q y suena %q", i, anim.Name, frames().Anims[r.mascot.anim].Name)
		}
		seen := []string{r.frameName()}
		for r.mascot.playing {
			r.Update(mascotTickMsg{gen: r.mascot.gen})
			if r.mascot.playing {
				seen = append(seen, r.frameName())
				if !mascotMatches(t, r, r.frameName()) {
					t.Fatalf("%s: el cuadro %q no se ve en pantalla", anim.Name, r.frameName())
				}
			}
		}
		if strings.Join(seen, " ") != strings.Join(anim.Frames, " ") {
			t.Errorf("%s: cuadros %v, se esperaban %v", anim.Name, seen, anim.Frames)
		}
		if r.frameName() != "sleep" || !mascotMatches(t, r, "sleep") {
			t.Errorf("%s: al terminar debe volver a dormir", anim.Name)
		}
	}
	// después de la última, la secuencia vuelve a empezar
	r.mascotClick(rect.X+1, rect.Y+1)
	if frames().Anims[r.mascot.anim].Name != frames().Anims[0].Name {
		t.Error("tras la última animación empieza otra vez por la primera")
	}
	// un clic fuera de la mascota no hace nada
	r.stopMascot()
	if _, ok := r.mascotClick(rect.X-2, rect.Y); ok {
		t.Error("un clic fuera de la mascota no debe animarla")
	}
	// un tick de una animación anterior se ignora
	r.mascotClick(rect.X+1, rect.Y+1)
	old := r.mascot.gen
	r.mascotClick(rect.X+1, rect.Y+1)
	step := r.mascot.step
	r.Update(mascotTickMsg{gen: old})
	if r.mascot.step != step {
		t.Error("el tick de una generación anterior debe ignorarse")
	}
	// con un popup abierto: no se anima, y abrir uno corta la animación
	r.stopMascot()
	press(r.AppModel, "?")
	if _, ok := r.mascotClick(rect.X+1, rect.Y+1); ok || r.mascot.playing {
		t.Error("con un popup abierto no debe animarse")
	}
	press(r.AppModel, "esc")
	r.mascotClick(rect.X+1, rect.Y+1)
	press(r.AppModel, "?")
	r.Update(mascotTickMsg{gen: r.mascot.gen})
	if r.mascot.playing {
		t.Error("al abrir un popup la animación debe cortarse")
	}
}

// TestMascotClickViaMouse: el clic real del mouse (MouseClickMsg) llega a la mascota.
func TestMascotClickViaMouse(t *testing.T) {
	r := newEmptyRig(t, 120, 35, false)
	rect := r.mascotRect()
	click(r.AppModel, rect.X+8, rect.Y+4)
	if !r.mascot.playing {
		t.Error("el clic del mouse en la mascota debe lanzar la animación")
	}
}
