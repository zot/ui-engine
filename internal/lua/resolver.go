// CRC: crc-LuaResolver.md
// Spec: libraries.md
// Sequence: seq-lua-resolve.md
package lua

import (
	"fmt"
	"reflect"

	lua "github.com/yuin/gopher-lua"
	changetracker "github.com/zot/change-tracker"
)

// LuaResolver implements changetracker.Resolver for Lua tables and Go wrappers.
// It navigates Lua tables using GetField/RawGetInt and converts values appropriately.
type LuaResolver struct {
	Session *LuaSession
}

// Ensure LuaResolver implements the Resolver interface.
var _ changetracker.Resolver = (*LuaResolver)(nil)

// Get retrieves a value from an object at the given path element.
func (r *LuaResolver) Get(obj any, pathElement any) (any, error) {
	// Handle ViewList wrapper
	if vl, ok := obj.(*ViewList); ok {
		if prop, ok := pathElement.(string); ok && prop == "items" {
			switch prop {
			case "items":
				vl.mu.RLock()
				defer vl.mu.RUnlock()
				return r.luaValueToGo(vl.Items)
			}
		}
		return nil, fmt.Errorf("Unknown ViewList property: %v", pathElement)
	}

	slice := reflect.ValueOf(obj)
	// Handle []*ViewListItem slice
	if slice.Kind() == reflect.Array || slice.Kind() == reflect.Slice {
		if index, ok := pathElement.(int); !ok {
			return nil, fmt.Errorf("[]*ViewListItem resolution only supports number indexes")
		} else if index < 0 || index >= slice.Len() {
			return nil, fmt.Errorf("ViewList index %d out of range", index)
		} else {
			//r.Session.Log(4, "  Returning ViewListItem #%d: %v", index, slice.Index(index).Interface())
			return r.luaValueToGo(slice.Index(index).Interface())
		}
	}

	// Handle ViewListItem wrapper
	if vli, ok := obj.(*ViewListItem); ok {
		prop, ok := pathElement.(string)
		if !ok {
			return nil, fmt.Errorf("ViewListItem resolution only supports string property")
		}
		switch prop {
		case "item":
			return r.luaValueToGo(vli.Item)
		case "index":
			return vli.Index, nil
		case "list":
			return vli.List, nil
		case "type":
			return "lua.ViewListItem", nil
		default:
			return nil, fmt.Errorf("Unknown ViewListItem property: %s", prop)
		}
	}

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
		val = r.Session.State.GetField(tbl, pe)
	case int:
		val = r.Session.State.RawGetInt(tbl, pe+1) // Lua is 1-indexed
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
	method := r.Session.State.GetField(tbl, methodName)
	if method == lua.LNil {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	fn, ok := method.(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", methodName)
	}

	// Call the method with self (the table) as first argument
	r.Session.State.Push(fn)
	r.Session.State.Push(tbl) // self
	nargs := 1
	if hasArg {
		// TODO: Pass the update value as argument
		// For now, just call with self only
		_ = hasArg
	}

	if err := r.Session.State.PCall(nargs, 1, nil); err != nil {
		return nil, fmt.Errorf("method call failed: %w", err)
	}

	// Get the result
	result := r.Session.State.Get(-1)
	r.Session.State.Pop(1)

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
	method := r.Session.State.GetField(tbl, methodName)
	if method == lua.LNil {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	fn, ok := method.(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", methodName)
	}

	// Call the method with self (the table) as first argument
	r.Session.State.Push(fn)
	r.Session.State.Push(tbl) // self

	if err := r.Session.State.PCall(1, 1, nil); err != nil {
		return nil, fmt.Errorf("method call failed: %w", err)
	}

	// Get the result
	result := r.Session.State.Get(-1)
	r.Session.State.Pop(1)

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
	method := r.Session.State.GetField(tbl, methodName)
	if method == lua.LNil {
		return fmt.Errorf("method %s not found", methodName)
	}

	fn, ok := method.(*lua.LFunction)
	if !ok {
		return fmt.Errorf("%s is not a function", methodName)
	}

	// Call the method with self and value arguments
	r.Session.State.Push(fn)
	r.Session.State.Push(tbl)              // self
	r.Session.State.Push(r.goToLua(value)) // value argument

	if err := r.Session.State.PCall(2, 0, nil); err != nil {
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

	lval := r.goToLua(value)

	switch pe := pathElement.(type) {
	case string:
		r.Session.State.SetField(tbl, pe, lval)
	case int:
		r.Session.State.RawSetInt(tbl, pe+1, lval) // Lua is 1-indexed
	default:
		return fmt.Errorf("LuaResolver.Set: unsupported path element type %T", pathElement)
	}

	return nil
}

// luaValueToGo converts a Lua value to a Go value for path navigation.
// - Primitives: bool, number, string -> Go equivalents
// - Tables (both arrays and objects): keep as *lua.LTable for navigation
// - ToValueJSON handles the conversion to []any for arrays during serialization
func (r *LuaResolver) luaValueToGo(obj any) (any, error) {
	if val, ok := obj.(lua.LValue); ok {
		switch v := val.(type) {
		case lua.LBool:
			return bool(v), nil
		case lua.LNumber:
			return float64(v), nil
		case lua.LString:
			return string(v), nil
		case *lua.LTable:
			// Keep tables as *lua.LTable for proper navigation and mutation
			// ToValueJSON will convert arrays to []any during serialization
			return v, nil
		case *lua.LNilType:
			return nil, nil
		}
	}
	return obj, nil
}

// tableToSlice converts a Lua array table to a Go slice.
// Elements can be *lua.LTable refs (for objects that will become {"obj": id}).
// Nested arrays are an error.
func (r *LuaResolver) tableToSlice(tbl *lua.LTable) ([]any, error) {
	length := tbl.Len()
	result := make([]any, length)

	for i := 1; i <= length; i++ {
		elem := r.Session.State.RawGetInt(tbl, i)

		if elemTbl, ok := elem.(*lua.LTable); ok {
			if r.Session.isArray(elemTbl) {
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

// CreateValue creates a value for the given variable.
// CRC: crc-LuaResolver.md
// Spec: protocol.md (Variable Wrappers section)
func (r *LuaResolver) CreateValue(variable *changetracker.Variable, typ string, value any) any {
	if typ == "" {
		return nil
	} else if factory, ok := GetGlobalCreateFactory(typ); ok {
		return factory(r.Session, value)
	} else if valueClass := r.Session.State.GetGlobal(typ); valueClass == lua.LNil {
		return nil // No Lua global by that name
	} else if valueTable, ok := valueClass.(*lua.LTable); !ok {
		return nil // Not a table
	} else if newFn := r.Session.State.GetField(valueTable, "new"); newFn == lua.LNil {
		return nil // No new() method
	} else if fn, ok := newFn.(*lua.LFunction); !ok {
		return nil // new is not a function
	} else {
		// Call WrapperType:new(variable)
		r.Session.State.Push(fn)
		r.Session.State.Push(valueTable)       // self (the class table)
		r.Session.State.Push(r.goToLua(value)) // the value arg
		if err := r.Session.State.PCall(2, 1, nil); err != nil {
			return nil // Constructor failed
		}
		// Get the result
		result := r.Session.State.Get(-1)
		r.Session.State.Pop(1)
		if result == lua.LNil {
			return nil // Constructor returned nil
		}
		return result
	}
}

// CreateWrapper creates a wrapper object for the given variable.
// The wrapper stands in for the variable's value when child variables navigate paths.
// Returns the existing wrapper if one exists (for wrapper reuse).
// CRC: crc-LuaResolver.md
// Spec: protocol.md (Variable Wrappers section)
func (r *LuaResolver) CreateWrapper(variable *changetracker.Variable) any {
	r.Session.Log(4, "CREATE WRAPPER %#v", variable)
	// Check if wrapper property is set
	wrapperType := variable.GetProperty("wrapper")
	if wrapperType == "" {
		return nil
	}

	// Check for existing wrapper
	if existing := variable.WrapperValue; existing != nil {
		// Update wrapper's value property (reuse pattern)
		if luaWrapper, ok := existing.(*lua.LTable); ok {
			r.Session.State.SetField(luaWrapper, "value", r.goToLua(variable.Value))
			// Call sync() if it exists (for ViewList sync)
			syncFn := r.Session.State.GetField(luaWrapper, "sync")
			if syncFn != lua.LNil {
				if fn, ok := syncFn.(*lua.LFunction); ok {
					r.Session.State.Push(fn)
					r.Session.State.Push(luaWrapper)
					r.Session.State.PCall(1, 0, nil) // ignore errors
				}
			}
		} else if vl, ok := existing.(*ViewList); ok {
			// Update Go wrapper
			vl.Update(variable.Value)
		}
		return existing
	}

	// Try Go registry first
	if factory, ok := GetGlobalWrapperFactory(wrapperType); ok {
		if r.Session != nil {
			// Create adapter for WrapperVariable
			wrapperVar := &TrackerVariableAdapter{Variable: variable, Session: r.Session}
			wrapper := factory(r.Session, wrapperVar)
			if wrapper != nil {
				variable.WrapperValue = wrapper
				variable.SetProperty("type", wrapperType)
				return wrapper
			}
		}
	}

	// Look up wrapper type in Lua globals
	wrapperClass := r.Session.State.GetGlobal(wrapperType)
	if wrapperClass == lua.LNil {
		return nil // Wrapper type not found
	}

	wrapperTable, ok := wrapperClass.(*lua.LTable)
	if !ok {
		return nil // Not a table
	}

	// Check for 'new' method
	newFn := r.Session.State.GetField(wrapperTable, "new")
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
	r.Session.State.Push(fn)
	r.Session.State.Push(wrapperTable) // self (the class table)
	r.Session.State.Push(luaVar)       // variable argument

	if err := r.Session.State.PCall(2, 1, nil); err != nil {
		return nil // Constructor failed
	}

	// Get the result
	result := r.Session.State.Get(-1)
	r.Session.State.Pop(1)

	if result == lua.LNil {
		return nil
	}

	// Store wrapper on variable for reuse
	if luaWrapper, ok := result.(*lua.LTable); ok {
		variable.WrapperValue = luaWrapper
		variable.SetProperty("type", wrapperType)
		return luaWrapper
	}

	return result
}

// GetType returns a value's type, given the variable as context.
// CRC: crc-LuaResolver.md
// Spec: protocol.md (Variable Wrappers section)
func (r *LuaResolver) GetType(variable *changetracker.Variable, obj any) string {
	typ := GetType(r.Session.State, obj)
	return typ
}

func GetType(L *lua.LState, obj any) string {
	if lObj, ok := obj.(*lua.LTable); ok {
		// First check metatable
		mt := L.GetMetatable(lObj)
		if mt != lua.LNil {
			if mtTbl, ok := mt.(*lua.LTable); ok {
				if typeVal := L.GetField(mtTbl, "type"); typeVal != lua.LNil {
					return lua.LVAsString(typeVal)
				}
			}
		}
		// Fall back to direct "type" field
		if typeVal := L.GetField(lObj, "type"); typeVal != lua.LNil {
			return lua.LVAsString(typeVal)
		}
	} else if obj != nil {
		v := reflect.ValueOf(obj)
		typ := v.Type()
		if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.UnsafePointer {
			typ = typ.Elem()
		}
		typename := typ.String()
		_, ok1 := GetGlobalCreateFactory(typename)
		_, ok2 := GetGlobalWrapperFactory(typename)
		if ok1 || ok2 {
			return typename
		}
	}
	return ""
}

// createLuaVariableWrapper creates a Lua table that wraps a change-tracker Variable.
// This provides the Lua-accessible interface to the Variable.
func (r *LuaResolver) createLuaVariableWrapper(v *changetracker.Variable) *lua.LTable {
	r.Session.Log(4, "CREATE LUA VARIABLE WRAPPER %#v", v)
	wrapper := r.Session.State.NewTable()

	// Store the variable ID for reference
	r.Session.State.SetField(wrapper, "_id", lua.LNumber(v.ID))

	// getValue() - returns the current value
	r.Session.State.SetField(wrapper, "getValue", r.Session.State.NewFunction(func(L *lua.LState) int {
		L.Push(r.goToLua(v.Value))
		return 1
	}))

	// getProperty(name) - returns a property value
	r.Session.State.SetField(wrapper, "getProperty", r.Session.State.NewFunction(func(L *lua.LState) int {
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
	r.Session.State.SetField(wrapper, "getWrapper", r.Session.State.NewFunction(func(L *lua.LState) int {
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

func (r *LuaResolver) goToLua(value any) lua.LValue {
	return r.Session.goToLua(r.Session.State, value)
}

// ConvertToValueJSON implements the Resolver interface for Lua values.
// Handles *lua.LTable specially: arrays become []any, objects become ObjectRef.
func (r *LuaResolver) ConvertToValueJSON(tracker *changetracker.Tracker, value any) any {
	tbl, ok := value.(*lua.LTable)
	if !ok {
		// Not a Lua table - return unchanged for tracker to handle
		return value
	}

	// If no session, can't determine array vs object - let tracker handle it
	if r.Session == nil {
		return value
	}

	// Check if it's an array table
	if r.Session.isArray(tbl) {
		// Convert array to slice, recursively calling tracker.ToValueJSON for elements
		length := tbl.Len()
		result := make([]any, length)
		for i := 1; i <= length; i++ {
			elem := r.Session.State.RawGetInt(tbl, i)
			// Recursively convert elements (handles nested tables, etc.)
			result[i-1] = tracker.ToValueJSON(r.luaElementToGo(elem))
		}
		return result
	}

	// Object table - return as-is for tracker to register
	return tbl
}

// luaElementToGo converts a Lua value to Go for ToValueJSON serialization.
// Unlike luaValueToGo, this does NOT convert array tables to slices (that's handled by ToValueJSON).
func (r *LuaResolver) luaElementToGo(val lua.LValue) any {
	switch v := val.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// Keep tables as-is; ToValueJSON will handle them
		return v
	case *lua.LNilType:
		return nil
	default:
		return val
	}
}
