package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderBoxWithTitle(t *testing.T) {
	ApplyPalette(CatppuccinMocha)

	testCases := []struct {
		w, h   int
		title  string
		badge  string
		active bool
	}{
		{20, 5, "[1] Notes", "(10)", true},
		{20, 5, "[1] Notes", "(10)", false},
		{40, 10, "[2] Tasks", "[pending]", true},
		{80, 24, "[4] Preview", "", false},
		{15, 4, "[3] VeryLongTitleThatNeedsTruncation", "(99)", true},
		{12, 3, "", "", false},
	}

	for _, tc := range testCases {
		res := RenderBoxWithTitle(tc.title, tc.badge, "Línea 1\nLínea 2", tc.w, tc.h, tc.active)
		lines := strings.Split(res, "\n")

		if len(lines) != tc.h {
			t.Fatalf("Para %dx%d, se esperaban %d líneas, obtenidas %d", tc.w, tc.h, tc.h, len(lines))
		}

		for idx, l := range lines {
			w := ansi.StringWidth(l)
			if w != tc.w {
				t.Fatalf("Para %dx%d (%s), línea %d mide %d columnas != %d esperadas:\n%s", tc.w, tc.h, tc.title, idx, w, tc.w, l)
			}
		}

		// Validar bordes
		topClean := ansi.Strip(lines[0])
		if !strings.HasPrefix(topClean, "╭") || !strings.HasSuffix(topClean, "╮") {
			t.Errorf("Línea superior sin esquinas correctas: %s", topClean)
		}
		bottomClean := ansi.Strip(lines[len(lines)-1])
		if !strings.HasPrefix(bottomClean, "╰") || !strings.HasSuffix(bottomClean, "╯") {
			t.Errorf("Línea inferior sin esquinas correctas: %s", bottomClean)
		}
	}
}
