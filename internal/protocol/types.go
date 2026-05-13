package protocol

import "encoding/json"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientCapabilities struct{}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct{}

type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func OKResponse(id int, result any) Response {
	b, _ := json.Marshal(result)
	return Response{JSONRPC: "2.0", ID: id, Result: b}
}

func ErrResponse(id, code int, msg string) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: msg}}
}
