package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/charmbracelet/x/ansi"
)

// copyFixtures copia testdata/notes a un directorio temporal para que el test
// nunca toque las notas reales ni modifique los fixtures versionados.
func copyFixtures(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("testdata", "notes")
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copiando fixtures: %v", err)
	}
	return dst
}

// TestViewFitsTerminal comprueba que View() nunca desborda la terminal:
// cada línea cabe en el ancho y la altura es exactamente la de la terminal.
func TestViewFitsTerminal(t *testing.T) {
	sizes := []struct{ w, h int }{{120, 35}, {100, 30}, {80, 24}}
	states := []struct {
		name  string
		setup func(m *AppModel)
	}{
		{"base", func(m *AppModel) {}},
		{"cheatsheet", func(m *AppModel) { m.showCheatsheet = true }},
		{"settings", func(m *AppModel) { m.showSettings = true }},
		{"preview-enfocado", func(m *AppModel) { m.activePanel = PanelPreview }},
		{"tareas", func(m *AppModel) { m.activePanel = PanelTasks; m.lastLeftPanel = PanelTasks }},
		{"tags", func(m *AppModel) { m.activePanel = PanelTags; m.lastLeftPanel = PanelTags }},
		{"kanban", func(m *AppModel) { m.currentSheet = SheetKanban }},
		{"kanban-cheatsheet", func(m *AppModel) { m.currentSheet = SheetKanban; m.showCheatsheet = true }},
	}

	for _, sz := range sizes {
		for _, st := range states {
			name := fmt.Sprintf("%s/%dx%d", st.name, sz.w, sz.h)
			t.Run(name, func(t *testing.T) {
				m, err := New(config.DefaultConfig(copyFixtures(t)))
				if err != nil {
					t.Fatalf("New: %v", err)
				}
				m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
				st.setup(m)

				lines := strings.Split(m.View().Content, "\n")
				if len(lines) != sz.h {
					t.Errorf("altura = %d líneas, se esperaban %d", len(lines), sz.h)
				}
				for i, line := range lines {
					if w := ansi.StringWidth(line); w > sz.w {
						t.Errorf("línea %d: ancho %d > %d: %q", i, w, sz.w, ansi.Strip(line))
					}
				}
			})
		}
	}
}

// lastCell devuelve la última celda visible de una línea renderizada.
func lastCell(line string) string {
	w := ansi.StringWidth(line)
	return ansi.Strip(ansi.Cut(line, w-1, w))
}

// TestPopupKeepsRightFrame (H0-3): con un popup anclado abajo a la derecha,
// el borde derecho de los paneles de fondo debe seguir intacto.
func TestPopupKeepsRightFrame(t *testing.T) {
	sizes := []struct{ w, h int }{{120, 35}, {100, 30}, {80, 24}}
	sheets := []struct {
		name  string
		sheet AppSheet
	}{{"lazygit", SheetLazygit}, {"kanban", SheetKanban}}

	for _, sz := range sizes {
		for _, sh := range sheets {
			t.Run(fmt.Sprintf("%s/%dx%d", sh.name, sz.w, sz.h), func(t *testing.T) {
				m, err := New(config.DefaultConfig(copyFixtures(t)))
				if err != nil {
					t.Fatalf("New: %v", err)
				}
				m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
				m.currentSheet = sh.sheet

				base := strings.Split(m.View().Content, "\n")
				m.showCheatsheet = true
				withPopup := strings.Split(m.View().Content, "\n")

				// La última fila es el footer, que no tiene marco.
				for y := 0; y < sz.h-1; y++ {
					want, got := lastCell(base[y]), lastCell(withPopup[y])
					if want != got {
						t.Errorf("fila %d: borde derecho %q con el popup abierto, %q sin popup", y, got, want)
					}
				}
			})
		}
	}
}
