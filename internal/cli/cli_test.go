package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

func TestCLINoteCommands(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-cli-note-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	st := storage.New(tempDir)
	note, err := st.CreateNote("Nota Para CLI")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}

	// 1. Probar `note list` en texto plano
	var outBuf bytes.Buffer
	err = RunNoteWithWriter(&outBuf, []string{"list", "--dir", tempDir}, tempDir)
	if err != nil {
		t.Fatalf("Error en note list: %v", err)
	}
	if !strings.Contains(strings.ToLower(outBuf.String()), "nota para cli") {
		t.Errorf("Se esperaba encontrar 'nota para cli' en salida: %s", outBuf.String())
	}

	// 2. Probar `note list --json`
	var jsonBuf bytes.Buffer
	err = RunNoteWithWriter(&jsonBuf, []string{"list", "--json", "--dir", tempDir}, tempDir)
	if err != nil {
		t.Fatalf("Error en note list --json: %v", err)
	}
	var notesJSON []NoteJSON
	if err := json.Unmarshal(jsonBuf.Bytes(), &notesJSON); err != nil {
		t.Fatalf("Error al decodificar JSON de notas: %v, salida: %s", err, jsonBuf.String())
	}
	if len(notesJSON) != 1 || !strings.EqualFold(notesJSON[0].Title, "nota para cli") {
		t.Errorf("Salida JSON incorrecta: %+v", notesJSON)
	}

	// 3. Probar `note get <path>`
	var getBuf bytes.Buffer
	err = RunNoteWithWriter(&getBuf, []string{"get", note.Path}, tempDir)
	if err != nil {
		t.Fatalf("Error en note get: %v", err)
	}
	if !strings.Contains(getBuf.String(), "# Nota Para CLI") {
		t.Errorf("Contenido de nota incorrecto: %s", getBuf.String())
	}
}

func TestCLITaskCommands(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-cli-task-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	st := storage.New(tempDir)
	note, err := st.CreateNote("Nota Tareas CLI")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}

	// Agregar tareas variadas
	content := "# Nota Tareas CLI\n\n- [ ] Tarea Uno\n- [ ] Tarea Dos #doing\n- [x] Tarea Tres\n"
	if err := os.WriteFile(note.Path, []byte(content), 0644); err != nil {
		t.Fatalf("Error al escribir nota: %v", err)
	}

	// 1. Probar `task list` en texto plano
	var listBuf bytes.Buffer
	err = RunTaskWithWriter(&listBuf, []string{"list", "--dir", tempDir}, tempDir)
	if err != nil {
		t.Fatalf("Error en task list: %v", err)
	}
	if !strings.Contains(listBuf.String(), "Tarea Uno") || !strings.Contains(listBuf.String(), "Tarea Tres") {
		t.Errorf("Salida de task list incompleta: %s", listBuf.String())
	}

	// 2. Probar `task list --json --pending`
	var jsonBuf bytes.Buffer
	err = RunTaskWithWriter(&jsonBuf, []string{"list", "--json", "--pending", "--dir", tempDir}, tempDir)
	if err != nil {
		t.Fatalf("Error en task list --json --pending: %v", err)
	}
	var tasksJSON []TaskJSON
	if err := json.Unmarshal(jsonBuf.Bytes(), &tasksJSON); err != nil {
		t.Fatalf("Error al decodificar JSON de tareas: %v", err)
	}
	if len(tasksJSON) != 2 {
		t.Errorf("Se esperaban 2 tareas pendientes, obtenidas %d", len(tasksJSON))
	}

	// 3. Probar `task toggle --path <note.Path> --line 3`
	var toggleBuf bytes.Buffer
	err = RunTaskWithWriter(&toggleBuf, []string{"toggle", "--path", note.Path, "--line", "3", "--dir", tempDir}, tempDir)
	if err != nil {
		t.Fatalf("Error en task toggle: %v", err)
	}

	// Verificar que Tarea Uno (línea 3) ahora está completada
	notesReloaded, _ := st.ListNotes()
	foundToggled := false
	for _, n := range notesReloaded {
		for _, task := range n.Tasks {
			if task.Line == 3 && task.Done {
				foundToggled = true
			}
		}
	}
	if !foundToggled {
		t.Errorf("La tarea no persistió como completada tras toggle")
	}
}
