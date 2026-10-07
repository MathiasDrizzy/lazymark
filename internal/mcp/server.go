package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"io"
	"math"
	"os"
	"sort"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/ops"
)

// JSONRPCRequest representa una petición JSON-RPC 2.0
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse representa una respuesta JSON-RPC 2.0
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"` // sin omitempty: un error sobre una petición ilegible lleva "id": null
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError define un error en JSON-RPC 2.0
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolContent representa un bloque de contenido devuelto por una herramienta MCP
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult estructura de salida para tools/call
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Server encapsula el servidor MCP de Lazymark
type Server struct {
	notesDir string
}

// NewServer crea una nueva instancia del servidor MCP
func NewServer(notesDir string) *Server {
	if notesDir == "" {
		notesDir = config.DefaultNotesDir()
	}
	return &Server{notesDir: notesDir}
}

// RunServer ejecuta el servidor MCP sobre stdin y stdout
func RunServer(notesDir string) error {
	s := NewServer(notesDir)
	return s.Serve(os.Stdin, os.Stdout)
}

// Serve procesa el stream de JSON-RPC línea por línea
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	ops.SetAgentLanguage(s.notesDir) // el idioma de la configuración o, sin uno, inglés
	reader := bufio.NewReaderSize(r, 64*1024)
	encoder := json.NewEncoder(ops.C1Escaper(w)) // lo que venga de una nota con controles C1 sale escapado en el JSON

	for {
		line, tooLong, rerr := readLine(reader, maxLineBytes)
		if tooLong { // una línea de más de 1 MiB no cierra el servidor: se contesta con un error y se sigue con la siguiente
			if err := encoder.Encode(rpcError(nil, -32600, i18n.E("Invalid Request: la petición pasa de 1 MiB", "Invalid Request: the request is over 1 MiB"), nil)); err != nil {
				return err
			}
			if rerr != nil {
				return nilIfEOF(rerr)
			}
			continue
		}
		if len(line) == 0 {
			if rerr != nil {
				return nilIfEOF(rerr)
			}
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error: " + err.Error()},
			}
			_ = encoder.Encode(resp)
			continue
		}

		if bad := invalidID(line); bad {
			if err := encoder.Encode(rpcError(nil, -32600, i18n.E("Invalid Request: el id debe ser un texto o un número", "Invalid Request: the id must be a string or a number"), nil)); err != nil {
				return err
			}
			continue
		}

		resp := s.handleRequest(&req)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
		if rerr != nil {
			return nilIfEOF(rerr)
		}
	}
}

// maxLineBytes es el tamaño máximo de una petición (una línea JSON).
const maxLineBytes = 1 << 20

func nilIfEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// readLine lee una línea de hasta max bytes (sin el salto). Si es más larga, la descarta entera sin guardarla (tooLong) y deja el lector en el
// comienzo de la siguiente. err es el error de lectura (io.EOF al final, con la última línea sin salto incluida).
func readLine(r *bufio.Reader, max int) (line []byte, tooLong bool, err error) {
	var buf []byte
	for {
		chunk, e := r.ReadSlice('\n')
		if !tooLong {
			if len(buf)+len(chunk) > max+2 { // +2: el salto (\n o \r\n); el tope es sobre el contenido
				tooLong, buf = true, nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		if e == bufio.ErrBufferFull {
			continue
		}
		line = bytes.TrimRight(buf, "\r\n")
		if len(line) > max {
			tooLong, line = true, nil
		}
		return line, tooLong, e
	}
}

// Versiones del protocolo que habla el servidor. La vigente (2026-07-28) no tiene sesiones ni handshake `initialize`: cada
// petición lleva su versión en `_meta` y el servidor la acepta o la rechaza (https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning).
// Las anteriores (de 2025-11-25 hacia atrás) abren con `initialize`; el servidor es "de dos épocas" y elige según cómo abre el
// cliente: una petición con `_meta` de la versión moderna se atiende sin estado, y un `initialize` usa la versión vieja negociada.
const (
	modernVersion   = "2026-07-28"
	metaVersionKey  = "io.modelcontextprotocol/protocolVersion"
	metaCapsKey     = "io.modelcontextprotocol/clientCapabilities"
	metaServerInfo  = "io.modelcontextprotocol/serverInfo"
	errUnsupported  = -32022 // UnsupportedProtocolVersionError
	errInvalidParam = -32602
)

// legacyVersions son las versiones con `initialize` que acepta el servidor (las herramientas se leen y se llaman igual en todas).
var legacyVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// supportedVersions son todas las que se anuncian: primero la moderna.
func supportedVersions() []string { return append([]string{modernVersion}, legacyVersions...) }

func (s *Server) serverInfo() obj { return obj{"name": config.AppName, "version": config.Version} }

// requestMeta lee el `_meta` de los parámetros de una petición.
func requestMeta(params json.RawMessage) map[string]json.RawMessage {
	var p struct {
		Meta map[string]json.RawMessage `json:"_meta"`
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &p)
	}
	return p.Meta
}

func rpcError(id interface{}, code int, msg string, data interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: msg, Data: data}}
}

// invalidID indica si la línea trae un "id" que no es un texto ni un número (la spec: "Unlike base JSON-RPC, the ID MUST NOT be null";
// "Requests MUST include a string or integer ID"). Una petición sin "id" es una notificación y no cuenta.
func invalidID(line []byte) bool {
	var probe struct {
		ID json.RawMessage `json:"id"`
	}
	if json.Unmarshal(line, &probe) != nil || probe.ID == nil {
		return false
	}
	id := bytes.TrimSpace(probe.ID)
	if len(id) == 0 {
		return true
	}
	return !(id[0] == '"' || id[0] == '-' || (id[0] >= '0' && id[0] <= '9'))
}

func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	// Notificaciones (sin ID): no tienen respuesta
	if req.ID == nil {
		return nil
	}
	meta := requestMeta(req.Params)
	// una petición es de la época moderna si lleva en `_meta` alguno de los campos por petición; si le falta uno obligatorio, está
	// mal formada (-32602) y no se atiende como si fuera de la época anterior
	_, hasVersion := meta[metaVersionKey]
	_, hasCaps := meta[metaCapsKey]
	_, hasInfo := meta["io.modelcontextprotocol/clientInfo"]
	modern := hasVersion || hasCaps || hasInfo

	if modern { // época moderna: la versión y las capacidades vienen en cada petición
		if !hasVersion {
			return rpcError(req.ID, errInvalidParam, i18n.E("Invalid params: falta _meta.", "Invalid params: missing _meta.")+metaVersionKey, nil)
		}
		var version string
		if err := json.Unmarshal(meta[metaVersionKey], &version); err != nil {
			return rpcError(req.ID, errInvalidParam, fmt.Sprintf(i18n.E("Invalid params: _meta.%s debe ser un texto", "Invalid params: _meta.%s must be a string"), metaVersionKey), nil)
		}
		if version != modernVersion {
			if !contains(legacyVersions, version) || req.Method != "server/discover" {
				return rpcError(req.ID, errUnsupported, "Unsupported protocol version", obj{"supported": supportedVersions(), "requested": version})
			}
		}
		if _, ok := meta[metaCapsKey]; !ok {
			return rpcError(req.ID, errInvalidParam, i18n.E("Invalid params: falta _meta.", "Invalid params: missing _meta.")+metaCapsKey, nil)
		}
	}
	ok := func(result obj) *JSONRPCResponse {
		if modern {
			result["resultType"] = "complete"
			result["_meta"] = obj{metaServerInfo: s.serverInfo()}
		}
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	}

	switch req.Method {
	case "server/discover":
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: obj{
			"resultType":        "complete",
			"supportedVersions": supportedVersions(),
			"capabilities":      obj{"tools": obj{}},
			"_meta":             obj{metaServerInfo: s.serverInfo()},
			"ttlMs":             3600000,
			"cacheScope":        "public",
		}}

	case "initialize": // época anterior: se responde con la versión pedida si se conoce, y si no con la más nueva de las viejas
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := legacyVersions[0]
		if contains(legacyVersions, p.ProtocolVersion) {
			version = p.ProtocolVersion
		}
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: obj{
			"protocolVersion": version,
			"capabilities":    obj{"tools": obj{}},
			"serverInfo":      s.serverInfo(),
		}}

	case "tools/list":
		// en la época moderna el resultado lleva su pista de caché (CacheableResult: ttlMs y cacheScope); la lista es fija y va en orden
		return ok(obj{"tools": s.getToolsList(), "ttlMs": 3600000, "cacheScope": "public"})

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return rpcError(req.ID, errInvalidParam, "Invalid params: "+err.Error(), nil)
		}
		// una herramienta que no existe es un error de protocolo (-32602), no un resultado con isError (spec 2026-07-28, server/tools,
		// "Error Handling": "Unknown tool" … "returned as standard JSON-RPC errors")
		if !s.hasTool(callParams.Name) {
			return rpcError(req.ID, errInvalidParam, "Unknown tool: "+callParams.Name, nil)
		}
		res := s.callTool(callParams.Name, callParams.Arguments)
		content := make([]obj, len(res.Content))
		for i, c := range res.Content {
			content[i] = obj{"type": c.Type, "text": c.Text}
		}
		out := obj{"content": content}
		if res.IsError {
			out["isError"] = true
		}
		return ok(out)

	default:
		return rpcError(req.ID, -32601, fmt.Sprintf("Method '%s' not found", req.Method), nil)
	}
}

// hasTool indica si name es una de las herramientas del servidor.
func (s *Server) hasTool(name string) bool {
	for _, t := range s.getToolsList() {
		if t["name"] == name {
			return true
		}
	}
	return false
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// obj y los demás construyen los esquemas de entrada de las herramientas.
type obj = map[string]interface{}

func str(desc string) obj { return obj{"type": "string", "description": desc} }

func schema(required []string, props obj) obj {
	out := obj{"type": "object", "properties": props}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func (s *Server) getToolsList() []obj {
	idDesc := i18n.E("Id estable de la tarea (el campo `id` de list_tasks y get_kanban): <nota.md>#<huella>.", "Stable task id (the `id` field of list_tasks and get_kanban): <note.md>#<hash>.")
	return []obj{
		{
			"name":        "list_notes",
			"description": i18n.E("Lista las notas de Lazymark con su ruta, título, etiquetas y cantidad de tareas.", "Lists the Lazymark notes with their path, title, tags and number of tasks."),
			"inputSchema": schema(nil, obj{}),
		},
		{
			"name":        "read_note",
			"description": i18n.E("Lee el contenido Markdown de una nota de la carpeta de notas.", "Reads the Markdown content of a note in the notes folder."),
			"inputSchema": schema([]string{"path"}, obj{"path": str(i18n.E("Ruta de la nota .md: absoluta o relativa a la carpeta de notas (el `id` que devuelven list_notes y create_note sirve tal cual). Debe quedar dentro de ella.", "Path of the .md note: absolute or relative to the notes folder (the `id` that list_notes and create_note return works as is). It must be inside it."))}),
		},
		{
			"name":        "create_note",
			"description": i18n.E("Crea una nota nueva (con plantilla, o solo con su título si empty es true) y devuelve su ruta.", "Creates a new note (from the template, or with just its title if empty is true) and returns its path."),
			"inputSchema": schema([]string{"title"}, obj{
				"title":  str(i18n.E("Título de la nota.", "Title of the note.")),
				"folder": str(i18n.E("Subcarpeta de la carpeta de notas donde crearla (opcional; debe existir).", "Subfolder of the notes folder to create it in (optional; it must exist).")),
				"empty":  obj{"type": "boolean", "description": i18n.E("Si es true, la nota solo lleva su título.", "If true, the note only has its title.")},
			}),
		},
		{
			"name":        "search_notes",
			"description": i18n.E("Busca texto en todas las notas (sin distinguir mayúsculas). Devuelve nota, línea y el contexto de cada coincidencia, ordenadas por nota y línea.", "Searches text in all the notes (case-insensitive). Returns the note, line and context of each match, ordered by note and line."),
			"inputSchema": schema([]string{"query"}, obj{
				"query":          str(i18n.E("El texto a buscar, o una expresión regular si regex es true.", "The text to search for, or a regular expression if regex is true.")),
				"regex":          obj{"type": "boolean", "description": i18n.E("Si es true, query es una expresión regular (RE2).", "If true, query is a regular expression (RE2).")},
				"case_sensitive": obj{"type": "boolean", "description": i18n.E("Si es true, distingue mayúsculas de minúsculas.", "If true, the search is case-sensitive.")},
				"limit":          obj{"type": "integer", "description": i18n.E("Máximo de coincidencias (por defecto 500).", "Maximum number of matches (default 500).")},
			}),
		},
		{
			"name":        "list_tasks",
			"description": i18n.E("Lista las tareas con su id, texto, columna del tablero y si están hechas.", "Lists the tasks with their id, text, board column and whether they are done."),
			"inputSchema": schema(nil, obj{
				"pending_only": obj{"type": "boolean", "description": i18n.E("Si es true, omite las tareas hechas.", "If true, leaves out the finished tasks.")},
				"column":       str(i18n.E("Opcional: solo las de esta columna (su id, como todo, doing o done).", "Optional: only the tasks in this column (its id, such as todo, doing or done).")),
				"note_path":    str(i18n.E("Opcional: solo las de esta nota.", "Optional: only the tasks of this note.")),
			}),
		},
		{
			"name":        "move_task",
			"description": i18n.E("Lleva una tarea a otra columna del tablero Kanban. Reescribe solo la línea de la tarea; si la nota cambió en disco mientras tanto, no escribe y devuelve un error.", "Moves a task to another column of the Kanban board. It only rewrites the task's line; if the note changed on disk in the meantime, it writes nothing and returns an error."),
			"inputSchema": schema([]string{"id", "column"}, obj{"id": str(idDesc), "column": str(i18n.E("Id de la columna destino (get_kanban muestra las que hay).", "Id of the destination column (get_kanban shows the existing ones)."))}),
		},
		{
			"name":        "set_task_date",
			"description": i18n.E("Pone o quita la fecha de inicio (🛫) o de vencimiento (📅) de una tarea, en el formato de Obsidian Tasks. Reescribe solo la línea de la tarea. La fecha de completada (✅) la maneja sola el movimiento a la columna de hecho.", "Sets or removes the start date or the due date of a task, in an Obsidian Tasks format (the configured date format). It only rewrites the task's line. The completion date is handled by moving the task to the done column."),
			"inputSchema": schema([]string{"id", "field", "date"}, obj{
				"id":    str(idDesc),
				"field": obj{"type": "string", "enum": []string{"start", "due"}, "description": i18n.E("start (inicio) o due (vencimiento).", "start (start date) or due (due date).")},
				"date":  str(i18n.E("La fecha AAAA-MM-DD (debe existir en el calendario), o \"none\" para quitarla.", "The date YYYY-MM-DD (it must exist in the calendar), or \"none\" to remove it.")),
			}),
		},
		{
			"name":        "toggle_task",
			"description": i18n.E("Marca una tarea como hecha (la lleva a la columna de hecho) o, si ya lo estaba, la devuelve a la primera columna.", "Marks a task as done (moves it to the done column) or, if it already was, returns it to the first column."),
			"inputSchema": schema(nil, obj{
				"id":   str(idDesc),
				"path": str(i18n.E("Forma anterior: ruta de la nota (con line).", "Previous form: path of the note (with line).")),
				"line": obj{"type": "integer", "description": i18n.E("Forma anterior: línea de la tarea, desde 1 (con path).", "Previous form: line of the task, from 1 (with path).")},
			}),
		},
		{
			"name":        "get_kanban",
			"description": i18n.E("Devuelve el tablero Kanban: las columnas configuradas, en orden, cada una con sus tarjetas.", "Returns the Kanban board: the configured columns, in order, each with its cards."),
			"inputSchema": schema(nil, obj{}),
		},
	}
}

func fail(err error) CallToolResult {
	return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf(i18n.E("Error (código %d): %v", "Error (code %d): %v"), ops.Code(err), err)}}}
}

func ok(v interface{}) CallToolResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	data = ops.EscapeC1(data)
	return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}
}

func missing(names string) CallToolResult {
	return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + names + i18n.E(" requerido(s)", " required")}}}
}

func (s *Server) callTool(name string, args map[string]interface{}) CallToolResult {
	svc, err := ops.NewForAgent(s.notesDir)
	if err != nil {
		return fail(err)
	}
	if msg := checkArgs(args); msg != "" {
		return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + msg}}}
	}
	text := func(k string) string { v, _ := args[k].(string); return v }

	switch name {
	case "list_notes":
		notes, err := svc.ListNotes()
		if err != nil {
			return fail(err)
		}
		return ok(notes)

	case "read_note":
		if text("path") == "" {
			return missing("'path'")
		}
		n, err := svc.ShowNote(text("path"))
		if err != nil {
			return fail(err)
		}
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: n.Content}}}

	case "create_note":
		if text("title") == "" {
			return missing("'title'")
		}
		empty, _ := args["empty"].(bool)
		n, err := svc.NewNote(text("title"), text("folder"), empty)
		if err != nil {
			return fail(err)
		}
		return ok(n)

	case "search_notes":
		if text("query") == "" {
			return missing("'query'")
		}
		regex, _ := args["regex"].(bool)
		cs, _ := args["case_sensitive"].(bool)
		limit, _ := args["limit"].(float64)
		res, err := svc.Search(text("query"), regex, cs, int(limit))
		if err != nil {
			return fail(err)
		}
		return ok(res)

	case "list_tasks":
		pending, _ := args["pending_only"].(bool)
		tasks, err := svc.ListTasks(ops.TaskFilter{PendingOnly: pending, Column: text("column"), Note: text("note_path")})
		if err != nil {
			return fail(err)
		}
		return ok(tasks)

	case "move_task":
		if text("id") == "" || text("column") == "" {
			return missing("'id' y 'column'")
		}
		t, err := svc.MoveTask(text("id"), text("column"))
		if err != nil {
			return fail(err)
		}
		return ok(t)

	case "set_task_date":
		if text("id") == "" || text("field") == "" || text("date") == "" {
			return missing("'id', 'field' y 'date'")
		}
		t, err := svc.SetDate(text("id"), text("field"), text("date"))
		if err != nil {
			return fail(err)
		}
		return ok(t)

	case "toggle_task":
		id := text("id")
		if id == "" { // forma anterior: path + line
			line, hasLine := args["line"].(float64)
			if text("path") == "" || !hasLine {
				return missing("'id' (o 'path' y 'line')")
			}
			if id, err = svc.IDByLine(text("path"), int(line)); err != nil {
				return fail(err)
			}
		}
		t, err := svc.ToggleTask(id)
		if err != nil {
			return fail(err)
		}
		return ok(t)

	case "get_kanban":
		b, err := svc.Board()
		if err != nil {
			return fail(err)
		}
		return ok(b)

	default:
		return CallToolResult{
			IsError: true,
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Herramienta '%s' no encontrada", name)}},
		}
	}
}

// argKinds son los tipos de los argumentos de las herramientas: un argumento de otro tipo es un error, no se ignora en silencio.
var argKinds = map[string]string{
	"path": "texto", "title": "texto", "folder": "texto", "query": "texto", "column": "texto", "note_path": "texto", "id": "texto", "field": "texto", "date": "texto",
	"empty": "booleano", "regex": "booleano", "case_sensitive": "booleano", "pending_only": "booleano",
	"limit": "entero", "line": "entero",
}

// checkArgs devuelve el error del primer argumento (por orden alfabético) con un tipo que no es el suyo, o "".
func checkArgs(args map[string]interface{}) string {
	names := make([]string, 0, len(args))
	for k := range args {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		kind, known := argKinds[k]
		if !known {
			continue
		}
		v := args[k]
		if v == nil { // null en un argumento opcional es "no enviado"
			continue
		}
		ok := false
		switch kind {
		case "texto":
			_, ok = v.(string)
		case "booleano":
			_, ok = v.(bool)
		case "entero":
			f, isNum := v.(float64)
			ok = isNum && f == math.Trunc(f) && f >= 0 && f <= 1e9
		}
		if !ok {
			return fmt.Sprintf(i18n.E("el argumento '%s' debe ser %s", "the argument '%s' must be %s"), k, map[string]string{
				"texto":    i18n.E("un texto", "a string"),
				"booleano": i18n.E("verdadero o falso", "true or false"),
				"entero":   i18n.E("un número entero (0 o mayor)", "a whole number (0 or more)"),
			}[kind])
		}
	}
	return ""
}
