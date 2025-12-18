// CRC: crc-VariableStore.md
// Spec: protocol.md, data-models.md
package variable

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

// propertyWatcher tracks a callback for a specific property.
type propertyWatcher struct {
	varID    int64
	property string
	callback func(value interface{})
}

// Store manages all variables and their relationships.
type Store struct {
	variables         map[int64]*Variable
	standardVariables map[string]int64 // @NAME -> ID
	nextID            atomic.Int64
	verbosity         int
	propertyWatchers  []propertyWatcher
	mu                sync.RWMutex
}

// NewStore creates a new variable store.
func NewStore() *Store {
	s := &Store{
		variables:         make(map[int64]*Variable),
		standardVariables: make(map[string]int64),
	}
	s.nextID.Store(1)
	return s
}

// SetVerbosity sets the verbosity level for variable operation logging.
func (s *Store) SetVerbosity(level int) {
	s.verbosity = level
}

// CreateOptions holds options for creating a variable.
type CreateOptions struct {
	ID         int64 // If non-zero, use this ID instead of generating one
	ParentID   int64
	Value      json.RawMessage
	Properties map[string]string
	NoWatch    bool
	Unbound    bool
}

// Create creates a new variable and returns its ID.
func (s *Store) Create(opts CreateOptions) (int64, error) {
	var id int64
	if opts.ID != 0 {
		// Use explicit ID and bump counter past it
		id = opts.ID
		for {
			current := s.nextID.Load()
			if current > id {
				break
			}
			if s.nextID.CompareAndSwap(current, id+1) {
				break
			}
		}
	} else {
		id = s.nextID.Add(1) - 1
	}

	v := NewVariable(id)
	v.ParentID = opts.ParentID
	v.Value = opts.Value
	v.Unbound = opts.Unbound

	if opts.Properties != nil {
		v.SetProperties(opts.Properties)
	}

	s.mu.Lock()
	s.variables[id] = v
	verbosity := s.verbosity
	s.mu.Unlock()

	// Log variable creation (verbosity level 3)
	if verbosity >= 3 {
		log.Printf("[v3] Variable created: id=%d parent=%d", id, opts.ParentID)
	}
	// Log variable value (verbosity level 4)
	if verbosity >= 4 && opts.Value != nil {
		log.Printf("[v4] Variable %d value: %s", id, string(opts.Value))
	}

	return id, nil
}

// Get retrieves a variable by ID.
func (s *Store) Get(id int64) (*Variable, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.variables[id]
	return v, ok
}

// GetByName retrieves a standard variable by @NAME.
func (s *Store) GetByName(name string) (*Variable, bool) {
	s.mu.RLock()
	id, ok := s.standardVariables[name]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return s.Get(id)
}

// Update updates a variable's value and/or properties.
func (s *Store) Update(id int64, value json.RawMessage, properties map[string]string) error {
	v, ok := s.Get(id)
	if !ok {
		return fmt.Errorf("variable %d not found", id)
	}

	if value != nil {
		v.SetValue(value)
	}
	if properties != nil {
		v.SetProperties(properties)
	}

	s.mu.RLock()
	verbosity := s.verbosity
	s.mu.RUnlock()

	// Log variable update (verbosity level 3)
	if verbosity >= 3 {
		log.Printf("[v3] Variable updated: id=%d", id)
	}
	// Log variable value (verbosity level 4)
	if verbosity >= 4 && value != nil {
		log.Printf("[v4] Variable %d value: %s", id, string(value))
	}

	// Notify property watchers
	if properties != nil {
		s.notifyPropertyWatchers(id, properties)
	}

	return nil
}

// WatchProperty registers a callback for when a property changes on a variable.
// This implements the VariableUpdater interface for Lua runtime integration.
func (s *Store) WatchProperty(varID int64, property string, callback func(value interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.propertyWatchers = append(s.propertyWatchers, propertyWatcher{
		varID:    varID,
		property: property,
		callback: callback,
	})
}

// notifyPropertyWatchers calls all registered callbacks for changed properties.
func (s *Store) notifyPropertyWatchers(varID int64, properties map[string]string) {
	s.mu.RLock()
	watchers := make([]propertyWatcher, len(s.propertyWatchers))
	copy(watchers, s.propertyWatchers)
	s.mu.RUnlock()

	for _, w := range watchers {
		if w.varID == varID {
			if value, ok := properties[w.property]; ok {
				w.callback(value)
			}
		}
	}
}

// Destroy removes a variable and all its children recursively.
func (s *Store) Destroy(id int64) error {
	// First find all children
	children := s.GetChildren(id)

	// Recursively destroy children
	for _, child := range children {
		if err := s.Destroy(child.ID); err != nil {
			return err
		}
	}

	// Remove the variable
	s.mu.Lock()
	delete(s.variables, id)

	// Remove from standard variables if registered
	for name, varID := range s.standardVariables {
		if varID == id {
			delete(s.standardVariables, name)
			break
		}
	}
	verbosity := s.verbosity
	s.mu.Unlock()

	// Log variable destruction (verbosity level 3)
	if verbosity >= 3 {
		log.Printf("[v3] Variable destroyed: id=%d", id)
	}

	return nil
}

// RegisterStandardVariable associates @NAME with a variable ID.
func (s *Store) RegisterStandardVariable(name string, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.variables[id]; !ok {
		return fmt.Errorf("variable %d not found", id)
	}

	s.standardVariables[name] = id
	return nil
}

// GetChildren returns all variables with the given parent ID.
func (s *Store) GetChildren(parentID int64) []*Variable {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var children []*Variable
	for _, v := range s.variables {
		if v.ParentID == parentID {
			children = append(children, v)
		}
	}
	return children
}

// ResolveObjectReference gets object data for {obj: ID} references.
func (s *Store) ResolveObjectReference(ref json.RawMessage) (json.RawMessage, error) {
	id, ok := GetObjectReferenceID(ref)
	if !ok {
		return nil, fmt.Errorf("not an object reference")
	}

	v, ok := s.Get(id)
	if !ok {
		return nil, fmt.Errorf("object %d not found", id)
	}

	// The value is an interface{}, so we need to marshal it to JSON.
	valBytes, err := json.Marshal(v.GetValue())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object value: %w", err)
	}

	return valBytes, nil
}

// GetAll returns all variables (for debugging/testing).
func (s *Store) GetAll() []*Variable {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vars := make([]*Variable, 0, len(s.variables))
	for _, v := range s.variables {
		vars = append(vars, v)
	}
	return vars
}

// Count returns the number of variables.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.variables)
}

