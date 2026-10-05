package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// mascotMatches comprueba, celda por celda, que la mascota de la pantalla es el cuadro name en bloques de cuadrante.
func mascotMatches(t *testing.T, r *imageRig, name string) bool {
	t.Helper()
	want := sprite.Quadrants(frames().Grids[name])
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
// y sin selección, la mascota (10x5 celdas, sin Kitty en bloques de cuadrante con los colores exactos del
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
			if rect.X+rect.W != pv.X+pv.W-2 || rect.Y+rect.H != pv.Y+pv.H-1 || rect.W != mascotCols || rect.H != mascotRows {
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
		if strings.ContainsAny(plain(r.AppModel), quadrantRunes) {
			t.Errorf("%dx%d: con una nota con contenido no debe verse la mascota", sz.w, sz.h)
		}
	}
	// a 60x20 (el mínimo) la mascota de 10x5 cabe debajo del texto, sin romper el layout ni taparlo
	small := newEmptyRig(t, 60, 20, false)
	if !strings.Contains(plain(small.AppModel), "carpeta de notas está vacía") {
		t.Errorf("a 60x20 debe verse el texto:\n%s", plain(small.AppModel))
	}
	if rect := small.mascotRect(); small.mascotVisible() && !small.layout.Preview.Contains(rect.X, rect.Y) {
		t.Errorf("a 60x20 la mascota %+v se sale de la vista previa %+v", rect, small.layout.Preview)
	}
	for i, l := range screen(small.AppModel) {
		if w := ansi.StringWidth(l); w != 60 {
			t.Errorf("a 60x20 la línea %d mide %d", i, w)
		}
	}
}

// TestMascotKitty (M1, M2): con Kitty la mascota son 10x5 celdas de placeholders de una imagen cuyo tamaño
// en píxeles es el de esas celdas (con el tamaño de celda que contestó la terminal), una sola transmisión por cuadro.
func TestMascotKitty(t *testing.T) {
	r := newEmptyRig(t, 120, 35, true)
	r.Update(uv.CellSizeEvent{Width: 10, Height: 21})
	r.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	if r.cellW != 10 || r.cellH != 21 {
		t.Fatalf("el tamaño de celda no se guardó: %dx%d", r.cellW, r.cellH)
	}
	if n := placeholderCells(r.AppModel); n != mascotCols*mascotRows {
		t.Errorf("celdas de la mascota = %d, se esperaban %d", n, mascotCols*mascotRows)
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
	img, k := frames36().Grids["sleep"].Image(mascotCols*10, mascotRows*21)
	if b := img.Bounds(); b.Dx() != 100 || b.Dy() != 105 || k != 2 {
		t.Errorf("imagen %dx%d con factor %d, se esperaba 100x105 con 2 (el cuadro de 40x40)", b.Dx(), b.Dy(), k)
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
	click(r.AppModel, rect.X+3, rect.Y+2)
	if r.mascot.playing {
		t.Error("un clic donde estaba no debe animar nada")
	}
	r2 := newEmptyRig(t, 120, 35, false)
	r2.c.cfg.Mascot = false
	if mascotVisibleOnScreen(r2) {
		t.Error("sin Kitty y con Mascota = no tampoco debe verse")
	}
}

// quadrantRunes son los bloques con los que se dibuja la mascota sin Kitty.
const quadrantRunes = "▀▄▘▝▖▗▌▐▚▞▛▜▙▟█"

func mascotVisibleOnScreen(r *imageRig) bool {
	return strings.ContainsAny(plain(r.AppModel), quadrantRunes) || placeholderCells(r.AppModel) > 0
}

// TestMascotAnimates (M4): un clic en la mascota lanza una animación por ticks (cada clic la siguiente: abrir los
// ojos, saludar, bailar, saltar, dar una vuelta), cada cuadro se ve en pantalla, al terminar vuelve a dormir; con un popup
// abierto no se anima y abrir uno corta la que suena.
func TestMascotAnimates(t *testing.T) {
	r := newEmptyRig(t, 120, 35, false)
	rect := r.mascotRect()
	for i, anim := range frames().Anims {
		if cmd, ok := r.mascotClick(rect.X+5, rect.Y+3); !ok || cmd == nil {
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
	click(r.AppModel, rect.X+4, rect.Y+2)
	if !r.mascot.playing {
		t.Error("el clic del mouse en la mascota debe lanzar la animación")
	}
}

// TestMascotTimingAndIdle (C.2): cada animación dura de 1 a 2 s a 15-20 cuadros por segundo; el reloj solo corre
// mientras suena (al terminar no queda ningún tick pendiente: sin CPU en reposo) y un tick suelto no lo reactiva.
func TestMascotTimingAndIdle(t *testing.T) {
	if fps := time.Second / mascotInterval; fps < 15 || fps > 20 {
		t.Errorf("%d cuadros por segundo, se piden 15-20", fps)
	}
	for _, a := range frames().Anims {
		if d := time.Duration(len(a.Frames)) * mascotInterval; d < time.Second || d > 2*time.Second {
			t.Errorf("%s dura %v, se piden 1-2 s", a.Name, d)
		}
	}
	r := newEmptyRig(t, 120, 35, false)
	rect := r.mascotRect()
	if cmd := r.mascot.playing; cmd {
		t.Fatal("en reposo no hay animación")
	}
	if _, cmd := r.Update(mascotTickMsg{gen: r.mascot.gen}); cmd != nil {
		t.Error("un tick en reposo no debe pedir otro")
	}
	r.mascotClick(rect.X+1, rect.Y+1)
	for r.mascot.playing {
		_, cmd := r.Update(mascotTickMsg{gen: r.mascot.gen})
		if r.mascot.playing && cmd == nil {
			t.Fatal("mientras suena hace falta el siguiente tick")
		}
		if !r.mascot.playing && cmd != nil {
			t.Error("al terminar no debe quedar un tick pendiente")
		}
	}
}

// TestMascotKittyAnimationTransmitsOnce (C.2): con Kitty cada cuadro distinto se transmite una sola vez: repetir la
// animación no vuelve a enviar imágenes.
func TestMascotKittyAnimationTransmitsOnce(t *testing.T) {
	r := newEmptyRig(t, 120, 35, true)
	r.Update(uv.CellSizeEvent{Width: 18, Height: 36})
	r.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	r.take()
	rect := r.mascotRect()
	play := func() int {
		n := 0
		r.mascot.next = 0 // siempre la misma animación
		r.mascotClick(rect.X+1, rect.Y+1)
		for r.mascot.playing {
			r.Update(mascotTickMsg{gen: r.mascot.gen})
			_ = r.View()
			n += transmits(r.take())
		}
		return n
	}
	first := play()
	if first < 5 {
		t.Errorf("la primera vez deben transmitirse los cuadros nuevos: %d", first)
	}
	if second := play(); second != 0 {
		t.Errorf("la segunda vez no debe transmitirse nada: %d", second)
	}
}

// TestCellSizeIsClamped: un tamaño de celda absurdo contestado por la terminal se ignora (no se reserva una imagen enorme).
func TestCellSizeIsClamped(t *testing.T) {
	r := newEmptyRig(t, 120, 35, true)
	r.Update(uv.CellSizeEvent{Width: 18, Height: 36})
	for _, bad := range []uv.CellSizeEvent{{Width: 1 << 20, Height: 1 << 20}, {Width: 0, Height: 0}, {Width: -4, Height: 9}, {Width: 300, Height: 20}} {
		r.Update(bad)
		if r.cellW != 18 || r.cellH != 36 {
			t.Errorf("%+v cambió el tamaño de celda a %dx%d", bad, r.cellW, r.cellH)
		}
	}
	_ = r.View()
}

// TestEmbeddedMascotHasSleepFrame: los dos sprites tienen el cuadro "sleep" (el de reposo) con el tamaño esperado.
func TestEmbeddedMascotHasSleepFrame(t *testing.T) {
	for name, f := range map[string]*sprite.Frames{"cuadrantes": frames(), "kitty": frames36()} {
		if f.Grids["sleep"] == nil {
			t.Errorf("%s: falta el cuadro sleep", name)
		}
		for _, a := range f.Anims {
			for _, fr := range a.Frames {
				if f.Grids[fr] == nil {
					t.Errorf("%s/%s: falta el cuadro %s", name, a.Name, fr)
				}
			}
		}
	}
	if w, h := frames().Grids["sleep"].Size(); w != 2*mascotCols || h != 2*mascotRows {
		t.Errorf("el sprite de cuadrantes mide %dx%d y %dx%d celdas piden %dx%d", w, h, mascotCols, mascotRows, 2*mascotCols, 2*mascotRows)
	}
}
