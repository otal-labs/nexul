// Package mcp implements the Model Context Protocol server adapter over Streamable HTTP and stdio.
package mcp

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the legacy MCP version kept for the 12-month back-compat window (2025-03-26).
const ProtocolVersion = "2025-03-26"

// ProtocolVersion2026 is the stateless revision: per-request _meta version + capabilities, no initialize handshake.
const ProtocolVersion2026 = "2026-07-28"

// SupportedVersions lists every protocol version accepted by the per-request _meta check, newest first.
var SupportedVersions = []string{ProtocolVersion2026, ProtocolVersion}

// ServerInfo identifies this server in the initialize handshake.
const (
	ServerName    = "nexul"
	ServerVersion = "0.1.0"
)

// JSON-RPC 2.0 method names (MCP 2025-03-26 unless noted).
const (
	MethodInitialize        = "initialize"
	MethodPing              = "ping"
	MethodServerDiscover    = "server/discover" // MCP 2026-07-28
	MethodToolsList         = "tools/list"
	MethodToolsCall         = "tools/call"
	MethodResourcesList     = "resources/list"
	MethodResourcesRead     = "resources/read"
	MethodPromptsList       = "prompts/list"
	MethodPromptsGet        = "prompts/get"
	MethodNotifyInitialized = "notifications/initialized"
	MethodNotifyCancelled   = "notifications/cancelled"
)

// _meta keys for MCP 2026-07-28 per-request and per-result metadata.
const (
	MetaKeyProtocolVersion    = "io.modelcontextprotocol/protocolVersion"
	MetaKeyClientCapabilities = "io.modelcontextprotocol/clientCapabilities"
	MetaKeyClientInfo         = "io.modelcontextprotocol/clientInfo"
	MetaKeyServerInfo         = "io.modelcontextprotocol/serverInfo"
)

// JSON-RPC error codes plus the domain-error mappings; -32000..-32019 are implementation-defined, -32020+ is spec-reserved.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeUnauthorized   = -32001
	CodeNotFound       = -32002

	CodeHeaderMismatch             = -32020 // MCP 2026-07-28
	CodeUnsupportedProtocolVersion = -32022 // MCP 2026-07-28
)

// Request is a JSON-RPC 2.0 request or notification. A notification has no ID.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response. Exactly one of Result or Error is set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object. Data carries optional extra detail.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func errResponse(code int, format string, args ...any) *Response {
	return &Response{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: code, Message: fmt.Sprintf(format, args...)},
	}
}
