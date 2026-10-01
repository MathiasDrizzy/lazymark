package views

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/charmbracelet/x/ansi"
)

// TestTagPreviewCutsByCells: un título largo con acentos y emoji se corta por
// celdas, sin partir caracteres UTF-8 y sin desbordar el panel.
func TestTagPreviewCutsByCells(t *testing.T) {
	notes := []storage.Note{{Title: strings.Repeat("ñáé⚠️", 20), Tags: []string{"x"}, Path: "/n.md"}}
	for w := 30; w <= 60; w++ {
		out := RenderTagPreview(notes, "x", w, 10, true)
		if !utf8.ValidString(out) {
			t.Fatalf("ancho %d: la salida tiene UTF-8 inválido", w)
		}
		for i, l := range strings.Split(out, "\n") {
			if ansi.StringWidth(l) != w {
				t.Fatalf("ancho %d: línea %d mide %d", w, i, ansi.StringWidth(l))
			}
		}
	}
}
