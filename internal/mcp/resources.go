// Package mcp implements the Model Context Protocol server.
// CRC: crc-MCPResource.md
// Spec: interfaces.md
// Sequence: seq-mcp-get-state.md
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerResources() {
	// ui://state (defaults to session 1)
	s.mcpServer.AddResource(mcp.NewResource("ui://state", "Current Session State",
		mcp.WithResourceDescription("Current JSON state of session 1 (Variable 1)"),
		mcp.WithMIMEType("application/json"),
	), s.handleGetStateResource)

	// ui://state/{sessionId}
	s.mcpServer.AddResource(mcp.NewResource("ui://state/{sessionId}", "Session State",
		mcp.WithResourceDescription("Current JSON state of the session (Variable 1)"),
		mcp.WithMIMEType("application/json"),
	), s.handleGetStateResource)
}

func (s *Server) handleGetStateResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	uri := request.Params.URI
	
	// Simple parsing of URI to get sessionId
	var sessionID string
	if uri == "ui://state" {
		sessionID = "1"
	} else {
		n, err := fmt.Sscanf(uri, "ui://state/%s", &sessionID)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("invalid URI format")
		}
	}

	session, ok := s.runtime.GetLuaSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	tracker := session.GetTracker()
	if tracker == nil {
		return nil, fmt.Errorf("tracker not found")
	}

	var result interface{}

	// 1. Try to use the explicit MCP state variable
	if session.McpStateID != 0 {
		v := tracker.GetVariable(session.McpStateID)
		if v != nil {
			val := v.NavigationValue()
			jsonVal, err := tracker.ToValueJSONBytes(val)
			if err == nil {
				result = map[string]interface{}{
					"id":         v.ID,
					"properties": v.Properties,
					"value":      json.RawMessage(jsonVal),
				}
			}
		}
	}

	// 2. Fallback to Variable 1 (App)
	if result == nil {
		v1 := tracker.GetVariable(1)
		if v1 == nil {
			return nil, fmt.Errorf("variable 1 not found")
		}

		val := v1.NavigationValue()
		jsonVal, err := tracker.ToValueJSONBytes(val)
		if err != nil {
			return nil, fmt.Errorf("marshaling error: %v", err)
		}

		result = map[string]interface{}{
			"id":         1,
			"properties": v1.Properties,
			"value":      json.RawMessage(jsonVal),
		}
	}

	content, _ := json.MarshalIndent(result, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(content),
		},
	}, nil
}
