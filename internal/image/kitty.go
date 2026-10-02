// Package image muestra imágenes en el preview con el protocolo de gráficos de
// Kitty usando placeholders Unicode: la imagen se transmite una vez y en la
// pantalla son celdas de texto normales (U+10EEEE con el ID en el color), así
// viajan por el renderer de Bubble Tea, la composición de popups y los
// multiplexores sin dejar imágenes fantasma.
//
// Especificación: https://sw.kovidgoyal.net/kitty/graphics-protocol/
package image

import (
	"bytes"
	"errors"
	"fmt"
	goimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

const (
	// queryID identifica la consulta de soporte (a=q) y su respuesta.
	queryID = 31
	// maxSide limita el lado mayor de lo que se transmite: la terminal escala
	// la imagen a las celdas, no hace falta mandarle una captura de varios MB.
	maxSide = 1024
	// Una celda mide aproximadamente 1 de ancho por 2 de alto; 8 px de ancho es
	// el tamaño natural de una columna.
	cellAspect = 2.0
	pxPerCol   = 8
)

// QuerySequence consulta si la terminal soporta gráficos (a=q) y pide los
// atributos primarios (DA1) detrás: si llega la respuesta de gráficos, hay
// soporte; si llega solo la de DA1, no lo hay.
func QuerySequence() string {
	return ansi.KittyGraphics([]byte("AAAA"),
		"i="+strconv.Itoa(queryID), "s=1", "v=1", "a=q", "t=d", "f=24") + ansi.RequestPrimaryDeviceAttributes
}

// IsSupportReply indica si e es la respuesta afirmativa a QuerySequence.
func IsSupportReply(e uv.KittyGraphicsEvent) bool {
	return e.Options.ID == queryID && strings.HasPrefix(string(e.Payload), "OK")
}

// DeleteAllSequence borra todas las imágenes visibles y libera sus datos
// (a=d con d=A, en silencio).
func DeleteAllSequence() string {
	return ansi.KittyGraphics(nil, "a=d", "d=A", "q=2")
}

// Fit calcula las celdas que ocupa una imagen de imgW x imgH píxeles dentro de
// maxCols x maxRows, conservando la proporción y sin ampliarla más allá de su
// tamaño natural.
func Fit(imgW, imgH, maxCols, maxRows int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 || maxCols < 1 || maxRows < 1 {
		return 0, 0
	}
	cols = min(maxCols, max(2, imgW/pxPerCol))
	rows = int(float64(cols)*float64(imgH)/float64(imgW)/cellAspect + 0.5)
	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows)*cellAspect*float64(imgW)/float64(imgH) + 0.5)
	}
	return max(1, min(cols, maxCols)), max(1, rows)
}

// Placeholders devuelve las rows líneas de cols celdas que muestran la imagen
// id: cada celda es U+10EEEE más los diacríticos de su fila y su columna, y el
// ID va en el color de primer plano (24 bits, sin tercer diacrítico).
func Placeholders(id, cols, rows int) []string {
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", id>>16&0xff, id>>8&0xff, id&0xff)
	lines := make([]string, rows)
	for r := range lines {
		var b strings.Builder
		b.WriteString(fg)
		for c := 0; c < cols; c++ {
			b.WriteRune(kitty.Placeholder)
			b.WriteRune(kitty.Diacritic(r))
			b.WriteRune(kitty.Diacritic(c))
		}
		b.WriteString("\x1b[0m")
		lines[r] = b.String()
	}
	return lines
}

// Label es el texto de reemplazo cuando no se puede mostrar la imagen.
func Label(path string) string {
	return fmt.Sprintf("[%s: %s]", i18n.T("imagen", "image"), filepath.Base(path))
}

// placed es una imagen ya transmitida (o pendiente de transmitir).
type placed struct {
	id         int
	cols, rows int
}

// Job es el trabajo pesado de una imagen: decodificarla, reducirla y codificarla. Va fuera del
// ciclo de Update (Encode corre en una goroutine) y se identifica por su clave: ruta, mtime,
// tamaño del archivo y celdas, así un archivo cambiado o un ancho distinto no reusan el resultado.
type Job struct {
	Key, Path  string
	Data       []byte // imagen embebida en el binario (si no es nil, se usa en vez de Path)
	Cols, Rows int
}

// maxReady limita la caché de imágenes ya codificadas (secuencias de transmisión listas).
const maxReady = 24

// Client lleva el estado de los gráficos: si la terminal los soporta, qué
// imágenes están transmitidas, cuáles están codificadas y esperando, y las
// secuencias que faltan por enviar. Se usa desde un solo hilo (el de Update/View de
// Bubble Tea); solo Encode, que no toca el Client, corre en otras goroutines.
type Client struct {
	supported bool
	visible   bool
	gen       int
	nextID    int
	live      map[string]placed
	pending   []string

	sel      int            // selección actual: cambia al cambiar de nota (NewSelection)
	wanted   []Job          // lo que pide el render y aún no se lanzó
	inflight map[string]int // trabajos lanzados, con la selección que los pidió
	ready    map[string]string
	order    []string // claves de ready, de la más vieja a la más nueva
	failed   map[string]bool
	dims     map[string][2]int
}

// New crea un cliente sin soporte: hasta que la terminal conteste a la
// consulta (SetSupported) las imágenes se muestran como texto.
func New() *Client {
	return &Client{visible: true, nextID: 1, live: map[string]placed{}, inflight: map[string]int{},
		ready: map[string]string{}, failed: map[string]bool{}, dims: map[string][2]int{}}
}

// Supported indica si la terminal contestó afirmativamente a la consulta.
func (c *Client) Supported() bool { return c.supported }

// SetSupported registra el resultado de la detección.
func (c *Client) SetSupported(v bool) {
	if c.supported != v {
		c.supported = v
		c.gen++
	}
}

// Generation cambia cada vez que lo que Block devuelve puede ser distinto
// (soporte, visibilidad, imágenes borradas o recién listas); sirve para invalidar cachés de render.
func (c *Client) Generation() int { return c.gen }

// SetVisible oculta o vuelve a mostrar las imágenes (p. ej. mientras el Kanban está abierto).
// Al ocultarlas se borran de la terminal.
func (c *Client) SetVisible(v bool) {
	if c.visible == v {
		return
	}
	c.visible = v
	if !v {
		c.Reset()
		return
	}
	c.gen++
}

// Reset borra todas las imágenes de la terminal (a=d); las que se vuelvan a
// pedir se transmiten de nuevo (desde la caché, sin volver a codificarlas). Se llama al cambiar de
// nota, volver del editor y al salir.
func (c *Client) Reset() {
	if len(c.live) > 0 || len(c.pending) > 0 {
		c.pending = append(c.pending[:0], DeleteAllSequence())
		clear(c.live)
	}
	c.gen++
}

// NewSelection marca que la selección cambió: lo que se pidió antes deja de importar. Los
// trabajos en vuelo se descartan al terminar (no se dibujan ni se transmiten).
func (c *Client) NewSelection() {
	c.sel++
	c.wanted = c.wanted[:0]
	clear(c.inflight)
}

// Selection devuelve la selección actual.
func (c *Client) Selection() int { return c.sel }

// HasLive indica si hay imágenes transmitidas en la terminal.
func (c *Client) HasLive() bool { return len(c.live) > 0 }

// HasWanted indica si el render pidió imágenes que todavía no se lanzaron.
func (c *Client) HasWanted() bool { return len(c.wanted) > 0 }

// TakeJobs devuelve los trabajos pedidos y los marca como lanzados.
func (c *Client) TakeJobs() []Job {
	jobs := append([]Job(nil), c.wanted...)
	for _, j := range jobs {
		c.inflight[j.Key] = c.sel
	}
	c.wanted = c.wanted[:0]
	return jobs
}

// Done recibe el resultado de Encode. Si la selección ya cambió, el resultado no se dibuja ni se
// transmite (se guarda en la caché por si se vuelve a esa nota) y devuelve false.
func (c *Client) Done(sel int, j Job, tmpl string, err error) bool {
	current := c.inflight[j.Key] == sel && sel == c.sel
	delete(c.inflight, j.Key)
	if err != nil {
		if current {
			c.failed[j.Key] = true
			c.gen++
		}
		return current
	}
	c.store(j.Key, tmpl)
	if current {
		c.gen++
	}
	return current
}

// store guarda una secuencia en la caché, expulsando la más vieja si hay más de maxReady.
func (c *Client) store(key, tmpl string) {
	if _, ok := c.ready[key]; !ok {
		c.order = append(c.order, key)
	}
	c.ready[key] = tmpl
	for len(c.order) > maxReady {
		delete(c.ready, c.order[0])
		c.order = c.order[1:]
	}
}

// TakePending devuelve las secuencias por enviar a la terminal, en orden, y las vacía.
func (c *Client) TakePending() []string {
	out := c.pending
	c.pending = nil
	return out
}

// Block devuelve las líneas de placeholders de la imagen en path, ajustada a
// maxCols x maxRows. ok es false si no hay soporte, las imágenes están ocultas
// o el archivo no se puede leer: el llamador usa entonces Label.
//
// Block nunca espera: leer las dimensiones es barato, pero decodificar, reducir y codificar la
// imagen no. Si la imagen no está lista, pide el trabajo (se lanza con TakeJobs) y devuelve
// mientras tanto un bloque de la misma altura con su texto de reemplazo, así el texto de
// alrededor no salta cuando la imagen llega. Si ya está codificada, la deja en la cola de
// transmisión.
func (c *Client) Block(path string, maxCols, maxRows int) (lines []string, ok bool) {
	if !c.supported || !c.visible {
		return nil, false
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	dkey := fmt.Sprintf("%s|%d|%d", path, fi.ModTime().UnixNano(), fi.Size())
	d, known := c.dims[dkey]
	if !known {
		f, err := os.Open(path)
		if err != nil {
			return nil, false
		}
		cfg, _, err := goimage.DecodeConfig(f)
		_ = f.Close()
		if err != nil {
			return nil, false
		}
		if checkPixels(cfg.Width, cfg.Height) != nil {
			return nil, false // se queda como texto, sin decodificar
		}
		d = [2]int{cfg.Width, cfg.Height}
		c.dims[dkey] = d
	}
	cols, rows := Fit(d[0], d[1], maxCols, maxRows)
	if cols == 0 {
		return nil, false
	}
	key := fmt.Sprintf("%s|%d|%d", dkey, cols, rows)
	if p, ok := c.live[key]; ok {
		return Placeholders(p.id, p.cols, p.rows), true
	}
	if c.failed[key] {
		return nil, false
	}
	if tmpl, ok := c.ready[key]; ok {
		p := placed{id: c.nextID, cols: cols, rows: rows}
		c.nextID++
		c.live[key] = p
		c.pending = append(c.pending, withID(tmpl, p.id))
		return Placeholders(p.id, p.cols, p.rows), true
	}
	if _, running := c.inflight[key]; !running && !c.isWanted(key) {
		c.wanted = append(c.wanted, Job{Key: key, Path: path, Cols: cols, Rows: rows})
	}
	loading := make([]string, rows)
	loading[0] = Label(path)
	return loading, true
}

// BlockData es Block para una imagen embebida en el binario (data), que se muestra en exactamente
// cols x rows celdas. A diferencia de Block, mientras no esté lista devuelve ok=false (el llamador
// dibuja su alternativa, por ejemplo medios bloques) y pide el trabajo igual.
func (c *Client) BlockData(name string, data []byte, cols, rows int) (lines []string, ok bool) {
	if !c.supported || !c.visible || cols < 1 || rows < 1 {
		return nil, false
	}
	key := fmt.Sprintf("data:%s|%d|%d", name, cols, rows)
	if p, ok := c.live[key]; ok {
		return Placeholders(p.id, p.cols, p.rows), true
	}
	if c.failed[key] {
		return nil, false
	}
	if tmpl, ok := c.ready[key]; ok {
		p := placed{id: c.nextID, cols: cols, rows: rows}
		c.nextID++
		c.live[key] = p
		c.pending = append(c.pending, withID(tmpl, p.id))
		return Placeholders(p.id, p.cols, p.rows), true
	}
	if _, running := c.inflight[key]; !running && !c.isWanted(key) {
		c.wanted = append(c.wanted, Job{Key: key, Data: data, Cols: cols, Rows: rows})
	}
	return nil, false
}

func (c *Client) isWanted(key string) bool {
	for _, j := range c.wanted {
		if j.Key == key {
			return true
		}
	}
	return false
}

// MaxPixels es el tope de píxeles de una imagen para mostrarla (unos 40 MP): más que eso se rechaza sin decodificar.
const MaxPixels = 40_000_000

// ErrTooLarge es el error de una imagen con más de MaxPixels.
var ErrTooLarge = errors.New("la imagen tiene demasiados píxeles")

func checkPixels(w, h int) error {
	if w <= 0 || h <= 0 || int64(w)*int64(h) > MaxPixels {
		return fmt.Errorf("%w (%dx%d, máximo %d)", ErrTooLarge, w, h, MaxPixels)
	}
	return nil
}

// sentinelID es el ID con el que Encode codifica: withID lo cambia por el real al transmitir.
const sentinelID = 16777215

// Encode hace el trabajo pesado de j: decodifica la imagen, la reduce y arma la secuencia de
// transmisión (PNG en base64, en trozos). No toca ningún estado: es seguro correrlo en una goroutine.
func Encode(j Job) (string, error) {
	open := func() (io.Reader, func(), error) {
		if j.Data != nil {
			return bytes.NewReader(j.Data), func() {}, nil
		}
		f, err := os.Open(j.Path)
		if err != nil {
			return nil, nil, err
		}
		return f, func() { _ = f.Close() }, nil
	}
	// primero solo la cabecera: una imagen "bomba" (un PNG chico que declara miles de millones de píxeles) no se
	// decodifica nunca
	src, closeSrc, err := open()
	if err != nil {
		return "", err
	}
	cfg, _, err := goimage.DecodeConfig(src)
	closeSrc()
	if err != nil {
		return "", err
	}
	if err := checkPixels(cfg.Width, cfg.Height); err != nil {
		return "", err
	}
	if src, closeSrc, err = open(); err != nil {
		return "", err
	}
	defer closeSrc()
	img, _, err := goimage.Decode(src)
	if err != nil {
		return "", err
	}
	return encodeImage(scaleDown(img, maxSide), j.Cols, j.Rows)
}

// encodeImage arma la secuencia de transmisión de img (PNG en base64, en trozos) para mostrarla en
// cols x rows celdas, con el ID de marcador.
func encodeImage(img goimage.Image, cols, rows int) (string, error) {
	var b strings.Builder
	err := kitty.EncodeGraphics(&b, img, &kitty.Options{
		Action:           kitty.TransmitAndPut,
		ID:               sentinelID,
		Format:           kitty.PNG,
		Transmission:     kitty.Direct,
		Chunk:            true,
		Columns:          cols,
		Rows:             rows,
		VirtualPlacement: true,
		Quiet:            2,
	})
	return b.String(), err
}

// BlockImage es Block para una imagen que ya está en memoria y es pequeña (la mascota): se codifica
// aquí mismo, sin goroutine, y se muestra tal cual en cols x rows celdas. name identifica la imagen: con
// el mismo nombre y tamaño se reutiliza lo ya transmitido. Sin soporte devuelve ok=false.
func (c *Client) BlockImage(name string, img goimage.Image, cols, rows int) (lines []string, ok bool) {
	if !c.supported || !c.visible || cols < 1 || rows < 1 {
		return nil, false
	}
	key := fmt.Sprintf("img:%s|%d|%d|%dx%d", name, cols, rows, img.Bounds().Dx(), img.Bounds().Dy())
	if p, ok := c.live[key]; ok {
		return Placeholders(p.id, p.cols, p.rows), true
	}
	if c.failed[key] {
		return nil, false
	}
	tmpl, err := encodeImage(img, cols, rows)
	if err != nil {
		c.failed[key] = true
		return nil, false
	}
	p := placed{id: c.nextID, cols: cols, rows: rows}
	c.nextID++
	c.live[key] = p
	c.pending = append(c.pending, withID(tmpl, p.id))
	return Placeholders(p.id, p.cols, p.rows), true
}

// withID pone el ID real en la primera parte de control de una secuencia hecha por Encode (las
// demás partes del trozo solo llevan m y q).
func withID(tmpl string, id int) string {
	end := strings.IndexByte(tmpl, ';')
	if end < 0 {
		return tmpl
	}
	return strings.Replace(tmpl[:end], "i="+strconv.Itoa(sentinelID), "i="+strconv.Itoa(id), 1) + tmpl[end:]
}

// scaleDown reduce src para que su lado mayor no pase de limit, promediando
// los píxeles de cada bloque. Devuelve src si ya cabe.
func scaleDown(src goimage.Image, limit int) goimage.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= limit && h <= limit {
		return src
	}
	nw, nh := max(1, w*limit/max(w, h)), max(1, h*limit/max(w, h))
	dst := goimage.NewRGBA(goimage.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		y0, y1 := b.Min.Y+y*h/nh, b.Min.Y+max(y*h/nh+1, (y+1)*h/nh)
		for x := 0; x < nw; x++ {
			x0, x1 := b.Min.X+x*w/nw, b.Min.X+max(x*w/nw+1, (x+1)*w/nw)
			var r, g, bl, a, n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					pr, pg, pb, pa := src.At(sx, sy).RGBA()
					r, g, bl, a, n = r+pr, g+pg, bl+pb, a+pa, n+1
				}
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = uint8(r/n>>8), uint8(g/n>>8), uint8(bl/n>>8), uint8(a/n>>8)
		}
	}
	return dst
}
