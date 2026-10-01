package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/charmbracelet/x/ansi"
)

// TestPreviewKeepsBlankLines (H1-4): el preview respeta los saltos simples y
// la cantidad exacta de líneas en blanco escritas en el editor.
func TestPreviewKeepsBlankLines(t *testing.T) {
	path := filepath.Join("testdata", "notes", "parrafos.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := ansi.Strip(renderMarkdown(&storage.Note{Path: path, Content: string(data)}, 70))
	checks := map[string]string{
		"salto simple":          `UNO_A[^\n]*\n[^\n]*UNO_B`,
		"una línea en blanco":   `UNO_B[^\n]*\n *\n *DOS_A`,
		"dos líneas en blanco":  `DOS_A[^\n]*\n *\n *\n *TRES_A`,
		"tres líneas en blanco": `TRES_A[^\n]*\n *\n *\n *\n *CUATRO_A`,
		"código intacto":        `codigo *\n *\n *\n *sigue_codigo`,
	}
	for name, re := range checks {
		if !regexp.MustCompile(re).MatchString(out) {
			t.Errorf("%s: no coincide %q en:\n%s", name, re, out)
		}
	}
	if strings.Contains(out, " ") {
		t.Error("quedaron marcadores internos en la salida")
	}
}
