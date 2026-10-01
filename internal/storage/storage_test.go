package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-test-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := New(tempDir)

	// 1. Probar crear nota
	note, err := s.CreateNote("Mi Primera Nota")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}
	if note.Title != "Mi Primera Nota" {
		t.Errorf("Título incorrecto: esperado 'Mi Primera Nota', obtenido '%s'", note.Title)
	}

	// 2. Probar listar notas
	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("Error al listar notas: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("Se esperaba 1 nota, se obtuvieron %d", len(notes))
	}

	// 3. Probar extracción de tags y tareas
	if len(notes[0].Tags) == 0 || notes[0].Tags[0] != "general" {
		t.Errorf("Tag 'general' no detectado")
	}
	if len(notes[0].Tasks) != 1 {
		t.Errorf("Tarea pendiente no detectada")
	}

	// 4. Probar crear nota con imágenes
	contentWithImg := "# Nota con Imagen\n\n![Diagrama](assets/diagram.png)\n"
	imgPath := filepath.Join(tempDir, "nota-img.md")
	if err := os.WriteFile(imgPath, []byte(contentWithImg), 0644); err != nil {
		t.Fatalf("Error al escribir nota con imagen: %v", err)
	}

	notesAfterImg, err := s.ListNotes()
	if err != nil || len(notesAfterImg) != 2 {
		t.Fatalf("Fallo al listar 2 notas: %v", err)
	}

	foundImg := false
	for _, n := range notesAfterImg {
		if len(n.Images) > 0 && n.Images[0] == "assets/diagram.png" {
			foundImg = true
			break
		}
	}
	if !foundImg {
		t.Errorf("No se extrajo la referencia a assets/diagram.png")
	}

	// 5. Probar eliminar nota
	if err := s.DeleteNote(note.Path); err != nil {
		t.Fatalf("Fallo al borrar nota: %v", err)
	}
	remaining, _ := s.ListNotes()
	if len(remaining) != 1 {
		t.Errorf("Se esperaba 1 nota restante, hay %d", len(remaining))
	}
}

func TestTreeHierarchyAndFolderOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-tree-test-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := New(tempDir)

	// Crear carpeta docs y notas dentro
	folderPath, err := s.CreateFolderInDir(tempDir, "docs")
	if err != nil {
		t.Fatalf("Error al crear carpeta docs: %v", err)
	}

	_, err = s.CreateNoteInDir(folderPath, "DEVELOPMENT_PLAN")
	if err != nil {
		t.Fatalf("Error al crear nota en docs: %v", err)
	}
	_, err = s.CreateNoteInDir(folderPath, "STATUS_AND_TODO")
	if err != nil {
		t.Fatalf("Error al crear segunda nota en docs: %v", err)
	}

	// Crear nota en raíz
	_, err = s.CreateNoteInDir(tempDir, "Root Note")
	if err != nil {
		t.Fatalf("Error al crear nota en raíz: %v", err)
	}

	// 1. Probar conteo de elementos en carpeta
	count := s.CountFolderItems(folderPath)
	if count != 2 {
		t.Errorf("Se esperaban 2 elementos en 'docs', obtenidos %d", count)
	}

	// 2. Probar ListTreeEntries con docs expandida (por defecto)
	entriesExpanded, err := s.ListTreeEntries(nil)
	if err != nil {
		t.Fatalf("Error al listar árbol expandido: %v", err)
	}
	// Deben ser: docs (depth 0), 2 notas hijas (depth 1), 1 nota en raíz (depth 0) = 4 entries
	if len(entriesExpanded) != 4 {
		t.Fatalf("Se esperaban 4 entries en árbol expandido, obtenidos %d", len(entriesExpanded))
	}
	if entriesExpanded[0].Name != "docs" || entriesExpanded[0].Depth != 0 {
		t.Errorf("La primera entry debería ser la carpeta 'docs' con depth 0, obtenida: %+v", entriesExpanded[0])
	}
	if entriesExpanded[1].Depth != 1 || entriesExpanded[2].Depth != 1 {
		t.Errorf("Las notas hijas en docs deben tener depth 1")
	}

	// 3. Probar ListTreeEntries con docs colapsada
	expandedMap := map[string]bool{folderPath: false}
	entriesCollapsed, err := s.ListTreeEntries(expandedMap)
	if err != nil {
		t.Fatalf("Error al listar árbol colapsado: %v", err)
	}
	// Deben ser: docs (depth 0, colapsada) y Root Note (depth 0) = 2 entries
	if len(entriesCollapsed) != 2 {
		t.Fatalf("Se esperaban 2 entries en árbol colapsado, obtenidos %d", len(entriesCollapsed))
	}
}

func TestTrashLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-trash-test-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := New(tempDir)

	// Crear una nota
	note, err := s.CreateNote("Nota Para Papelera")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}

	// 1. Mover a papelera
	item, err := s.MoveToTrash(note.Path)
	if err != nil {
		t.Fatalf("Error al mover nota a papelera: %v", err)
	}
	if item.Name != "nota-para-papelera.md" {
		t.Errorf("Nombre esperado 'nota-para-papelera.md', obtenido '%s'", item.Name)
	}

	// Comprobar que ya no está en la ubicación original
	if _, err := os.Stat(note.Path); !os.IsNotExist(err) {
		t.Errorf("El archivo original aún existe después de moverlo a la papelera")
	}

	// Comprobar contador de papelera
	count := s.CountTrash()
	if count != 1 {
		t.Errorf("Se esperaba 1 elemento en papelera, obtenidos %d", count)
	}

	// 2. Restaurar elemento
	if err := s.RestoreTrashItem(item.ID); err != nil {
		t.Fatalf("Error al restaurar nota: %v", err)
	}
	if _, err := os.Stat(note.Path); err != nil {
		t.Errorf("El archivo restaurado no existe en su ubicación original")
	}

	// Comprobar que la papelera quedó vacía
	if s.CountTrash() != 0 {
		t.Errorf("La papelera debería estar vacía tras restaurar")
	}
}

func TestToggleTaskAtomic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-toggle-test-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := New(tempDir)
	_, err = s.CreateNote("Nota Con Tareas")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}

	notes, err := s.ListNotes()
	if err != nil || len(notes) == 0 || len(notes[0].Tasks) == 0 {
		t.Fatalf("No se encontraron notas o tareas iniciales")
	}

	task := notes[0].Tasks[0]
	if task.Done {
		t.Errorf("La tarea inicial debería estar pendiente")
	}

	// 1. Toggle a completada (Done = true)
	newStatus, err := s.ToggleTask(task.NotePath, task.Line)
	if err != nil {
		t.Fatalf("Error al alternar tarea a completada: %v", err)
	}
	if !newStatus {
		t.Errorf("Se esperaba que la tarea estuviera completada (true), obtenido false")
	}

	// Recargar y verificar
	notesReloaded, err := s.ListNotes()
	if err != nil || len(notesReloaded[0].Tasks) == 0 {
		t.Fatalf("Fallo al recargar notas tras toggle")
	}
	if !notesReloaded[0].Tasks[0].Done {
		t.Errorf("La tarea en disco debería persistir como completada")
	}

	// 2. Toggle de vuelta a pendiente (Done = false)
	newStatus2, err := s.ToggleTask(task.NotePath, task.Line)
	if err != nil {
		t.Fatalf("Error al alternar tarea a pendiente: %v", err)
	}
	if newStatus2 {
		t.Errorf("Se esperaba que la tarea estuviera pendiente (false), obtenido true")
	}

	notesReloaded2, err := s.ListNotes()
	if err != nil || notesReloaded2[0].Tasks[0].Done {
		t.Errorf("La tarea en disco debería persistir como pendiente")
	}
}

func TestUpdateTaskStage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-stage-test-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := New(tempDir)
	_, err = s.CreateNote("Kanban Stage Note")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}

	notes, err := s.ListNotes()
	if err != nil || len(notes) == 0 || len(notes[0].Tasks) == 0 {
		t.Fatalf("No se encontraron notas o tareas iniciales")
	}

	task := notes[0].Tasks[0]
	if GetTaskStage(task) != StageTodo {
		t.Fatalf("Se esperaba etapa inicial StageTodo, obtenido %v", GetTaskStage(task))
	}

	// 1. Mover a StageDoing
	if err := s.UpdateTaskStage(task.NotePath, task.Line, StageDoing); err != nil {
		t.Fatalf("Error al mover tarea a StageDoing: %v", err)
	}

	notesReloaded, _ := s.ListNotes()
	taskReloaded := notesReloaded[0].Tasks[0]
	if GetTaskStage(taskReloaded) != StageDoing {
		t.Errorf("Se esperaba etapa StageDoing, obtenido %v (texto: %s)", GetTaskStage(taskReloaded), taskReloaded.Text)
	}

	// 2. Mover a StageDone
	if err := s.UpdateTaskStage(task.NotePath, task.Line, StageDone); err != nil {
		t.Fatalf("Error al mover tarea a StageDone: %v", err)
	}

	notesReloaded2, _ := s.ListNotes()
	taskReloaded2 := notesReloaded2[0].Tasks[0]
	if GetTaskStage(taskReloaded2) != StageDone {
		t.Errorf("Se esperaba etapa StageDone, obtenido %v (texto: %s, done: %v)", GetTaskStage(taskReloaded2), taskReloaded2.Text, taskReloaded2.Done)
	}

	// 3. Mover de vuelta a StageTodo
	if err := s.UpdateTaskStage(task.NotePath, task.Line, StageTodo); err != nil {
		t.Fatalf("Error al mover tarea a StageTodo: %v", err)
	}

	notesReloaded3, _ := s.ListNotes()
	taskReloaded3 := notesReloaded3[0].Tasks[0]
	if GetTaskStage(taskReloaded3) != StageTodo {
		t.Errorf("Se esperaba etapa StageTodo, obtenido %v (texto: %s, done: %v)", GetTaskStage(taskReloaded3), taskReloaded3.Text, taskReloaded3.Done)
	}
}

