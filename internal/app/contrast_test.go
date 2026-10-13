package app

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/charmbracelet/x/ansi"

	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

var dateText = regexp.MustCompile(`\d{4}-\d{2}-\d{2}|\d{1,2} [A-Z][a-z]{2}`)

// hasWords reports whether s has letters or digits: runs made only of box-drawing glyphs and icons are decoration.
func hasWords(s string) bool {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n >= 2
}

// TestTextReadsInEveryTheme sweeps the main views in every built-in theme and checks the contrast of each run of text against the background it
// is drawn on: 3:1 at least (WCAG for bold text and interface), the same floor as theme.Button. Dates are left out because they have their own
// policy (views/datecolors.go) and decoration (borders, icons) is not text. It reproduces the report that the selected row, the dim text and
// the titles and counters were unreadable in solarized-dark, nord and catppuccin-latte.
func TestTextReadsInEveryTheme(t *testing.T) {
	prev := theme.CurrentThemeName
	t.Cleanup(func() { theme.ApplyThemeByName(prev) })
	screens := []struct {
		name string
		keys []string
	}{{"notes", nil}, {"kanban", []string{"W"}}, {"tasks", []string{"2"}}, {"settings", []string{","}}, {"cheatsheet", []string{"?"}},
		{"dates", []string{"2", "d"}}, {"move", []string{"m"}}, {"kanban dates", []string{"W", "d"}}}
	var failures []string
	for _, name := range theme.ThemeNames() {
		for _, v := range screens {
			m, note := hideModel(t, nil)
			if err := os.WriteFile(note, []byte(richNote), 0o644); err != nil {
				t.Fatal(err)
			}
			m.c.reload()
			m.afterChange()
			theme.ApplyThemeByName(name) // after New, which applies the theme of the config
			press(m, "down", "down")
			press(m, v.keys...)
			seen := map[string]bool{}
			for _, line := range strings.Split(m.View().Content, "\n") {
				for _, sg := range colorSegments(line) {
					txt := strings.TrimSpace(ansi.Strip(sg.text))
					if sg.fg == nil || !hasWords(txt) || dateText.MatchString(txt) {
						continue
					}
					bg := sg.bg
					if bg == nil {
						bg = theme.ColorBase
					}
					if r := themeContrast(sg.fg, bg); r < theme.ButtonMinContrast && !seen[txt] {
						seen[txt] = true
						failures = append(failures, fmt.Sprintf("%-20s %-10s %.1f:1 %q", name, v.name, r, txt))
					}
				}
			}
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		shown := failures
		if len(shown) > 400 {
			shown = shown[:400]
		}
		t.Errorf("%d runs of text below %.0f:1:\n%s", len(failures), theme.ButtonMinContrast, strings.Join(shown, "\n"))
	}
}

func themeContrast(a, b interface{ RGBA() (r, g, b, a uint32) }) float64 {
	return theme.Contrast(a, b)
}

// richNote exercises what the Markdown preview and its code blocks paint: headings, a quote, a rule, a link, a table, code and strikethrough.
const richNote = "# Title\n\n## Section\n\nText with **bold**, *italic*, ~~gone~~ and [a link](https://example.com).\n\n> a quote\n\n---\n\n" +
	"| a | b |\n|---|---|\n| 1 | 2 |\n\n```go\n// comment\nfunc main() { println(\"hi\", 42) }\n```\n\n- [ ] alpha\n- [x] gamma\n"
