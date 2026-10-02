package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

func TestMCPServerLifecycleAndTools(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-mcp-test-*")
	if err != nil {
		t.Fatalf("Fallo al crear temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	st := storage.New(tempDir)
	note, err := st.CreateNote("Nota Para MCP")
	if err != nil {
		t.Fatalf("Error al crear nota: %v", err)
	}

	content := "# Nota Para MCP\n\n- [ ] Tarea Todo\n- [ ] Tarea Doing #doing\n- [x] Tarea Done\n"
	if err := os.WriteFile(note.Path, []byte(content), 0644); err != nil {
		t.Fatalf("Error al escribir nota: %v", err)
	}

	server := NewServer(tempDir)

	// Helper para ejecutar una petición JSON-RPC y obtener la respuesta decodificada
	sendRequest := func(method string, id interface{}, params interface{}) JSONRPCResponse {
		var req JSONRPCRequest
		req.JSONRPC = "2.0"
		req.ID = id
		req.Method = method
		if params != nil {
			pBytes, _ := json.Marshal(params)
			req.Params = pBytes
		}

		inBytes, _ := json.Marshal(req)
		inBuf := bytes.NewBuffer(append(inBytes, '\n'))
		var outBuf bytes.Buffer

		if err := server.Serve(inBuf, &outBuf); err != nil {
			t.Fatalf("Error al servir request %s: %v", method, err)
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
			t.Fatalf("Error al decodificar respuesta JSON-RPC: %v, raw: %s", err, outBuf.String())
		}
		return resp
	}

	// 1. Probar `initialize`
	initResp := sendRequest("initialize", 1, map[string]interface{}{})
	if initResp.Error != nil {
		t.Fatalf("Error en initialize: %+v", initResp.Error)
	}
	resultMap, ok := initResp.Result.(map[string]interface{})
	if !ok || resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("Respuesta de initialize inválida: %+v", initResp.Result)
	}

	// 2. Probar `tools/list`
	toolsResp := sendRequest("tools/list", 2, nil)
	if toolsResp.Error != nil {
		t.Fatalf("Error en tools/list: %+v", toolsResp.Error)
	}
	toolsMap := toolsResp.Result.(map[string]interface{})
	toolsList := toolsMap["tools"].([]interface{})
	if len(toolsList) != 5 {
		t.Errorf("Se esperaban 5 herramientas MCP, obtenidas %d", len(toolsList))
	}

	// 3. Probar tool `list_notes`
	callListNotes := sendRequest("tools/call", 3, map[string]interface{}{
		"name":      "list_notes",
		"arguments": map[string]interface{}{},
	})
	if callListNotes.Error != nil {
		t.Fatalf("Error en call list_notes: %+v", callListNotes.Error)
	}

	// 4. Probar tool `read_note`
	callReadNote := sendRequest("tools/call", 4, map[string]interface{}{
		"name": "read_note",
		"arguments": map[string]interface{}{
			"path": note.Path,
		},
	})
	if callReadNote.Error != nil {
		t.Fatalf("Error en call read_note: %+v", callReadNote.Error)
	}

	// 5. Probar tool `list_tasks`
	callListTasks := sendRequest("tools/call", 5, map[string]interface{}{
		"name": "list_tasks",
		"arguments": map[string]interface{}{
			"pending_only": true,
		},
	})
	if callListTasks.Error != nil {
		t.Fatalf("Error en call list_tasks: %+v", callListTasks.Error)
	}

	// 6. Probar tool `get_kanban`
	callKanban := sendRequest("tools/call", 6, map[string]interface{}{
		"name":      "get_kanban",
		"arguments": map[string]interface{}{},
	})
	if callKanban.Error != nil {
		t.Fatalf("Error en call get_kanban: %+v", callKanban.Error)
	}
	callResult := callKanban.Result.(map[string]interface{})
	contents := callResult["content"].([]interface{})
	text := contents[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "Tarea Todo") || !strings.Contains(text, "Tarea Doing") || !strings.Contains(text, "Tarea Done") {
		t.Errorf("Salida de get_kanban incompleta: %s", text)
	}

	// 7. Probar tool `toggle_task`
	callToggle := sendRequest("tools/call", 7, map[string]interface{}{
		"name": "toggle_task",
		"arguments": map[string]interface{}{
			"path": note.Path,
			"line": 3,
		},
	})
	if callToggle.Error != nil {
		t.Fatalf("Error en call toggle_task: %+v", callToggle.Error)
	}
}

// TestMCPStaysInsideNotes (seguridad): read_note y toggle_task rechazan archivos de fuera de la carpeta de
// notas (ruta absoluta, "..") con IsError y sin tocarlos. Antes leían y escribían cualquier archivo.
func TestMCPStaysInsideNotes(t *testing.T) {
	root := t.TempDir()
	notes := root + "/notas"
	os.MkdirAll(notes, 0o755)
	secret := root + "/secreto.md"
	os.WriteFile(secret, []byte("clave\n- [ ] fuera\n"), 0o644)
	os.WriteFile(notes+"/a.md", []byte("# A\n- [ ] dentro\n"), 0o644)
	server := NewServer(notes)
	for _, p := range []string{secret, notes + "/../secreto.md", "/etc/hosts"} {
		r := server.callTool("read_note", map[string]interface{}{"path": p})
		if !r.IsError || strings.Contains(r.Content[0].Text, "clave") {
			t.Errorf("read_note %q debía rechazarse: %+v", p, r)
		}
		r = server.callTool("toggle_task", map[string]interface{}{"path": p, "line": float64(2)})
		if !r.IsError {
			t.Errorf("toggle_task %q debía rechazarse: %+v", p, r)
		}
	}
	if b, _ := os.ReadFile(secret); string(b) != "clave\n- [ ] fuera\n" {
		t.Errorf("se modificó un archivo de fuera: %q", b)
	}
	if r := server.callTool("read_note", map[string]interface{}{"path": notes + "/a.md"}); r.IsError {
		t.Errorf("read_note dentro: %+v", r)
	}
}
