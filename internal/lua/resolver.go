// CRC: crc-LuaResolver.md
// Spec: libraries.md
// Sequence: seq-lua-resolve.md
package lua

import (
	"fmt"

	changetracker "github.com/zot/change-tracker"
	lua "github.com/yuin/gopher-lua"
)

// LuaResolver implements changetracker.Resolver for Lua tables.
// It navigates Lua tables using GetField/RawGetInt and converts values appropriately.
type LuaResolver struct {
	L *lua.LState
}

// Ensure LuaResolver implements the Resolver interface.
var _ changetracker.Resolver = (*LuaResolver)(nil)

// Get retrieves a value from a Lua table at the given path element.
// Path elements can be:
//   - string: table field name, or method call if ends with "()"
//   - int: array index (0-based, converted to Lua's 1-based)
// Spec: viewdefs.md - Method calls: method() or method(_)
func (r *LuaResolver) Get(obj any, pathElement any) (any, error) {
	tbl, ok := obj.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("LuaResolver.Get: expected *lua.LTable, got %T", obj)
	}

	var val lua.LValue
	switch pe := pathElement.(type) {
	case string:
		// Check for method call syntax: name() or name(_)
		if isMethodCall(pe) {
			return r.callMethod(tbl, pe)
		}
		val = r.L.GetField(tbl, pe)
	case int:
		val = r.L.RawGetInt(tbl, pe+1) // Lua is 1-indexed
	default:
		return nil, fmt.Errorf("LuaResolver.Get: unsupported path element type %T", pathElement)
	}

	return r.luaValueToGo(val)
}

// isMethodCall checks if a path element is a method call (ends with "()").
func isMethodCall(s string) bool {
	return len(s) > 2 && s[len(s)-2:] == "()" ||
	       len(s) > 3 && s[len(s)-3:] == "(_)"
}

// callMethod calls a method on a Lua table and returns the result.
// Supports: method() - no args, method(_) - with value arg (not yet implemented)
func (r *LuaResolver) callMethod(tbl *lua.LTable, methodCall string) (any, error) {
	// Parse method name from "name()" or "name(_)"
	var methodName string
	var hasArg bool
	if len(methodCall) > 3 && methodCall[len(methodCall)-3:] == "(_)" {
		methodName = methodCall[:len(methodCall)-3]
		hasArg = true
	} else if len(methodCall) > 2 && methodCall[len(methodCall)-2:] == "()" {
		methodName = methodCall[:len(methodCall)-2]
		hasArg = false
	} else {
		return nil, fmt.Errorf("invalid method call syntax: %s", methodCall)
	}

	// Get the method from the table (checks metatable too)
	method := r.L.GetField(tbl, methodName)
	if method == lua.LNil {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	fn, ok := method.(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", methodName)
	}

	// Call the method with self (the table) as first argument
	r.L.Push(fn)
	r.L.Push(tbl) // self
	nargs := 1
	if hasArg {
		// TODO: Pass the update value as argument
		// For now, just call with self only
		_ = hasArg
	}

	if err := r.L.PCall(nargs, 1, nil); err != nil {
		return nil, fmt.Errorf("method call failed: %w", err)
	}

	// Get the result
	result := r.L.Get(-1)
	r.L.Pop(1)

	return r.luaValueToGo(result)
}

// Call invokes a zero-argument method on a Lua table and returns the result.
// Used for computed getters like compute().
func (r *LuaResolver) Call(obj any, methodName string) (any, error) {
	tbl, ok := obj.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("LuaResolver.Call: expected *lua.LTable, got %T", obj)
	}

	// Get the method from the table (checks metatable too)
	method := r.L.GetField(tbl, methodName)
	if method == lua.LNil {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	fn, ok := method.(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", methodName)
	}

	// Call the method with self (the table) as first argument
	r.L.Push(fn)
	r.L.Push(tbl) // self

	if err := r.L.PCall(1, 1, nil); err != nil {
		return nil, fmt.Errorf("method call failed: %w", err)
	}

	// Get the result
	result := r.L.Get(-1)
	r.L.Pop(1)

	return r.luaValueToGo(result)
}

// CallWith invokes a one-argument method on a Lua table with the given value.
// Used for computed setters like setValue(_).
func (r *LuaResolver) CallWith(obj any, methodName string, value any) error {
	tbl, ok := obj.(*lua.LTable)
	if !ok {
		return fmt.Errorf("LuaResolver.CallWith: expected *lua.LTable, got %T", obj)
	}

	// Get the method from the table (checks metatable too)
	method := r.L.GetField(tbl, methodName)
	if method == lua.LNil {
		return fmt.Errorf("method %s not found", methodName)
	}

	fn, ok := method.(*lua.LFunction)
	if !ok {
		return fmt.Errorf("%s is not a function", methodName)
	}

	// Call the method with self and value arguments
	r.L.Push(fn)
	r.L.Push(tbl)                    // self
	r.L.Push(r.goToLuaValue(value)) // value argument

	if err := r.L.PCall(2, 0, nil); err != nil {
		return fmt.Errorf("method call failed: %w", err)
	}

	return nil
}

// Set assigns a value in a Lua table at the given path element.
func (r *LuaResolver) Set(obj any, pathElement any, value any) error {
	tbl, ok := obj.(*lua.LTable)
	if !ok {
		return fmt.Errorf("LuaResolver.Set: expected *lua.LTable, got %T", obj)
	}

	lval := r.goToLuaValue(value)

	switch pe := pathElement.(type) {
	case string:
		r.L.SetField(tbl, pe, lval)
	case int:
		r.L.RawSetInt(tbl, pe+1, lval) // Lua is 1-indexed
	default:
		return fmt.Errorf("LuaResolver.Set: unsupported path element type %T", pathElement)
	}

	return nil
}

// luaValueToGo converts a Lua value to a Go value.
// - Primitives: bool, number, string -> Go equivalents
// - Array tables: -> []any (elements can be *lua.LTable refs for objects)
// - Object tables: -> keep as *lua.LTable (will be registered by tracker)
// - Nested arrays: ERROR
func (r *LuaResolver) luaValueToGo(val lua.LValue) (any, error) {
	switch v := val.(type) {
	case lua.LBool:
		return bool(v), nil
	case lua.LNumber:
		return float64(v), nil
	case lua.LString:
		return string(v), nil
	case *lua.LTable:
		if r.isArray(v) {
			return r.tableToSlice(v)
		}
		// Object table: return as *lua.LTable for tracker registration
		return v, nil
	case *lua.LNilType:
		return nil, nil
	default:
		return nil, nil
	}
}

// isArray checks if a Lua table is an array (sequential integer keys starting at 1).
// Returns true if the table has only numeric keys with no string keys (excluding _ prefixed).
func (r *LuaResolver) isArray(tbl *lua.LTable) bool {
	hasNumericKeys := false
	hasStringKeys := false

	tbl.ForEach(func(key, _ lua.LValue) {
		switch k := key.(type) {
		case lua.LNumber:
			hasNumericKeys = true
		case lua.LString:
			// Skip internal fields (prefixed with _)
			keyStr := string(k)
			if len(keyStr) > 0 && keyStr[0] != '_' {
				hasStringKeys = true
			}
		}
	})

	// Pure array: only numeric keys, no string keys
	return hasNumericKeys && !hasStringKeys
}

// tableToSlice converts a Lua array table to a Go slice.
// Elements can be *lua.LTable refs (for objects that will become {"obj": id}).
// Nested arrays are an error.
func (r *LuaResolver) tableToSlice(tbl *lua.LTable) ([]any, error) {
	length := tbl.Len()
	result := make([]any, length)

	for i := 1; i <= length; i++ {
		elem := r.L.RawGetInt(tbl, i)

		if elemTbl, ok := elem.(*lua.LTable); ok {
			if r.isArray(elemTbl) {
				return nil, fmt.Errorf("nested arrays not supported")
			}
			// Keep table ref for tracker registration
			result[i-1] = elemTbl
		} else {
			val, err := r.luaValueToGo(elem)
			if err != nil {
				return nil, err
			}
			result[i-1] = val
		}
	}

	return result, nil
}

// CreateWrapper creates a wrapper object for the given variable.
// The wrapper stands in for the variable's value when child variables navigate paths.
// Returns the existing wrapper if one exists (for wrapper reuse).
// CRC: crc-LuaResolver.md
// Spec: protocol.md (Variable Wrappers section)
func (r *LuaResolver) CreateWrapper(variable *changetracker.Variable) any {
	// Check if wrapper property is set
	wrapperType := variable.GetProperty("wrapper")
	if wrapperType == "" {
		return nil
	}

	// Check for existing wrapper
	if existing := variable.WrapperValue; existing != nil {
		// Update wrapper's value property (reuse pattern)
		if luaWrapper, ok := existing.(*lua.LTable); ok {
			r.L.SetField(luaWrapper, "value", r.goToLuaValue(variable.Value))
			// Call sync() if it exists (for ViewList sync)
			syncFn := r.L.GetField(luaWrapper, "sync")
			if syncFn != lua.LNil {
				if fn, ok := syncFn.(*lua.LFunction); ok {
					r.L.Push(fn)
					r.L.Push(luaWrapper)
					r.L.PCall(1, 0, nil) // ignore errors
				}
			}
		}
		return existing
	}

	// Look up wrapper type in Lua globals
	wrapperClass := r.L.GetGlobal(wrapperType)
	if wrapperClass == lua.LNil {
		return nil // Wrapper type not found
	}

	wrapperTable, ok := wrapperClass.(*lua.LTable)
	if !ok {
		return nil // Not a table
	}

	// Check for 'new' method
	newFn := r.L.GetField(wrapperTable, "new")
	if newFn == lua.LNil {
		return nil // No new() method
	}

	fn, ok := newFn.(*lua.LFunction)
	if !ok {
		return nil // new is not a function
	}

	// Create a LuaVariable wrapper to pass to the constructor
	luaVar := r.createLuaVariableWrapper(variable)

	// Call WrapperType:new(variable)
	r.L.Push(fn)
	r.L.Push(wrapperTable) // self (the class table)
	r.L.Push(luaVar)       // variable argument

	if err := r.L.PCall(2, 1, nil); err != nil {
		return nil // Constructor failed
	}

	// Get the result
	result := r.L.Get(-1)
	r.L.Pop(1)

	if result == lua.LNil {
		return nil
	}

	// Store wrapper on variable for reuse
	if luaWrapper, ok := result.(*lua.LTable); ok {
		variable.WrapperValue = luaWrapper
		return luaWrapper
	}

	return result
}

// createLuaVariableWrapper creates a Lua table that wraps a change-tracker Variable.
// This provides the Lua-accessible interface to the Variable.
func (r *LuaResolver) createLuaVariableWrapper(v *changetracker.Variable) *lua.LTable {
	wrapper := r.L.NewTable()

	// Store the variable ID for reference
	r.L.SetField(wrapper, "_id", lua.LNumber(v.ID))

	// getValue() - returns the current value
	r.L.SetField(wrapper, "getValue", r.L.NewFunction(func(L *lua.LState) int {
		L.Push(r.goToLuaValue(v.Value))
		return 1
	}))

	// getProperty(name) - returns a property value
	r.L.SetField(wrapper, "getProperty", r.L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		prop := v.GetProperty(name)
		if prop == "" {
			L.Push(lua.LNil)
		} else {
			L.Push(lua.LString(prop))
		}
		return 1
	}))

	// getWrapper() - returns existing wrapper or nil
	r.L.SetField(wrapper, "getWrapper", r.L.NewFunction(func(L *lua.LState) int {
		existing := v.WrapperValue
		if existing == nil {
			L.Push(lua.LNil)
		} else if tbl, ok := existing.(*lua.LTable); ok {
			L.Push(tbl)
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))

	return wrapper
}

// goToLuaValue converts a Go value to a Lua value.
func (r *LuaResolver) goToLuaValue(value any) lua.LValue {
	if value == nil {
		return lua.LNil
	}

	switch v := value.(type) {
	case bool:
		return lua.LBool(v)
	case float64:
		return lua.LNumber(v)
	case float32:
		return lua.LNumber(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case *lua.LTable:
		return v
	case []any:
		tbl := r.L.NewTable()
		for i, elem := range v {
			r.L.RawSetInt(tbl, i+1, r.goToLuaValue(elem))
		}
		return tbl
	default:
		return lua.LNil
	}
}
