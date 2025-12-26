// CRC: crc-LuaBackend.md
// Spec: main.md (UI Server Architecture - Hosted Backend), protocol.md (Session-Based Communication)
// Sequence: seq-session-create-backend.md, seq-backend-watch.md, seq-backend-detect-changes.md
package backend

import (
	"encoding/json"
	"sync"

	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui-engine/internal/config"
)

// LuaBackend implements Backend for hosted Lua sessions.
// It owns a per-session change-tracker and manages watch subscriptions.
// This replaces the global WatchManager with per-session watch management.
type LuaBackend struct {
	config            *config.Config
	sessionID         string
	tracker           *changetracker.Tracker
	watchCounts       map[int64]int      // variable ID -> observer count
	watchers          map[int64][]string // variable ID -> connection IDs
	inactiveVariables map[int64]struct{} // variable IDs marked inactive
	varToSession      map[int64]struct{} // track variables owned by this session
	mu                sync.RWMutex
}

// NewLuaBackend creates a new LuaBackend for a session.
// The resolver is used for path navigation and wrapper creation.
// CRC: crc-LuaBackend.md
// Sequence: seq-session-create-backend.md
func NewLuaBackend(cfg *config.Config, sessionID string, resolver changetracker.Resolver) *LuaBackend {
	tracker := changetracker.NewTracker()
	tracker.Resolver = resolver

	return &LuaBackend{
		config:            cfg,
		sessionID:         sessionID,
		tracker:           tracker,
		watchCounts:       make(map[int64]int),
		watchers:          make(map[int64][]string),
		inactiveVariables: make(map[int64]struct{}),
		varToSession:      make(map[int64]struct{}),
	}
}

// Log logs a message via the config.
func (lb *LuaBackend) Log(level int, format string, args ...interface{}) {
	lb.config.Log(level, format, args...)
}

// GetSessionID returns the session ID.
func (lb *LuaBackend) GetSessionID() string {
	return lb.sessionID
}

// GetTracker returns the change-tracker instance for this session.
func (lb *LuaBackend) GetTracker() *changetracker.Tracker {
	return lb.tracker
}

// Watch adds an observer for a variable.
// Returns WatchResult indicating if the watch should be forwarded (for bound variables).
// CRC: crc-LuaBackend.md
// Sequence: seq-backend-watch.md
func (lb *LuaBackend) Watch(varID int64, connectionID string) WatchResult {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	prevCount := lb.watchCounts[varID]
	lb.watchCounts[varID] = prevCount + 1

	// Add connection to watchers list
	watchers := lb.watchers[varID]
	found := false
	for _, id := range watchers {
		if id == connectionID {
			found = true
			break
		}
	}
	if !found {
		lb.watchers[varID] = append(watchers, connectionID)
	}

	// On 0->1 transition, mark variable as active for change detection
	if prevCount == 0 {
		v := lb.tracker.GetVariable(varID)
		if v != nil {
			v.SetActive(true)
		}
	}

	// For LuaBackend, we don't forward watch messages since we handle variables locally
	// ShouldForward would be true for ProxiedBackend
	return WatchResult{
		ShouldForward: false,
		Count:         prevCount + 1,
	}
}

// Unwatch removes an observer from a variable.
// Returns UnwatchResult indicating if the unwatch should be forwarded (for bound variables).
func (lb *LuaBackend) Unwatch(varID int64, connectionID string) UnwatchResult {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	prevCount := lb.watchCounts[varID]
	if prevCount <= 0 {
		return UnwatchResult{ShouldForward: false, Count: 0}
	}

	lb.watchCounts[varID] = prevCount - 1
	if prevCount-1 == 0 {
		delete(lb.watchCounts, varID)
	}

	// Remove connection from watchers list
	watchers := lb.watchers[varID]
	for i, id := range watchers {
		if id == connectionID {
			lb.watchers[varID] = append(watchers[:i], watchers[i+1:]...)
			break
		}
	}
	if len(lb.watchers[varID]) == 0 {
		delete(lb.watchers, varID)
	}

	// On 1->0 transition, mark variable as inactive for change detection
	if prevCount == 1 {
		v := lb.tracker.GetVariable(varID)
		if v != nil {
			v.SetActive(false)
		}
	}

	// For LuaBackend, we don't forward unwatch messages
	return UnwatchResult{
		ShouldForward: false,
		Count:         prevCount - 1,
	}
}

// UnwatchAll removes all watches for a connection (on disconnect).
// Returns list of variable IDs that were unwatched.
func (lb *LuaBackend) UnwatchAll(connectionID string) []int64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var unwatched []int64
	var deactivated []int64

	for varID, watchers := range lb.watchers {
		for i, id := range watchers {
			if id == connectionID {
				lb.watchers[varID] = append(watchers[:i], watchers[i+1:]...)
				lb.watchCounts[varID]--
				if lb.watchCounts[varID] <= 0 {
					delete(lb.watchCounts, varID)
					deactivated = append(deactivated, varID)
				}
				unwatched = append(unwatched, varID)
				break
			}
		}
		if len(lb.watchers[varID]) == 0 {
			delete(lb.watchers, varID)
		}
	}

	// Mark deactivated variables as inactive in tracker
	for _, varID := range deactivated {
		v := lb.tracker.GetVariable(varID)
		if v != nil {
			v.SetActive(false)
		}
	}

	return unwatched
}

// GetWatchers returns all connection IDs watching a variable.
func (lb *LuaBackend) GetWatchers(varID int64) []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	watchers := lb.watchers[varID]
	if watchers == nil {
		return nil
	}

	// Return a copy
	result := make([]string, len(watchers))
	copy(result, watchers)
	return result
}

// GetWatcherCount returns the current observer count for a variable.
func (lb *LuaBackend) GetWatcherCount(varID int64) int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.watchCounts[varID]
}

// SetInactive marks a variable as inactive (updates not relayed).
func (lb *LuaBackend) SetInactive(varID int64, inactive bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if inactive {
		lb.inactiveVariables[varID] = struct{}{}
	} else {
		delete(lb.inactiveVariables, varID)
	}
}

// IsInactive checks if a variable or any ancestor has the inactive property set.
func (lb *LuaBackend) IsInactive(varID int64) bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.isInactiveUnsafe(varID)
}

// isInactiveUnsafe checks inactive status without locking (caller must hold lock).
func (lb *LuaBackend) isInactiveUnsafe(varID int64) bool {
	// Check if this variable is inactive
	if _, ok := lb.inactiveVariables[varID]; ok {
		return true
	}

	// Check ancestors via tracker
	v := lb.tracker.GetVariable(varID)
	if v == nil {
		return false
	}

	if v.ParentID != 0 {
		return lb.isInactiveUnsafe(v.ParentID)
	}

	return false
}

// DetectChanges computes and returns changes for watched variables.
// CRC: crc-LuaBackend.md
// Sequence: seq-backend-detect-changes.md
func (lb *LuaBackend) DetectChanges() []VariableUpdate {
	lb.tracker.DetectChanges()
	changes := lb.tracker.GetChanges()
	if len(changes) == 0 {
		return nil
	}

	var updates []VariableUpdate
	for _, change := range changes {
		if change.ValueChanged {
			v := lb.tracker.GetVariable(change.VariableID)
			if v == nil {
				continue
			}

			jsonBytes, err := lb.tracker.ToValueJSONBytes(v.Value)
			if err != nil {
				continue
			}

			updates = append(updates, VariableUpdate{
				VarID:      change.VariableID,
				Value:      json.RawMessage(jsonBytes),
				Properties: v.Properties,
			})
		}
	}

	return updates
}

// TrackVariable records that this session owns a variable.
// Used for cleanup and session-scoped variable lookup.
func (lb *LuaBackend) TrackVariable(varID int64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.varToSession[varID] = struct{}{}
}

// UntrackVariable removes a variable from session ownership.
func (lb *LuaBackend) UntrackVariable(varID int64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	delete(lb.varToSession, varID)
}

// OwnsVariable checks if this session owns the given variable.
func (lb *LuaBackend) OwnsVariable(varID int64) bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	_, ok := lb.varToSession[varID]
	return ok
}

// Shutdown cleans up backend resources.
func (lb *LuaBackend) Shutdown() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Clear all maps
	lb.watchCounts = nil
	lb.watchers = nil
	lb.inactiveVariables = nil
	lb.varToSession = nil
}
