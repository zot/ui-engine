// Package lua provides wrapper support for variable value transformation.
// CRC: crc-Wrapper.md, crc-Variable.md
// Spec: protocol.md
// Sequence: seq-wrapper-transform.md
package lua

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Wrapper is the interface for variable value transformers.
// Wrappers compute the stored value sent to the frontend from raw values.
//
// Wrapper lifecycle:
// 1. Constructor receives the variable (for property access)
// 2. Wrapper instance is stored internally in the variable
// 3. On value changes: ComputeValue(rawValue) returns stored value
// 4. On variable destroy: Destroy() cleans up
type Wrapper interface {
	// ComputeValue transforms the raw value into the stored value.
	// Called when the monitored value changes.
	ComputeValue(rawValue json.RawMessage) (json.RawMessage, error)

	// Destroy cleans up any managed objects when the variable is destroyed.
	Destroy() error
}

// WrapperVariable provides the interface wrappers need from a variable.
// The variable reference is passed to the wrapper constructor.
type WrapperVariable interface {
	GetID() int64
	GetValue() json.RawMessage
	GetProperty(name string) string
}

// WrapperFactory creates a new wrapper instance for a variable.
// The factory receives the runtime and the variable being wrapped.
// The wrapper constructor can access variable properties (e.g., "item" type).
type WrapperFactory func(runtime *Runtime, variable WrapperVariable) Wrapper

// Global wrapper registry for auto-registration via init().
// Go wrappers register themselves here at package initialization.
var globalWrapperFactories = struct {
	factories map[string]WrapperFactory
	mu        sync.RWMutex
}{
	factories: make(map[string]WrapperFactory),
}

// RegisterWrapperType registers a Go wrapper factory globally.
// Called from init() functions to auto-register wrappers.
// This enables frictionless development - just define a wrapper and it's available by name.
func RegisterWrapperType(typeName string, factory WrapperFactory) {
	globalWrapperFactories.mu.Lock()
	defer globalWrapperFactories.mu.Unlock()
	globalWrapperFactories.factories[typeName] = factory
}

// GetGlobalWrapperFactory retrieves a globally registered wrapper factory.
func GetGlobalWrapperFactory(typeName string) (WrapperFactory, bool) {
	globalWrapperFactories.mu.RLock()
	defer globalWrapperFactories.mu.RUnlock()
	factory, ok := globalWrapperFactories.factories[typeName]
	return factory, ok
}

// WrapperRegistry manages registered wrapper types.
// Combines global auto-registered wrappers with instance-specific registrations.
type WrapperRegistry struct {
	wrappers map[string]WrapperFactory
	mu       sync.RWMutex
}

// NewWrapperRegistry creates a new wrapper registry.
func NewWrapperRegistry() *WrapperRegistry {
	return &WrapperRegistry{
		wrappers: make(map[string]WrapperFactory),
	}
}

// Register adds a wrapper factory for a type name (instance-specific).
func (r *WrapperRegistry) Register(typeName string, factory WrapperFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wrappers[typeName] = factory
}

// Get retrieves a wrapper factory by type name.
// Checks instance registry first, then global auto-registered wrappers.
func (r *WrapperRegistry) Get(typeName string) (WrapperFactory, bool) {
	// Check instance registry first
	r.mu.RLock()
	factory, ok := r.wrappers[typeName]
	r.mu.RUnlock()
	if ok {
		return factory, true
	}

	// Fall back to global auto-registered wrappers
	return GetGlobalWrapperFactory(typeName)
}

// WrapperManager handles wrapper creation for variables.
// Note: Wrapper instances are stored internally in variables, not in this manager.
// This manager only handles wrapper creation via the registry.
type WrapperManager struct {
	registry *WrapperRegistry
	runtime  *Runtime
	mu       sync.RWMutex
}

// NewWrapperManager creates a new wrapper manager.
func NewWrapperManager(runtime *Runtime, registry *WrapperRegistry) *WrapperManager {
	return &WrapperManager{
		registry: registry,
		runtime:  runtime,
	}
}

// CreateWrapper creates a new wrapper instance for a variable.
// The wrapper is returned to be stored internally in the variable.
// Returns nil if the variable has no wrapper property.
//
// Wrapper resolution order:
// 1. Check registry for explicitly registered wrapper (Go built-ins like ViewList)
// 2. Look up Lua global by name (auto-discovery for Lua-defined wrappers)
func (m *WrapperManager) CreateWrapper(variable WrapperVariable) (Wrapper, error) {
	wrapperType := variable.GetProperty("wrapper")
	if wrapperType == "" {
		return nil, nil // No wrapper
	}

	// First, check registry for explicitly registered wrappers (Go built-ins)
	factory, ok := m.registry.Get(wrapperType)
	if ok {
		wrapper := factory(m.runtime, variable)
		return wrapper, nil
	}

	// Fall back to Lua global lookup (auto-discovery)
	if m.runtime != nil {
		luaTable := m.runtime.GetGlobalTable(wrapperType)
		if luaTable != nil {
			return NewLuaWrapper(m.runtime, luaTable, variable), nil
		}
	}

	return nil, fmt.Errorf("unknown wrapper type: %s", wrapperType)
}

// ComputeStoredValue computes the stored value for a variable.
// If wrapper is provided, uses wrapper.ComputeValue(rawValue).
// Otherwise returns the raw value unchanged.
func ComputeStoredValue(wrapper Wrapper, rawValue json.RawMessage) (json.RawMessage, error) {
	if wrapper == nil {
		// No wrapper - return raw value
		return rawValue, nil
	}

	return wrapper.ComputeValue(rawValue)
}

// LuaWrapper wraps a Lua table as a Go Wrapper interface.
// Used for wrappers defined in Lua via ui.registerWrapper.
type LuaWrapper struct {
	runtime  *Runtime
	template luaTable // The registered Lua table (prototype)
	instance luaTable // The instance for this variable
	variable WrapperVariable
}

// luaTable is an interface to abstract Lua tables for testing
type luaTable interface{}

// NewLuaWrapper creates a wrapper from a Lua table definition.
// The table should have computeValue(self, rawValue) and optionally destroy(self).
func NewLuaWrapper(runtime *Runtime, template luaTable, variable WrapperVariable) *LuaWrapper {
	return &LuaWrapper{
		runtime:  runtime,
		template: template,
		instance: template, // For now, use template directly (stateless)
		variable: variable,
	}
}

// ComputeValue calls the Lua table's computeValue method.
func (lw *LuaWrapper) ComputeValue(rawValue json.RawMessage) (json.RawMessage, error) {
	if lw.runtime == nil {
		return rawValue, nil
	}

	result, err := lw.runtime.CallLuaWrapperMethod(lw.instance, "computeValue", rawValue)
	if err != nil {
		return nil, err
	}

	// Convert result back to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wrapper result: %w", err)
	}

	return data, nil
}

// Destroy calls the Lua table's destroy method if it exists.
func (lw *LuaWrapper) Destroy() error {
	if lw.runtime == nil {
		return nil
	}

	_, _ = lw.runtime.CallLuaWrapperMethod(lw.instance, "destroy")
	return nil
}

// Ensure LuaWrapper implements Wrapper
var _ Wrapper = (*LuaWrapper)(nil)
