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
