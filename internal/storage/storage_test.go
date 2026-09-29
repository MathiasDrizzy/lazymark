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
