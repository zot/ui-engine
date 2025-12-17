// Package lua provides wrapper support for variable value transformation.
// CRC: crc-Wrapper.md, crc-Variable.md
// Spec: protocol.md
// Sequence: seq-wrapper-transform.md
package lua

import (
	"fmt"
	"sync"
)

// WrapperVariable provides the interface wrappers need from a variable.
// The variable reference is passed to the wrapper constructor.
type WrapperVariable interface {
	GetID() int64
	GetValue() interface{}
	GetProperty(name string) string
}

// --- Create Factory Registry ---

// CreateFactory creates a new object instance from a value.
type CreateFactory func(runtime *Runtime, value interface{}) interface{}

var globalCreateFactories = struct {
	factories map[string]CreateFactory
	mu        sync.RWMutex
}{
	factories: make(map[string]CreateFactory),
}

// RegisterCreateFactory registers a Go create factory globally.
func RegisterCreateFactory(typeName string, factory CreateFactory) {
	globalCreateFactories.mu.Lock()
	defer globalCreateFactories.mu.Unlock()
	globalCreateFactories.factories[typeName] = factory
}

// GetGlobalCreateFactory retrieves a globally registered create factory.
func GetGlobalCreateFactory(typeName string) (CreateFactory, bool) {
	globalCreateFactories.mu.RLock()
	defer globalCreateFactories.mu.RUnlock()
	factory, ok := globalCreateFactories.factories[typeName]
	return factory, ok
}

// --- Wrapper Factory Registry ---

// WrapperFactory creates a new wrapper instance for a variable.
type WrapperFactory func(runtime *Runtime, variable WrapperVariable) interface{}

var globalWrapperFactories = struct {
	factories map[string]WrapperFactory
	mu        sync.RWMutex
}{
	factories: make(map[string]WrapperFactory),
}

// RegisterWrapperType registers a Go wrapper factory globally.
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
func (r *WrapperRegistry) Get(typeName string) (WrapperFactory, bool) {
	r.mu.RLock()
	factory, ok := r.wrappers[typeName]
	r.mu.RUnlock()
	if ok {
		return factory, true
	}
	return GetGlobalWrapperFactory(typeName)
}

// WrapperManager handles wrapper creation for variables.
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
func (m *WrapperManager) CreateWrapper(variable WrapperVariable) (interface{}, error) {
	wrapperType := variable.GetProperty("wrapper")
	if wrapperType == "" {
		return nil, nil // No wrapper
	}

	factory, ok := m.registry.Get(wrapperType)
	if ok {
		wrapper := factory(m.runtime, variable)
		return wrapper, nil
	}

	if m.runtime != nil {
		luaTable := m.runtime.GetGlobalTable(wrapperType)
		if luaTable != nil {
			return NewLuaWrapper(m.runtime, luaTable, variable), nil
		}
	}

	return nil, fmt.Errorf("unknown wrapper type: %s", wrapperType)
}

// LuaWrapper wraps a Lua table as a Go Wrapper interface.
type LuaWrapper struct {
	runtime  *Runtime
	template luaTable // The registered Lua table (prototype)
	instance luaTable // The instance for this variable
	variable WrapperVariable
}

// luaTable is an interface to abstract Lua tables for testing
type luaTable interface{}

// NewLuaWrapper creates a wrapper from a Lua table definition.
func NewLuaWrapper(runtime *Runtime, template luaTable, variable WrapperVariable) *LuaWrapper {
	return &LuaWrapper{
		runtime:  runtime,
		template: template,
		instance: template, // For now, use template directly (stateless)
		variable: variable,
	}
}
