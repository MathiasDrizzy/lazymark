package image

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// withTimeout falla si f tarda más de 3 s (abrir un FIFO o /dev/tty sin datos se queda esperando para siempre).
func withTimeout(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s: se colgó (más de 3 s): abrió un archivo que no es regular", what)
	}
}

// TestNeverOpensNonRegularFiles (ORD-015 C.5 S1): una imagen que es un FIFO o un dispositivo (![x](/dev/tty)) no se abre: Block devuelve false
// y Encode da un error, ambos sin esperar.
func TestNeverOpensNonRegularFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sin FIFO en Windows")
	}
	fifo := filepath.Join(t.TempDir(), "trampa.png")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skip("no se puede crear un FIFO:", err)
	}
	for _, p := range []string{fifo, "/dev/tty", "/dev/zero", "/dev/null"} {
		c := New()
		c.SetSupported(true)
		withTimeout(t, "Block "+p, func() {
			if lines, ok := c.Block(p, 40, 10); ok {
				t.Errorf("%s: Block no debe dar una imagen: %q", p, lines)
			}
		})
		withTimeout(t, "Encode "+p, func() {
			if _, err := Encode(Job{Path: p}); err == nil {
				t.Errorf("%s: Encode debe fallar", p)
			}
		})
	}
}

// TestFileSizeCap (ORD-015 C.5 S1): una imagen de más de MaxFileBytes no se lee.
func TestFileSizeCap(t *testing.T) {
	p := filepath.Join(t.TempDir(), "enorme.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	f.Truncate(MaxFileBytes + 1) // archivo disperso: no ocupa disco
	f.Close()
	c := New()
	c.SetSupported(true)
	if _, ok := c.Block(p, 40, 10); ok {
		t.Error("una imagen sobre el tope no se muestra")
	}
	if _, err := Encode(Job{Path: p}); err == nil {
		t.Error("Encode rechaza una imagen sobre el tope")
	}
}

// TestResolveHook (ORD-015 C.5 S1): con Resolve, Block solo acepta lo que el hook aprueba y usa la ruta real que devuelve.
func TestResolveHook(t *testing.T) {
	dir := t.TempDir()
	p := writePNG(t, dir, "a.png", 100, 80)
	c := New()
	c.SetSupported(true)
	c.Resolve = func(string) (string, bool) { return "", false }
	if _, ok := c.Block(p, 40, 10); ok {
		t.Error("el hook rechaza: no hay imagen")
	}
	c.Resolve = func(string) (string, bool) { return p, true }
	if _, ok := c.Block("cualquier-otra-ruta.png", 40, 10); !ok {
		t.Error("el hook aprueba y da la ruta real")
	}
}
