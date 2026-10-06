package app

import (
	"image"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/assets/brand"
	"github.com/MathiasDrizzy/lazymark/internal/ui/sprite"
	"github.com/charmbracelet/x/ansi"
)

// La mascota: con Kitty, el logo de 32x32 (variante c2) sobre un lienzo de 40x40 con sitio para saltar y moverse
// (ningún cuadro de ninguna animación toca el borde: ORD-015; el de 36x36 cortaba el salto y el baile), y sus
// animaciones, hechas desde sus capas, escalados en un factor entero; sin Kitty, un perezoso de 14x7 dibujado
// a mano sobre bloques de cuadrante (2x2 píxeles por celda, lienzo de 20x10). El sprite mide lo mismo que antes:
// solo creció el lienzo. Ocupa 10 columnas x 5 filas y duerme abajo a la
// derecha de los estados de reposo. Un clic en ella lanza una animación corta (despertar, saludar con el lápiz,
// bailar, saltar, dar una vuelta), una distinta en cada clic y en orden, y vuelve a dormir sola. Cada animación
// dura de 1 a 1,7 s a unos 16 cuadros por segundo, con cuadros intermedios (aplastar, estirar, giro).
const (
	mascotCols = 10
	mascotRows = 5
	// mascotInterval es lo que dura cada cuadro de la animación (~16 fps).
	mascotInterval = 60 * time.Millisecond
	// Tamaño de celda que se supone mientras la terminal no contesta a CSI 16 t.
	defaultCellW, defaultCellH = 9, 18
	// maxCellPx es el mayor lado de celda que se acepta de la terminal.
	maxCellPx = 256
)

var (
	mascotOnce        sync.Once
	mascot8, mascot36 *sprite.Frames
)

// frames devuelve los cuadros de cuadrantes (20x10) y frames36 los de Kitty (40x40), leídos una vez. Tienen los
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

	clicked   bool      // ya le hicieron clic en esta sesión: el "click me!" no vuelve
	lastInput time.Time // la última tecla o clic: el "click me!" cuenta el tiempo sin interacción desde aquí
}

// El "click me!" sobre la mascota (pedido de Mathias: muy sutil y solo algunas veces). Cadencia: tras 20 segundos sin tocar nada aparece 15 segundos y se repite cada
// 60 segundos de quietud (antes 2 min / 3 s / 4 min, y luego 20 s / 6 s / 90 s: una ventana de 6 s se pierde con solo mirar a otro lado); cualquier tecla o clic reinicia la cuenta, así nunca sale mientras se trabaja, y tras el primer clic en la mascota no vuelve más.
const (
	hintIdleAfter = 20 * time.Second
	hintEvery     = 60 * time.Second
	hintShowFor   = 15 * time.Second
	hintText      = "click me!"
)

// hintNow es el reloj del "click me!" (las pruebas lo reemplazan).
var hintNow = time.Now

// hintTickMsg despierta al modelo en la próxima frontera del "click me!" (cuando toca aparecer o desaparecer), para que se redibuje sin que nadie toque nada.
// No es un tick fijo: en quietud son un par de avisos por ciclo.
type hintTickMsg struct{}

// hintWait es cuánto falta, con idle de quietud, para la próxima frontera: aparecer tras hintIdleAfter, irse hintShowFor después y volver cada hintEvery.
func hintWait(idle time.Duration, ht hintTimes) time.Duration {
	if idle < 0 {
		return ht.Idle // un reloj que retrocede: se vuelve a mirar tras la quietud, nunca en un bucle de ticks inmediatos
	}
	if idle < ht.Idle {
		return ht.Idle - idle
	}
	phase := (idle - ht.Idle) % ht.Every
	if phase < ht.Show {
		return ht.Show - phase
	}
	return ht.Every - phase
}

// hintTimes es la cadencia del "click me!": quietud antes de aparecer, cuánto se ve y cada cuánto vuelve (click_hint_*_seconds en la configuración).
type hintTimes struct{ Idle, Show, Every time.Duration }

// defaultHintTimes es la cadencia por defecto (las constantes de arriba).
var defaultHintTimes = hintTimes{hintIdleAfter, hintShowFor, hintEvery}

// hintTimes lee la cadencia de la configuración; un valor fuera de rango (config a mano sin validar) cae en el de por defecto.
func (m *AppModel) hintTimes() hintTimes {
	c := m.c.cfg
	idle, show, every := time.Duration(c.ClickHintIdleSeconds)*time.Second, time.Duration(c.ClickHintShowSeconds)*time.Second, time.Duration(c.ClickHintEverySeconds)*time.Second
	if idle < 5*time.Second || show < 5*time.Second || every <= show {
		return defaultHintTimes
	}
	return hintTimes{idle, show, every}
}

func (m *AppModel) hintTickCmd() tea.Cmd {
	wait := hintWait(hintNow().Sub(m.mascot.lastInput), m.hintTimes()) + 50*time.Millisecond // un poco después de la frontera, ya del otro lado
	return tea.Tick(wait, func(time.Time) tea.Msg { return hintTickMsg{} })
}

// hintShowing dice si en este instante toca mostrar el "click me!" sobre la mascota del panel r: con mouse, la mascota dibujada, sin popups, sin haberle hecho clic
// y en la ventana de la cadencia.
func (m *AppModel) hintShowing(kind string, r Rect) bool {
	st := &m.mascot
	if st.clicked || st.playing || !m.c.cfg.ClickHint || !m.c.cfg.MouseClick || len(m.c.popups) > 0 || !m.mascotShows(kind, r) {
		return false
	}
	ht := m.hintTimes()
	idle := hintNow().Sub(st.lastInput) - ht.Idle
	return idle >= 0 && idle%ht.Every < ht.Show
}

// mascotTickMsg avanza un cuadro de la animación de la generación gen.
type mascotTickMsg struct{ gen int }

// mascotRequestSequence pide a la terminal el tamaño de la celda en píxeles (CSI 16 t), con el que se
// escala la imagen en múltiplos enteros del sprite: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// ("CSI 1 6 t: report xterm char cell size in pixels").
func mascotRequestSequence() string { return ansi.WindowOp(16) } // 16: pedir el tamaño de la celda en píxeles

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

// mascotLines devuelve las 5 filas de la mascota: con Kitty, la imagen (el cuadro de 40x40 escalado en un
// factor entero, con vecino más cercano, sobre un lienzo del tamaño exacto en píxeles de las celdas: la
// terminal no tiene que reescalarla ni suavizarla); sin Kitty, bloques de cuadrante del cuadro de 20x10.
func (m *AppModel) mascotLines() []string {
	lines, _, _ := m.mascotBlock()
	return lines
}

// mascotBlock es mascotLines más dónde está la parte visible del sprite dentro del bloque de 10x5 celdas: la fila (0 a 4) y la columna de la celda donde empieza la
// cabeza (la primera con píxeles no transparentes) y la columna central de lo visible. El sprite tiene filas transparentes arriba (la imagen se apoya abajo), así que
// el "click me!" se coloca desde esta fila y no desde el borde del bloque.
func (m *AppModel) mascotBlock() (lines []string, headRow, headCol int) {
	name := m.frameName()
	cw, ch := m.cellSize()
	img, _ := frames36().Grids[name].Image(mascotCols*cw, mascotRows*ch)
	if lines, ok := m.c.kitty.BlockImage("mascot-"+name, img, mascotCols, mascotRows); ok {
		x0, y0, x1, _ := opaqueBounds(img)
		return lines, min(max(y0/ch, 0), mascotRows-1), min(max((x0+x1)/2/cw, 0), mascotCols-1)
	}
	g := frames().Grids[name]
	gw, gh := g.Size()
	x0, y0, x1 := gw, gh, -1
	for y := 0; y < gh; y++ {
		for x := 0; x < gw; x++ {
			if x < len(g[y]) && g[y][x].Set {
				x0, y0, x1 = min(x0, x), min(y0, y), max(x1, x)
			}
		}
	}
	if x1 < 0 { // cuadro vacío
		return sprite.Quadrants(g), 0, mascotCols / 2
	}
	// cada celda de cuadrantes son 2x2 píxeles del cuadro de 20x10
	q := sprite.Quadrants(g)
	return q, min(max(y0/2, 0), max(len(q)-1, 0)), min(max((x0+x1)/2/2, 0), mascotCols-1)
}

// opaqueBounds son los límites (píxeles, x1 inclusive, y1 inclusive) de lo que no es transparente en img; sin nada, un cuadro vacío en (0,0,0,0).
func opaqueBounds(img *image.NRGBA) (x0, y0, x1, y1 int) {
	b := img.Bounds()
	x0, y0, x1, y1 = b.Max.X, b.Max.Y, -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.NRGBAAt(x, y).A != 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return 0, 0, 0, 0
	}
	return x0, y0, x1, y1
}

// mascotClick maneja un clic: si cae en la mascota, lanza la siguiente animación.
func (m *AppModel) mascotClick(x, y int) (tea.Cmd, bool) {
	if len(m.c.popups) > 0 || !m.mascotVisible() || !m.mascotRect().Contains(x, y) {
		return nil, false
	}
	st := &m.mascot
	st.clicked = true // ya la encontró: el "click me!" no vuelve en esta sesión
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
