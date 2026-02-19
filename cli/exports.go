// Package cli provides the command-line interface for remote-ui.
// This file re-exports internal packages for MCP integration.
package cli

import (
	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui-engine/internal/bundle"
	"github.com/zot/ui-engine/internal/lua"
	"github.com/zot/ui-engine/internal/server"
	"github.com/zot/ui-engine/internal/viewdef"
)

// Re-export server types for MCP integration
type (
	Server         = server.Server
	LuaRuntime     = lua.LuaSession
	ViewdefManager = viewdef.ViewdefManager
	// Change-tracker types for variable inspection
	Variable = changetracker.Variable
	Tracker  = changetracker.Tracker
	// Debug types
	DebugVariable = server.DebugVariable
)

// Re-export server constructor
var (
	NewServer = server.New
)

// Re-export bundle functions for MCP integration
var (
	IsBundled                = bundle.IsBundled
	BundleListFiles          = bundle.ListFilesInDir
	BundleListFilesRecursive = bundle.ListFilesInDirRecursive
	BundleListFilesWithInfo  = bundle.ListFilesWithInfo
	BundleReadFile           = bundle.ReadFile
	TypeName                 = lua.TypeName
)

// Re-export bundle types
type BundleFileInfo = bundle.FileInfo

// Re-export Lua utilities
var (
	LuaToGo = lua.LuaToGo
)
