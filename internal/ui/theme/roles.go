package theme

import "image/color"

// Roles of text that the palette names with one color but that some palettes cannot read: the dim text, the titles, the accents, the selected row and the footer. Each role
// keeps the color the theme always used and moves to the next one of the same palette only where that color falls below ButtonMinContrast on the
// backgrounds the role is drawn on, so the identity of each theme stays and nothing is mixed (TestThemeColorsComeFromPalette).
var (
	// ColorSelection is the background of the selected row.
	ColorSelection color.Color = CatppuccinMocha.Surface1
	// ColorMuted is dim text (dates, hints, empty messages) over the base and the title bar.
	ColorMuted color.Color = CatppuccinMocha.Overlay0
	// ColorSoft is the secondary text of an inactive panel title and of a hint.
	ColorSoft color.Color = CatppuccinMocha.Subtext0
	// ColorActive is the border and the title of the focused panel, and the green of what is done or went well.
	ColorActive color.Color = CatppuccinMocha.Green
	// ColorAccent is accent text (the pointer, a value, a key hint) over the base and over the selected row.
	ColorAccent color.Color = CatppuccinMocha.Peach
)

// legible returns the first candidate that reads on every background at ButtonMinContrast, or else the one with the best worst contrast.
func legible(candidates []color.Color, bgs ...color.Color) color.Color {
	worst := func(c color.Color) float64 {
		w := 100.0
		for _, bg := range bgs {
			if r := Contrast(c, bg); r < w {
				w = r
			}
		}
		return w
	}
	best, bestWorst := candidates[0], -1.0
	for _, c := range candidates {
		w := worst(c)
		if w >= ButtonMinContrast {
			return c
		}
		if w > bestWorst {
			best, bestWorst = c, w
		}
	}
	return best
}

// applyRoles resolves the roles of a palette; ApplyPalette calls it before rebuilding the styles.
func applyRoles(p Palette) {
	ColorSelection = legible([]color.Color{p.Surface1, p.Surface0}, p.Text) // the text must read on it: not the case of Surface1 in solarized-dark
	ColorMuted = legible([]color.Color{p.Overlay0, p.Subtext0, p.Text}, p.Base, p.Mantle)
	ColorSoft = legible([]color.Color{p.Subtext0, p.Text}, p.Base, p.Mantle)
	ColorActive = legible([]color.Color{p.Green, p.Teal, p.Blue, p.Mauve, p.Peach, p.Yellow, p.Text}, p.Base)
	ColorAccent = legible([]color.Color{p.Peach, p.Yellow, p.Mauve, p.Teal, p.Blue, p.Green, p.Text}, p.Base, ColorSelection)
}

// readable is c where it reads on the backgrounds, or else the text color: the fallback of the Markdown and code colors, which are many and
// carry no theme identity beyond their hue.
func readable(c color.Color, bgs ...color.Color) color.Color {
	return legible([]color.Color{c, ColorText}, bgs...)
}

// h1Colors are the text and the background of a level-1 heading: Base over Peach where that reads (every theme but latte), or else the
// first accent of the palette on which Base or Text does.
func h1Colors() (fg, bg color.Color) {
	for _, bg := range []color.Color{ColorPeach, ColorYellow, ColorBlue, ColorMauve, ColorTeal, ColorGreen} {
		for _, fg := range []color.Color{ColorBase, ColorText} {
			if Contrast(fg, bg) >= ButtonMinContrast {
				return fg, bg
			}
		}
	}
	return ColorBase, ColorPeach
}
