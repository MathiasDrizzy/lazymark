package image

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// RenderInlineToAnsi abre la imagen y la convierte a un bloque de texto ANSI TrueColor usando medios bloques (▀)
func RenderInlineToAnsi(imagePath string, maxWidth, maxHeight int) (string, error) {
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		absPath = imagePath
	}

	f, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if origW == 0 || origH == 0 {
		return "", fmt.Errorf("dimensiones inválidas de imagen")
	}

	// Cada carácter de celda de terminal es ~2:1 vertical (un carácter de ancho equivale a 2 filas de píxeles)
	// usando el carácter medio bloque '▀' (arriba foreground, abajo background).
	targetW := maxWidth
	if targetW > 48 {
		targetW = 48
	}
	if targetW < 10 {
		targetW = 10
	}

	// Calcular altura manteniendo la relación de aspecto
	// Como cada celda contiene 2 píxeles verticales, multiplicamos por 2 para la cuadrícula de píxeles
	pixelHeight := (origH * targetW * 2) / (origW * 2)
	maxPixelHeight := maxHeight * 2
	if maxPixelHeight > 36 {
		maxPixelHeight = 36
	}
	if pixelHeight > maxPixelHeight {
		pixelHeight = maxPixelHeight
		targetW = (origW * pixelHeight) / (origH * 2)
		if targetW < 10 {
			targetW = 10
		}
	}

	var sb strings.Builder
	// Recorrer de 2 en 2 filas de píxeles
	for py := 0; py < pixelHeight; py += 2 {
		for px := 0; px < targetW; px++ {
			// Mapear coordenadas a la imagen original
			srcX := bounds.Min.X + (px * origW) / targetW
			srcYTop := bounds.Min.Y + (py * origH) / pixelHeight
			srcYBottom := bounds.Min.Y + ((py + 1) * origH) / pixelHeight

			r1, g1, b1, _ := img.At(srcX, srcYTop).RGBA()
			topR, topG, topB := uint8(r1>>8), uint8(g1>>8), uint8(b1>>8)

			if py+1 < pixelHeight {
				r2, g2, b2, _ := img.At(srcX, srcYBottom).RGBA()
				botR, botG, botB := uint8(r2>>8), uint8(g2>>8), uint8(b2>>8)
				// Foreground: top, Background: bottom, Carácter: ▀
				sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", topR, topG, topB, botR, botG, botB))
			} else {
				// Solo top
				sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[49m▀", topR, topG, topB))
			}
		}
		sb.WriteString("\x1b[0m\n")
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}
