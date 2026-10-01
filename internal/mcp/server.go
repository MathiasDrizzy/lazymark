package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
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
	storage  *storage.Storage
}

// NewServer crea una nueva instancia del servidor MCP
func NewServer(notesDir string) *Server {
	if notesDir == "" {
		notesDir = config.DefaultNotesDir()
	}
	return &Server{
		notesDir: notesDir,
		storage:  storage.New(notesDir),
	}
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

func (s *Server) getToolsList() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "list_notes",
			"description": "Lista todas las notas de Lazymark con sus rutas, títulos, etiquetas y conteo de tareas.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "read_note",
			"description": "Lee el contenido Markdown completo de una nota específica.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Ruta absoluta o relativa del archivo de nota Markdown.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "list_tasks",
			"description": "Lista las tareas de todas las notas o de una nota específica, con filtro de pendientes.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pending_only": map[string]interface{}{
						"type":        "boolean",
						"description": "Si es true, solo retorna las tareas que no están completadas.",
					},
					"note_path": map[string]interface{}{
						"type":        "string",
						"description": "Opcional: filtrar tareas de una nota en específico.",
					},
				},
			},
		},
		{
			"name":        "toggle_task",
			"description": "Alterna atómicamente el estado de una tarea (- [ ] <-> - [x]) en el archivo físico.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Ruta de la nota Markdown donde reside la tarea.",
					},
					"line": map[string]interface{}{
						"type":        "integer",
						"description": "Número de línea 1-indexed de la tarea en el archivo.",
					},
				},
				"required": []string{"path", "line"},
			},
		},
		{
			"name":        "get_kanban",
			"description": "Devuelve el tablero Kanban completo con tareas agrupadas en columnas: todo, doing, done.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func (s *Server) callTool(name string, args map[string]interface{}) CallToolResult {
	switch name {
	case "list_notes":
		notes, err := s.storage.ListNotes()
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}}}
		}
		type noteDTO struct {
			ID         string   `json:"id"`
			Title      string   `json:"title"`
			Path       string   `json:"path"`
			Tags       []string `json:"tags"`
			TasksCount int      `json:"tasks_count"`
			ModTime    string   `json:"mod_time"`
		}
		var dtoList []noteDTO
		for _, n := range notes {
			dtoList = append(dtoList, noteDTO{
				ID:         n.ID,
				Title:      n.Title,
				Path:       n.Path,
				Tags:       n.Tags,
				TasksCount: len(n.Tasks),
				ModTime:    n.ModTime.Format(time.RFC3339),
			})
		}
		data, _ := json.MarshalIndent(dtoList, "", "  ")
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}

	case "read_note":
		p, _ := args["path"].(string)
		if p == "" {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: parámetro 'path' requerido"}}}
		}
		bytes, err := os.ReadFile(p)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error al leer archivo: " + err.Error()}}}
		}
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(bytes)}}}

	case "list_tasks":
		pendingOnly, _ := args["pending_only"].(bool)
		notePathFilter, _ := args["note_path"].(string)

		notes, err := s.storage.ListNotes()
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}}}
		}

		filter := views.TaskFilterAll
		if pendingOnly {
			filter = views.TaskFilterPending
		}

		flatTasks := views.CollectTasks(notes, filter)
		type taskDTO struct {
			Line      int    `json:"line"`
			Text      string `json:"text"`
			CleanText string `json:"clean_text"`
			Done      bool   `json:"done"`
			Stage     string `json:"stage"`
			NoteTitle string `json:"note_title"`
			NotePath  string `json:"note_path"`
		}

		var dtoList []taskDTO
		for _, t := range flatTasks {
			if notePathFilter != "" && t.NotePath != notePathFilter {
				continue
			}
			stage := storage.GetTaskStage(t.Task)
			stageStr := "todo"
			if stage == storage.StageDoing {
				stageStr = "doing"
			} else if stage == storage.StageDone {
				stageStr = "done"
			}
			dtoList = append(dtoList, taskDTO{
				Line:      t.Line,
				Text:      t.Text,
				CleanText: storage.CleanTaskText(t.Text),
				Done:      t.Done,
				Stage:     stageStr,
				NoteTitle: t.NoteTitle,
				NotePath:  t.NotePath,
			})
		}
		data, _ := json.MarshalIndent(dtoList, "", "  ")
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}

	case "toggle_task":
		p, _ := args["path"].(string)
		lineFloat, ok := args["line"].(float64)
		if !ok || p == "" {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: parámetros 'path' y 'line' requeridos"}}}
		}
		line := int(lineFloat)
		newStatus, err := s.storage.ToggleTask(p, line)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}}}
		}
		res, _ := json.MarshalIndent(map[string]interface{}{
			"success": true,
			"path":    p,
			"line":    line,
			"done":    newStatus,
		}, "", "  ")
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(res)}}}

	case "get_kanban":
		notes, err := s.storage.ListNotes()
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}}}
		}
		board := views.CollectKanban(notes)

		type cardDTO struct {
			Text      string `json:"text"`
			NoteTitle string `json:"note_title"`
			NotePath  string `json:"note_path"`
			Line      int    `json:"line"`
		}

		mapCards := func(cards []views.KanbanCard) []cardDTO {
			var res []cardDTO
			for _, c := range cards {
				res = append(res, cardDTO{
					Text:      c.CleanText,
					NoteTitle: c.NoteTitle,
					NotePath:  c.NotePath,
					Line:      c.Task.Line,
				})
			}
			return res
		}

		kanbanDTO := map[string]interface{}{
			"todo":  mapCards(board.Todo),
			"doing": mapCards(board.Doing),
			"done":  mapCards(board.Done),
		}

		data, _ := json.MarshalIndent(kanbanDTO, "", "  ")
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}

	default:
		return CallToolResult{
			IsError: true,
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Herramienta '%s' no encontrada", name)}},
		}
	}
}
