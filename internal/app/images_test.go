package app

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

const deleteAll = "\x1b_Ga=d,d=A,q=2\x1b\\"

// imageModel crea el modelo con las secuencias a la terminal registradas (nada
// se escribe de verdad) y, si supported, con la terminal marcada como Kitty.
type imageRig struct {
	*AppModel
	sent []string
}

func newImageRig(t *testing.T, w, h int, supported bool) *imageRig {
	t.Helper()
	r := &imageRig{AppModel: newTestModel(t, w, h)}
	r.emit = func(seq any) tea.Cmd {
		r.sent = append(r.sent, seq.(string))
		return nil
	}
	r.c.kitty.SetSupported(supported)
	return r
}

// take devuelve lo enviado desde la última vez y lo vacía.
func (r *imageRig) take() []string {
	out := r.sent
	r.sent = nil
	return out
}

// show pone el cursor del árbol sobre la nota (como un usuario que navega) y deja
// que Update prepare los gráficos.
func (r *imageRig) show(name string) {
	r.showNoLoad(name)
	r.settle()
}

// showNoLoad es show sin esperar a que lleguen las imágenes: lo que se ve mientras cargan.
func (r *imageRig) showNoLoad(name string) {
	r.notes.selectPath(filepath.Join(r.c.store.BaseDir, name))
	r.Update(tea.WindowSizeMsg{Width: r.w, Height: r.h})
}

// settle hace lo que haría el programa real al pasar el debounce: lanza lo que la selección pidió, lo
// ejecuta y entrega los resultados a Update. Las imágenes ya no se cargan dentro de Update.
func (r *imageRig) settle() {
	for i := 0; i < 5; i++ {
		jobs := r.c.kitty.TakeJobs()
		if len(jobs) == 0 {
			return
		}
		sel := r.c.kitty.Selection()
		for _, j := range jobs {
			tmpl, err := image.Encode(j)
			r.Update(imageLoadedMsg{sel: sel, job: j, tmpl: tmpl, err: err})
		}
	}
}

func transmits(seqs []string) (n int) {
	for _, s := range seqs {
		if strings.Contains(s, "a=T") {
			n++
		}
	}
	return
}

func placeholderCells(m *AppModel) int {
	return strings.Count(m.View().Content, string(kitty.Placeholder))
}

// TestImageInlineAligned (H3-1, C2): con soporte, la imagen se ve en su posición
// dentro del texto, alineada con él y sin el texto crudo de markdown.
func TestImageInlineAligned(t *testing.T) {
	r := newImageRig(t, 120, 35, true)
	r.show("imagen.md")

	cols, rows := image.Fit(240, 120, 70, maxImageRows)
	if got := placeholderCells(r.AppModel); got != cols*rows {
		t.Fatalf("celdas de imagen en pantalla = %d, se esperaban %d (%dx%d)", got, cols*rows, cols, rows)
	}
	out := plain(r.AppModel)
	for _, bad := range []string{"![", "](assets", "foto.png"} {
		if strings.Contains(out, bad) {
			t.Errorf("se ve texto crudo %q junto a la imagen:\n%s", bad, out)
		}
	}
	if got := r.take(); transmits(got) != 1 || strings.Contains(strings.Join(got, ""), deleteAll) {
		t.Fatalf("debía transmitirse una imagen (la rota no) sin borrar nada: %q", got)
	}

	// posición: entre el texto de antes y el de después, y alineada con el texto
	var rowsText []string
	for _, l := range screen(r.AppModel) {
		rowsText = append(rowsText, l)
	}
	before, firstImg, after, textX, imgX := -1, -1, -1, -1, -1
	for y, l := range rowsText {
		s := ansi.Strip(l)
		switch {
		case strings.Contains(s, "Texto antes de la imagen"):
			before, textX = y, ansi.StringWidth(s[:strings.Index(s, "Texto antes")])
		case strings.Contains(s, string(kitty.Placeholder)) && firstImg < 0:
			firstImg = y
			imgX = ansi.StringWidth(s[:strings.Index(s, string(kitty.Placeholder))])
		case strings.Contains(s, "Texto después de la imagen"):
			after = y
		}
	}
	if !(before >= 0 && before < firstImg && firstImg+rows <= after) {
		t.Errorf("la imagen no está entre los dos párrafos: antes=%d imagen=%d..%d después=%d", before, firstImg, firstImg+rows-1, after)
	}
	if textX < 0 || imgX != textX {
		t.Errorf("la imagen empieza en la columna %d y el texto en %d: no están alineados", imgX, textX)
	}
}

// TestImageFallbackLabel (H3-4, C5): sin soporte, o si el archivo no existe, la
// imagen es un texto de reemplazo y no se transmite nada.
func TestImageFallbackLabel(t *testing.T) {
	r := newImageRig(t, 120, 35, false)
	r.show("imagen.md")
	out := plain(r.AppModel)
	for _, want := range []string{"[imagen: foto.png]", "[imagen: no-existe.png]"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta el texto de reemplazo %q:\n%s", want, out)
		}
	}
	if placeholderCells(r.AppModel) != 0 || len(r.take()) != 0 {
		t.Error("sin soporte no debe haber celdas de imagen ni secuencias")
	}

	// con soporte, la que existe se muestra y la ausente sigue siendo texto
	r = newImageRig(t, 120, 35, true)
	r.show("imagen.md")
	out = plain(r.AppModel)
	if placeholderCells(r.AppModel) == 0 || strings.Contains(out, "[imagen: foto.png]") {
		t.Error("con soporte, foto.png debería verse como imagen")
	}
	if !strings.Contains(out, "[imagen: no-existe.png]") {
		t.Errorf("la imagen ausente debería quedar como texto:\n%s", out)
	}
}

// TestImageFitsTerminal (X4, X5): con imágenes (o su texto) el View sigue
// midiendo exactamente el ancho y el alto, también con scroll y popups abiertos.
func TestImageFitsTerminal(t *testing.T) {
	sizes := []struct{ w, h int }{{120, 35}, {100, 30}, {80, 24}, {60, 20}}
	for _, supported := range []bool{true, false} {
		for _, sz := range sizes {
			for _, keys := range [][]string{nil, {"4", "right", "right", "down", "down"}, {"?"}, {"4", "w"}} {
				r := newImageRig(t, sz.w, sz.h, supported)
				r.show("imagen.md")
				press(r.AppModel, keys...)
				lines := screen(r.AppModel)
				if len(lines) != sz.h {
					t.Errorf("soporte=%v %dx%d teclas=%v: %d líneas", supported, sz.w, sz.h, keys, len(lines))
				}
				for i, l := range lines {
					if w := ansi.StringWidth(l); w != sz.w {
						t.Errorf("soporte=%v %dx%d teclas=%v línea %d mide %d", supported, sz.w, sz.h, keys, i, w)
						break
					}
				}
			}
		}
	}
}

// TestImageDeleteEvents (H3-3, C4): cambiar de nota, abrir el editor, volver de él y
// salir emiten a=d (borrar todo) y la imagen se transmite de nuevo al volver. Abrir un
// popup ya NO borra la imagen (ORD-007 C.4, ver TestImageSurvivesPopups).
func TestImageDeleteEvents(t *testing.T) {
	r := newImageRig(t, 120, 35, true)
	r.show("imagen.md")
	if transmits(r.take()) != 1 {
		t.Fatal("la primera vez debe transmitirse la imagen")
	}

	// sin cambios no se manda nada
	r.Update(tea.WindowSizeMsg{Width: r.w, Height: r.h})
	press(r.AppModel, "4", "down", "up")
	if got := r.take(); len(got) != 0 {
		t.Fatalf("sin cambios de nota ni popups no debe enviarse nada: %q", got)
	}

	// popup abierto: la imagen se queda; no se emite nada y no se retransmite al cerrar
	press(r.AppModel, "?")
	if got := r.take(); len(got) != 0 {
		t.Fatalf("abrir un popup no debe emitir nada: %q", got)
	}
	press(r.AppModel, "esc")
	if got := r.take(); len(got) != 0 {
		t.Errorf("cerrar el popup no debe emitir nada: %q", got)
	}
	if placeholderCells(r.AppModel) == 0 {
		t.Error("al cerrar el popup la imagen debe seguir viéndose")
	}

	// cambiar de nota
	r.show("compras.md")
	if got := r.take(); len(got) != 1 || got[0] != deleteAll {
		t.Fatalf("cambiar de nota debe emitir a=d: %q", got)
	}
	if placeholderCells(r.AppModel) != 0 {
		t.Error("tras cambiar de nota no debe quedar la imagen")
	}

	// volver del editor: borra y retransmite, en ese orden
	r.show("imagen.md")
	r.take()
	r.Update(EditorFinishedMsg{Path: filepath.Join(r.c.store.BaseDir, "imagen.md")})
	got := r.take()
	if len(got) != 2 || got[0] != deleteAll || !strings.Contains(got[1], "a=T") {
		t.Fatalf("al volver del editor debe borrar y retransmitir, en ese orden: %q", got)
	}

	// abrir el editor: borra al abrir y no retransmite mientras está abierto
	r.take()
	press(r.AppModel, "e")
	if got := r.take(); len(got) != 1 || got[0] != deleteAll {
		t.Fatalf("abrir el editor debe emitir solo a=d: %q", got)
	}
	r.Update(tea.WindowSizeMsg{Width: r.w, Height: r.h})
	if got := r.take(); len(got) != 0 {
		t.Fatalf("con el editor abierto no se debe transmitir nada: %q", got)
	}
	r.Update(EditorFinishedMsg{Path: filepath.Join(r.c.store.BaseDir, "imagen.md")})
	if got := r.take(); len(got) != 1 || !strings.Contains(got[0], "a=T") {
		t.Fatalf("al volver del editor debe retransmitirse: %q", got)
	}

	// salir
	r.Update(keyMsg("q"))
	if got := r.take(); len(got) == 0 || got[0] != deleteAll {
		t.Errorf("al salir debe emitirse a=d: %q", got)
	}
}

// TestImageDetection: Init consulta con a=q y solo la respuesta OK activa las
// imágenes; sin respuesta (DA1 sin gráficos) todo sigue en texto.
func TestImageDetection(t *testing.T) {
	r := newImageRig(t, 120, 35, false)
	r.Init()
	if got := r.take(); len(got) != 1 || got[0] != image.QuerySequence() {
		t.Fatalf("Init debe consultar a=q: %q", got)
	}
	r.show("imagen.md")
	if r.c.kitty.Supported() || placeholderCells(r.AppModel) != 0 {
		t.Fatal("sin respuesta no debe haber soporte")
	}
	// una respuesta de otro tipo no activa nada
	r.Update(uv.PrimaryDeviceAttributesEvent{1, 2})
	if r.c.kitty.Supported() {
		t.Error("la respuesta a DA1 sola significa que no hay soporte")
	}
	var reply uv.KittyGraphicsEvent
	reply.Options.ID, reply.Payload = 31, []byte("OK")
	r.Update(reply)
	if !r.c.kitty.Supported() {
		t.Fatal("la respuesta OK debe activar el soporte")
	}
	r.settle() // la imagen se carga fuera de Update
	if got := r.take(); transmits(got) != 1 || placeholderCells(r.AppModel) == 0 {
		t.Errorf("al detectar soporte la imagen debe transmitirse y verse: %q", got)
	}
}

// TestImageSurvivesPopups (I1): con un popup abierto la imagen de la vista previa se
// sigue viendo: las celdas fuera del popup siguen siendo placeholders Kitty, el
// texto de reemplazo "[imagen: …]" no aparece y no se emite a=d. Solo las celdas que
// tapa el popup quedan ocultas. Antes, abrir cualquier popup borraba la imagen
// (a=d) y la reemplazaba por el texto.
func TestImageSurvivesPopups(t *testing.T) {
	for _, keys := range [][]string{{"?"}, {","}, {"x"}, {"r"}} {
		r := newImageRig(t, 120, 35, true)
		r.show("imagen.md")
		r.take()
		total := placeholderCells(r.AppModel)
		if total == 0 {
			t.Fatal("sin popup debe verse la imagen")
		}
		press(r.AppModel, keys...)
		p := r.c.top()
		if p == nil {
			t.Fatalf("%v: no abrió un popup", keys)
		}
		if got := r.take(); len(got) != 0 {
			t.Errorf("%v: abrir el popup no debe emitir nada (a=d borra la imagen): %q", keys, got)
		}
		out := plain(r.AppModel)
		if strings.Contains(out, "[imagen:") {
			t.Errorf("%v: la imagen se reemplazó por su texto con el popup abierto:\n%s", keys, out)
		}
		visible := placeholderCells(r.AppModel)
		if visible == 0 {
			t.Errorf("%v: con el popup abierto no queda ninguna celda de imagen", keys)
		}
		// ninguna celda de imagen dentro del rectángulo del popup
		rect := popupRect(r.layout, p, p.render(r.layout))
		for y, line := range screen(r.AppModel) {
			if y < rect.Y || y >= rect.Y+rect.H {
				continue
			}
			inside := ansi.Strip(ansi.Cut(line, rect.X, rect.X+rect.W))
			if strings.Contains(inside, string(kitty.Placeholder)) {
				t.Errorf("%v: hay celdas de imagen dentro del popup, fila %d", keys, y)
			}
		}
		// al cerrar, la imagen sigue entera y no se retransmite
		press(r.AppModel, "esc")
		if got := r.take(); len(got) != 0 {
			t.Errorf("%v: cerrar el popup no debe emitir nada: %q", keys, got)
		}
		if after := placeholderCells(r.AppModel); after != total {
			t.Errorf("%v: tras cerrar, celdas de imagen = %d, se esperaban %d", keys, after, total)
		}
	}
}
