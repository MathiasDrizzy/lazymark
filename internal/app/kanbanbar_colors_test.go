package app

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// barSegment is a run of visible text with the colors it is drawn with.
type barSegment struct {
	text   string
	fg, bg color.Color
}

// colorSegments walks the SGR sequences of a rendered line and returns each run of text with its foreground and background.
func colorSegments(s string) []barSegment {
	var out []barSegment
	var fg, bg color.Color
	pos := 0
	emit := func(end int) {
		if end > pos {
			out = append(out, barSegment{s[pos:end], fg, bg})
		}
	}
	for _, loc := range sgrRe.FindAllStringSubmatchIndex(s, -1) {
		emit(loc[0])
		pos = loc[1]
		params := strings.Split(s[loc[2]:loc[3]], ";")
		for i := 0; i < len(params); i++ {
			switch params[i] {
			case "", "0":
				fg, bg = nil, nil
			case "39":
				fg = nil
			case "49":
				bg = nil
			case "38", "48":
				if i+4 < len(params) && params[i+1] == "2" {
					r, _ := strconv.Atoi(params[i+2])
					g, _ := strconv.Atoi(params[i+3])
					b, _ := strconv.Atoi(params[i+4])
					c := color.RGBA{uint8(r), uint8(g), uint8(b), 255}
					if params[i] == "38" {
						fg = c
					} else {
						bg = c
					}
					i += 4
				}
			}
		}
	}
	emit(len(s))
	return out
}

// colorsOf returns the colors the first run containing label is drawn with.
func colorsOf(t *testing.T, line, label string) (fg, bg color.Color) {
	t.Helper()
	for _, sg := range colorSegments(line) {
		if strings.Contains(sg.text, label) {
			if sg.fg == nil || sg.bg == nil {
				t.Fatalf("%q must set both its foreground and its background, got fg=%v bg=%v", label, sg.fg, sg.bg)
			}
			return sg.fg, sg.bg
		}
	}
	t.Fatalf("%q is not in the bar: %q", label, ansi.Strip(line))
	return nil, nil
}

// TestKanbanBarButtonsReadInEveryTheme is the check behind "the Hide done and Tag done buttons are grey and unreadable with another theme":
// in every built-in theme each button of the Kanban bar must be readable (WCAG 4.5:1 for its text, 3:1 against the bar), and the
// pressed state of the hide button must look different from the idle one.
func TestKanbanBarButtonsReadInEveryTheme(t *testing.T) {
	const barMin = 3.0
	prev := theme.CurrentThemeName
	t.Cleanup(func() { theme.ApplyThemeByName(prev) })
	for _, name := range theme.ThemeNames() {
		m, _ := hideModel(t, nil)
		if !theme.ApplyThemeByName(name) { // after New, which applies the theme of the config
			t.Fatalf("unknown theme %q", name)
		}
		press(m, "W")
		idle := m.renderKanbanBar()
		press(m, "o") // hiding on: "Show done (o)" and "3 hidden"
		active := m.renderKanbanBar()
		barBg := theme.ColorMantle
		// AA (4.5:1) wherever some accent of the palette can give it; solarized-light has none, and there the floor is 3:1
		textMin := theme.ButtonMinContrast
		for _, a := range []color.Color{theme.ColorBlue, theme.ColorTeal, theme.ColorMauve, theme.ColorPeach, theme.ColorGreen, theme.ColorYellow} {
			for _, f := range []color.Color{theme.ColorBase, theme.ColorMantle, theme.ColorText} {
				if views.Contrast(f, a) >= theme.ButtonGoodContrast {
					textMin = theme.ButtonGoodContrast
				}
			}
		}
		for _, c := range []struct{ line, label string }{
			{idle, "Notes (Esc)"}, {idle, "Hide done (o)"}, {idle, "Tag done (a)"},
			{active, "Show done (o)"}, {active, "Tag done (a)"}, {active, "3 hidden"},
		} {
			fg, bg := colorsOf(t, c.line, c.label)
			want := theme.ButtonMinContrast // the pressed state and the counter have to differ from the idle button, which can leave them the second-best color
			if c.label == "Notes (Esc)" || strings.HasPrefix(c.label, "Hide") || strings.HasPrefix(c.label, "Tag") {
				want = textMin
			}
			if r := views.Contrast(fg, bg); r < want {
				t.Errorf("%s: %q text contrast %.1f:1, want >= %.1f", name, c.label, r, want)
			}
			if r := views.Contrast(bg, barBg); r < barMin {
				t.Errorf("%s: %q stands out from the bar by only %.1f:1, want >= %.1f", name, c.label, r, barMin)
			}
		}
		_, idleBg := colorsOf(t, idle, "Hide done (o)")
		_, onBg := colorsOf(t, active, "Show done (o)")
		if fmt.Sprint(idleBg) == fmt.Sprint(onBg) {
			t.Errorf("%s: the hide button looks the same idle and pressed", name)
		}
	}
}
