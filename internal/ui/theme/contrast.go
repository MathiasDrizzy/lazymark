package theme

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
)

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		x := float64(v>>8) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// Contrast es la razón de contraste WCAG (1 a 21) entre dos colores.
func Contrast(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

const (
	// ButtonGoodContrast es el contraste AA de WCAG para texto: lo que busca un botón si la paleta lo permite.
	ButtonGoodContrast = 4.5
	// ButtonMinContrast es el mínimo: el 3:1 de WCAG para texto en negrita e interfaz, el mismo que usan las fechas.
	// Algunas paletas claras (solarized-light) no dan más con ningún par de sus colores.
	ButtonMinContrast = 3.0
)

// accents son los colores de acento de la paleta activa, en el orden en que un botón los prueba después de los que pidió. El rojo queda fuera: en una interfaz
// significa peligro, y un botón de ocultar no debe parecer una advertencia.
func accents() []color.Color {
	return []color.Color{ColorBlue, ColorTeal, ColorMauve, ColorPeach, ColorGreen, ColorYellow}
}

// Button devuelve el estilo de un botón de la barra (en negrita): el fondo es un color de acento de la paleta activa y el texto el mejor de Base, Mantle y
// Text. Elige entre los colores que se piden y, después, el resto de los acentos: el primero con contraste de texto >= ButtonGoodContrast que además se
// distingue de la barra (>= ButtonMinContrast); si ninguno llega, baja la exigencia por pasos hasta el de más contraste. Nunca mezcla colores: todo sale de la paleta, así que cambia con el tema y la
// prueba de que solo se pintan colores de la paleta se sigue cumpliendo.
func Button(preferred ...color.Color) lipgloss.Style { return ButtonExcept(nil, preferred...) }

// ButtonExcept es Button sin usar el color de fondo skip, para que dos botones que deben distinguirse (reposo y activo) no acaben iguales en un tema
// que repite o escasea colores legibles.
func ButtonExcept(skip color.Color, preferred ...color.Color) lipgloss.Style {
	var candidates []color.Color
	for _, c := range append(append([]color.Color{}, preferred...), accents()...) {
		if skip == nil || !sameColor(c, skip) {
			candidates = append(candidates, c)
		}
	}
	var bestBg, bestFg color.Color
	best := -1.0
	// de más a menos exigente: texto y contraste con la barra (el fondo es Mantle) a la vez, y después solo el texto
	for _, tier := range []struct {
		text  float64
		stand bool
	}{{ButtonGoodContrast, true}, {ButtonMinContrast, true}, {ButtonGoodContrast, false}, {ButtonMinContrast, false}} {
		for _, bg := range candidates {
			fg, r := readableOn(bg)
			if r >= tier.text && (!tier.stand || Contrast(bg, ColorMantle) >= ButtonMinContrast) {
				return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true)
			}
		}
	}
	for _, bg := range candidates {
		if fg, r := readableOn(bg); r > best {
			best, bestBg, bestFg = r, bg, fg
		}
	}
	return lipgloss.NewStyle().Foreground(bestFg).Background(bestBg).Bold(true)
}

func sameColor(a, b color.Color) bool {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	return ar == br && ag == bg && ab == bb
}

// readableOn es, de los colores de texto de la paleta, el que más contrasta con bg, y ese contraste.
func readableOn(bg color.Color) (color.Color, float64) {
	var fg color.Color
	best := -1.0
	for _, c := range []color.Color{ColorBase, ColorMantle, ColorText} {
		if r := Contrast(c, bg); r > best {
			best, fg = r, c
		}
	}
	return fg, best
}
