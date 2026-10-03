package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError define un error en JSON-RPC 2.0
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
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
	scanner := bufio.NewScanner(r)
	// Permitir líneas de hasta 1MB por si se envían notas grandes
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	encoder := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
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

		resp := s.handleRequest(&req)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	// Notificaciones (sin ID)
	if req.ID == nil {
		if req.Method == "notifications/initialized" {
			return nil
		}
		return nil
	}

	switch req.Method {
	case "initialize":
		result := map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    config.AppName,
				"version": config.Version,
			},
		}
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"tools": s.getToolsList()},
		}

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32602, Message: "Invalid params: " + err.Error()},
			}
		}

		toolResult := s.callTool(callParams.Name, callParams.Arguments)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  toolResult,
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Method '%s' not found", req.Method)},
		}
	}
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
	const idDesc = "Id estable de la tarea (el campo `id` de list_tasks y get_kanban): <nota.md>#<huella>."
	return []obj{
		{
			"name":        "list_notes",
			"description": "Lista las notas de Lazymark con su ruta, título, etiquetas y cantidad de tareas.",
			"inputSchema": schema(nil, obj{}),
		},
		{
			"name":        "read_note",
			"description": "Lee el contenido Markdown de una nota de la carpeta de notas.",
			"inputSchema": schema([]string{"path"}, obj{"path": str("Ruta de la nota .md: absoluta o relativa a la carpeta de notas. Debe quedar dentro de ella.")}),
		},
		{
			"name":        "create_note",
			"description": "Crea una nota nueva (con plantilla, o solo con su título si empty es true) y devuelve su ruta.",
			"inputSchema": schema([]string{"title"}, obj{
				"title":  str("Título de la nota."),
				"folder": str("Subcarpeta de la carpeta de notas donde crearla (opcional; debe existir)."),
				"empty":  obj{"type": "boolean", "description": "Si es true, la nota solo lleva su título."},
			}),
		},
		{
			"name":        "list_tasks",
			"description": "Lista las tareas con su id, texto, columna del tablero y si están hechas.",
			"inputSchema": schema(nil, obj{
				"pending_only": obj{"type": "boolean", "description": "Si es true, omite las tareas hechas."},
				"column":       str("Opcional: solo las de esta columna (su id, como todo, doing o done)."),
				"note_path":    str("Opcional: solo las de esta nota."),
			}),
		},
		{
			"name":        "move_task",
			"description": "Lleva una tarea a otra columna del tablero Kanban. Reescribe solo la línea de la tarea; si la nota cambió en disco mientras tanto, no escribe y devuelve un error.",
			"inputSchema": schema([]string{"id", "column"}, obj{"id": str(idDesc), "column": str("Id de la columna destino (get_kanban muestra las que hay).")}),
		},
		{
			"name":        "set_task_date",
			"description": "Pone o quita la fecha de inicio (🛫) o de vencimiento (📅) de una tarea, en el formato de Obsidian Tasks. Reescribe solo la línea de la tarea. La fecha de completada (✅) la maneja sola el movimiento a la columna de hecho.",
			"inputSchema": schema([]string{"id", "field", "date"}, obj{
				"id":    str(idDesc),
				"field": obj{"type": "string", "enum": []string{"start", "due"}, "description": "start (inicio) o due (vencimiento)."},
				"date":  str("La fecha AAAA-MM-DD (debe existir en el calendario), o \"none\" para quitarla."),
			}),
		},
		{
			"name":        "toggle_task",
			"description": "Marca una tarea como hecha (la lleva a la columna de hecho) o, si ya lo estaba, la devuelve a la primera columna.",
			"inputSchema": schema(nil, obj{
				"id":   str(idDesc),
				"path": str("Forma anterior: ruta de la nota (con line)."),
				"line": obj{"type": "integer", "description": "Forma anterior: línea de la tarea, desde 1 (con path)."},
			}),
		},
		{
			"name":        "get_kanban",
			"description": "Devuelve el tablero Kanban: las columnas configuradas, en orden, cada una con sus tarjetas.",
			"inputSchema": schema(nil, obj{}),
		},
	}
}

func fail(err error) CallToolResult {
	return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error (código %d): %v", ops.Code(err), err)}}}
}

func ok(v interface{}) CallToolResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}
}

func missing(names string) CallToolResult {
	return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + names + " requerido(s)"}}}
}

func (s *Server) callTool(name string, args map[string]interface{}) CallToolResult {
	svc, err := ops.New(s.notesDir)
	if err != nil {
		return fail(err)
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
