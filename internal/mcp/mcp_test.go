package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if !strings.Contains(string(init), `"protocolVersion":"2025-11-25"`) || !strings.Contains(string(init), `"tools"`) {
		t.Errorf("initialize: %s", init)
	}

	// tools/list: las 8 herramientas, cada una con descripción y esquema de entrada
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
	if got := strings.Join(names, ","); got != "list_notes,read_note,create_note,list_tasks,move_task,set_task_date,toggle_task,get_kanban" {
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
		call(11, "set_task_date", map[string]interface{}{"id": todoID, "field": "due", "date": "2026-01-02"}),
		call(12, "set_task_date", map[string]interface{}{"id": todoID, "field": "start", "date": "2026-02-30"}),
		call(13, "set_task_date", map[string]interface{}{"id": todoID, "field": "done", "date": "2026-01-02"}),
		call(14, "set_task_date", map[string]interface{}{"id": todoID, "field": "due", "date": "none"}),
		call(15, "set_task_date", map[string]interface{}{"id": todoID, "field": "due"}),
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
	if txt, isErr := toolText(t, r[11]); isErr || !strings.Contains(txt, `"due": "2026-01-02"`) || !strings.Contains(txt, `"overdue": false`) /* está hecha: no vence */ || !strings.Contains(txt, `"completed": "`) {
		t.Errorf("set_task_date due: %v %s", isErr, txt)
	}
	for id, want := range map[float64]string{12: "código 2", 13: "código 2"} {
		if txt, isErr := toolText(t, r[id]); !isErr || !strings.Contains(txt, want) {
			t.Errorf("respuesta %v debía ser un error con %q: %v %s", id, want, isErr, txt)
		}
	}
	if txt, isErr := toolText(t, r[14]); isErr || !strings.Contains(txt, `"due": ""`) {
		t.Errorf("quitar el vencimiento: %v %s", isErr, txt)
	}
	if _, isErr := toolText(t, r[15]); !isErr {
		t.Error("set_task_date sin date debía ser un error")
	}
	if b, _ := os.ReadFile(secreto); string(b) != "clave\n" {
		t.Errorf("se tocó un archivo de fuera: %q", b)
	}
	if b, _ := os.ReadFile(proyecto); !strings.Contains(string(b), "- [x] Tarea Todo ✅ ") || !strings.Contains(string(b), "- [x] Tarea Doing ✅ ") {
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

func modernMeta(version string) map[string]interface{} {
	return map[string]interface{}{
		metaVersionKey:                       version,
		metaCapsKey:                          map[string]interface{}{},
		"io.modelcontextprotocol/clientInfo": map[string]interface{}{"name": "test", "version": "1"},
	}
}

func modernReq(id int, method string, extra map[string]interface{}, meta map[string]interface{}) map[string]interface{} {
	params := map[string]interface{}{"_meta": meta}
	for k, v := range extra {
		params[k] = v
	}
	return map[string]interface{}{"id": id, "method": method, "params": params}
}

// TestMCPModernEra (C.0): la versión vigente (2026-07-28) no tiene sesión ni `initialize`: cada petición lleva su versión y sus
// capacidades en `_meta`, el servidor la atiende sin estado (resultType "complete" y serverInfo en `_meta`) o la rechaza con
// UnsupportedProtocolVersion (-32022, con las versiones que soporta) o Invalid params (-32602) si falta una capacidad. Una
// misma conexión puede mezclar las dos épocas.
func TestMCPModernEra(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	notes, _ := filepath.EvalSymlinks(t.TempDir())
	os.WriteFile(filepath.Join(notes, "a.md"), []byte("# A\n- [ ] tarea\n"), 0o644)

	noCaps := map[string]interface{}{metaVersionKey: modernVersion}
	r := session(t, notes,
		modernReq(1, "server/discover", nil, modernMeta(modernVersion)), // sin initialize previo
		modernReq(2, "tools/list", nil, modernMeta(modernVersion)),
		modernReq(3, "tools/call", map[string]interface{}{"name": "list_notes", "arguments": map[string]interface{}{}}, modernMeta(modernVersion)),
		modernReq(4, "tools/list", nil, modernMeta("1900-01-01")), // versión desconocida
		modernReq(5, "tools/list", nil, noCaps),                   // falta clientCapabilities
		modernReq(6, "tools/call", map[string]interface{}{"name": "nada"}, modernMeta(modernVersion)),
		modernReq(7, "ping", nil, modernMeta(modernVersion)),
		// la misma conexión, época anterior: initialize clásico y herramientas sin _meta
		map[string]interface{}{"id": 8, "method": "initialize", "params": map[string]interface{}{"protocolVersion": "2025-06-18"}},
		map[string]interface{}{"id": 9, "method": "initialize", "params": map[string]interface{}{"protocolVersion": "1999-01-01"}},
		map[string]interface{}{"id": 10, "method": "tools/list"},
		modernReq(11, "server/discover", nil, modernMeta("2025-11-25")), // pide discover con una versión de la época anterior: se contesta
	)
	disc, _ := json.Marshal(r[1].Result)
	for _, want := range []string{`"resultType":"complete"`, `"supportedVersions":["2026-07-28","2025-11-25","2025-06-18","2025-03-26","2024-11-05"]`, `"tools":{}`, metaServerInfo, `"ttlMs"`} {
		if !strings.Contains(string(disc), want) {
			t.Errorf("server/discover: falta %s en %s", want, disc)
		}
	}
	list, _ := json.Marshal(r[2].Result)
	if !strings.Contains(string(list), `"resultType":"complete"`) || !strings.Contains(string(list), `"name":"list_notes"`) || !strings.Contains(string(list), metaServerInfo) {
		t.Errorf("tools/list moderno: %s", list)
	}
	if txt, isErr := toolText(t, r[3]); isErr || !strings.Contains(txt, `"id": "a.md"`) {
		t.Errorf("tools/call moderno: %v %s", isErr, txt)
	}
	if b, _ := json.Marshal(r[3].Result); !strings.Contains(string(b), `"resultType":"complete"`) {
		t.Errorf("tools/call moderno sin resultType: %s", b)
	}
	if e := r[4].Error; e == nil || e.Code != -32022 || !strings.Contains(fmt.Sprint(e.Data), "2026-07-28") || !strings.Contains(fmt.Sprint(e.Data), "1900-01-01") {
		t.Errorf("versión desconocida: %+v", e)
	}
	if e := r[5].Error; e == nil || e.Code != -32602 {
		t.Errorf("sin clientCapabilities: %+v", e)
	}
	if txt, isErr := toolText(t, r[6]); !isErr || !strings.Contains(txt, "no encontrada") {
		t.Errorf("herramienta inexistente: %v %s", isErr, txt)
	}
	if r[7].Error != nil {
		t.Errorf("ping: %+v", r[7].Error)
	}
	for id, want := range map[float64]string{8: `"protocolVersion":"2025-06-18"`, 9: `"protocolVersion":"2025-11-25"`} {
		if b, _ := json.Marshal(r[id].Result); !strings.Contains(string(b), want) {
			t.Errorf("initialize %v: %s", id, b)
		}
	}
	if b, _ := json.Marshal(r[10].Result); strings.Contains(string(b), "resultType") || !strings.Contains(string(b), `"tools"`) {
		t.Errorf("tools/list de la época anterior no lleva resultType: %s", b)
	}
	if r[11].Error != nil {
		t.Errorf("discover con versión anterior: %+v", r[11].Error)
	}
}
