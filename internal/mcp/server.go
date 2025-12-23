// Package mcp implements the Model Context Protocol server.
// CRC: crc-MCPServer.md
// Spec: interfaces.md (MCP Server)
package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/lua"
	"github.com/zot/ui/internal/viewdef"
)

// Server implements an MCP server for AI integration.
type Server struct {
	mcpServer *server.MCPServer
	cfg       *config.Config
	runtime   *lua.Runtime
	viewdefs  *viewdef.ViewdefManager
}

// NewServer creates a new MCP server.
func NewServer(cfg *config.Config, runtime *lua.Runtime, viewdefs *viewdef.ViewdefManager) *Server {
	s := server.NewMCPServer("ui-server", "0.1.0")
	srv := &Server{
		mcpServer: s,
		cfg:       cfg,
		runtime:   runtime,
		viewdefs:  viewdefs,
	}
	srv.registerTools()
	srv.registerResources()
	return srv
}

// ServeStdio starts the MCP server on Stdin/Stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}