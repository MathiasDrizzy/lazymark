package textwidth

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestWidthGraphemes(t *testing.T) {
	cases := map[string]int{
		"abc":              3,
		"⚠️":               2, // U+26A0 + VS16
		"❤️":               2,
		"📝":                2,
		"":                1, // NerdFont (área de uso privado)
		"日本":               4,
		"\x1b[31mx\x1b[0m": 1,
	}
	for s, want := range cases {
		if got := Width(s); got != want {
			t.Errorf("Width(%q) = %d, se esperaba %d", s, got, want)
		}
	}
}

func TestFitExactWidth(t *testing.T) {
	inputs := []string{"", "corto", "una línea bastante larga con ⚠️ y ❤️ y 日本語", " nota.md", "\x1b[1mnegrita larga larga\x1b[0m"}
	for _, s := range inputs {
		for w := 1; w <= 20; w++ {
			got := Fit(s, w)
			if Width(got) != w {
				t.Errorf("Fit(%q, %d) mide %d: %q", s, w, Width(got), ansi.Strip(got))
			}
		}
	}
	if got := ansi.Strip(Fit("abcdef", 4)); got != "abc…" {
		t.Errorf("Fit corta con elipsis: %q", got)
	}
}

func TestCutKeepsWidthOnWideGlyph(t *testing.T) {
	// Cortar en medio de un glifo ancho debe conservar el ancho pedido.
	s := "a日b"
	for l := 0; l < 4; l++ {
		for r := l; r <= 4; r++ {
			if got := Cut(s, l, r); Width(got) != r-l {
				t.Errorf("Cut(%q,%d,%d) mide %d", s, l, r, Width(got))
			}
		}
	}
}
