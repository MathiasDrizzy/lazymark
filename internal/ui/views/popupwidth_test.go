package views

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/charmbracelet/x/ansi"
)

// TestPopupOuterWidth fija el ancho exterior (con borde) de cada popup.
// Lip Gloss v2 cuenta el borde dentro de Width(); sin compensarlo, los popups
// quedan 2 columnas más angostos y su contenido baja de renglón.
func TestPopupOuterWidth(t *testing.T) {
	ht := mouse.NewHitTester()
	cases := []struct {
		name  string
		out   string
		width int
	}{
		{"cheatsheet", RenderCheatsheet(120, 35), 60},
		{"confirm", RenderConfirmModal("Título", "Mensaje", 120, 35, ht), 54},
		{"move", RenderMoveModal([]string{"a", "b"}, "/tmp", 0, "nota", 120, 35, ht), 52},
		{"settings", RenderSettingsModal(config.DefaultConfig(t.TempDir()), 0, 120, 35, ht), 54},
		{"trash", RenderTrashModal(nil, 0, 120, 35, ht), 58},
	}
	for _, c := range cases {
		for i, line := range strings.Split(c.out, "\n") {
			if w := ansi.StringWidth(line); w != c.width {
				t.Errorf("%s línea %d: ancho %d, se esperaba %d: %q", c.name, i, w, c.width, ansi.Strip(line))
				break
			}
		}
	}
}
