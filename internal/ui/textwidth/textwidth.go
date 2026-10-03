// Package textwidth es el único lugar de lazymark que mide y corta texto en
// celdas de terminal. Todo se mide por grapheme clusters (como Lip Gloss v2),
// entiende secuencias ANSI y nunca parte un glifo ancho.
package textwidth

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Ellipsis es el marcador de texto cortado.
const Ellipsis = "…"

// Width devuelve el ancho en celdas de s, ignorando secuencias ANSI.
func Width(s string) int {
	return ansi.StringWidth(s)
}

// Strip quita las secuencias ANSI de s.
func Strip(s string) string {
	return ansi.Strip(s)
}

// Truncate corta s a w celdas como máximo; si corta, termina en tail.
func Truncate(s string, w int, tail string) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, tail)
}

// Pad rellena s con espacios a la derecha hasta w celdas. No corta.
func Pad(s string, w int) string {
	if d := w - Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// PadLeft rellena s con espacios a la izquierda hasta w celdas. No corta.
func PadLeft(s string, w int) string {
	if d := w - Width(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return s
}

// Fit deja s en exactamente w celdas: la corta con "…" si sobra y la rellena
// con espacios si falta.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if Width(s) > w {
		s = Truncate(s, w, Ellipsis)
	}
	return Pad(s, w)
}

// Cut devuelve las celdas [left, right) de s. Si el corte cae en medio de un
// glifo ancho, rellena con espacios para conservar el ancho pedido.
func Cut(s string, left, right int) string {
	if right <= left {
		return ""
	}
	if left < 0 {
		left = 0
	}
	rest := s
	if left > 0 {
		// ansi.TruncateLeft conserva un glifo ancho partido por el borde
		// izquierdo; se reemplaza por espacios para no correr las columnas.
		target := Width(s) - left
		if target <= 0 {
			return Repeat(" ", right-left)
		}
		rest = ansi.TruncateLeft(s, left, "")
		for k := Width(rest) - target; k > 0 && Width(rest) > target; k++ {
			if r := ansi.TruncateLeft(rest, k, ""); Width(r) <= target {
				rest = Repeat(" ", target-Width(r)) + r
			}
		}
	}
	return Pad(ansi.Truncate(rest, right-left, ""), right-left)
}

// Repeat repite r n veces; con n <= 0 devuelve "".
func Repeat(r string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(r, n)
}

// NoControl cambia por "?" los caracteres de control (también el ESC de una secuencia): lo que viene de nombres de archivo o del texto de las
// notas no es de fiar y no debe llegar a la terminal como orden.
func NoControl(s string) string {
	for _, r := range s {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return strings.Map(func(r rune) rune {
				if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
					return '?'
				}
				return r
			}, s)
		}
	}
	return s
}
