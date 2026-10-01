package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MathiasDrizzy/lazymark/internal/app"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	home, _ := os.UserHomeDir()
	notesDir := filepath.Join(home, "Documents", "notes")
	os.MkdirAll(notesDir, 0755)

	cfg := &config.Config{
		NotesDir:       notesDir,
		Editor:         "micro",
		Theme:          "catppuccin-mocha",
		MouseClick:   true,
		ShowTagsTab:  true,
		ShowTasksTab: true,
		SidebarRatio: 0.33,
	}

	appModel, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resolutions := []struct {
		name string
		w, h int
	}{
		{"compact_80x24", 80, 24},
		{"standard_120x35", 120, 35},
		{"fullscreen_160x45", 160, 45},
	}

	for _, res := range resolutions {
		// Simular WindowSizeMsg
		updatedModel, _ := appModel.Update(tea.WindowSizeMsg{
			Width:  res.w,
			Height: res.h,
		})
		m := updatedModel.(*app.AppModel)

		view := m.View()
		outFile := filepath.Join("sandbox", fmt.Sprintf("snapshot_%s.ans", res.name))
		err := os.WriteFile(outFile, []byte(view), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outFile, err)
		} else {
			fmt.Printf("Generado snapshot: %s (%dx%d, %d bytes)\n", outFile, res.w, res.h, len(view))
		}

		// Generar snapshot de la Hoja 2 (Tablero Kanban)
		kanbanModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}})
		km := kanbanModel.(*app.AppModel)
		kanbanView := km.View()
		kanbanOutFile := filepath.Join("sandbox", fmt.Sprintf("snapshot_kanban_%s.ans", res.name))
		if err := os.WriteFile(kanbanOutFile, []byte(kanbanView), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", kanbanOutFile, err)
		} else {
			fmt.Printf("Generado snapshot: %s (%dx%d, %d bytes)\n", kanbanOutFile, res.w, res.h, len(kanbanView))
		}
	}
}
