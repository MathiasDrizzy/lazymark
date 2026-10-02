package image

import (
	"bytes"
	"encoding/base64"
	"errors"
	goimage "image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

// block pide la imagen como lo haría la interfaz y deja que "termine de cargar" antes de volver a
// pedirla: Block ya no decodifica dentro de la llamada (ver async_test.go).
func block(t *testing.T, c *Client, path string, cols, rows int) ([]string, bool) {
	t.Helper()
	lines, ok := c.Block(path, cols, rows)
	if !ok {
		return lines, ok
	}
	jobs := c.TakeJobs()
	if len(jobs) == 0 {
		return lines, ok
	}
	for _, j := range jobs {
		tmpl, err := Encode(j)
		c.Done(c.Selection(), j, tmpl, err)
	}
	return c.Block(path, cols, rows)
}

// writePNG crea un PNG con ruido (no se comprime) de w x h en dir.
func writePNG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := goimage.NewRGBA(goimage.Rect(0, 0, w, h))
	rnd := rand.New(rand.NewSource(1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(rnd.Intn(256)), uint8(rnd.Intn(256)), uint8(rnd.Intn(256)), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestQueryAndReply(t *testing.T) {
	q := QuerySequence()
	if !strings.HasPrefix(q, "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\") || !strings.HasSuffix(q, "\x1b[c") {
		t.Errorf("consulta distinta de la de la especificación: %q", q)
	}
	var ev uv.KittyGraphicsEvent
	ev.Options.ID, ev.Payload = 31, []byte("OK")
	if !IsSupportReply(ev) {
		t.Error("la respuesta OK a i=31 debería contar como soporte")
	}
	ev.Options.ID = 7
	if IsSupportReply(ev) {
		t.Error("una respuesta de otra imagen no es la de la consulta")
	}
	ev.Options.ID, ev.Payload = 31, []byte("ENOENT:error")
	if IsSupportReply(ev) {
		t.Error("una respuesta de error no es soporte")
	}
}

func TestFit(t *testing.T) {
	cases := []struct {
		w, h, mc, mr       int
		wantCols, wantRows int
	}{
		{1600, 900, 60, 30, 60, 17}, // apaisada: llena el ancho
		{900, 1600, 60, 10, 11, 10}, // vertical: manda el alto
		{32, 32, 60, 30, 4, 2},      // chica: no se amplía
		{1600, 900, 5, 30, 5, 1},    // columna angosta
		{0, 0, 60, 30, 0, 0},
	}
	for _, c := range cases {
		cols, rows := Fit(c.w, c.h, c.mc, c.mr)
		if cols != c.wantCols || rows != c.wantRows {
			t.Errorf("Fit(%dx%d en %dx%d) = %dx%d, se esperaba %dx%d", c.w, c.h, c.mc, c.mr, cols, rows, c.wantCols, c.wantRows)
		}
		if cols > c.mc || rows > c.mr {
			t.Errorf("Fit(%dx%d) se pasa del recuadro %dx%d: %dx%d", c.w, c.h, c.mc, c.mr, cols, rows)
		}
	}
}

// TestPlaceholdersCells: cada celda es un único grapheme de ancho 1 con el
// marcador, los diacríticos de su fila y columna, y el ID en el color.
func TestPlaceholdersCells(t *testing.T) {
	for _, id := range []int{1, 255, 70000, 0xABCDEF} {
		lines := Placeholders(id, 6, 3)
		if len(lines) != 3 {
			t.Fatalf("%d líneas", len(lines))
		}
		canvas := lipgloss.NewCanvas(6, 3)
		canvas.Compose(lipgloss.NewLayer(strings.Join(lines, "\n")))
		wantFg := color.RGBA{uint8(id >> 16), uint8(id >> 8), uint8(id), 255}
		for r := 0; r < 3; r++ {
			if w := ansi.StringWidth(lines[r]); w != 6 {
				t.Errorf("id %d línea %d mide %d celdas", id, r, w)
			}
			for c := 0; c < 6; c++ {
				cell := canvas.CellAt(c, r)
				want := string(kitty.Placeholder) + string(kitty.Diacritic(r)) + string(kitty.Diacritic(c))
				if cell == nil || cell.Content != want || cell.Width != 1 {
					t.Fatalf("id %d celda (%d,%d) = %+v, se esperaba %q de ancho 1", id, c, r, cell, want)
				}
				if got := cell.Style.Fg; got == nil || !sameColor(got, wantFg) {
					t.Fatalf("id %d celda (%d,%d): fg %v, se esperaba %v", id, c, r, got, wantFg)
				}
			}
		}
	}
}

func sameColor(a color.Color, b color.RGBA) bool {
	r, g, bl, _ := a.RGBA()
	return uint8(r>>8) == b.R && uint8(g>>8) == b.G && uint8(bl>>8) == b.B
}

// apcs separa las secuencias APC (ESC _ G … ESC \) de un flujo.
func apcs(t *testing.T, stream string) (opts, payloads []string) {
	t.Helper()
	for _, part := range strings.Split(stream, "\x1b\\") {
		if part == "" {
			continue
		}
		body, ok := strings.CutPrefix(part, "\x1b_G")
		if !ok {
			t.Fatalf("trozo que no empieza con ESC _ G: %q", part[:min(20, len(part))])
		}
		o, p, _ := strings.Cut(body, ";")
		opts, payloads = append(opts, o), append(payloads, p)
	}
	return
}

// TestTransmitSequence (C2): la imagen se transmite por trozos de ≤4096 bytes,
// el primero con todas las claves y los demás solo con m (y q); al juntar los
// trozos sale el PNG, reducido a ≤1024 px.
func TestTransmitSequence(t *testing.T) {
	path := writePNG(t, t.TempDir(), "grande.png", 1500, 800)
	c := New()
	c.SetSupported(true)
	lines, ok := block(t, c, path, 60, 30)
	if !ok {
		t.Fatal("Block no devolvió la imagen")
	}
	pend := c.TakePending()
	if len(pend) != 1 {
		t.Fatalf("pendientes = %d, se esperaba 1 transmisión", len(pend))
	}
	opts, payloads := apcs(t, pend[0])
	if len(opts) < 3 {
		t.Fatalf("el ruido de 1024 px debería ir en varios trozos, hubo %d", len(opts))
	}
	first := strings.Split(opts[0], ",")
	for _, want := range []string{"a=T", "U=1", "i=1", "f=100", "q=2"} {
		if !contains(first, want) {
			t.Errorf("al primer trozo le falta %s: %v", want, first)
		}
	}
	if !contains(first, "m=1") {
		t.Errorf("el primer trozo debería llevar m=1: %v", first)
	}
	cols, rows := Fit(1500, 800, 60, 30)
	if !contains(first, "c="+strconv.Itoa(cols)) || !contains(first, "r="+strconv.Itoa(rows)) {
		t.Errorf("c/r distintos de %dx%d: %v", cols, rows, first)
	}
	var data strings.Builder
	for i, o := range opts {
		if len(payloads[i]) > 4096 {
			t.Errorf("trozo %d de %d bytes (máx. 4096)", i, len(payloads[i]))
		}
		if i > 0 {
			for _, k := range strings.Split(o, ",") {
				if !strings.HasPrefix(k, "m=") && !strings.HasPrefix(k, "q=") {
					t.Errorf("el trozo %d lleva la clave %q: solo se permiten m y q", i, k)
				}
			}
		}
		if i < len(opts)-1 && len(payloads[i])%4 != 0 {
			t.Errorf("trozo %d: su tamaño no es múltiplo de 4", i)
		}
		data.WriteString(payloads[i])
	}
	if last := opts[len(opts)-1]; !contains(strings.Split(last, ","), "m=0") {
		t.Errorf("el último trozo debería llevar m=0: %q", last)
	}
	raw, err := base64.StdEncoding.DecodeString(data.String())
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("los trozos no forman un PNG: %v", err)
	}
	if b := img.Bounds(); max(b.Dx(), b.Dy()) != maxSide || b.Dx() <= b.Dy() {
		t.Errorf("la imagen transmitida mide %v, se esperaba el lado mayor en %d y proporción apaisada", b, maxSide)
	}
	if len(lines) != rows || ansi.StringWidth(lines[0]) != cols {
		t.Errorf("placeholders de %dx%d, se esperaba %dx%d", ansi.StringWidth(lines[0]), len(lines), cols, rows)
	}

	// pedirla otra vez no la vuelve a transmitir
	if _, ok := block(t, c, path, 60, 30); !ok || len(c.TakePending()) != 0 {
		t.Error("la segunda vez no debería haber nada pendiente")
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// TestResetEmitsDelete (H3-3): Reset borra con a=d y lo que se vuelve a pedir se
// transmite de nuevo con un ID nuevo; ocultar también borra.
func TestResetEmitsDelete(t *testing.T) {
	path := writePNG(t, t.TempDir(), "a.png", 80, 40)
	c := New()
	c.SetSupported(true)
	block(t, c, path, 60, 30)
	c.TakePending()
	if !c.HasLive() {
		t.Fatal("debería haber una imagen viva")
	}

	gen := c.Generation()
	c.Reset()
	pend := c.TakePending()
	if len(pend) != 1 || pend[0] != "\x1b_Ga=d,d=A,q=2\x1b\\" {
		t.Fatalf("Reset debía emitir solo a=d,d=A: %q", pend)
	}
	if c.HasLive() || c.Generation() == gen {
		t.Error("Reset debe vaciar lo vivo y cambiar la generación")
	}
	block(t, c, path, 60, 30)
	if pend := c.TakePending(); len(pend) != 1 || !strings.Contains(pend[0], "a=T") || !strings.Contains(pend[0], "i=2") {
		t.Errorf("tras Reset la imagen debe transmitirse de nuevo con otro ID: %q", pend)
	}

	// ocultar (popup abierto): borra y Block devuelve texto
	c.SetVisible(false)
	if pend := c.TakePending(); len(pend) != 1 || !strings.HasPrefix(pend[0], "\x1b_Ga=d") {
		t.Errorf("ocultar debía emitir a=d: %q", pend)
	}
	if _, ok := block(t, c, path, 60, 30); ok {
		t.Error("con las imágenes ocultas Block no debe devolver placeholders")
	}
	if len(c.TakePending()) != 0 {
		t.Error("oculto no debe transmitir nada")
	}
	c.SetVisible(true)
	if _, ok := block(t, c, path, 60, 30); !ok || len(c.TakePending()) != 1 {
		t.Error("al volver a mostrar debe retransmitir")
	}

	// sin nada vivo, Reset no emite basura
	empty := New()
	empty.Reset()
	if len(empty.TakePending()) != 0 {
		t.Error("Reset sin imágenes no debería emitir nada")
	}
}

// TestFallback (H3-4): sin soporte, o con un archivo ausente o corrupto, Block
// no devuelve placeholders ni transmite nada; el texto de reemplazo nombra el archivo.
func TestFallback(t *testing.T) {
	dir := t.TempDir()
	good := writePNG(t, dir, "x.png", 80, 40)
	bad := filepath.Join(dir, "roto.png")
	_ = os.WriteFile(bad, []byte("no soy un png"), 0o644)

	c := New() // sin soporte
	if _, ok := block(t, c, good, 60, 30); ok {
		t.Error("sin soporte no debe haber placeholders")
	}
	c.SetSupported(true)
	for _, p := range []string{filepath.Join(dir, "no-existe.png"), bad} {
		if _, ok := block(t, c, p, 60, 30); ok {
			t.Errorf("%s no debería poder mostrarse", p)
		}
	}
	if len(c.TakePending()) != 0 || c.HasLive() {
		t.Error("los fallos no deben dejar transmisiones")
	}
	if got := Label(good); got != "[imagen: x.png]" && got != "[image: x.png]" {
		t.Errorf("texto de reemplazo = %q", got)
	}
}

func TestScaleDown(t *testing.T) {
	small := goimage.NewRGBA(goimage.Rect(0, 0, 100, 50))
	if scaleDown(small, 1024) != goimage.Image(small) {
		t.Error("una imagen que ya cabe no debería copiarse")
	}
	big := goimage.NewRGBA(goimage.Rect(0, 0, 3000, 1000))
	for i := range big.Pix {
		big.Pix[i] = 200
	}
	out := scaleDown(big, 1024).Bounds()
	if out.Dx() != 1024 || out.Dy() != 341 {
		t.Errorf("3000x1000 -> %dx%d, se esperaba 1024x341", out.Dx(), out.Dy())
	}
}

// TestEncodeRejectsPixelBomb (S6): una imagen que declara más de MaxPixels se rechaza leyendo solo la cabecera, sin
// decodificarla (un PNG de 420 KB de 20000x20000 pedía 416 MB).
func TestEncodeRejectsPixelBomb(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, goimage.NewGray(goimage.Rect(0, 0, 8000, 6000))); err != nil { // 48 MP
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := Encode(Job{Data: buf.Bytes(), Cols: 10, Rows: 5})
	runtime.ReadMemStats(&after)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("se esperaba ErrTooLarge: %v", err)
	}
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 8<<20 {
		t.Errorf("rechazar una imagen grande reservó %d MB: se decodificó", grown>>20)
	}
	// una imagen normal sigue funcionando
	buf.Reset()
	png.Encode(&buf, goimage.NewGray(goimage.Rect(0, 0, 64, 64)))
	if _, err := Encode(Job{Data: buf.Bytes(), Cols: 4, Rows: 2}); err != nil {
		t.Errorf("imagen normal: %v", err)
	}
	// y Block la deja como texto (sin placeholders) en lugar de encolarla
	dir := t.TempDir()
	big := filepath.Join(dir, "bomba.png")
	var b2 bytes.Buffer
	png.Encode(&b2, goimage.NewGray(goimage.Rect(0, 0, 8000, 6000)))
	os.WriteFile(big, b2.Bytes(), 0o644)
	c := New()
	c.supported, c.visible = true, true
	if _, ok := c.Block(big, 20, 10); ok || len(c.TakeJobs()) != 0 {
		t.Error("una imagen sobre el tope no se encola")
	}
}
