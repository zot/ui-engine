// Package cli provides the command-line interface for remote-ui.
// This file re-exports internal packages for MCP integration.
package cli

import (
	"github.com/zot/ui-engine/internal/bundle"
	"github.com/zot/ui-engine/internal/lua"
	"github.com/zot/ui-engine/internal/server"
	"github.com/zot/ui-engine/internal/viewdef"
)

// Re-export server types for MCP integration
type (
	Server         = server.Server
	LuaRuntime     = lua.Runtime
	ViewdefManager = viewdef.ViewdefManager
)

// Re-export server constructor
var (
	NewServer = server.New
)

// Re-export bundle functions for MCP integration
var (
	IsBundled        = bundle.IsBundled
	BundleListFiles  = bundle.ListFilesInDir
	BundleReadFile   = bundle.ReadFile
)
