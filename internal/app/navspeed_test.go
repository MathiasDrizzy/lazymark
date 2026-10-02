package app

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/charmbracelet/x/ansi"
)

// bigImage escribe una imagen de w x h con ruido (no se comprime bien, como una foto) en path.
func bigImage(t testing.TB, path string, w, h int) {
	t.Helper()
	rng := rand.New(rand.NewSource(42))
	img := goimage.NewRGBA(goimage.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			n := uint8(rng.Intn(60))
			img.SetRGBA(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 120 + n/2, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if strings.HasSuffix(path, ".jpg") {
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	} else {
		err = png.Encode(f, img)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// newNavRig crea el modelo con nNotes notas, cada una con su propia copia de una imagen grande
// (PNG las pares, JPG las impares), todas con soporte Kitty. Las notas son nav-00.md, nav-01.md…
func newNavRig(t testing.TB, nNotes, px int) *imageRig {
	t.Helper()
	tt, _ := t.(*testing.T)
	if tt == nil {
		t.Fatal("newNavRig necesita *testing.T")
	}
	r := newImageRig(tt, 120, 35, true)
	dir := r.c.store.BaseDir
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	src := t.TempDir()
	bigImage(t, filepath.Join(src, "a.png"), px, px*3/4)
	bigImage(t, filepath.Join(src, "b.jpg"), px, px*3/4)
	for i := 0; i < nNotes; i++ {
		name, ext := fmt.Sprintf("nav-%02d", i), ".png"
		from := "a.png"
		if i%2 == 1 {
			ext, from = ".jpg", "b.jpg"
		}
		b, _ := os.ReadFile(filepath.Join(src, from))
		os.WriteFile(filepath.Join(dir, "assets", name+ext), b, 0o644)
		md := fmt.Sprintf("# Nota %d\n\nTexto antes.\n\n![](assets/%s%s)\n\nTexto después.\n", i, name, ext)
		os.WriteFile(filepath.Join(dir, name+".md"), []byte(md), 0o644)
	}
	r.afterChange()
	r.showNoLoad("nav-00.md")
	r.take()
	return r
}

// navStats mide Update+View por tecla y los bytes que se mandan a la terminal.
type navStats struct {
	per        []time.Duration
	bytes      int
	transmits  int
	deleteAlls int
}

func (s navStats) p(q float64) time.Duration {
	d := append([]time.Duration(nil), s.per...)
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[min(len(d)-1, int(float64(len(d))*q))]
}

func (s navStats) String() string {
	var sum time.Duration
	for _, d := range s.per {
		sum += d
	}
	return fmt.Sprintf("teclas=%d  media=%v  p50=%v  p95=%v  max=%v  | transmisiones=%d  a=d=%d  bytes Kitty=%d",
		len(s.per), (sum / time.Duration(len(s.per))).Round(time.Microsecond), s.p(.5).Round(time.Microsecond),
		s.p(.95).Round(time.Microsecond), s.p(1).Round(time.Microsecond), s.transmits, s.deleteAlls, s.bytes)
}

// navigate pulsa keys con foco en Notas y mide cada tecla (Update + View).
func navigate(r *imageRig, keys []string) navStats {
	var s navStats
	for _, k := range keys {
		start := time.Now()
		press(r.AppModel, k)
		_ = r.View()
		s.per = append(s.per, time.Since(start))
	}
	for _, seq := range r.take() {
		s.bytes += len(seq)
		if strings.Contains(seq, "a=T") {
			s.transmits++
		}
		if seq == deleteAll {
			s.deleteAlls++
		}
	}
	return s
}

// budget es el presupuesto por tecla: 16 ms (un cuadro a 60 fps); con -race, 8 veces más.
func budget() time.Duration {
	if raceOn {
		return 8 * 16 * time.Millisecond
	}
	return 16 * time.Millisecond
}

// runCmd ejecuta un comando de Bubble Tea y devuelve los mensajes que produce (con tea.Batch, uno por comando).
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch m := cmd().(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			out = append(out, runCmd(c)...)
		}
		return out
	case nil:
		return nil
	default:
		return []tea.Msg{m}
	}
}

// TestNavigationNeverWaitsForImages (N1): con imágenes de 2000 px, Update+View de cada tecla de
// navegación tarda menos de un cuadro (p95 < 16 ms) y mientras se navega no se manda ningún byte de
// imagen. Antes (medido, 8 notas con PNG/JPG de 2000 px): p95 = 138 ms por tecla, 12 transmisiones y
// 11,5 MB de secuencias Kitty en 37 teclas, porque cada selección decodificaba, reducía y
// codificaba la imagen dentro de Update.
func TestNavigationNeverWaitsForImages(t *testing.T) {
	r := newNavRig(t, 8, 2000)
	var keys []string
	for i := 0; i < 30; i++ {
		keys = append(keys, "down")
	}
	for i := 0; i < 7; i++ {
		keys = append(keys, "up")
	}
	st := navigate(r, keys)
	t.Log(st)
	if p95 := st.p(.95); p95 >= budget() {
		t.Errorf("p95 por tecla = %v, el presupuesto es %v", p95, budget())
	}
	if st.transmits != 0 || st.bytes > 1024 {
		t.Errorf("mientras se navega no debe mandarse ninguna imagen: %d transmisiones, %d bytes", st.transmits, st.bytes)
	}
}

// TestLoadedImageCostsLittle (N1): lo que sí se hace en Update cuando una imagen llega (armar la secuencia y
// repintar) también cabe en un cuadro.
func TestLoadedImageCostsLittle(t *testing.T) {
	r := newNavRig(t, 2, 2000)
	jobs := r.c.kitty.TakeJobs()
	if len(jobs) != 1 {
		t.Fatalf("la nota 0 debía pedir 1 imagen, pidió %d", len(jobs))
	}
	tmpl, err := image.Encode(jobs[0])
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	r.Update(imageLoadedMsg{sel: r.c.kitty.Selection(), job: jobs[0], tmpl: tmpl, err: nil})
	_ = r.View()
	took := time.Since(start)
	t.Logf("Update+View al llegar una imagen de 2000 px: %v (secuencia de %d KB)", took.Round(time.Microsecond), len(tmpl)/1024)
	if took >= budget() {
		t.Errorf("tardó %v, el presupuesto es %v", took, budget())
	}
}

// TestOnlyFinalSelectionTransmits (N2): 30 ↓ rápidas sobre notas con imágenes. Los ticks de las
// selecciones intermedias no cargan nada; solo la selección final lanza su imagen y es la única que
// se transmite, y después del cambio no llega ninguna transmisión de una selección vieja.
func TestOnlyFinalSelectionTransmits(t *testing.T) {
	r := newNavRig(t, 12, 600)
	var ticks []int
	// 30 teclas (↓↓↓↑↑ seis veces) que pasan por varias notas con imagen y terminan en otra
	for i := 0; i < 6; i++ {
		for _, k := range []string{"down", "down", "down", "up", "up"} {
			press(r.AppModel, k)
			ticks = append(ticks, r.c.kitty.Selection())
		}
	}
	if !strings.HasPrefix(filepath.Base(r.displayedNote().Path), "nav-") {
		t.Fatalf("la selección final debía ser una nota con imagen: %s", r.displayedNote().Path)
	}
	final := r.c.kitty.Selection()
	finalNote := r.displayedNote().Path
	if r.take(); r.c.kitty.HasLive() {
		t.Fatal("no debe haber imágenes transmitidas mientras se navega")
	}
	var loaded []tea.Msg
	for _, sel := range ticks { // los ticks llegan en orden, cada uno 40 ms después de su tecla
		loaded = append(loaded, runCmd(r.startImageJobs(imageTickMsg{sel: sel}))...)
	}
	if len(loaded) != 1 {
		t.Fatalf("de 30 selecciones solo la final debe cargar su imagen: %d imágenes lanzadas", len(loaded))
	}
	msg := loaded[0].(imageLoadedMsg)
	if msg.sel != final || !strings.Contains(msg.job.Path, strings.TrimSuffix(filepath.Base(finalNote), ".md")) {
		t.Errorf("la imagen cargada no es la de la selección final: %+v (nota final %s)", msg.job.Path, finalNote)
	}
	r.Update(msg)
	got := r.take()
	if transmits(got) != 1 {
		t.Fatalf("debía transmitirse exactamente 1 imagen, hubo %d", transmits(got))
	}
	if placeholderCells(r.AppModel) == 0 {
		t.Error("la imagen de la selección final debe verse")
	}
}

// TestLateImageIsDropped (N3): una imagen que termina de cargar después de cambiar de nota no se
// dibuja ni se transmite, y no deja restos al volver; la que llega a tiempo sí.
func TestLateImageIsDropped(t *testing.T) {
	r := newNavRig(t, 3, 600)
	oldSel := r.c.kitty.Selection()
	jobs := r.c.kitty.TakeJobs() // nav-00 pidió su imagen y está "en vuelo"
	if len(jobs) != 1 {
		t.Fatalf("nav-00 debía pedir 1 imagen: %d", len(jobs))
	}
	slow, err := image.Encode(jobs[0])
	if err != nil {
		t.Fatal(err)
	}
	r.showNoLoad("nav-01.md") // la selección cambió antes de que terminara
	r.take()
	r.Update(imageLoadedMsg{sel: oldSel, job: jobs[0], tmpl: slow})
	if got := r.take(); transmits(got) != 0 {
		t.Errorf("la imagen tardía de la selección vieja se transmitió: %q", got)
	}
	if n := placeholderCells(r.AppModel); n != 0 {
		t.Errorf("la imagen tardía se dibujó: %d celdas", n)
	}
	// la de la selección actual llega a tiempo y se ve
	r.settle()
	if placeholderCells(r.AppModel) == 0 {
		t.Error("la imagen de la selección actual debe verse")
	}
	// y al volver a la nota vieja, su resultado ya está en la caché: se ve al instante, sin volver a decodificar
	r.show("nav-00.md")
	if len(r.c.kitty.TakeJobs()) != 0 || placeholderCells(r.AppModel) == 0 {
		t.Error("al volver a una imagen ya codificada debe verse al instante")
	}
}

// TestLoadingKeepsLayout: mientras la imagen carga ocupa las mismas filas que tendrá, así el texto de
// alrededor no salta cuando llega; y un archivo que no se puede decodificar queda como texto.
func TestLoadingKeepsLayout(t *testing.T) {
	r := newNavRig(t, 2, 600)
	rowOf := func(sub string) int {
		for y, l := range screen(r.AppModel) {
			if strings.Contains(ansi.Strip(l), sub) {
				return y
			}
		}
		return -1
	}
	before := rowOf("Texto después")
	if before < 0 || placeholderCells(r.AppModel) != 0 || !strings.Contains(plain(r.AppModel), "[imagen: nav-00.png]") {
		t.Fatalf("antes de cargar debe verse el texto de reemplazo: fila=%d\n%s", before, plain(r.AppModel))
	}
	r.settle()
	if after := rowOf("Texto después"); after != before || placeholderCells(r.AppModel) == 0 {
		t.Errorf("el texto de después saltó de la fila %d a la %d al llegar la imagen", before, after)
	}
}

// syncBuffer es un io.Writer seguro entre goroutines para capturar lo que escribe el programa.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// TestProgramLoadsImageAfterDebounce (N2, de punta a punta): con el programa de Bubble Tea de
// verdad (tea.Tick y comandos en goroutines), tras teclas rápidas la imagen de la selección
// final llega sola, una vez, y no antes del debounce; ninguna transmisión ocurre mientras se
// navega.
func TestProgramLoadsImageAfterDebounce(t *testing.T) {
	r := newNavRig(t, 6, 1200)
	out := &syncBuffer{}
	m := r.AppModel
	m.emit = tea.Raw // el programa real escribe las secuencias a la terminal
	p := tea.NewProgram(m, tea.WithInput(nil), tea.WithOutput(out), tea.WithWindowSize(120, 35))
	done := make(chan struct{})
	go func() { _, _ = p.Run(); close(done) }()
	time.Sleep(150 * time.Millisecond)
	for i := 0; i < 3; i++ {
		p.Send(keyMsg("down"))
		time.Sleep(5 * time.Millisecond)
	}
	if strings.Contains(out.String(), "a=T") {
		t.Error("no debe transmitirse ninguna imagen mientras se navega")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && strings.Count(out.String(), "a=T") == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond) // por si llegara una segunda transmisión
	p.Quit()
	<-done
	if n := strings.Count(out.String(), "a=T"); n != 1 {
		t.Errorf("tras asentarse la selección debía transmitirse 1 imagen, hubo %d", n)
	}
}
