package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// session ejecuta UNA sesión stdio contra el servidor: escribe todas las líneas JSON-RPC por stdin, como lo
// hace un cliente MCP, y devuelve las respuestas por id.
func session(t *testing.T, dir string, reqs ...map[string]interface{}) map[float64]JSONRPCResponse {
	t.Helper()
	var in bytes.Buffer
	for _, r := range reqs {
		r["jsonrpc"] = "2.0"
		b, _ := json.Marshal(r)
		in.Write(b)
		in.WriteByte('\n')
	}
	var out bytes.Buffer
	if err := NewServer(dir).Serve(&in, &out); err != nil {
		t.Fatal(err)
	}
	resps := map[float64]JSONRPCResponse{}
	dec := json.NewDecoder(&out)
	for dec.More() {
		var r JSONRPCResponse
		if err := dec.Decode(&r); err != nil {
			t.Fatalf("respuesta inválida: %v", err)
		}
		id, _ := r.ID.(float64)
		resps[id] = r
	}
	return resps
}

func call(id int, tool string, args map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"id": id, "method": "tools/call", "params": map[string]interface{}{"name": tool, "arguments": args}}
}

// toolText devuelve el texto de una respuesta de herramienta y si fue un error.
func toolText(t *testing.T, r JSONRPCResponse) (string, bool) {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("error JSON-RPC: %+v", r.Error)
	}
	b, _ := json.Marshal(r.Result)
	var res CallToolResult
	if err := json.Unmarshal(b, &res); err != nil || len(res.Content) == 0 {
		t.Fatalf("resultado inválido: %s", b)
	}
	return res.Content[0].Text, res.IsError
}

// TestMCPSession (MC1): una sesión completa por stdio: initialize, tools/list y una llamada a cada herramienta.
func TestMCPSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	notes := filepath.Join(dir, "notas")
	os.MkdirAll(notes, 0o755)
	proyecto := filepath.Join(notes, "proyecto.md")
	os.WriteFile(proyecto, []byte("# Proyecto\n\n- [ ] Tarea Todo\n- [ ] Tarea Doing #kb/doing\n- [x] Tarea Done\n"), 0o644)
	secreto := filepath.Join(dir, "secreto.md")
	os.WriteFile(secreto, []byte("clave\n"), 0o644)

	r := session(t, notes,
		map[string]interface{}{"id": 1, "method": "initialize", "params": map[string]interface{}{}},
		map[string]interface{}{"method": "notifications/initialized"}, // sin id: no tiene respuesta
		map[string]interface{}{"id": 2, "method": "tools/list"},
		call(3, "list_notes", nil),
		call(4, "read_note", map[string]interface{}{"path": "proyecto.md"}),
		call(5, "list_tasks", map[string]interface{}{"pending_only": true}),
		call(6, "get_kanban", nil),
		call(7, "create_note", map[string]interface{}{"title": "Desde MCP", "empty": true}),
	)
	if len(r) != 7 {
		t.Fatalf("7 respuestas (la notificación no tiene), hay %d", len(r))
	}

	// initialize
	init, _ := json.Marshal(r[1].Result)
	if !strings.Contains(string(init), `"protocolVersion":"2024-11-05"`) || !strings.Contains(string(init), `"tools"`) {
		t.Errorf("initialize: %s", init)
	}

	// tools/list: las 7 herramientas, cada una con descripción y esquema de entrada
	lst, _ := json.Marshal(r[2].Result)
	var tl struct {
		Tools []struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			InputSchema map[string]interface{} `json:"inputSchema"`
		} `json:"tools"`
	}
	json.Unmarshal(lst, &tl)
	var names []string
	for _, tool := range tl.Tools {
		names = append(names, tool.Name)
		if tool.Description == "" || tool.InputSchema["type"] != "object" {
			t.Errorf("%s: falta la descripción o el esquema", tool.Name)
		}
	}
	if got := strings.Join(names, ","); got != "list_notes,read_note,create_note,list_tasks,move_task,toggle_task,get_kanban" {
		t.Errorf("herramientas: %s", got)
	}

	if txt, isErr := toolText(t, r[3]); isErr || !strings.Contains(txt, `"id": "proyecto.md"`) || !strings.Contains(txt, `"tasks_count": 3`) {
		t.Errorf("list_notes: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[4]); isErr || !strings.HasPrefix(txt, "# Proyecto") {
		t.Errorf("read_note: %v %s", isErr, txt)
	}
	txt, isErr := toolText(t, r[5])
	var pending []struct {
		ID, Text, Column string
		Done             bool
	}
	if err := json.Unmarshal([]byte(txt), &pending); err != nil || isErr || len(pending) != 2 {
		t.Fatalf("list_tasks pending: %v %v %s", err, isErr, txt)
	}
	if pending[1].Column != "doing" || pending[0].Column != "todo" || pending[1].Text != "Tarea Doing" {
		t.Errorf("columnas/texto limpio: %+v", pending)
	}
	if txt, isErr := toolText(t, r[6]); isErr || !strings.Contains(txt, `"id": "todo"`) || !strings.Contains(txt, `"id": "done"`) || !strings.Contains(txt, "Tarea Done") {
		t.Errorf("get_kanban: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[7]); isErr || !strings.Contains(txt, "desde-mcp.md") {
		t.Errorf("create_note: %v %s", isErr, txt)
	}
	if b, _ := os.ReadFile(filepath.Join(notes, "desde-mcp.md")); string(b) != "# Desde MCP\n" {
		t.Errorf("create_note empty escribió %q", b)
	}

	// otra sesión: mover y marcar por id, y los errores
	todoID := pending[0].ID
	r = session(t, notes,
		call(1, "move_task", map[string]interface{}{"id": todoID, "column": "doing"}),
		call(2, "toggle_task", map[string]interface{}{"id": todoID}),
		call(3, "toggle_task", map[string]interface{}{"path": "proyecto.md", "line": 4}), // forma anterior
		call(4, "move_task", map[string]interface{}{"id": todoID, "column": "cancelada"}),
		call(5, "move_task", map[string]interface{}{"id": "proyecto.md#00000000", "column": "doing"}),
		call(6, "read_note", map[string]interface{}{"path": secreto}),
		call(7, "read_note", map[string]interface{}{"path": "../secreto.md"}),
		call(8, "create_note", map[string]interface{}{"title": "X", "folder": ".."}),
		call(9, "move_task", map[string]interface{}{"id": todoID}),
		call(10, "borrar_todo", nil),
	)
	if txt, isErr := toolText(t, r[1]); isErr || !strings.Contains(txt, `"column": "doing"`) {
		t.Errorf("move_task: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[2]); isErr || !strings.Contains(txt, `"column": "done"`) || !strings.Contains(txt, `"done": true`) {
		t.Errorf("toggle_task por id: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[3]); isErr || !strings.Contains(txt, `"column": "done"`) {
		t.Errorf("toggle_task por path+line: %v %s", isErr, txt)
	}
	for id, want := range map[float64]string{4: "código 2", 5: "código 3", 6: "código 2", 7: "código 2", 8: "código 2"} {
		if txt, isErr := toolText(t, r[id]); !isErr || !strings.Contains(txt, want) {
			t.Errorf("respuesta %v debía ser un error con %q: %v %s", id, want, isErr, txt)
		}
	}
	if _, isErr := toolText(t, r[9]); !isErr {
		t.Error("move_task sin column debía ser un error")
	}
	if _, isErr := toolText(t, r[10]); !isErr {
		t.Error("una herramienta inexistente debía ser un error")
	}
	if b, _ := os.ReadFile(secreto); string(b) != "clave\n" {
		t.Errorf("se tocó un archivo de fuera: %q", b)
	}
	if b, _ := os.ReadFile(proyecto); !strings.Contains(string(b), "- [x] Tarea Todo\n") || !strings.Contains(string(b), "- [x] Tarea Doing\n") {
		t.Errorf("proyecto.md tras las llamadas:\n%s", b)
	}
}

// TestMCPProtocolErrors: JSON inválido y método desconocido dan los errores estándar de JSON-RPC.
func TestMCPProtocolErrors(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("{no es json\n" + `{"jsonrpc":"2.0","id":9,"method":"nada"}` + "\n")
	if err := NewServer(t.TempDir()).Serve(in, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "-32700") || !strings.Contains(out.String(), "-32601") {
		t.Errorf("respuestas: %s", out.String())
	}
}
