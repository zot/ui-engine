// CRC: crc-Backend.md
// Spec: main.md (UI Server Architecture section)
// Package backend provides the Backend interface and implementations for
// the UI server's backend layer. Each session has one Backend instance.
package backend

import (
	"encoding/json"

	changetracker "github.com/zot/change-tracker"
)

// WatchResult indicates whether a watch should be forwarded to backend.
type WatchResult struct {
	ShouldForward bool // True if tally changed 0->1 for bound variables
	Count         int  // New watch count
}

// UnwatchResult indicates whether an unwatch should be forwarded to backend.
type UnwatchResult struct {
	ShouldForward bool // True if tally changed 1->0 for bound variables
	Count         int  // New watch count
}

// VariableUpdate represents a detected change to be sent to the frontend.
type VariableUpdate struct {
	VarID int64
	Value json.RawMessage
}

// Backend is the interface for hosted (Lua) and proxied backends.
// Each session has exactly one Backend instance.
// CRC: crc-Backend.md
type Backend interface {
	// Watch subscribes a connection to variable changes.
	// For LuaBackend: manages tally, registers with tracker if new.
	// For ProxiedBackend: relays to external backend.
	Watch(varID int64, connectionID string) WatchResult

	// Unwatch removes a connection's subscription to variable changes.
	// For LuaBackend: decrements tally, unregisters from tracker if zero.
	// For ProxiedBackend: relays to external backend.
	Unwatch(varID int64, connectionID string) UnwatchResult

	// UnwatchAll removes all watches for a connection (on disconnect).
	// Returns list of variable IDs that were unwatched.
	UnwatchAll(connectionID string) []int64

	// GetWatchers returns all connection IDs watching a variable.
	GetWatchers(varID int64) []string

	// GetWatcherCount returns the current observer count for a variable.
	GetWatcherCount(varID int64) int

	// DetectChanges computes and returns changes for watched variables.
	// Only meaningful for LuaBackend; ProxiedBackend returns nil.
	DetectChanges() []VariableUpdate

	// GetTracker returns the change-tracker instance for this session.
	// Only meaningful for LuaBackend; ProxiedBackend returns nil.
	GetTracker() *changetracker.Tracker

	// SetInactive marks a variable as inactive (updates not relayed).
	SetInactive(varID int64, inactive bool)

	// IsInactive checks if a variable or any ancestor is inactive.
	IsInactive(varID int64) bool

	// Shutdown cleans up backend resources.
	Shutdown()
}
