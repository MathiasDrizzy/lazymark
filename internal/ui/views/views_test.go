package views

import (
	"strings"
	"testing"

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

func TestCollectKanbanAndRender(t *testing.T) {
	notes := []storage.Note{
		{
			ID:    "nota1.md",
			Title: "Sprint 1",
			Path:  "/notes/sprint1.md",
			Tasks: []storage.Task{
				{NoteTitle: "Sprint 1", Line: 5, Text: "Escribir especificación", Done: false},
				{NoteTitle: "Sprint 1", Line: 6, Text: "Implementar backend #doing", Done: false},
				{NoteTitle: "Sprint 1", Line: 7, Text: "Diseñar logo", Done: true},
			},
		},
		{
			ID:    "nota2.md",
			Title: "Sprint 2",
			Path:  "/notes/sprint2.md",
			Tasks: []storage.Task{
				{NoteTitle: "Sprint 2", Line: 10, Text: "Configurar CI/CD #wip", Done: false},
				{NoteTitle: "Sprint 2", Line: 11, Text: "Deploy en producción", Done: false},
			},
		},
	}

	board := CollectKanban(notes, storage.DefaultColumns, []string{"To do", "In progress", "Done"})

	// #doing y #wip son el formato anterior: se leen como #kb/doing
	if n := len(board.ColumnCards(0)); n != 2 {
		t.Errorf("Se esperaban 2 tareas en Todo, obtenidas %d", n)
	}
	if n := len(board.ColumnCards(1)); n != 2 {
		t.Errorf("Se esperaban 2 tareas en Doing, obtenidas %d", n)
	}
	if n := len(board.ColumnCards(2)); n != 1 {
		t.Errorf("Se esperaba 1 tarea en Done, obtenida %d", n)
	}

	if board.TotalCards() != 5 {
		t.Errorf("Total de tarjetas esperado 5, obtenido %d", board.TotalCards())
	}

	// Probar renderizado y registro de zonas
	ht := mouse.NewHitTester()
	selectedRows := []int{0, 1, 0}
	rendered := RenderKanban(board, 1, selectedRows, 90, 20, ht, 1, KanbanDrag{})

	if !strings.Contains(rendered, "[1] Por Hacer") && !strings.Contains(rendered, "[1] To Do") {
		t.Errorf("No se encontró cabecera de columna [1]")
	}
	if !strings.Contains(rendered, "[2] En Progreso") && !strings.Contains(rendered, "[2] In Progress") {
		t.Errorf("No se encontró cabecera de columna [2]")
	}
	if !strings.Contains(rendered, "[3] Completado") && !strings.Contains(rendered, "[3] Done") {
		t.Errorf("No se encontró cabecera de columna [3]")
	}

	// Verificar registro de zonas en HitTester
	// Al hacer clic en columna 0 (x=5, y=5) debe detectar col 0 o tarjeta
	zone, found := ht.Check(5, 5)
	if !found {
		t.Errorf("No se detectó zona para clic en kanban (x=5, y=5)")
	} else if zone.Type != mouse.ZoneKanbanCol && zone.Type != mouse.ZoneKanbanCard {
		t.Errorf("Tipo de zona inesperado: %v", zone.Type)
	}
}
