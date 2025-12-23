// Package mcp implements the Model Context Protocol server.
// CRC: crc-MCPServer.md
// Spec: specs/mcp.md
package mcp

import (
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/server"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/lua"
	"github.com/zot/ui/internal/viewdef"
)

// State represents the lifecycle state of the MCP server.
type State int

const (
	Unconfigured State = iota
	Configured
	Running
)

// Server implements an MCP server for AI integration.
type Server struct {
		mcpServer *server.MCPServer
		cfg       *config.Config
		runtime   *lua.Runtime
		viewdefs  *viewdef.ViewdefManager
		startFunc func(port int) (string, error) // Callback to start HTTP server
		onViewdefUploaded func(typeName string) // Callback when a viewdef is uploaded
	
		mu      sync.RWMutex
		state   State
	

		baseDir string

		url     string

	}

	

	// NewServer creates a new MCP server.

	

	func NewServer(cfg *config.Config, runtime *lua.Runtime, viewdefs *viewdef.ViewdefManager, startFunc func(port int) (string, error), onViewdefUploaded func(typeName string)) *Server {

	

		s := server.NewMCPServer("ui-server", "0.1.0")

	

		srv := &Server{

	

			mcpServer: s,

	

			cfg:       cfg,

	

			runtime:   runtime,

	

			viewdefs:  viewdefs,

	

			startFunc: startFunc,

	

			onViewdefUploaded: onViewdefUploaded,

	

			state:     Unconfigured,

	

		}

	

	

		srv.registerTools()

		srv.registerResources()

		return srv

	}

	

	// ServeStdio starts the MCP server on Stdin/Stdout.

	func (s *Server) ServeStdio() error {

		return server.ServeStdio(s.mcpServer)

	}

	

	// Configure transitions the server to the Configured state.

	

	// Spec: mcp.md

	

	// CRC: crc-MCPServer.md

	

	func (s *Server) Configure(baseDir string) error {

	

	

		s.mu.Lock()

		defer s.mu.Unlock()

	

		if s.state == Running {

			return fmt.Errorf("Cannot reconfigure while running")

		}

	

		// In a real implementation, we would set up I/O redirection here.

		// For now, we just update the state and baseDir.

		s.baseDir = baseDir

		s.state = Configured

	

		return nil

	}

	

	// Start transitions the server to the Running state and starts the HTTP server.

	

	// Spec: mcp.md

	

	// CRC: crc-MCPServer.md

	

	func (s *Server) Start() (string, error) {

	

	

		s.mu.Lock()

		defer s.mu.Unlock()

	

		if s.state == Unconfigured {

			return "", fmt.Errorf("Server not configured")

		}

		if s.state == Running {

			return "", fmt.Errorf("Server already running")

		}

	

		// Select random port (0)

		url, err := s.startFunc(0)

		if err != nil {

			return "", err

		}

	

		s.state = Running

		s.url = url

		return url, nil

	}

	