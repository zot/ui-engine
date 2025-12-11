// CRC: crc-WatchManager.md
// Spec: protocol.md
package variable

import (
	"sync"
)

// WatchManager manages watch subscriptions for variables.
type WatchManager struct {
	watchCounts       map[int64]int        // variable ID -> observer count
	watchers          map[int64][]string   // variable ID -> connection IDs
	inactiveVariables map[int64]struct{}   // variable IDs marked inactive
	store             *Store
	mu                sync.RWMutex
}

// NewWatchManager creates a new WatchManager.
func NewWatchManager(store *Store) *WatchManager {
	return &WatchManager{
		watchCounts:       make(map[int64]int),
		watchers:          make(map[int64][]string),
		inactiveVariables: make(map[int64]struct{}),
		store:             store,
	}
}

// WatchResult indicates whether a watch should be forwarded to backend.
type WatchResult struct {
	ShouldForward bool // True if tally changed 0->1 for bound variables
	Count         int  // New watch count
}

// Watch adds an observer for a variable.
func (wm *WatchManager) Watch(varID int64, connectionID string) WatchResult {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	prevCount := wm.watchCounts[varID]
	wm.watchCounts[varID] = prevCount + 1

	// Add connection to watchers list
	watchers := wm.watchers[varID]
	found := false
	for _, id := range watchers {
		if id == connectionID {
			found = true
			break
		}
	}
	if !found {
		wm.watchers[varID] = append(watchers, connectionID)
	}

	// Check if this is a bound variable
	v, ok := wm.store.Get(varID)
	shouldForward := ok && !v.IsUnbound() && prevCount == 0

	return WatchResult{
		ShouldForward: shouldForward,
		Count:         prevCount + 1,
	}
}

// UnwatchResult indicates whether an unwatch should be forwarded to backend.
type UnwatchResult struct {
	ShouldForward bool // True if tally changed 1->0 for bound variables
	Count         int  // New watch count
}

// Unwatch removes an observer from a variable.
func (wm *WatchManager) Unwatch(varID int64, connectionID string) UnwatchResult {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	prevCount := wm.watchCounts[varID]
	if prevCount <= 0 {
		return UnwatchResult{ShouldForward: false, Count: 0}
	}

	wm.watchCounts[varID] = prevCount - 1
	if prevCount-1 == 0 {
		delete(wm.watchCounts, varID)
	}

	// Remove connection from watchers list
	watchers := wm.watchers[varID]
	for i, id := range watchers {
		if id == connectionID {
			wm.watchers[varID] = append(watchers[:i], watchers[i+1:]...)
			break
		}
	}
	if len(wm.watchers[varID]) == 0 {
		delete(wm.watchers, varID)
	}

	// Check if this is a bound variable
	v, ok := wm.store.Get(varID)
	shouldForward := ok && !v.IsUnbound() && prevCount == 1

	return UnwatchResult{
		ShouldForward: shouldForward,
		Count:         prevCount - 1,
	}
}

// GetWatchers returns all connection IDs watching a variable.
func (wm *WatchManager) GetWatchers(varID int64) []string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	watchers := wm.watchers[varID]
	if watchers == nil {
		return nil
	}

	// Return a copy
	result := make([]string, len(watchers))
	copy(result, watchers)
	return result
}

// GetWatcherCount returns the current observer count for a variable.
func (wm *WatchManager) GetWatcherCount(varID int64) int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.watchCounts[varID]
}

// SetInactive marks a variable as inactive (updates not relayed).
func (wm *WatchManager) SetInactive(varID int64, inactive bool) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if inactive {
		wm.inactiveVariables[varID] = struct{}{}
	} else {
		delete(wm.inactiveVariables, varID)
	}
}

// IsInactive checks if a variable or any ancestor has the inactive property set.
func (wm *WatchManager) IsInactive(varID int64) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.isInactiveUnsafe(varID)
}

// isInactiveUnsafe checks inactive status without locking (caller must hold lock).
func (wm *WatchManager) isInactiveUnsafe(varID int64) bool {
	// Check if this variable is inactive
	if _, ok := wm.inactiveVariables[varID]; ok {
		return true
	}

	// Check ancestors
	v, ok := wm.store.Get(varID)
	if !ok {
		return false
	}

	if v.ParentID != 0 {
		return wm.isInactiveUnsafe(v.ParentID)
	}

	return false
}

// UnwatchAll removes all watches for a connection (e.g., on disconnect).
func (wm *WatchManager) UnwatchAll(connectionID string) []int64 {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	var unwatched []int64

	for varID, watchers := range wm.watchers {
		for i, id := range watchers {
			if id == connectionID {
				wm.watchers[varID] = append(watchers[:i], watchers[i+1:]...)
				wm.watchCounts[varID]--
				if wm.watchCounts[varID] <= 0 {
					delete(wm.watchCounts, varID)
				}
				unwatched = append(unwatched, varID)
				break
			}
		}
		if len(wm.watchers[varID]) == 0 {
			delete(wm.watchers, varID)
		}
	}

	return unwatched
}
