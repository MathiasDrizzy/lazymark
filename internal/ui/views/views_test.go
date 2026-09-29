package views

import (
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
)

func TestCollectTagsAndNotesForTag(t *testing.T) {
	notes := []storage.Note{
		{
			ID:    "nota1.md",
			Title: "Nota 1",
			Tags:  []string{"golang", "tui"},
		},
		{
			ID:    "nota2.md",
			Title: "Nota 2",
			Tags:  []string{"golang", "productividad"},
		},
	}

	tags := CollectTags(notes)
	if len(tags) != 3 {
		t.Fatalf("se esperaban 3 tags únicos, obtenidos %d", len(tags))
	}

	foundGo := false
	for _, tag := range tags {
		if tag.Name == "golang" {
			foundGo = true
			if tag.NoteCount != 2 {
				t.Errorf("conteo de tag 'golang' esperado 2, obtenido %d", tag.NoteCount)
			}
		}
	}
	if !foundGo {
		t.Errorf("no se encontró el tag 'golang'")
	}

	goNotes := NotesForTag(notes, "golang")
	if len(goNotes) != 2 {
		t.Errorf("se esperaban 2 notas con tag golang, obtenidas %d", len(goNotes))
	}

	tuiNotes := NotesForTag(notes, "tui")
	if len(tuiNotes) != 1 {
		t.Errorf("se esperaba 1 nota con tag tui, obtenida %d", len(tuiNotes))
	}
}

func TestCollectTasksAndFilters(t *testing.T) {
	notes := []storage.Note{
		{
			ID:    "nota1.md",
			Title: "Tareas",
			Path:  "/tmp/nota1.md",
			Tasks: []storage.Task{
				{NoteTitle: "Tareas", Line: 5, Text: "Hacer café", Done: false},
				{NoteTitle: "Tareas", Line: 6, Text: "Revisar email", Done: true},
			},
		},
	}

	all := CollectTasks(notes, TaskFilterAll)
	if len(all) != 2 {
		t.Fatalf("se esperaban 2 tareas totales, obtenidas %d", len(all))
	}

	pending := CollectTasks(notes, TaskFilterPending)
	if len(pending) != 1 || pending[0].Text != "Hacer café" {
		t.Fatalf("se esperaba 1 tarea pendiente ('Hacer café'), obtenidas %d", len(pending))
	}

	done := CollectTasks(notes, TaskFilterDone)
	if len(done) != 1 || done[0].Text != "Revisar email" {
		t.Fatalf("se esperaba 1 tarea completada ('Revisar email'), obtenidas %d", len(done))
	}

	label := TaskFilterLabel(TaskFilterPending)
	if label != "Pendientes" && label != "Pending" {
		t.Errorf("etiqueta inesperada para TaskFilterPending: %s", label)
	}
}

func TestCollectImages(t *testing.T) {
	notes := []storage.Note{
		{
			ID:     "nota1.md",
			Title:  "Con Imagen",
			Path:   "/tmp/nota1.md",
			Images: []string{"assets/screenshot.png"},
		},
	}

	imgs := CollectImages(notes)
	if len(imgs) != 1 {
		t.Fatalf("se esperaba 1 imagen, obtenidas %d", len(imgs))
	}
	if imgs[0].Path != "assets/screenshot.png" {
		t.Errorf("ruta de imagen incorrecta: %s", imgs[0].Path)
	}
}

func TestRenderTabsAndHitTest(t *testing.T) {
	ht := mouse.NewHitTester()
	rendered := RenderTabs(0, 80, ht)
	if rendered == "" {
		t.Fatalf("RenderTabs devolvió string vacío")
	}

	zone, ok := ht.Check(2, 0)
	if !ok || zone.Type != mouse.ZoneTab || zone.Index != 0 {
		t.Errorf("Hit-testing de pestaña 0 falló: %+v", zone)
	}
}

func TestRenderNoteListAndPreview(t *testing.T) {
	notes := []storage.Note{
		{
			ID:      "test.md",
			Title:   "Nota de Prueba",
			Path:    "/tmp/test.md",
			Content: "# Encabezado\nContenido de prueba",
			ModTime: time.Now(),
		},
	}
	ht := mouse.NewHitTester()
	listOut := RenderNoteList(notes, 0, 40, 20, true, ht, 1)
	if listOut == "" {
		t.Errorf("RenderNoteList devolvió string vacío")
	}

	prevOut := RenderPreview(&notes[0], 60, 20, false, nil)
	if prevOut == "" {
		t.Errorf("RenderPreview devolvió string vacío")
	}
}

func TestRenderSettingsView(t *testing.T) {
	cfg := &config.Config{
		Editor:         "micro",
		Theme:          "catppuccin-mocha",
		Language:       "es",
		ShowTagsTab:    true,
		ShowTasksTab:   true,
		ShowGalleryTab: true,
	}
	ht := mouse.NewHitTester()
	out := RenderSettingsView(cfg, SectionEditor, 0, 80, 24, ht)
	if out == "" {
		t.Errorf("RenderSettingsView devolvió string vacío")
	}

	// Comprobar que los botones de acción se registraron
	zone, ok := ht.Check(5, 4)
	if !ok || zone.Type != mouse.ZoneAction {
		t.Errorf("hit-test en configuración falló")
	}
}
