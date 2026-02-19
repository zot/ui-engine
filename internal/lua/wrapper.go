// Package lua provides wrapper support for variable value transformation.
// CRC: crc-Wrapper.md, crc-Variable.md
// Spec: protocol.md
// Sequence: seq-wrapper-transform.md
//
// WrapperRegistry and factory functions for creating wrapper instances.
// Note: WrapperManager was removed - wrapper functionality is per-session via LuaSession.
package lua

import (
	"reflect"
	"sync"
)

// --- Create Factory Registry ---

// CreateFactory creates a new object instance from a value.
type CreateFactory func(session *LuaSession, value interface{}) interface{}

var globalCreateFactories = struct {
	factories map[string]CreateFactory
	mu        sync.RWMutex
}{
	factories: make(map[string]CreateFactory),
}

// RegisterCreateFactory registers a Go create factory globally.
func RegisterCreateFactory(typeName string, typ reflect.Type, factory CreateFactory) {
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
type WrapperFactory func(session *LuaSession, variable *TrackerVariableAdapter) interface{}

var globalWrapperFactories = struct {
	factories map[string]WrapperFactory
	mu        sync.RWMutex
}{
	factories: make(map[string]WrapperFactory),
}

// RegisterWrapperType registers a Go wrapper factory globally.
func RegisterWrapperType(typeName string, typ reflect.Type, factory WrapperFactory) {
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

// Remove removes a wrapper from the registry.
func (r *WrapperRegistry) Remove(typeName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.wrappers, typeName)
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

// LuaWrapper wraps a Lua table as a Go Wrapper interface.
type LuaWrapper struct {
	session  *LuaSession
	template luaTable // The registered Lua table (prototype)
	instance luaTable // The instance for this variable
	variable *TrackerVariableAdapter
}

// luaTable is an interface to abstract Lua tables for testing
type luaTable interface{}

// NewLuaWrapper creates a wrapper from a Lua table definition.
func NewLuaWrapper(session *LuaSession, template luaTable, variable *TrackerVariableAdapter) *LuaWrapper {
	return &LuaWrapper{
		session:  session,
		template: template,
		instance: template, // For now, use template directly (stateless)
		variable: variable,
	}
}

// Value returns the wrapped Lua table instance.
func (w *LuaWrapper) Value() any {
	return w.instance
}
