// MCP protocol types
// Spec: interfaces.md
package mcp

import "encoding/json"

// Message represents an MCP JSON-RPC message.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC error.
type ErrorObject struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeResult is returned from initialize.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities describes server capabilities.
type ServerCapabilities struct {
	Resources *ResourceCapabilities `json:"resources,omitempty"`
	Tools     *ToolCapabilities     `json:"tools,omitempty"`
	Prompts   *PromptCapabilities   `json:"prompts,omitempty"`
}

// ResourceCapabilities for resources.
type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolCapabilities for tools.
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptCapabilities for prompts.
type PromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo identifies the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ResourceInfo describes a resource.
type ResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourcesResult from resources/list.
type ListResourcesResult struct {
	Resources []ResourceInfo `json:"resources"`
}

// ReadResourceParams for resources/read.
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ResourceContent is content from a resource.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ReadResourceResult from resources/read.
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ToolInfo describes a tool.
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema"`
}

// ListToolsResult from tools/list.
type ListToolsResult struct {
	Tools []ToolInfo `json:"tools"`
}

// ToolCallParams for tools/call.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolContent is content in a tool result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolCallResult from tools/call.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}
