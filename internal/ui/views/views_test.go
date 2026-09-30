package views

import (
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	entries := []storage.NoteEntry{
		{
			Name: "test.md",
			Path: "/tmp/test.md",
			Type: storage.EntryNote,
			Note: &notes[0],
		},
	}
	ht := mouse.NewHitTester()
	listOut := RenderNoteList(entries, nil, 0, 40, 20, true, ht, 1)
	if listOut == "" {
		t.Errorf("RenderNoteList devolvió string vacío")
	}

	prevOut := RenderPreview(&notes[0], 60, 20, false, nil, 0, 0)
	if prevOut == "" {
		t.Errorf("RenderPreview devolvió string vacío")
	}
}

func TestRenderConfirmModal(t *testing.T) {
	ht := mouse.NewHitTester()
	notes := []storage.Note{
		{Title: "Nota de prueba", Path: "/notes/test.md"},
	}
	entries := []storage.NoteEntry{
		{Name: "folder-8", Path: "/notes/folder-8", Type: storage.EntryFolder},
		{Name: "nota-1", Path: "/notes/nota-1.md", Type: storage.EntryNote, Note: &notes[0]},
	}
	listOut := RenderNoteList(entries, nil, 0, 26, 20, true, ht, 1)
	prevOut := RenderPreview(&notes[0], 46, 20, false, nil, 0, 0)
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, listOut, prevOut)
	fullView := lipgloss.JoinVertical(lipgloss.Left, "TABS", mainView, "FOOTER")

	modal := RenderConfirmModal("󰀪  Delete Folder with Items", "The folder 'folder-8' contains 2 item(s).\nDo you want to move it and all its notes to trash?", 80, 24, ht)
	if modal == "" {
		t.Fatalf("RenderConfirmModal devolvió string vacío")
	}

	over := OverlayLayers(fullView, modal, 80, 24, false)
	for idx, l := range strings.Split(over, "\n") {
		w := ansi.StringWidth(l)
		if idx >= 7 && idx <= 15 && w != 80 {
			t.Errorf("Línea %d tiene ancho %d, se esperaba 80: %q", idx, w, l)
		}
	}
}

func TestRenderSettingsModal(t *testing.T) {
	cfg := &config.Config{
		Editor:         "micro",
		Theme:          "catppuccin-mocha",
		Language:       "es",
		ShowTagsTab:    true,
		ShowTasksTab:   true,
		ShowGalleryTab: true,
	}
	ht := mouse.NewHitTester()
	out := RenderSettingsModal(cfg, ItemLanguage, 80, 24, ht)
	if out == "" {
		t.Errorf("RenderSettingsModal devolvió string vacío")
	}

	// Comprobar que los ítems del modal se registraron en la esquina inferior derecha
	zone, ok := ht.Check(30, 11)
	if !ok || zone.Type != mouse.ZoneAction {
		t.Errorf("hit-test en modal de configuración falló: %+v", zone)
	}
}

func TestRenderCheatsheetAndOverlay(t *testing.T) {
	sheet := RenderCheatsheet(80, 24)
	if sheet == "" {
		t.Fatalf("RenderCheatsheet devolvió string vacío")
	}

	// Simular base con borde derecho en la columna 79
	baseLine := "LEFT COLUMN" + strings.Repeat(" ", 67) + "│"
	var baseLines []string
	for i := 0; i < 24; i++ {
		baseLines = append(baseLines, baseLine)
	}
	base := strings.Join(baseLines, "\n")

	over := OverlayLayers(base, sheet, 80, 24, true)
	if over == "" {
		t.Fatalf("OverlayLayers devolvió string vacío")
	}

	// Comprobar que en las líneas del overlay, no se arrastra el borde derecho del panel base '│' a la derecha del modal
	lines := strings.Split(over, "\n")
	for idx, l := range lines {
		// En las líneas inferiores donde está el overlay, la línea no debe terminar con el '│' del fondo
		if idx >= 6 && idx < 23 {
			trimmed := strings.TrimRight(l, " ")
			// Si termina en borde de modal o texto, no debe tener un segundo '│' a la derecha
			if strings.Count(trimmed, "│") > 2 {
				t.Errorf("Línea %d tiene bordes fantasma: %s", idx, l)
			}
		}
	}
}

func TestRenderMoveModal(t *testing.T) {
	ht := mouse.NewHitTester()
	folders := []string{"/tmp/notes", "/tmp/notes/sub"}
	modal := RenderMoveModal(folders, "/tmp/notes", 0, "test.md", 80, 24, ht)
	if modal == "" {
		t.Fatalf("RenderMoveModal devolvió string vacío")
	}
}

func TestRenderTrashModal(t *testing.T) {
	ht := mouse.NewHitTester()
	items := []storage.TrashItem{
		{
			ID:           "test-item-1",
			OriginalPath: "/notes/test.md",
			Name:         "test.md",
			IsDir:        false,
			DeletedAt:    time.Now().Add(-2 * 24 * time.Hour),
		},
	}
	modal := RenderTrashModal(items, 0, 80, 24, ht)
	if modal == "" {
		t.Fatalf("RenderTrashModal devolvió string vacío")
	}
	if !strings.Contains(modal, "test.md") {
		t.Errorf("RenderTrashModal no contiene el nombre de la nota")
	}
	if !strings.Contains(modal, "18d") {
		t.Errorf("RenderTrashModal no calculó los días restantes correctamente")
	}
}

func TestNarrowNoteListNoWrap(t *testing.T) {
	notes := []storage.Note{
		{
			Title:   "nueva-nota-con-nombre-largo.md",
			Path:    "/notes/nueva-nota.md",
			ModTime: time.Now(),
		},
	}
	entries := []storage.NoteEntry{
		{
			Name:    "nueva-nota-con-nombre-largo.md",
			Path:    "/notes/nueva-nota.md",
			Type:    storage.EntryNote,
			Note:    &notes[0],
			Depth:   2,
			ModTime: time.Now(),
		},
	}
	ht := mouse.NewHitTester()
	narrowOut := RenderNoteList(entries, nil, 0, 20, 10, true, ht, 1)
	lines := strings.Split(narrowOut, "\n")
	for idx, l := range lines {
		w := ansi.StringWidth(l)
		if w > 24 {
			t.Errorf("Línea %d excede el ancho: w=%d, contenido: %q", idx, w, l)
		}
		trimmed := strings.TrimSpace(l)
		if trimmed == "Sep" || trimmed == "Jan" {
			t.Errorf("Se encontró mes desbordado en su propia línea: %q", l)
		}
	}
}
