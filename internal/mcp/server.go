// CRC: crc-MCPServer.md
// Spec: interfaces.md
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

// Server implements an MCP server for AI integration.
type Server struct {
	resources    map[string]*Resource
	tools        map[string]*Tool
	input        io.Reader
	output       io.Writer
	mu           sync.RWMutex
	shutdown     chan struct{}
	onToolCall   func(name string, args map[string]interface{}) (interface{}, error)
	onResource   func(uri string) (interface{}, error)
}

// NewServer creates a new MCP server.
func NewServer(input io.Reader, output io.Writer) *Server {
	return &Server{
		resources: make(map[string]*Resource),
		tools:     make(map[string]*Tool),
		input:     input,
		output:    output,
		shutdown:  make(chan struct{}),
	}
}

// SetToolHandler sets the callback for tool calls.
func (s *Server) SetToolHandler(handler func(name string, args map[string]interface{}) (interface{}, error)) {
	s.onToolCall = handler
}

// SetResourceHandler sets the callback for resource queries.
func (s *Server) SetResourceHandler(handler func(uri string) (interface{}, error)) {
	s.onResource = handler
}

// RegisterResource adds a resource to the server.
func (s *Server) RegisterResource(r *Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[r.URI] = r
}

// RegisterTool adds a tool to the server.
func (s *Server) RegisterTool(t *Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[t.Name] = t
}

// Start begins processing MCP messages.
func (s *Server) Start() error {
	scanner := bufio.NewScanner(s.input)
	// Handle large messages
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-s.shutdown:
			return nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("[mcp] Failed to parse message: %v", err)
			continue
		}

		response := s.handleMessage(&msg)
		if response != nil {
			s.sendResponse(response)
		}
	}

	return scanner.Err()
}

// Shutdown stops the server.
func (s *Server) Shutdown() {
	close(s.shutdown)
}

// handleMessage processes an incoming MCP message.
func (s *Server) handleMessage(msg *Message) *Message {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "resources/list":
		return s.handleListResources(msg)
	case "resources/read":
		return s.handleReadResource(msg)
	case "tools/list":
		return s.handleListTools(msg)
	case "tools/call":
		return s.handleToolCall(msg)
	case "notifications/initialized":
		// Client acknowledgment, no response needed
		return nil
	default:
		return s.errorResponse(msg.ID, -32601, "Method not found")
	}
}

// handleInitialize responds to the initialize request.
func (s *Server) handleInitialize(msg *Message) *Message {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Resources: &ResourceCapabilities{},
			Tools:     &ToolCapabilities{},
		},
		ServerInfo: ServerInfo{
			Name:    "ui-server",
			Version: "0.1.0",
		},
	}
	return s.successResponse(msg.ID, result)
}

// handleListResources returns available resources.
func (s *Server) handleListResources(msg *Message) *Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]ResourceInfo, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, ResourceInfo{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MimeType:    r.MimeType,
		})
	}

	return s.successResponse(msg.ID, ListResourcesResult{Resources: resources})
}

// handleReadResource reads a specific resource.
func (s *Server) handleReadResource(msg *Message) *Message {
	var params ReadResourceParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, -32602, "Invalid params")
	}

	s.mu.RLock()
	resource, ok := s.resources[params.URI]
	s.mu.RUnlock()

	if !ok {
		return s.errorResponse(msg.ID, -32602, "Resource not found")
	}

	var content interface{}
	var err error

	if resource.Handler != nil {
		content, err = resource.Handler()
	} else if s.onResource != nil {
		content, err = s.onResource(params.URI)
	} else {
		return s.errorResponse(msg.ID, -32603, "No resource handler")
	}

	if err != nil {
		return s.errorResponse(msg.ID, -32603, err.Error())
	}

	contentJSON, _ := json.Marshal(content)
	return s.successResponse(msg.ID, ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      params.URI,
				MimeType: resource.MimeType,
				Text:     string(contentJSON),
			},
		},
	})
}

// handleListTools returns available tools.
func (s *Server) handleListTools(msg *Message) *Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]ToolInfo, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, ToolInfo{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	return s.successResponse(msg.ID, ListToolsResult{Tools: tools})
}

// handleToolCall executes a tool.
func (s *Server) handleToolCall(msg *Message) *Message {
	var params ToolCallParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, -32602, "Invalid params")
	}

	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return s.errorResponse(msg.ID, -32602, "Tool not found")
	}

	var result interface{}
	var err error

	if tool.Handler != nil {
		result, err = tool.Handler(params.Arguments)
	} else if s.onToolCall != nil {
		result, err = s.onToolCall(params.Name, params.Arguments)
	} else {
		return s.errorResponse(msg.ID, -32603, "No tool handler")
	}

	if err != nil {
		return s.successResponse(msg.ID, ToolCallResult{
			Content: []ToolContent{
				{Type: "text", Text: fmt.Sprintf("Error: %v", err)},
			},
			IsError: true,
		})
	}

	resultJSON, _ := json.Marshal(result)
	return s.successResponse(msg.ID, ToolCallResult{
		Content: []ToolContent{
			{Type: "text", Text: string(resultJSON)},
		},
	})
}

// SendNotification sends a notification to the client.
func (s *Server) SendNotification(method string, params interface{}) error {
	msg := Message{
		JSONRPC: "2.0",
		Method:  method,
	}
	if params != nil {
		paramsJSON, _ := json.Marshal(params)
		msg.Params = paramsJSON
	}
	return s.sendResponse(&msg)
}

// successResponse creates a success response.
func (s *Server) successResponse(id interface{}, result interface{}) *Message {
	resultJSON, _ := json.Marshal(result)
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultJSON,
	}
}

// errorResponse creates an error response.
func (s *Server) errorResponse(id interface{}, code int, message string) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
		},
	}
}

// sendResponse writes a response to the output.
func (s *Server) sendResponse(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.output, "%s\n", data)
	return err
}
