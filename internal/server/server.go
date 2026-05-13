package server

import (
	"encoding/json"
	"mcp-bench/internal/protocol"
	"strings"
)

func Handle(req protocol.Request) (protocol.Response, bool) {
	switch req.Method {
	case "initialize":
		return handleInitialize(req), true

	case "notifications/initialized":
		// уведомление, ответ не нужен
		return protocol.Response{}, false

	case "tools/list":
		return handleToolsList(req), true

	case "tools/call":
		var p protocol.ToolCallParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return protocol.ErrResponse(req.ID, -32600, "invalid params"), true
		}
		return callTool(req.ID, p), true

	default:
		return protocol.ErrResponse(req.ID, -32601, "method not found"), true
	}
}

func handleInitialize(req protocol.Request) protocol.Response {
	result := protocol.InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo:      protocol.ServerInfo{Name: "mcp-bench-server", Version: "1.0.0"},
		Capabilities:    protocol.ServerCapabilities{Tools: &protocol.ToolsCapability{}},
	}
	return protocol.OKResponse(req.ID, result)
}

func handleToolsList(req protocol.Request) protocol.Response {
	result := protocol.ToolsListResult{
		Tools: []protocol.ToolDefinition{
			{Name: "echo", Description: "Returns the input back"},
			{Name: "heavy", Description: "Returns 10KB payload"},
		},
	}
	return protocol.OKResponse(req.ID, result)
}

func callTool(id int, p protocol.ToolCallParams) protocol.Response {
	switch p.Name {
	case "echo":
		return protocol.OKResponse(id, map[string]any{"result": string(p.Arguments)})
	case "heavy":
		payload := strings.Repeat("x", 10*1024)
		return protocol.OKResponse(id, map[string]any{"result": payload})
	default:
		return protocol.ErrResponse(id, -32601, "unknown tool: "+p.Name)
	}
}
