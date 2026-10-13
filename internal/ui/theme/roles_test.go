package theme

import (
	"testing"
)

// TestRolesKeepTheThemeColorWhereItReads checks that a role only leaves the color the theme always used when that color is below
// ButtonMinContrast on the backgrounds the role is drawn on, so the identity of each theme is the same where nothing was wrong.
func TestRolesKeepTheThemeColorWhereItReads(t *testing.T) {
	prev := CurrentThemeName
	t.Cleanup(func() { ApplyThemeByName(prev) })
	for _, name := range ThemeNames() {
		p := AvailableThemes[name]
		ApplyPalette(p)
		if Contrast(p.Overlay0, p.Base) >= ButtonMinContrast && Contrast(p.Overlay0, p.Mantle) >= ButtonMinContrast && !sameColor(ColorMuted, p.Overlay0) {
			t.Errorf("%s: dim text left Overlay0 although it reads", name)
		}
		if Contrast(p.Green, p.Base) >= ButtonMinContrast && !sameColor(ColorActive, p.Green) {
			t.Errorf("%s: the active panel left Green although it reads", name)
		}
		if Contrast(p.Text, p.Surface1) >= ButtonMinContrast && !sameColor(ColorSelection, p.Surface1) {
			t.Errorf("%s: the selected row left Surface1 although the text reads on it", name)
		}
	}
}

// TestSelectionTextReadsInEveryTheme is the report "the selected row cannot be read in solarized-dark": Text over Surface1 was 1.7:1.
func TestSelectionTextReadsInEveryTheme(t *testing.T) {
	prev := CurrentThemeName
	t.Cleanup(func() { ApplyThemeByName(prev) })
	for _, name := range ThemeNames() {
		ApplyPalette(AvailableThemes[name])
		if r := Contrast(ColorText, ColorSelection); r < ButtonMinContrast {
			t.Errorf("%s: text on the selected row is %.1f:1", name, r)
		}
		if r := Contrast(ColorAccent, ColorSelection); r < ButtonMinContrast {
			t.Errorf("%s: accent on the selected row is %.1f:1", name, r)
		}
	}
}
