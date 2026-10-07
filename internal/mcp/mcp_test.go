package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	// tools/list: las 9 herramientas, cada una con descripción y esquema de entrada
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
	if got := strings.Join(names, ","); got != "list_notes,read_note,create_note,search_notes,list_tasks,move_task,set_task_date,toggle_task,get_kanban" {
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
		call(16, "search_notes", map[string]interface{}{"query": "TAREA TODO"}),
		call(17, "search_notes", map[string]interface{}{"query": "tarea (todo|doing)", "regex": true, "limit": 1}),
		call(18, "search_notes", map[string]interface{}{"query": "(", "regex": true}),
		call(19, "search_notes", map[string]interface{}{"query": "tarea todo", "case_sensitive": true}),
		call(20, "search_notes", map[string]interface{}{}),
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
	for id, want := range map[float64]string{4: "code 2", 5: "code 3", 6: "code 2", 7: "code 2", 8: "code 2"} {
		if txt, isErr := toolText(t, r[id]); !isErr || !strings.Contains(txt, want) {
			t.Errorf("respuesta %v debía ser un error con %q: %v %s", id, want, isErr, txt)
		}
	}
	if _, isErr := toolText(t, r[9]); !isErr {
		t.Error("move_task sin column debía ser un error")
	}
	if e := r[10].Error; e == nil || e.Code != -32602 || !strings.Contains(e.Message, "Unknown tool: borrar_todo") {
		t.Errorf("una herramienta inexistente es un error JSON-RPC -32602, no un resultado con isError: %+v %+v", e, r[10].Result)
	}
	if txt, isErr := toolText(t, r[11]); isErr || !strings.Contains(txt, `"due": "2026-01-02"`) || !strings.Contains(txt, `"overdue": false`) /* está hecha: no vence */ || !strings.Contains(txt, `"completed": "`) {
		t.Errorf("set_task_date due: %v %s", isErr, txt)
	}
	for id, want := range map[float64]string{12: "code 2", 13: "code 2"} {
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
	if txt, isErr := toolText(t, r[16]); isErr || !strings.Contains(txt, `"note": "proyecto.md"`) || !strings.Contains(txt, `"line": 3`) || !strings.Contains(txt, `"truncated": false`) {
		t.Errorf("search_notes: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[17]); isErr || !strings.Contains(txt, `"truncated": true`) {
		t.Errorf("search_notes con regex y límite 1 debía truncar: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[18]); !isErr || !strings.Contains(txt, "code 2") {
		t.Errorf("search_notes con regex inválida: %v %s", isErr, txt)
	}
	if txt, isErr := toolText(t, r[19]); isErr || !strings.Contains(txt, `"matches": []`) {
		t.Errorf("search_notes distinguiendo mayúsculas no halla 'tarea todo': %v %s", isErr, txt)
	}
	if _, isErr := toolText(t, r[20]); !isErr {
		t.Error("search_notes sin query debía ser un error")
	}
	if b, _ := os.ReadFile(secreto); string(b) != "clave\n" {
		t.Errorf("se tocó un archivo de fuera: %q", b)
	}
	if b, _ := os.ReadFile(proyecto); !strings.Contains(string(b), "- [x] Tarea Todo [completion:: ") || !strings.Contains(string(b), "- [x] Tarea Doing [completion:: ") {
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
		modernReq(7, "ping", nil, modernMeta(modernVersion)), // ping se quitó en 2026-07-28
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
	if !strings.Contains(string(list), `"resultType":"complete"`) || !strings.Contains(string(list), `"name":"list_notes"`) || !strings.Contains(string(list), metaServerInfo) ||
		!strings.Contains(string(list), `"ttlMs":3600000`) || !strings.Contains(string(list), `"cacheScope":"public"`) {
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
	if e := r[6].Error; e == nil || e.Code != -32602 || !strings.Contains(e.Message, "Unknown tool: nada") {
		t.Errorf("herramienta inexistente (spec: error JSON-RPC -32602): %+v", e)
	}
	if e := r[7].Error; e == nil || e.Code != -32601 {
		t.Errorf("ping ya no existe en 2026-07-28: %+v", r[7].Error)
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

// TestMCPMalformedRequests (C.0): lo mal formado se rechaza con el error que manda la spec, sin mezclar épocas ni dejar al cliente
// esperando: id null o no escalar → -32600 con id null; `_meta` moderno incompleto o con tipos erróneos → -32602; JSON roto → -32700.
func TestMCPMalformedRequests(t *testing.T) {
	notes := t.TempDir()
	lines := []string{
		`{"jsonrpc":"2.0","id":null,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":{"a":1},"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":[1],"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{}}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":2026,"io.modelcontextprotocol/clientCapabilities":{}}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list","params":null}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{"_meta":{"progressToken":"x"}}}`, // época anterior con progressToken: vale
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,                                  // notificación: sin respuesta
		`{roto`,
	}
	var out bytes.Buffer
	if err := NewServer(notes).Serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var got []JSONRPCResponse
	dec := json.NewDecoder(&out)
	for dec.More() {
		var r JSONRPCResponse
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if len(got) != 8 {
		t.Fatalf("%d respuestas, se esperaban 8 (la notificación no tiene): %+v", len(got), got)
	}
	codes := []int{-32600, -32600, -32600, -32602, -32602, 0, 0, -32700}
	for i, want := range codes {
		switch {
		case want == 0 && got[i].Error != nil:
			t.Errorf("respuesta %d: no debía ser un error: %+v", i, got[i].Error)
		case want != 0 && (got[i].Error == nil || got[i].Error.Code != want):
			t.Errorf("respuesta %d: se esperaba %d: %+v", i, want, got[i].Error)
		}
	}
	for _, i := range []int{0, 1, 2, 7} { // sin id usable: "id": null
		if got[i].ID != nil {
			t.Errorf("respuesta %d: el id debía ser null: %v", i, got[i].ID)
		}
	}
}

// TestMCPHugeLineKeepsServing (ORD-015 C.5 S5): una línea de más de 1 MiB no cierra el servidor: se contesta con un error JSON-RPC y la
// petición siguiente se atiende normalmente.
func TestMCPHugeLineKeepsServing(t *testing.T) {
	dir := t.TempDir()
	var in bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"pad":"` + strings.Repeat("x", 2<<20) + `"}}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	var out bytes.Buffer
	if err := NewServer(dir).Serve(&in, &out); err != nil {
		t.Fatalf("el servidor no debe terminar con error: %v", err)
	}
	var got []JSONRPCResponse
	dec := json.NewDecoder(&out)
	for dec.More() {
		var r JSONRPCResponse
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if len(got) != 2 {
		t.Fatalf("se esperaban 2 respuestas (error + tools/list), llegaron %d", len(got))
	}
	if got[0].Error == nil || got[0].Error.Code != -32600 || !strings.Contains(got[0].Error.Message, "1 MiB") {
		t.Errorf("la línea enorme da un error de petición inválida: %+v", got[0].Error)
	}
	if id, _ := got[1].ID.(float64); id != 2 || got[1].Error != nil {
		t.Errorf("la siguiente petición se atiende: %+v", got[1])
	}
}

// TestMCPArgumentTypes (ORD-015 C.5 S5): un argumento de tipo equivocado es un error de herramienta (isError) con el nombre del argumento, no un
// valor silenciosamente ignorado.
func TestMCPArgumentTypes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a\n- [ ] t\n"), 0o644)
	for i, c := range []struct {
		tool string
		args map[string]interface{}
		want string
	}{
		{"read_note", map[string]interface{}{"path": 123}, "path"},
		{"create_note", map[string]interface{}{"title": "x", "empty": "si"}, "empty"},
		{"search_notes", map[string]interface{}{"query": "a", "limit": "10"}, "limit"},
		{"search_notes", map[string]interface{}{"query": "a", "limit": 1.5}, "limit"},
		{"search_notes", map[string]interface{}{"query": "a", "regex": 1}, "regex"},
		{"list_tasks", map[string]interface{}{"pending_only": "true"}, "pending_only"},
		{"toggle_task", map[string]interface{}{"path": "a.md", "line": "2"}, "line"},
	} {
		r := session(t, dir, call(i+1, c.tool, c.args))[float64(i+1)]
		text, isErr := toolText(t, r)
		if !isErr || !strings.Contains(text, c.want) {
			t.Errorf("%s %v: se esperaba un error que nombre %q: %q (isError=%v)", c.tool, c.args, c.want, text, isErr)
		}
	}
	// los tipos correctos siguen funcionando
	r := session(t, dir, call(1, "search_notes", map[string]interface{}{"query": "a", "limit": 3, "regex": false}))[1]
	if _, isErr := toolText(t, r); isErr {
		t.Error("argumentos válidos no deben dar error")
	}
}

// TestMCPNullOptionalArguments (ORD-015 segunda opinión): un argumento opcional con valor null es "no enviado", no un error de tipo.
func TestMCPNullOptionalArguments(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a\n- [ ] t\n"), 0o644)
	r := session(t, dir, call(1, "search_notes", map[string]interface{}{"query": "a", "limit": nil, "regex": nil, "case_sensitive": nil}))[1]
	if text, isErr := toolText(t, r); isErr {
		t.Errorf("null en opcionales no es un error: %q", text)
	}
}

// TestReadLineLimitIsOnThePayload (ORD-015 segunda opinión): una línea de exactamente 1 MiB de contenido se acepta con \n o con \r\n; con un byte más, no.
func TestReadLineLimitIsOnThePayload(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		for _, c := range []struct {
			n       int
			tooLong bool
		}{{maxLineBytes, false}, {maxLineBytes + 1, true}} {
			r := bufio.NewReaderSize(strings.NewReader(strings.Repeat("x", c.n)+nl+"{}"+nl), 64*1024)
			line, tooLong, _ := readLine(r, maxLineBytes)
			if tooLong != c.tooLong || (!tooLong && len(line) != c.n) {
				t.Errorf("%q de %d bytes: tooLong=%v (se esperaba %v), len=%d", nl, c.n, tooLong, c.tooLong, len(line))
			}
			if next, _, _ := readLine(r, maxLineBytes); string(next) != "{}" {
				t.Errorf("la línea siguiente debe leerse bien: %q", next)
			}
		}
	}
}

// errAfter entrega unos datos y después falla con un error de E/S (no EOF): un stdin roto.
type errAfter struct {
	data []byte
	err  error
}

func (r *errAfter) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

// TestMCPStdinIOError (ORD-016 L2): un error de E/S en stdin (no un EOF) termina el servidor limpiamente con ese error, sin bucle: lo que se leyó antes
// se contesta, y con el error a mitad de una línea esa línea parcial se descarta o se atiende una sola vez.
func TestMCPStdinIOError(t *testing.T) {
	boom := errors.New("boom de E/S")
	for name, data := range map[string]string{
		"error tras una petición completa": `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n",
		"error a mitad de una línea":       `{"jsonrpc":"2.0","id":1,"meth`,
		"error sin datos":                  "",
	} {
		var out bytes.Buffer
		done := make(chan error, 1)
		go func() { done <- NewServer(t.TempDir()).Serve(&errAfter{data: []byte(data), err: boom}, &out) }()
		select {
		case err := <-done:
			if !errors.Is(err, boom) {
				t.Errorf("%s: Serve debe devolver el error de E/S: %v", name, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: el servidor no termina tras un error de E/S (bucle)", name)
		}
		if n := strings.Count(out.String(), "\n"); n > 2 {
			t.Errorf("%s: demasiadas respuestas (%d): %q", name, n, out.String())
		}
	}
}

// TestNoteIDIsRelativePathMCP (ORD-018 L13): el id de una nota es su ruta relativa a la carpeta de notas (como el campo `note` de las tareas): dos notas homónimas en
// carpetas distintas tienen ids distintos, y create_note en una subcarpeta devuelve un id con el que read_note lee esa nota.
func TestNoteIDIsRelativePathMCP(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	notes, _ := filepath.EvalSymlinks(t.TempDir())
	for _, f := range []string{"a/igual.md", "b/igual.md"} {
		os.MkdirAll(filepath.Join(notes, filepath.Dir(f)), 0o755)
		os.WriteFile(filepath.Join(notes, f), []byte("# Igual "+f+"\n"), 0o644)
	}
	os.MkdirAll(filepath.Join(notes, "trabajo"), 0o755)
	r := session(t, notes, call(1, "create_note", map[string]interface{}{"title": "Lanzamiento", "folder": "trabajo", "empty": true}), call(2, "list_notes", nil))
	txt, isErr := toolText(t, r[1])
	var created struct {
		ID string `json:"id"`
	}
	if isErr || json.Unmarshal([]byte(txt), &created) != nil || created.ID != "trabajo/lanzamiento.md" {
		t.Fatalf("create_note debe dar el id con su carpeta: %v %s", isErr, txt)
	}
	txt, _ = toolText(t, r[2])
	var list []struct{ ID, Title string }
	if err := json.Unmarshal([]byte(txt), &list); err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, n := range list {
		if ids[n.ID] {
			t.Errorf("id repetido: %s", n.ID)
		}
		ids[n.ID] = true
	}
	for _, want := range []string{"a/igual.md", "b/igual.md", "trabajo/lanzamiento.md"} {
		if !ids[want] {
			t.Errorf("falta el id %q en %v", want, ids)
		}
	}
	var reads []map[string]interface{}
	for i, id := range []string{"a/igual.md", "b/igual.md", "trabajo/lanzamiento.md"} {
		reads = append(reads, call(10+i, "read_note", map[string]interface{}{"path": id}))
	}
	rr := session(t, notes, reads...)
	for i, want := range []string{"# Igual a/igual.md", "# Igual b/igual.md", "# Lanzamiento"} {
		txt, isErr := toolText(t, rr[float64(10+i)]) // read_note devuelve el contenido
		if isErr || !strings.Contains(txt, want) {
			t.Errorf("read_note(id) debe leer su nota (%q): %v %s", want, isErr, txt)
		}
	}
}

// TestMCPErrorsLanguage (ORD-019 C.7 / L14): el MCP responde en inglés por defecto (un agente no tiene LANG) y en el idioma de la configuración si hay uno.
func TestMCPErrorsLanguage(t *testing.T) {
	for _, c := range []struct {
		cfg, want, other string
	}{
		{"", "Error (code 3): no such task", "no existe"},
		{`{"language":"en"}`, "Error (code 3): no such task", "no existe"},
		{`{"language":"es"}`, "Error (código 3): no existe esa tarea", "no such task"},
	} {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("AppData", filepath.Join(home, "AppData"))
		t.Setenv("LANG", "es_ES.UTF-8") // el idioma del sistema no manda en el MCP
		if c.cfg != "" {
			base, _ := os.UserConfigDir()
			os.MkdirAll(filepath.Join(base, "lazymark"), 0o755)
			os.WriteFile(filepath.Join(base, "lazymark", "config.json"), []byte(c.cfg), 0o644)
		}
		notes := filepath.Join(home, "notas")
		os.MkdirAll(notes, 0o755)
		os.WriteFile(filepath.Join(notes, "a.md"), []byte("- [ ] x\n"), 0o644)
		r := session(t, notes, call(1, "toggle_task", map[string]interface{}{"id": "a.md#00000000"}))
		txt, isErr := toolText(t, r[1])
		if !isErr || !strings.Contains(txt, c.want) || strings.Contains(txt, c.other) {
			t.Errorf("config %q: se esperaba %q: %v %s", c.cfg, c.want, isErr, txt)
		}
	}
}

// TestToolsListLanguage (ORD-020): las descripciones de las herramientas y de sus parámetros (tools/list) siguen la misma regla que los errores: inglés por defecto
// (un agente no tiene LANG) y el idioma de la configuración si hay uno. Ninguna descripción queda en español en el modo por defecto.
func TestToolsListLanguage(t *testing.T) {
	list := func(cfg string) string {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("AppData", filepath.Join(home, "AppData"))
		t.Setenv("LANG", "es_ES.UTF-8")
		if cfg != "" {
			base, _ := os.UserConfigDir()
			os.MkdirAll(filepath.Join(base, "lazymark"), 0o755)
			os.WriteFile(filepath.Join(base, "lazymark", "config.json"), []byte(cfg), 0o644)
		}
		notes := filepath.Join(home, "notas")
		os.MkdirAll(notes, 0o755)
		r := session(t, notes, map[string]interface{}{"id": 1, "method": "tools/list"})
		b, _ := json.Marshal(r[1].Result)
		return string(b)
	}
	en := list("")
	for _, want := range []string{"Lists the Lazymark notes", "Reads the Markdown content", "Searches text in all the notes", "Marks a task as done"} {
		if !strings.Contains(en, want) {
			t.Errorf("sin idioma configurado debe estar en inglés: falta %q", want)
		}
	}
	for _, spanish := range []string{"Lista las notas", "Lee el contenido", "Busca texto", "Ruta de la nota", "Título de la nota", "Marca una tarea"} {
		if strings.Contains(en, spanish) {
			t.Errorf("queda texto en español en tools/list por defecto: %q", spanish)
		}
	}
	if es := list(`{"language":"es"}`); !strings.Contains(es, "Lista las notas de Lazymark") || strings.Contains(es, "Lists the Lazymark notes") {
		t.Error("con language=es las descripciones salen en español")
	}
}

// TestMCPEscapesC1 (ORD-026 N3): el JSON-RPC y los resultados en JSON del MCP escapan los controles C1 que vengan de una nota.
func TestMCPEscapesC1(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	notes := filepath.Join(home, "notas")
	os.MkdirAll(notes, 0o755)
	os.WriteFile(filepath.Join(notes, "e.md"), []byte("# E\u009d\n- [ ] t \u009d52;c;x\u009c fin\n"), 0o644)
	var in bytes.Buffer
	for _, r := range []map[string]interface{}{call(1, "list_tasks", nil), call(2, "read_note", map[string]interface{}{"path": "e.md"})} {
		r["jsonrpc"] = "2.0"
		b, _ := json.Marshal(r)
		in.Write(b)
		in.WriteByte('\n')
	}
	var out bytes.Buffer
	if err := NewServer(notes).Serve(&in, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\xc2\x9d") || strings.Contains(out.String(), "\xc2\x9c") {
		t.Errorf("la salida del MCP lleva un C1 crudo:\n%q", out.String())
	}
	if !strings.Contains(out.String(), `\u009d`) {
		t.Errorf("lleva el escape: %.300s", out.String())
	}
}
