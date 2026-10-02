package app

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/assets/brand"
	"github.com/MathiasDrizzy/lazymark/internal/ui/sprite"
	"github.com/charmbracelet/x/ansi"
)

// La mascota: con Kitty, el logo de 32x32 (variante c2) sobre un lienzo de 36x36 con sitio para saltar, y sus
// animaciones, hechas desde sus capas, escalados en un factor entero; sin Kitty, un perezoso de 14x7 dibujado
// a mano sobre bloques de cuadrante (2x2 píxeles por celda). Ocupa 8 columnas x 4 filas y duerme abajo a la
// derecha de los estados de reposo. Un clic en ella lanza una animación corta (despertar, saludar con el lápiz,
// bailar, saltar, dar una vuelta), una distinta en cada clic y en orden, y vuelve a dormir sola. Cada animación
// dura de 1 a 1,7 s a unos 16 cuadros por segundo, con cuadros intermedios (aplastar, estirar, giro).
const (
	mascotCols = 8
	mascotRows = 4
	// mascotInterval es lo que dura cada cuadro de la animación (~16 fps).
	mascotInterval = 60 * time.Millisecond
	// Tamaño de celda que se supone mientras la terminal no contesta a CSI 16 t.
	defaultCellW, defaultCellH = 9, 18
)

var (
	mascotOnce        sync.Once
	mascot8, mascot36 *sprite.Frames
)

// frames devuelve los cuadros de cuadrantes (16x8) y frames36 los de Kitty (36x36), leídos una vez. Tienen los
// mismos nombres de cuadro y las mismas animaciones.
func loadMascot() {
	mascotOnce.Do(func() {
		var err error
		if mascot8, err = sprite.ParseFrames(brand.Sprite8); err != nil {
			panic("lazymark: la mascota de cuadrantes embebida no se puede leer: " + err.Error()) // un error del repo
		}
		if mascot36, err = sprite.ParseFrames(brand.Sprite36); err != nil {
			panic("lazymark: la mascota de Kitty embebida no se puede leer: " + err.Error())
		}
	})
}

func frames() *sprite.Frames   { loadMascot(); return mascot8 }
func frames36() *sprite.Frames { loadMascot(); return mascot36 }

// mascotState lleva la animación en curso.
type mascotState struct {
	playing bool
	anim    int // animación que suena
	step    int // cuadro dentro de ella
	next    int // la que sonará en el próximo clic
	gen     int // cambia con cada animación: los ticks de una anterior se ignoran
}

// mascotTickMsg avanza un cuadro de la animación de la generación gen.
type mascotTickMsg struct{ gen int }

// mascotRequestSequence pide a la terminal el tamaño de la celda en píxeles (CSI 16 t), con el que se
// escala la imagen en múltiplos enteros del sprite: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// ("CSI 1 6 t: report xterm char cell size in pixels").
func mascotRequestSequence() string { return ansi.WindowOp(ansi.RequestCellSizeWinOp) }

// cellSize devuelve el tamaño de celda en píxeles (el que contestó la terminal, o uno típico).
func (m *AppModel) cellSize() (w, h int) {
	if m.cellW > 0 && m.cellH > 0 {
		return m.cellW, m.cellH
	}
	return defaultCellW, defaultCellH
}

// mascotShows indica si la mascota se dibuja en el estado de reposo kind del panel r: está activada,
// el panel tiene sitio a su derecha y debajo del texto, y nada la tapa.
func (m *AppModel) mascotShows(kind string, r Rect) bool {
	if !m.c.cfg.Mascot || kind == "" || m.kanbanOn || m.layout.TooSmall {
		return false
	}
	inner, h := r.W-2, r.H-2
	msg := len(m.restMessage(kind))
	textEnd := max(0, (h-msg)/2) + msg // primera fila libre debajo del texto
	return inner >= mascotCols+6 && h-mascotRows > textEnd
}

// mascotVisible es mascotShows para el panel de la vista previa actual.
func (m *AppModel) mascotVisible() bool {
	return !(m.zoom && m.focus == panelPreview) && m.mascotShows(m.restKind(), m.layout.Preview)
}

// mascotRect es el rectángulo de la pantalla que ocupa la mascota (abajo a la derecha del panel).
func (m *AppModel) mascotRect() Rect {
	r := m.layout.Preview
	return Rect{X: r.X + r.W - 2 - mascotCols, Y: r.Y + r.H - 1 - mascotRows, W: mascotCols, H: mascotRows}
}

// frameName es el cuadro que se dibuja: el que toca de la animación, o el dormido.
func (m *AppModel) frameName() string {
	if st := m.mascot; st.playing {
		a := frames().Anims[st.anim]
		return a.Frames[min(st.step, len(a.Frames)-1)]
	}
	return "sleep"
}

// mascotLines devuelve las 4 filas de la mascota: con Kitty, la imagen (el cuadro de 36x36 escalado en un
// factor entero, con vecino más cercano, sobre un lienzo del tamaño exacto en píxeles de las celdas: la
// terminal no tiene que reescalarla ni suavizarla); sin Kitty, bloques de cuadrante del cuadro de 16x8.
func (m *AppModel) mascotLines() []string {
	name := m.frameName()
	cw, ch := m.cellSize()
	img, _ := frames36().Grids[name].Image(mascotCols*cw, mascotRows*ch)
	if lines, ok := m.c.kitty.BlockImage("mascot-"+name, img, mascotCols, mascotRows); ok {
		return lines
	}
	return sprite.Quadrants(frames().Grids[name])
}

// mascotClick maneja un clic: si cae en la mascota, lanza la siguiente animación.
func (m *AppModel) mascotClick(x, y int) (tea.Cmd, bool) {
	if len(m.c.popups) > 0 || !m.mascotVisible() || !m.mascotRect().Contains(x, y) {
		return nil, false
	}
	st := &m.mascot
	st.gen++
	st.playing, st.anim, st.step = true, st.next%len(frames().Anims), 0
	st.next++
	return m.mascotTickCmd(), true
}

func (m *AppModel) mascotTickCmd() tea.Cmd {
	gen := m.mascot.gen
	return tea.Tick(mascotInterval, func(time.Time) tea.Msg { return mascotTickMsg{gen: gen} })
}

// mascotTick avanza un cuadro. La animación termina (y la mascota vuelve a dormir) al acabar sus
// cuadros, y se corta al abrirse un popup, cambiar lo que se ve o apagar la mascota.
func (m *AppModel) mascotTick(t mascotTickMsg) tea.Cmd {
	st := &m.mascot
	if !st.playing || t.gen != st.gen {
		return nil
	}
	st.step++
	if len(m.c.popups) > 0 || !m.mascotVisible() || st.step >= len(frames().Anims[st.anim].Frames) {
		st.playing = false
		return nil
	}
	return m.mascotTickCmd()
}

// stopMascot corta la animación (la mascota vuelve a dormir).
func (m *AppModel) stopMascot() { m.mascot.playing = false }
