// Package mcp implements the Model Context Protocol server.
// CRC: crc-MCPTool.md
// Spec: interfaces.md
// Sequence: seq-mcp-run.md, seq-mcp-get-state.md
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerTools() {
	// ui_get_state
	s.mcpServer.AddTool(mcp.NewTool("ui_get_state",
		mcp.WithDescription("Get the current state (Variable 1) of a session"),
		mcp.WithString("sessionId", mcp.Description("The session ID to inspect (defaults to '1')")),
	), s.handleGetState)

	// ui_run
	s.mcpServer.AddTool(mcp.NewTool("ui_run",
		mcp.WithDescription("Execute Lua code in a session context"),
		mcp.WithString("code", mcp.Required(), mcp.Description("Lua code to execute")),
		mcp.WithString("sessionId", mcp.Description("The session ID to run in (defaults to '1')")),
	), s.handleRun)

	// ui_upload_viewdef
	s.mcpServer.AddTool(mcp.NewTool("ui_upload_viewdef",
		mcp.WithDescription("Upload a dynamic view definition"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Presenter type (e.g. 'MyPresenter')")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace (e.g. 'DEFAULT')")),
		mcp.WithString("content", mcp.Required(), mcp.Description("HTML content")),
	), s.handleUploadViewdef)
}

func (s *Server) handleGetState(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("arguments must be a map"), nil
	}
	
	sessionID, ok := args["sessionId"].(string)
	if !ok || sessionID == "" {
		sessionID = "1"
	}

	session, ok := s.runtime.GetLuaSession(sessionID)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("session %s not found", sessionID)), nil
	}

	tracker := session.GetTracker()
	if tracker == nil {
		return mcp.NewToolResultError("tracker not found"), nil
	}
	
	v1 := tracker.GetVariable(1)
	if v1 == nil {
		return mcp.NewToolResultError("variable 1 (app root) not found"), nil
	}

	val := v1.NavigationValue()
	jsonVal, err := tracker.ToValueJSONBytes(val)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal value: %v", err)), nil
	}

	result := map[string]interface{}{
		"id":         1,
		"properties": v1.Properties,
		"value":      json.RawMessage(jsonVal),
	}
	
	jsonResult, _ := json.MarshalIndent(result, "", "  ")

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleRun(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("arguments must be a map"), nil
	}

	code, ok := args["code"].(string)
	if !ok {
		return mcp.NewToolResultError("code must be a string"), nil
	}
	sessionID, ok := args["sessionId"].(string)
	if !ok || sessionID == "" {
		sessionID = "1"
	}
	
	err := s.runtime.ExecuteInSession(sessionID, func() error {
		return s.runtime.LoadCode("mcp-run", code)
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("execution failed: %v", err)), nil
	}

	return mcp.NewToolResultText("Executed successfully"), nil
}

func (s *Server) handleUploadViewdef(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("arguments must be a map"), nil
	}

	typeName, ok := args["type"].(string)
	if !ok {
		return mcp.NewToolResultError("type must be a string"), nil
	}
	namespace, ok := args["namespace"].(string)
	if !ok {
		return mcp.NewToolResultError("namespace must be a string"), nil
	}
	content, ok := args["content"].(string)
	if !ok {
		return mcp.NewToolResultError("content must be a string"), nil
	}

	key := fmt.Sprintf("%s.%s", typeName, namespace)
	s.viewdefs.AddViewdef(key, content)

	return mcp.NewToolResultText(fmt.Sprintf("Viewdef %s uploaded", key)), nil
}