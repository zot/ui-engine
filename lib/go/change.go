// CRC: crc-ChangeDetector.md
// Spec: libraries.md
package uiclient

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

// ChangeDetector tracks variable changes and sends updates.
type ChangeDetector struct {
	conn             *Connection
	navigator        *PathNavigator
	watchedVariables map[int64]string        // varID -> path
	previousValues   map[int64]interface{}   // varID -> last known value
	pendingRefresh   bool
	throttleInterval time.Duration
	lastRefresh      time.Time
	mu               sync.RWMutex
	refreshMu        sync.Mutex
	rootObject       interface{}
}

// NewChangeDetector creates a new change detector.
func NewChangeDetector(conn *Connection, nav *PathNavigator) *ChangeDetector {
	return &ChangeDetector{
		conn:             conn,
		navigator:        nav,
		watchedVariables: make(map[int64]string),
		previousValues:   make(map[int64]interface{}),
		throttleInterval: 50 * time.Millisecond, // Default throttle
	}
}

// SetRootObject sets the root object for path resolution.
func (d *ChangeDetector) SetRootObject(root interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rootObject = root
}

// SetThrottleInterval sets the minimum time between refreshes.
func (d *ChangeDetector) SetThrottleInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttleInterval = interval
}

// AddWatch starts tracking a variable for changes.
func (d *ChangeDetector) AddWatch(varID int64, path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.watchedVariables[varID] = path

	// Capture initial value
	if d.rootObject != nil {
		if val, err := d.navigator.Resolve(d.rootObject, path); err == nil {
			d.previousValues[varID] = d.cloneValue(val)
		}
	}
}

// RemoveWatch stops tracking a variable.
func (d *ChangeDetector) RemoveWatch(varID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.watchedVariables, varID)
	delete(d.previousValues, varID)
}

// IsWatched checks if a variable is being watched.
func (d *ChangeDetector) IsWatched(varID int64) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.watchedVariables[varID]
	return ok
}

// Refresh computes values for all watched variables and detects changes.
// Returns the number of changed variables.
func (d *ChangeDetector) Refresh() int {
	d.refreshMu.Lock()
	defer d.refreshMu.Unlock()

	d.mu.Lock()
	root := d.rootObject
	watched := make(map[int64]string)
	for k, v := range d.watchedVariables {
		watched[k] = v
	}
	d.mu.Unlock()

	if root == nil {
		return 0
	}

	changes := make(map[int64]interface{})

	for varID, path := range watched {
		currentVal, err := d.navigator.Resolve(root, path)
		if err != nil {
			continue
		}

		d.mu.RLock()
		prevVal, hasPrev := d.previousValues[varID]
		d.mu.RUnlock()

		if !hasPrev || !d.valuesEqual(prevVal, currentVal) {
			changes[varID] = currentVal
		}
	}

	// Send updates and update previous values
	for varID, val := range changes {
		d.sendUpdate(varID, val)

		d.mu.Lock()
		d.previousValues[varID] = d.cloneValue(val)
		d.mu.Unlock()
	}

	d.mu.Lock()
	d.lastRefresh = time.Now()
	d.pendingRefresh = false
	d.mu.Unlock()

	return len(changes)
}

// ScheduleRefresh queues a background-triggered refresh with throttling.
func (d *ChangeDetector) ScheduleRefresh() {
	d.mu.Lock()
	if d.pendingRefresh {
		d.mu.Unlock()
		return
	}

	elapsed := time.Since(d.lastRefresh)
	if elapsed < d.throttleInterval {
		d.pendingRefresh = true
		d.mu.Unlock()

		// Schedule delayed refresh
		time.AfterFunc(d.throttleInterval-elapsed, func() {
			d.Refresh()
		})
		return
	}

	d.mu.Unlock()

	// Refresh immediately
	go d.Refresh()
}

// AfterMessage triggers refresh after client message receipt.
func (d *ChangeDetector) AfterMessage() {
	d.ScheduleRefresh()
}

// sendUpdate sends an update message for a changed variable.
func (d *ChangeDetector) sendUpdate(varID int64, value interface{}) {
	if d.conn == nil || !d.conn.IsConnected() {
		return
	}

	d.conn.Update(varID, value, nil)
}

// valuesEqual compares two values for equality.
func (d *ChangeDetector) valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try JSON comparison for complex types
	aJSON, aErr := json.Marshal(a)
	bJSON, bErr := json.Marshal(b)

	if aErr == nil && bErr == nil {
		return string(aJSON) == string(bJSON)
	}

	// Fall back to reflect.DeepEqual
	return reflect.DeepEqual(a, b)
}

// cloneValue creates a deep copy of a value for comparison.
func (d *ChangeDetector) cloneValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	// Use JSON round-trip for deep copy
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var clone interface{}
	if err := json.Unmarshal(data, &clone); err != nil {
		return v
	}

	return clone
}

// WatchedCount returns the number of watched variables.
func (d *ChangeDetector) WatchedCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.watchedVariables)
}

// Clear removes all watches and resets state.
func (d *ChangeDetector) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.watchedVariables = make(map[int64]string)
	d.previousValues = make(map[int64]interface{})
	d.pendingRefresh = false
}
