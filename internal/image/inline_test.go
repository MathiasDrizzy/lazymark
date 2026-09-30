package image

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderInlineToAnsi(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")

	// Crear imagen de 4x4 píxeles con el paquete estándar image
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 250, G: 180, B: 130, A: 255})
		}
	}

	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("error creando archivo: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("error encoding png: %v", err)
	}
	f.Close()

	ansi, err := RenderInlineToAnsi(imgPath, 10, 10)
	if err != nil {
		t.Fatalf("RenderInlineToAnsi falló: %v", err)
	}
	if len(ansi) == 0 {
		t.Errorf("ANSI devuelto está vacío")
	}
}
