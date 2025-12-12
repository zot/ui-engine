// CRC: crc-LuaRuntime.md, crc-LuaSession.md, crc-LuaVariable.md
// Spec: interfaces.md, deployment.md, libraries.md
// Sequence: seq-lua-executor-init.md, seq-lua-execute.md, seq-lua-handle-action.md, seq-lua-session-init.md
package lua

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// WorkItem represents a unit of work for the executor.
type WorkItem struct {
	fn     func() (interface{}, error)
	result chan WorkResult
}

// WorkResult holds the result of a work item.
type WorkResult struct {
	Value interface{}
	Err   error
}

// WatchedVariable tracks a variable and its associated Lua object for change detection.
type WatchedVariable struct {
	ID          int64       // Variable ID
	LuaObject   *lua.LTable // Reference to the live Lua object
	CachedValue string      // Last JSON value sent to frontend
}

// LuaSession represents a per-frontend-session Lua environment.
// ID is the vended session ID (compact integer string like "1", "2") for backend communication,
// not the internal UUID which is used for URL paths.
type LuaSession struct {
	ID               string                      // Vended session ID (e.g., "1", "2", "3")
	sessionTable     *lua.LTable                 // The session object exposed to Lua
	appVariableID    int64                       // Variable 1 for this session (set by Lua code)
	appObject        *lua.LTable                 // Reference to the app Lua object
	watchedVariables map[int64]*WatchedVariable  // varID -> watched variable info
}

// Runtime manages embedded Lua VM execution with multiple sessions.
type Runtime struct {
	state          *lua.LState
	loadedModules  map[string]bool
	presenterTypes map[string]*PresenterType
	luaDir         string
	executorChan   chan WorkItem
	done           chan struct{}
	verbosity      int
	mu             sync.RWMutex

	// Session management
	sessions        map[string]*LuaSession // vendedID -> LuaSession
	variableStore   VariableStore          // Backend for variable operations
	mainLuaCode     string                 // Cached main.lua content (for bundle mode)
	wrapperRegistry *WrapperRegistry       // Shared wrapper registry (set by server)
}

// PresenterType represents a Lua-defined presenter type.
type PresenterType struct {
	Name    string
	Methods map[string]*lua.LFunction
	Table   *lua.LTable
}

// VariableStore interface for session operations.
type VariableStore interface {
	Create(parentID int64, value json.RawMessage, properties map[string]string) (int64, error)
	Get(id int64) (value json.RawMessage, properties map[string]string, ok bool)
	GetProperty(id int64, name string) (string, bool)
	Update(id int64, value json.RawMessage, properties map[string]string) error
	Destroy(id int64) error
}

// NewRuntime creates a new Lua runtime with executor goroutine.
func NewRuntime(luaDir string) (*Runtime, error) {
	L := lua.NewState()

	r := &Runtime{
		state:          L,
		loadedModules:  make(map[string]bool),
		presenterTypes: make(map[string]*PresenterType),
		sessions:       make(map[string]*LuaSession),
		luaDir:         luaDir,
		executorChan:   make(chan WorkItem, 100),
		done:           make(chan struct{}),
	}

	// Load standard libraries
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)
	lua.OpenOs(L)

	// Register UI module
	r.registerUIModule()

	// Start executor goroutine
	r.startExecutor()

	return r, nil
}

// SetVerbosity sets the verbosity level.
func (r *Runtime) SetVerbosity(level int) {
	r.verbosity = level
}

// SetVariableStore sets the variable store for session operations.
func (r *Runtime) SetVariableStore(store VariableStore) {
	r.variableStore = store
}

// SetWrapperRegistry sets the wrapper registry for registering Lua wrappers.
func (r *Runtime) SetWrapperRegistry(registry *WrapperRegistry) {
	r.wrapperRegistry = registry
}

// GetGlobalTable looks up a Lua global by name and returns it if it's a table.
// Used for auto-discovery of Lua-defined wrappers.
// Returns nil if the global doesn't exist or isn't a table.
func (r *Runtime) GetGlobalTable(name string) interface{} {
	var result interface{}
	r.execute(func() (interface{}, error) {
		L := r.state
		val := L.GetGlobal(name)
		if tbl, ok := val.(*lua.LTable); ok {
			result = tbl
		}
		return nil, nil
	})
	return result
}

// SetMainLuaCode sets the main.lua code to execute for each new session.
// Used when loading from bundle where filesystem access is not available.
func (r *Runtime) SetMainLuaCode(code string) {
	r.mainLuaCode = code
}

// CreateLuaSession creates a new Lua session for a frontend session.
// vendedID is the compact session ID (e.g., "1", "2") for backend communication.
// Loads and executes main.lua with a session global.
// Must be called after SetVariableStore.
func (r *Runtime) CreateLuaSession(vendedID string) (*LuaSession, error) {
	if r.variableStore == nil {
		return nil, fmt.Errorf("variable store not set")
	}

	var luaSession *LuaSession
	_, err := r.execute(func() (interface{}, error) {
		L := r.state

		// Create session table for this frontend session
		sessionTable := r.createSessionTable(L, vendedID)

		luaSession = &LuaSession{
			ID:               vendedID,
			sessionTable:     sessionTable,
			watchedVariables: make(map[int64]*WatchedVariable),
		}

		// Store in sessions map (keyed by vended ID)
		r.mu.Lock()
		r.sessions[vendedID] = luaSession
		r.mu.Unlock()

		// Set session global (will be replaced for each session's code execution)
		L.SetGlobal("session", sessionTable)

		// Load main.lua for this session
		if err := r.loadMainLua(L); err != nil {
			// Remove from sessions on failure
			r.mu.Lock()
			delete(r.sessions, vendedID)
			r.mu.Unlock()
			return nil, err
		}

		if r.verbosity >= 2 {
			log.Printf("[v2] LuaRuntime: created Lua session %s", vendedID)
		}

		// Run change detection after main.lua completes
		// This ensures any modifications made during init are synced to the store
		r.syncChangesInternal(luaSession)

		return nil, nil
	})

	if err != nil {
		return nil, err
	}
	return luaSession, nil
}

// loadMainLua loads main.lua from filesystem or cached bundle code.
func (r *Runtime) loadMainLua(L *lua.LState) error {
	// Try cached bundle code first
	if r.mainLuaCode != "" {
		if err := L.DoString(r.mainLuaCode); err != nil {
			return fmt.Errorf("failed to execute main.lua: %w", err)
		}
		return nil
	}

	// Try filesystem
	mainPath := filepath.Join(r.luaDir, "main.lua")
	if _, err := os.Stat(mainPath); err == nil {
		if err := L.DoFile(mainPath); err != nil {
			return fmt.Errorf("failed to load main.lua: %w", err)
		}
		return nil
	}

	// No main.lua found - this is OK for hybrid mode where backend creates variable 1
	if r.verbosity >= 2 {
		log.Printf("[v2] LuaRuntime: no main.lua found (hybrid mode or backend-only)")
	}
	return nil
}

// DestroyLuaSession removes a Lua session by its vended ID.
func (r *Runtime) DestroyLuaSession(vendedID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, vendedID)
	if r.verbosity >= 2 {
		log.Printf("[v2] LuaRuntime: destroyed Lua session %s", vendedID)
	}
}

// GetLuaSession retrieves a Lua session by its vended ID.
func (r *Runtime) GetLuaSession(vendedID string) (*LuaSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[vendedID]
	return session, ok
}

// createSessionTable creates the session object with all methods.
// vendedID is the compact session ID (e.g., "1", "2") exposed to Lua code.
func (r *Runtime) createSessionTable(L *lua.LState, vendedID string) *lua.LTable {
	session := L.NewTable()

	// Internal state
	variables := L.NewTable() // varId -> Variable wrapper
	watchers := L.NewTable()  // varId -> { property -> callbacks[] }

	L.SetField(session, "_variables", variables)
	L.SetField(session, "_watchers", watchers)
	L.SetField(session, "_sessionID", lua.LString(vendedID))

	// session:createAppVariable(object, properties)
	// Creates variable 1 (the app variable) - called by main.lua
	// The object is a Lua table that the variable references (for change detection)
	// If no type property is provided, extracts from object's metatable "type" field
	L.SetField(session, "createAppVariable", L.NewFunction(func(L *lua.LState) int {
		luaObject := L.CheckTable(2)
		propsTable := L.OptTable(3, nil)

		// Convert Lua object to JSON for initial value
		goValue := luaToGo(luaObject)
		var jsonValue json.RawMessage
		if goValue != nil {
			data, err := json.Marshal(goValue)
			if err != nil {
				L.RaiseError("failed to marshal value: %v", err)
				return 0
			}
			jsonValue = data
		}

		// Convert properties
		props := make(map[string]string)
		if propsTable != nil {
			propsTable.ForEach(func(k, v lua.LValue) {
				if ks, ok := k.(lua.LString); ok {
					props[string(ks)] = lua.LVAsString(v)
				}
			})
		}

		// Extract type from metatable's "type" field (frictionless convention)
		if props["type"] == "" {
			// First check metatable
			mt := L.GetMetatable(luaObject)
			if mt != lua.LNil {
				if mtTbl, ok := mt.(*lua.LTable); ok {
					if typeVal := L.GetField(mtTbl, "type"); typeVal != lua.LNil {
						props["type"] = lua.LVAsString(typeVal)
					}
				}
			}
			// Fall back to direct "type" field on object
			if props["type"] == "" {
				if typeVal := L.GetField(luaObject, "type"); typeVal != lua.LNil {
					props["type"] = lua.LVAsString(typeVal)
				}
			}
		}

		// Create variable 1 with parentID 0
		id, err := r.variableStore.Create(0, jsonValue, props)
		if err != nil {
			L.RaiseError("failed to create app variable: %v", err)
			return 0
		}

		// Store app variable ID and object reference in session
		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			luaSess.appVariableID = id
			luaSess.appObject = luaObject
			// Register for change detection
			luaSess.watchedVariables[id] = &WatchedVariable{
				ID:          id,
				LuaObject:   luaObject,
				CachedValue: string(jsonValue),
			}
		}

		if r.verbosity >= 2 {
			log.Printf("[v2] LuaRuntime: created app variable %d for session %s", id, vendedID)
		}

		// Return the variable ID (not a wrapper - Lua code should use getApp() to access the object)
		L.Push(lua.LNumber(id))
		return 1
	}))

	// session:getApp()
	// Returns the actual Lua app object (not a wrapper) for direct modification
	L.SetField(session, "getApp", L.NewFunction(func(L *lua.LState) int {
		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok || luaSess.appObject == nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(luaSess.appObject)
		return 1
	}))

	// session:getAppVariable() - DEPRECATED, use getApp() instead
	// Returns a variable wrapper for backward compatibility
	L.SetField(session, "getAppVariable", L.NewFunction(func(L *lua.LState) int {
		// Get app variable ID from session
		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok || luaSess.appVariableID == 0 {
			L.Push(lua.LNil)
			return 1
		}
		// Return variable wrapper for backward compatibility
		varWrapper := r.getOrCreateVariableWrapper(L, session, luaSess.appVariableID)
		L.Push(varWrapper)
		return 1
	}))

	// session:getVariable(id)
	L.SetField(session, "getVariable", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckInt64(2)
		varWrapper := r.getOrCreateVariableWrapper(L, session, id)
		L.Push(varWrapper)
		return 1
	}))

	// session:createVariable(parentId, object, properties)
	// Creates a child variable referencing a Lua object
	// parentId can be a number (variable ID) or a table (looked up via session's watched variables)
	// If no type property is provided, extracts from object's metatable "type" field
	L.SetField(session, "createVariable", L.NewFunction(func(L *lua.LState) int {
		// Parent can be ID (number) or object (table)
		var parentID int64
		parentArg := L.Get(2)
		switch p := parentArg.(type) {
		case lua.LNumber:
			parentID = int64(p)
		case *lua.LTable:
			// Look up parent ID by object reference
			luaSess, ok := r.GetLuaSession(vendedID)
			if ok {
				for id, wv := range luaSess.watchedVariables {
					if wv.LuaObject == p {
						parentID = id
						break
					}
				}
			}
			if parentID == 0 {
				L.RaiseError("parent object not found in watched variables")
				return 0
			}
		default:
			L.RaiseError("createVariable: parentId must be a number or table")
			return 0
		}

		luaObject := L.CheckTable(3)
		propsTable := L.OptTable(4, nil)

		// Convert Lua object to JSON for initial value
		goValue := luaToGo(luaObject)
		var jsonValue json.RawMessage
		if goValue != nil {
			data, err := json.Marshal(goValue)
			if err != nil {
				L.Push(lua.LNil)
				return 1
			}
			jsonValue = data
		}

		// Convert properties
		props := make(map[string]string)
		if propsTable != nil {
			propsTable.ForEach(func(k, v lua.LValue) {
				if ks, ok := k.(lua.LString); ok {
					props[string(ks)] = lua.LVAsString(v)
				}
			})
		}

		// Extract type from metatable's "type" field (frictionless convention)
		if props["type"] == "" {
			// First check metatable
			mt := L.GetMetatable(luaObject)
			if mt != lua.LNil {
				if mtTbl, ok := mt.(*lua.LTable); ok {
					if typeVal := L.GetField(mtTbl, "type"); typeVal != lua.LNil {
						props["type"] = lua.LVAsString(typeVal)
					}
				}
			}
			// Fall back to direct "type" field on object
			if props["type"] == "" {
				if typeVal := L.GetField(luaObject, "type"); typeVal != lua.LNil {
					props["type"] = lua.LVAsString(typeVal)
				}
			}
		}

		// Create variable
		id, err := r.variableStore.Create(parentID, jsonValue, props)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}

		// Register for change detection
		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			luaSess.watchedVariables[id] = &WatchedVariable{
				ID:          id,
				LuaObject:   luaObject,
				CachedValue: string(jsonValue),
			}
		}

		// Return the variable ID
		L.Push(lua.LNumber(id))
		return 1
	}))

	// session:destroyVariable(idOrObject)
	// Destroys a variable by ID or by object reference
	L.SetField(session, "destroyVariable", L.NewFunction(func(L *lua.LState) int {
		var id int64
		arg := L.Get(2)
		switch v := arg.(type) {
		case lua.LNumber:
			id = int64(v)
		case *lua.LTable:
			// Look up ID by object reference
			luaSess, ok := r.GetLuaSession(vendedID)
			if ok {
				for varID, wv := range luaSess.watchedVariables {
					if wv.LuaObject == v {
						id = varID
						break
					}
				}
			}
			if id == 0 {
				// Object not found - nothing to destroy
				return 0
			}
		default:
			L.RaiseError("destroyVariable: argument must be a number or table")
			return 0
		}

		if err := r.variableStore.Destroy(id); err != nil {
			if r.verbosity >= 2 {
				log.Printf("[v2] LuaRuntime: destroyVariable error: %v", err)
			}
		}

		// Remove from watched variables
		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			delete(luaSess.watchedVariables, id)
		}

		// Remove from cache
		variables := L.GetField(session, "_variables").(*lua.LTable)
		L.SetField(variables, fmt.Sprintf("%d", id), lua.LNil)

		return 0
	}))

	// session:watchVariable(id, callback)
	L.SetField(session, "watchVariable", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckInt64(2)
		callback := L.CheckFunction(3)
		r.addPropertyWatcher(L, session, id, "*", callback)
		return 0
	}))

	// session:watchProperty(id, property, callback)
	L.SetField(session, "watchProperty", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckInt64(2)
		property := L.CheckString(3)
		callback := L.CheckFunction(4)
		r.addPropertyWatcher(L, session, id, property, callback)
		return 0
	}))

	return session
}

// getOrCreateVariableWrapper gets or creates a Variable wrapper for an ID.
func (r *Runtime) getOrCreateVariableWrapper(L *lua.LState, session *lua.LTable, id int64) *lua.LTable {
	variables := L.GetField(session, "_variables").(*lua.LTable)
	key := fmt.Sprintf("%d", id)
	existing := L.GetField(variables, key)
	if existing != lua.LNil {
		return existing.(*lua.LTable)
	}

	// Create new wrapper
	wrapper := r.createVariableWrapper(L, session, id)
	L.SetField(variables, key, wrapper)
	return wrapper
}

// createVariableWrapper creates a Variable wrapper object.
func (r *Runtime) createVariableWrapper(L *lua.LState, session *lua.LTable, id int64) *lua.LTable {
	wrapper := L.NewTable()
	L.SetField(wrapper, "_id", lua.LNumber(id))
	L.SetField(wrapper, "_session", session)

	// var:getId()
	L.SetField(wrapper, "getId", L.NewFunction(func(L *lua.LState) int {
		self := L.CheckTable(1)
		varID := L.GetField(self, "_id")
		L.Push(varID)
		return 1
	}))

	// var:getValue()
	L.SetField(wrapper, "getValue", L.NewFunction(func(L *lua.LState) int {
		self := L.CheckTable(1)
		varID := int64(L.GetField(self, "_id").(lua.LNumber))
		value, _, ok := r.variableStore.Get(varID)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		// Parse JSON value
		var goVal interface{}
		if len(value) > 0 {
			json.Unmarshal(value, &goVal)
		}
		L.Push(goToLua(L, goVal))
		return 1
	}))

	// var:getProperty(name)
	L.SetField(wrapper, "getProperty", L.NewFunction(func(L *lua.LState) int {
		self := L.CheckTable(1)
		name := L.CheckString(2)
		varID := int64(L.GetField(self, "_id").(lua.LNumber))
		prop, ok := r.variableStore.GetProperty(varID, name)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LString(prop))
		return 1
	}))

	// var:update(value, properties)
	L.SetField(wrapper, "update", L.NewFunction(func(L *lua.LState) int {
		self := L.CheckTable(1)
		value := L.Get(2)
		propsTable := L.OptTable(3, nil)
		varID := int64(L.GetField(self, "_id").(lua.LNumber))

		// Convert value to JSON
		var jsonValue json.RawMessage
		if value != lua.LNil {
			goValue := luaToGo(value)
			if goValue != nil {
				data, _ := json.Marshal(goValue)
				jsonValue = data
			}
		}

		// Convert properties
		var props map[string]string
		if propsTable != nil {
			props = make(map[string]string)
			propsTable.ForEach(func(k, v lua.LValue) {
				if ks, ok := k.(lua.LString); ok {
					props[string(ks)] = lua.LVAsString(v)
				}
			})
		}

		if err := r.variableStore.Update(varID, jsonValue, props); err != nil {
			if r.verbosity >= 2 {
				log.Printf("[v2] LuaRuntime: update error: %v", err)
			}
		}
		return 0
	}))

	// var:updateProperties(properties)
	L.SetField(wrapper, "updateProperties", L.NewFunction(func(L *lua.LState) int {
		self := L.CheckTable(1)
		propsTable := L.CheckTable(2)
		varID := int64(L.GetField(self, "_id").(lua.LNumber))

		props := make(map[string]string)
		propsTable.ForEach(func(k, v lua.LValue) {
			if ks, ok := k.(lua.LString); ok {
				props[string(ks)] = lua.LVAsString(v)
			}
		})

		if err := r.variableStore.Update(varID, nil, props); err != nil {
			if r.verbosity >= 2 {
				log.Printf("[v2] LuaRuntime: updateProperties error: %v", err)
			}
		}
		return 0
	}))

	return wrapper
}

// addPropertyWatcher adds a watcher for a property on a variable.
func (r *Runtime) addPropertyWatcher(L *lua.LState, session *lua.LTable, varID int64, property string, callback *lua.LFunction) {
	watchers := L.GetField(session, "_watchers").(*lua.LTable)
	key := fmt.Sprintf("%d", varID)

	varWatchers := L.GetField(watchers, key)
	if varWatchers == lua.LNil {
		varWatchers = L.NewTable()
		L.SetField(watchers, key, varWatchers)
	}

	propWatchers := L.GetField(varWatchers.(*lua.LTable), property)
	if propWatchers == lua.LNil {
		propWatchers = L.NewTable()
		L.SetField(varWatchers.(*lua.LTable), property, propWatchers)
	}

	// Append callback to list
	tbl := propWatchers.(*lua.LTable)
	tbl.Append(callback)
}

// NotifyPropertyChange notifies Lua watchers of a property change for a session.
// Called by external code when a variable property changes.
// vendedID is the compact session ID (e.g., "1", "2").
func (r *Runtime) NotifyPropertyChange(vendedID string, varID int64, property string, value interface{}) {
	r.mu.RLock()
	luaSess, ok := r.sessions[vendedID]
	r.mu.RUnlock()
	if !ok || luaSess == nil {
		return
	}

	r.execute(func() (interface{}, error) {
		L := r.state
		r.notifyPropertyChangeInternal(L, luaSess.sessionTable, varID, property, value)
		return nil, nil
	})
}

// notifyPropertyChangeInternal notifies watchers (must be called from executor).
func (r *Runtime) notifyPropertyChangeInternal(L *lua.LState, session *lua.LTable, varID int64, property string, value interface{}) {
	watchers := L.GetField(session, "_watchers").(*lua.LTable)
	key := fmt.Sprintf("%d", varID)

	varWatchers := L.GetField(watchers, key)
	if varWatchers == lua.LNil {
		return
	}

	luaValue := goToLua(L, value)

	// Call property-specific watchers
	propWatchers := L.GetField(varWatchers.(*lua.LTable), property)
	if propWatchers != lua.LNil {
		r.callWatchers(L, propWatchers.(*lua.LTable), luaValue)
	}

	// Call wildcard watchers
	wildcardWatchers := L.GetField(varWatchers.(*lua.LTable), "*")
	if wildcardWatchers != lua.LNil {
		r.callWatchers(L, wildcardWatchers.(*lua.LTable), luaValue, lua.LString(property))
	}
}

// callWatchers calls all watcher callbacks in a table.
func (r *Runtime) callWatchers(L *lua.LState, watchers *lua.LTable, args ...lua.LValue) {
	watchers.ForEach(func(_, cb lua.LValue) {
		if fn, ok := cb.(*lua.LFunction); ok {
			L.Push(fn)
			for _, arg := range args {
				L.Push(arg)
			}
			if err := L.PCall(len(args), 0, nil); err != nil {
				log.Printf("[lua] Watcher callback error: %v", err)
			}
		}
	})
}

// startExecutor creates the goroutine that processes work items.
func (r *Runtime) startExecutor() {
	go func() {
		for {
			select {
			case <-r.done:
				return
			case work := <-r.executorChan:
				result, err := work.fn()
				work.result <- WorkResult{Value: result, Err: err}
			}
		}
	}()
}

// execute queues a function on the executor and blocks until complete.
func (r *Runtime) execute(fn func() (interface{}, error)) (interface{}, error) {
	result := make(chan WorkResult, 1)
	r.executorChan <- WorkItem{fn: fn, result: result}
	res := <-result
	return res.Value, res.Err
}

// VariableUpdate represents a detected change to be sent to the frontend.
type VariableUpdate struct {
	VarID int64
	Value json.RawMessage
}

// AfterBatch triggers change detection for a session after processing a message batch.
// Returns a list of variable updates that need to be sent to the frontend.
// vendedID is the compact session ID (e.g., "1", "2").
func (r *Runtime) AfterBatch(vendedID string) []VariableUpdate {
	r.mu.RLock()
	luaSess, ok := r.sessions[vendedID]
	r.mu.RUnlock()
	if !ok || luaSess == nil {
		return nil
	}

	var updates []VariableUpdate

	r.execute(func() (interface{}, error) {
		updates = r.syncChangesInternal(luaSess)
		return nil, nil
	})

	return updates
}

// syncChangesInternal performs change detection for a session.
// Must be called from within the executor goroutine.
func (r *Runtime) syncChangesInternal(luaSess *LuaSession) []VariableUpdate {
	var updates []VariableUpdate

	// Iterate all watched variables and check for changes
	for varID, wv := range luaSess.watchedVariables {
		// Compute current JSON value from Lua object
		goValue := luaToGo(wv.LuaObject)
		var currentJSON string
		if goValue != nil {
			data, err := json.Marshal(goValue)
			if err != nil {
				if r.verbosity >= 1 {
					log.Printf("[v1] syncChanges: failed to marshal variable %d: %v", varID, err)
				}
				continue
			}
			currentJSON = string(data)
		}

		// Compare to cached value
		if currentJSON != wv.CachedValue {
			if r.verbosity >= 2 {
				log.Printf("[v2] syncChanges: variable %d changed", varID)
			}

			// Update cache
			wv.CachedValue = currentJSON

			// Queue update
			updates = append(updates, VariableUpdate{
				VarID: varID,
				Value: json.RawMessage(currentJSON),
			})

			// Also update the variable store so watchers get notified
			if err := r.variableStore.Update(varID, json.RawMessage(currentJSON), nil); err != nil {
				if r.verbosity >= 1 {
					log.Printf("[v1] syncChanges: failed to update store for variable %d: %v", varID, err)
				}
			}
		}
	}

	return updates
}

// registerUIModule adds the ui.* API to Lua.
func (r *Runtime) registerUIModule() {
	L := r.state

	// Create ui module
	uiMod := L.NewTable()

	// ui.registerPresenter(name, table)
	L.SetField(uiMod, "registerPresenter", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		tbl := L.CheckTable(2)

		r.mu.Lock()
		r.presenterTypes[name] = &PresenterType{
			Name:    name,
			Methods: make(map[string]*lua.LFunction),
			Table:   tbl,
		}
		r.mu.Unlock()

		if r.verbosity >= 2 {
			log.Printf("[v2] LuaRuntime: registered presenter type %s", name)
		}

		return 0
	}))

	// ui.log(message)
	L.SetField(uiMod, "log", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		fmt.Printf("[lua] %s\n", msg)
		return 0
	}))

	// ui.json_encode(value)
	L.SetField(uiMod, "json_encode", L.NewFunction(func(L *lua.LState) int {
		val := L.Get(1)
		goVal := luaToGo(val)
		data, err := json.Marshal(goVal)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(data)))
		return 1
	}))

	// ui.json_decode(string)
	L.SetField(uiMod, "json_decode", L.NewFunction(func(L *lua.LState) int {
		str := L.CheckString(1)
		var val interface{}
		if err := json.Unmarshal([]byte(str), &val); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(goToLua(L, val))
		return 1
	}))

	// ui.registerWrapper(name, table)
	// Registers a Lua wrapper type for variable value transformation.
	// The table must have: computeValue(self, rawValue) -> storedValue
	// Optionally: destroy(self) for cleanup
	L.SetField(uiMod, "registerWrapper", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		tbl := L.CheckTable(2)

		if r.wrapperRegistry == nil {
			L.RaiseError("wrapper registry not configured")
			return 0
		}

		// Create a Lua wrapper factory that creates instances from the Lua table
		r.wrapperRegistry.Register(name, func(runtime *Runtime, variable WrapperVariable) Wrapper {
			return NewLuaWrapper(runtime, tbl, variable)
		})

		if r.verbosity >= 2 {
			log.Printf("[v2] LuaRuntime: registered wrapper type %s", name)
		}

		return 0
	}))

	L.SetGlobal("ui", uiMod)
}

// LoadFile loads and executes a Lua file via executor (relative to luaDir).
func (r *Runtime) LoadFile(filename string) error {
	path := filepath.Join(r.luaDir, filename)
	return r.LoadFileAbsolute(path)
}

// LoadFileAbsolute loads and executes a Lua file via executor (absolute path).
func (r *Runtime) LoadFileAbsolute(path string) error {
	_, err := r.execute(func() (interface{}, error) {
		r.mu.Lock()
		defer r.mu.Unlock()

		if r.loadedModules[path] {
			return nil, nil // Already loaded
		}

		if err := r.state.DoFile(path); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}

		r.loadedModules[path] = true
		return nil, nil
	})
	return err
}

// LoadCode loads and executes Lua code string via executor.
func (r *Runtime) LoadCode(name, code string) error {
	_, err := r.execute(func() (interface{}, error) {
		if err := r.state.DoString(code); err != nil {
			return nil, fmt.Errorf("failed to load code %s: %w", name, err)
		}
		return nil, nil
	})
	return err
}

// LoadDirectory loads all .lua files from a directory via executor.
func (r *Runtime) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".lua" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		_, err := r.execute(func() (interface{}, error) {
			r.mu.Lock()
			defer r.mu.Unlock()

			if r.loadedModules[path] {
				return nil, nil
			}

			if err := r.state.DoFile(path); err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", path, err)
			}
			r.loadedModules[path] = true
			return nil, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetPresenterType returns a registered presenter type.
func (r *Runtime) GetPresenterType(name string) (*PresenterType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pt, ok := r.presenterTypes[name]
	return pt, ok
}

// ListPresenterTypes returns all registered presenter type names.
func (r *Runtime) ListPresenterTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.presenterTypes))
	for name := range r.presenterTypes {
		names = append(names, name)
	}
	return names
}

// CallMethod invokes a method on a Lua presenter instance via executor.
func (r *Runtime) CallMethod(instance *lua.LTable, method string, args ...interface{}) (interface{}, error) {
	return r.execute(func() (interface{}, error) {
		L := r.state

		fn := L.GetField(instance, method)
		if fn == lua.LNil {
			return nil, fmt.Errorf("method %s not found", method)
		}

		lfn, ok := fn.(*lua.LFunction)
		if !ok {
			return nil, fmt.Errorf("%s is not a function", method)
		}

		// Push function and self
		L.Push(lfn)
		L.Push(instance)

		// Push arguments
		for _, arg := range args {
			L.Push(goToLua(L, arg))
		}

		// Call method (self + args)
		if err := L.PCall(len(args)+1, 1, nil); err != nil {
			return nil, err
		}

		// Get result
		result := L.Get(-1)
		L.Pop(1)

		return luaToGo(result), nil
	})
}

// CallLuaWrapperMethod invokes a method on a Lua wrapper table via executor.
// Used by LuaWrapper to call computeValue and destroy methods.
// The instance can be any interface{} but must be a *lua.LTable at runtime.
func (r *Runtime) CallLuaWrapperMethod(instance interface{}, method string, args ...interface{}) (interface{}, error) {
	tbl, ok := instance.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("wrapper instance is not a Lua table")
	}

	return r.execute(func() (interface{}, error) {
		L := r.state

		fn := L.GetField(tbl, method)
		if fn == lua.LNil {
			// Method not found is not an error for optional methods like destroy
			return nil, nil
		}

		lfn, ok := fn.(*lua.LFunction)
		if !ok {
			return nil, fmt.Errorf("%s is not a function", method)
		}

		// Push function and self
		L.Push(lfn)
		L.Push(tbl)

		// Push arguments (convert json.RawMessage specially)
		for _, arg := range args {
			if raw, ok := arg.(json.RawMessage); ok {
				// Parse JSON and convert to Lua
				var val interface{}
				if err := json.Unmarshal(raw, &val); err != nil {
					L.Push(lua.LNil)
				} else {
					L.Push(goToLua(L, val))
				}
			} else {
				L.Push(goToLua(L, arg))
			}
		}

		// Call method (self + args)
		if err := L.PCall(len(args)+1, 1, nil); err != nil {
			return nil, err
		}

		// Get result
		result := L.Get(-1)
		L.Pop(1)

		return luaToGo(result), nil
	})
}

// CreateInstance creates a new instance of a presenter type via executor.
func (r *Runtime) CreateInstance(typeName string, props map[string]interface{}) (*lua.LTable, error) {
	result, err := r.execute(func() (interface{}, error) {
		r.mu.RLock()
		pt, ok := r.presenterTypes[typeName]
		r.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("presenter type %s not found", typeName)
		}

		L := r.state

		// Create new instance table
		instance := L.NewTable()

		// Set metatable to inherit from presenter type
		mt := L.NewTable()
		L.SetField(mt, "__index", pt.Table)
		L.SetMetatable(instance, mt)

		// Set initial properties
		for k, v := range props {
			L.SetField(instance, k, goToLua(L, v))
		}

		// Call init method if exists
		initFn := L.GetField(pt.Table, "init")
		if initFn != lua.LNil {
			if lfn, ok := initFn.(*lua.LFunction); ok {
				L.Push(lfn)
				L.Push(instance)
				if err := L.PCall(1, 0, nil); err != nil {
					return nil, fmt.Errorf("init failed: %w", err)
				}
			}
		}

		return instance, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*lua.LTable), nil
}

// ItemWrapperInstance represents a created item wrapper (presenter).
type ItemWrapperInstance struct {
	objID    int64
	instance *lua.LTable
}

// ObjRef returns an object reference to this item wrapper.
func (i *ItemWrapperInstance) ObjRef() json.RawMessage {
	ref, _ := json.Marshal(map[string]int64{"obj": i.objID})
	return ref
}

// CreateItemWrapper creates an ItemWrapper instance for a ViewItem.
// The ItemWrapper constructor receives the ViewItem: ItemWrapper(viewItem).
// Returns nil if no itemType is specified or the type isn't registered.
func (r *Runtime) CreateItemWrapper(typeName string, viewItem *ViewItem) (*ItemWrapperInstance, error) {
	if typeName == "" {
		return nil, nil
	}

	result, err := r.execute(func() (interface{}, error) {
		r.mu.RLock()
		pt, ok := r.presenterTypes[typeName]
		r.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("item wrapper type %s not found", typeName)
		}

		L := r.state

		// Create new instance table
		instance := L.NewTable()

		// Set metatable to inherit from presenter type
		mt := L.NewTable()
		L.SetField(mt, "__index", pt.Table)
		L.SetMetatable(instance, mt)

		// Set ViewItem properties on the instance
		// The presenter can access: viewItem.baseItem, viewItem.list, viewItem.index
		L.SetField(instance, "viewItem", r.createViewItemLuaWrapper(L, viewItem))

		// Call init method if exists, passing the viewItem
		initFn := L.GetField(pt.Table, "init")
		if initFn != lua.LNil {
			if lfn, ok := initFn.(*lua.LFunction); ok {
				L.Push(lfn)
				L.Push(instance)
				if err := L.PCall(1, 0, nil); err != nil {
					return nil, fmt.Errorf("init failed: %w", err)
				}
			}
		}

		// Generate object ID for this presenter
		// Use negative IDs as per protocol.md
		objID := viewItem.ObjID - 1000000 // Offset from ViewItem ID

		return &ItemWrapperInstance{
			objID:    objID,
			instance: instance,
		}, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*ItemWrapperInstance), nil
}

// createViewItemLuaWrapper creates a Lua wrapper for a ViewItem.
func (r *Runtime) createViewItemLuaWrapper(L *lua.LState, viewItem *ViewItem) *lua.LTable {
	wrapper := L.NewTable()

	// viewItem.baseItem - object reference to domain object
	var baseItemRef map[string]interface{}
	json.Unmarshal(viewItem.BaseItem, &baseItemRef)
	L.SetField(wrapper, "baseItem", goToLua(L, baseItemRef))

	// viewItem.index - position in list
	L.SetField(wrapper, "index", lua.LNumber(viewItem.Index))

	// viewItem:remove() - removes this item from the list
	L.SetField(wrapper, "remove", L.NewFunction(func(L *lua.LState) int {
		if err := viewItem.Remove(); err != nil {
			if r.verbosity >= 1 {
				log.Printf("[v1] ViewItem.remove error: %v", err)
			}
		}
		return 0
	}))

	return wrapper
}

// GetValue gets a value from a Lua table via executor.
func (r *Runtime) GetValue(tbl *lua.LTable, key string) interface{} {
	result, _ := r.execute(func() (interface{}, error) {
		val := r.state.GetField(tbl, key)
		return luaToGo(val), nil
	})
	return result
}

// SetValue sets a value on a Lua table via executor.
func (r *Runtime) SetValue(tbl *lua.LTable, key string, value interface{}) {
	r.execute(func() (interface{}, error) {
		r.state.SetField(tbl, key, goToLua(r.state, value))
		return nil, nil
	})
}

// Shutdown cleans up the Lua VM and stops executor.
func (r *Runtime) Shutdown() {
	close(r.done)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.state.Close()
}

// goToLua converts a Go value to Lua.
func goToLua(L *lua.LState, val interface{}) lua.LValue {
	if val == nil {
		return lua.LNil
	}

	switch v := val.(type) {
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(float64(v))
	case int64:
		return lua.LNumber(float64(v))
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []interface{}:
		tbl := L.NewTable()
		for i, item := range v {
			L.RawSetInt(tbl, i+1, goToLua(L, item))
		}
		return tbl
	case map[string]interface{}:
		tbl := L.NewTable()
		for k, item := range v {
			L.SetField(tbl, k, goToLua(L, item))
		}
		return tbl
	default:
		return lua.LString(fmt.Sprintf("%v", v))
	}
}

// luaToGo converts a Lua value to Go.
// Fields prefixed with "_" are skipped (internal/private fields).
func luaToGo(val lua.LValue) interface{} {
	switch v := val.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// Count numeric and string keys to determine if array or map
		hasNumericKeys := false
		hasStringKeys := false
		maxN := 0
		v.ForEach(func(key, _ lua.LValue) {
			if n, ok := key.(lua.LNumber); ok {
				hasNumericKeys = true
				if int(n) > maxN {
					maxN = int(n)
				}
			} else if ks, ok := key.(lua.LString); ok {
				// Skip internal fields
				if !strings.HasPrefix(string(ks), "_") {
					hasStringKeys = true
				}
			}
		})

		// Pure array (only numeric keys)
		if hasNumericKeys && !hasStringKeys && maxN > 0 {
			arr := make([]interface{}, maxN)
			for i := 1; i <= maxN; i++ {
				arr[i-1] = luaToGo(v.RawGetInt(i))
			}
			return arr
		}

		// Object (string keys, possibly mixed with numeric)
		m := make(map[string]interface{})
		v.ForEach(func(key, value lua.LValue) {
			if ks, ok := key.(lua.LString); ok {
				keyStr := string(ks)
				// Skip internal fields (prefixed with _)
				if !strings.HasPrefix(keyStr, "_") {
					m[keyStr] = luaToGo(value)
				}
			}
		})
		return m
	case *lua.LNilType:
		return nil
	default:
		return nil
	}
}
