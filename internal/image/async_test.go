package image

import (
	goimage "image"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
)

// TestWithIDMatchesDirectEncoding: la secuencia que se codifica fuera de Update con un ID de
// marcador y se personaliza al transmitir es idéntica a codificarla directamente con ese ID.
func TestWithIDMatchesDirectEncoding(t *testing.T) {
	p := writePNG(t, t.TempDir(), "a.png", 300, 200)
	j := Job{Key: "k", Path: p, Cols: 20, Rows: 7}
	tmpl, err := Encode(j)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(p)
	img, _, _ := goimage.Decode(f)
	f.Close()
	var direct strings.Builder
	kitty.EncodeGraphics(&direct, scaleDown(img, maxSide), &kitty.Options{Action: kitty.TransmitAndPut, ID: 77, Format: kitty.PNG,
		Transmission: kitty.Direct, Chunk: true, Columns: 20, Rows: 7, VirtualPlacement: true, Quiet: 2})
	if got := withID(tmpl, 77); got != direct.String() {
		t.Errorf("la secuencia con ID real difiere de la codificada directamente:\n%.120q\n%.120q", got, direct.String())
	}
}

// TestBlockNeverDecodes: Block no decodifica nada: la primera vez devuelve un bloque "cargando" de
// la altura final y pide el trabajo; con el resultado entregado (Done) transmite desde la caché, y
// al cambiar de selección un resultado tardío no se dibuja pero queda en la caché.
func TestBlockNeverDecodes(t *testing.T) {
	p := writePNG(t, t.TempDir(), "a.png", 400, 300)
	c := New()
	c.SetSupported(true)

	lines, ok := c.Block(p, 40, 18)
	if !ok || strings.ContainsRune(strings.Join(lines, ""), kitty.Placeholder) || lines[0] != Label(p) {
		t.Fatalf("la primera vez debe ser el bloque de carga: %q %v", lines, ok)
	}
	if len(c.TakePending()) != 0 || !c.HasWanted() {
		t.Fatal("no debe transmitir nada y sí pedir el trabajo")
	}
	jobs := c.TakeJobs()
	if len(jobs) != 1 || len(lines) != jobs[0].Rows {
		t.Fatalf("1 trabajo y el bloque con sus %d filas: %v", jobs[0].Rows, jobs)
	}
	if again, _ := c.Block(p, 40, 18); len(again) != len(lines) || len(c.TakeJobs()) != 0 {
		t.Error("mientras está en vuelo no se pide dos veces")
	}
	tmpl, err := Encode(jobs[0])
	if err != nil {
		t.Fatal(err)
	}
	sel := c.Selection()
	c.NewSelection() // la selección cambió antes de que terminara
	if c.Done(sel, jobs[0], tmpl, nil) {
		t.Error("un resultado de una selección vieja no debe considerarse actual")
	}
	if len(c.TakePending()) != 0 {
		t.Error("un resultado tardío no se transmite")
	}
	// al volver a pedirla, ya está en la caché: se transmite sin lanzar trabajo
	got, ok := c.Block(p, 40, 18)
	if !ok || !strings.ContainsRune(strings.Join(got, ""), kitty.Placeholder) || len(c.TakeJobs()) != 0 {
		t.Fatal("con el resultado en la caché debe verse sin decodificar")
	}
	if seq := c.TakePending(); len(seq) != 1 || !strings.Contains(seq[0], "i=1,") {
		t.Errorf("debía transmitirse 1 vez con el ID 1: %.80q", seq)
	}
}

// TestFailedImageFallsBackToLabel: un archivo que no se puede decodificar termina como texto, sin reintentar.
func TestFailedImageFallsBackToLabel(t *testing.T) {
	// cabecera PNG válida (se leen las dimensiones) con datos corruptos: Encode falla
	p := writePNG(t, t.TempDir(), "rota.png", 50, 50)
	b, _ := os.ReadFile(p)
	os.WriteFile(p, append(b[:60], make([]byte, 40)...), 0o644)
	c := New()
	c.SetSupported(true)
	c.Block(p, 40, 18)
	job := c.TakeJobs()[0]
	_, err := Encode(job)
	if err == nil {
		t.Skip("el decodificador aceptó el archivo")
	}
	c.Done(c.Selection(), job, "", err)
	if lines, ok := c.Block(p, 40, 18); ok || lines != nil {
		t.Error("tras fallar debe usarse el texto de reemplazo")
	}
	if c.HasWanted() {
		t.Error("no se reintenta lo que falló")
	}
}

// TestReadyCacheIsBounded: la caché de imágenes codificadas no crece sin límite.
func TestReadyCacheIsBounded(t *testing.T) {
	c := New()
	for i := 0; i < maxReady+10; i++ {
		c.store(strings.Repeat("k", i+1), "x")
	}
	if len(c.ready) != maxReady || len(c.order) != maxReady {
		t.Errorf("la caché tiene %d entradas, el máximo es %d", len(c.ready), maxReady)
	}
	if _, ok := c.ready["k"]; ok {
		t.Error("debe expulsarse la más vieja")
	}
}
