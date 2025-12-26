// Package cli provides the command-line interface for remote-ui.
// This file re-exports config types from internal/config for public API.
package cli

import (
	"github.com/zot/ui-engine/internal/config"
)

// Re-export config types for public API
type (
	Config        = config.Config
	ServerConfig  = config.ServerConfig
	LuaConfig     = config.LuaConfig
	SessionConfig = config.SessionConfig
	LoggingConfig = config.LoggingConfig
	Duration      = config.Duration
)

// Re-export config functions for public API
var (
	DefaultConfig = config.DefaultConfig
	Load          = config.Load
)
