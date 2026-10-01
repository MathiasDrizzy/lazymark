// Package image muestra imágenes en el preview con el protocolo de gráficos de
// Kitty usando placeholders Unicode: la imagen se transmite una vez y en la
// pantalla son celdas de texto normales (U+10EEEE con el ID en el color), así
// viajan por el renderer de Bubble Tea, la composición de popups y los
// multiplexores sin dejar imágenes fantasma.
//
// Especificación: https://sw.kovidgoyal.net/kitty/graphics-protocol/
package image

import (
	"fmt"
	goimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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

// Client lleva el estado de los gráficos: si la terminal los soporta, qué
// imágenes están transmitidas y las secuencias que faltan por enviar. Se usa
// desde un solo hilo (el de Update/View de Bubble Tea).
type Client struct {
	supported bool
	visible   bool
	gen       int
	nextID    int
	live      map[string]placed
	pending   []string
}

// New crea un cliente sin soporte: hasta que la terminal conteste a la
// consulta (SetSupported) las imágenes se muestran como texto.
func New() *Client {
	return &Client{visible: true, nextID: 1, live: map[string]placed{}}
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
// (soporte, visibilidad o imágenes borradas); sirve para invalidar cachés de render.
func (c *Client) Generation() int { return c.gen }

// SetVisible oculta o vuelve a mostrar las imágenes (p. ej. mientras hay un
// popup abierto). Al ocultarlas se borran de la terminal.
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
// pedir se transmiten de nuevo. Se llama al cambiar de nota, abrir un popup,
// volver del editor y al salir.
func (c *Client) Reset() {
	if len(c.live) > 0 || len(c.pending) > 0 {
		c.pending = append(c.pending[:0], DeleteAllSequence())
		clear(c.live)
	}
	c.gen++
}

// HasLive indica si hay imágenes transmitidas en la terminal.
func (c *Client) HasLive() bool { return len(c.live) > 0 }

// TakePending devuelve las secuencias por enviar a la terminal, en orden, y las vacía.
func (c *Client) TakePending() []string {
	out := c.pending
	c.pending = nil
	return out
}

// Block devuelve las líneas de placeholders de la imagen en path, ajustada a
// maxCols x maxRows. ok es false si no hay soporte, las imágenes están ocultas
// o el archivo no se puede leer: el llamador usa entonces Label. Si la imagen
// todavía no está en la terminal, deja su transmisión en la cola de pendientes.
func (c *Client) Block(path string, maxCols, maxRows int) (lines []string, ok bool) {
	if !c.supported || !c.visible {
		return nil, false
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	cfg, _, err := goimage.DecodeConfig(f)
	_ = f.Close()
	if err != nil {
		return nil, false
	}
	cols, rows := Fit(cfg.Width, cfg.Height, maxCols, maxRows)
	if cols == 0 {
		return nil, false
	}
	key := fmt.Sprintf("%s|%d|%d|%d", path, fi.ModTime().UnixNano(), cols, rows)
	p, exists := c.live[key]
	if !exists {
		seq, err := c.sequence(path, c.nextID, cols, rows)
		if err != nil {
			return nil, false
		}
		p = placed{id: c.nextID, cols: cols, rows: rows}
		c.nextID++
		c.live[key] = p
		c.pending = append(c.pending, seq)
	}
	return Placeholders(p.id, p.cols, p.rows), true
}

// sequence codifica la transmisión de la imagen con el ID dado.
func (c *Client) sequence(path string, id, cols, rows int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck
	img, _, err := goimage.Decode(f)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = kitty.EncodeGraphics(&b, scaleDown(img, maxSide), &kitty.Options{
		Action:           kitty.TransmitAndPut,
		ID:               id,
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
