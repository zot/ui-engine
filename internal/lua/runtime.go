// CRC: crc-LuaRuntime.md, crc-LuaSession.md, crc-LuaVariable.md
// Spec: interfaces.md, deployment.md, libraries.md
// Sequence: seq-lua-executor-init.md, seq-lua-execute.md, seq-lua-handle-action.md, seq-lua-session-init.md
package lua

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui/internal/bundle"
	"github.com/zot/ui/internal/config"
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

// LuaSession represents a per-frontend-session Lua environment.
// ID is the vended session ID (compact integer string like "1", "2") for backend communication,
// not the internal UUID which is used for URL paths.
// Change detection is handled by the tracker in VariableStore.
type LuaSession struct {
	*Runtime
	ID            string      // Vended session ID (e.g., "1", "2", "3")
	sessionTable  *lua.LTable // The session object exposed to Lua
	appVariableID int64       // Variable 1 for this session (set by Lua code)
	appObject     *lua.LTable // Reference to the app Lua object
}

// Runtime manages embedded Lua VM execution with multiple sessions.
type Runtime struct {
	state          *lua.LState
	loadedModules  map[string]bool
	presenterTypes map[string]*PresenterType
	luaDir         string
	executorChan   chan WorkItem
	done           chan struct{}
	config         *config.Config
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
	// Session management - each session has its own tracker
	CreateSession(sessionID string, resolver changetracker.Resolver)
	DestroySession(sessionID string)
	GetTracker(sessionID string) *changetracker.Tracker

	// Variable operations (delegate to session's tracker)
	CreateVariable(sessionID string, parentID int64, luaObject *lua.LTable, properties map[string]string) (int64, error)
	Get(id int64) (value json.RawMessage, properties map[string]string, ok bool)
	GetProperty(id int64, name string) (string, bool)
	Update(id int64, value json.RawMessage, properties map[string]string) error
	Destroy(id int64) error

	// Change detection
	DetectChanges(sessionID string) []changetracker.Change
}

// NewRuntime creates a new Lua runtime with executor goroutine.
func NewRuntime(cfg *config.Config, luaDir string) (*Runtime, error) {
	L := lua.NewState()

	r := &Runtime{
		config:         cfg,
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

	// Register custom require() that works with both filesystem and bundle
	r.registerRequire()

	// Register UI module
	r.registerUIModule()

	// Try to load session module (lib/lua/session.lua) - optional for testing
	r.loadSessionModule()

	// Start executor goroutine
	r.startExecutor()

	return r, nil
}

// loadSessionModule tries to load lib/lua/session.lua and stores the module globally.
// Returns silently if module not found (allows tests to work without it).
func (r *Runtime) loadSessionModule() {
	L := r.state

	// Try to load the session module via require
	if err := L.DoString(`_SessionModule = require("session")`); err != nil {
		r.Log(2, "LuaRuntime: session module not found, using inline fallback")
		return
	}

	// Verify we got the module
	if L.GetGlobal("_SessionModule") == lua.LNil {
		r.Log(2, "LuaRuntime: session module returned nil")
	}
}

// Log logs a message via the config.
func (r *Runtime) Log(level int, format string, args ...interface{}) {
	r.config.Log(level, format, args...)
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

	// Create session in variable store with LuaResolver
	r.variableStore.CreateSession(vendedID, &LuaResolver{L: r.state})

	var luaSession *LuaSession
	_, err := r.execute(func() (interface{}, error) {
		L := r.state

		// Create session table for this frontend session
		sessionTable := r.createSessionTable(L, vendedID)

		luaSession = &LuaSession{
			Runtime:      r,
			ID:           vendedID,
			sessionTable: sessionTable,
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
			r.variableStore.DestroySession(vendedID)
			return nil, err
		}

		r.Log(2, "LuaRuntime: created Lua session %s", vendedID)

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
	r.Log(2, "LuaRuntime: no main.lua found (hybrid mode or backend-only)")
	return nil
}

// DestroyLuaSession removes a Lua session by its vended ID.
func (r *Runtime) DestroyLuaSession(vendedID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, vendedID)
	r.Log(2, "LuaRuntime: destroyed Lua session %s", vendedID)
}

// GetLuaSession retrieves a Lua session by its vended ID.
func (r *Runtime) GetLuaSession(vendedID string) (*LuaSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[vendedID]
	return session, ok
}

// createSessionTable creates the session object using lib/lua/session.lua module.
// Falls back to inline creation if module not loaded (for testing).
// vendedID is the compact session ID (e.g., "1", "2") exposed to Lua code.
func (r *Runtime) createSessionTable(L *lua.LState, vendedID string) *lua.LTable {
	// Get Session class from loaded module
	sessionModule := L.GetGlobal("_SessionModule")
	var session *lua.LTable

	if sessionModule != lua.LNil {
		// Use session module
		sessionModTbl := sessionModule.(*lua.LTable)
		SessionClass := L.GetField(sessionModTbl, "Session").(*lua.LTable)

		// Call Session.new() to create session instance (no backend = embedded mode)
		L.Push(L.GetField(SessionClass, "new"))
		L.Push(SessionClass)
		if err := L.PCall(1, 1, nil); err != nil {
			r.Log(0, "Session.new() failed: %v", err)
			session = r.createFallbackSessionTable(L, vendedID)
		} else {
			session = L.Get(-1).(*lua.LTable)
			L.Pop(1)
		}
	} else {
		// Fallback for tests - create minimal session table
		session = r.createFallbackSessionTable(L, vendedID)
	}

	// Store session ID
	L.SetField(session, "_sessionID", lua.LString(vendedID))

	// Inject Go functions (only if module-based session)
	if sessionModule != lua.LNil {
		r.injectSessionFunctions(L, session, vendedID)
	}

	// Add Go-specific methods that need access to Go structs
	r.addGoSessionMethods(L, session, vendedID)

	return session
}

// createFallbackSessionTable creates a minimal session table for testing when module not loaded.
func (r *Runtime) createFallbackSessionTable(L *lua.LState, vendedID string) *lua.LTable {
	session := L.NewTable()
	L.SetField(session, "_variables", L.NewTable())
	L.SetField(session, "_watchers", L.NewTable())
	// Create weak-keyed _objectToId table
	objectToId := L.NewTable()
	mt := L.NewTable()
	L.SetField(mt, "__mode", lua.LString("k"))
	L.SetMetatable(objectToId, mt)
	L.SetField(session, "_objectToId", objectToId)
	return session
}

// injectSessionFunctions injects Go backend functions into a Lua session.
func (r *Runtime) injectSessionFunctions(L *lua.LState, session *lua.LTable, vendedID string) {
	// _setGetValueFn - get variable value
	setGetValueFn := L.GetField(session, "_setGetValueFn")
	if setGetValueFn != lua.LNil {
		L.Push(setGetValueFn)
		L.Push(session)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			id := L.CheckInt64(1)
			value, _, ok := r.variableStore.Get(id)
			if !ok {
				L.Push(lua.LNil)
				return 1
			}
			var goVal interface{}
			if len(value) > 0 {
				json.Unmarshal(value, &goVal)
			}
			L.Push(goToLua(L, goVal))
			return 1
		}))
		L.PCall(2, 0, nil)
	}

	// _setGetPropertyFn - get variable property
	setGetPropertyFn := L.GetField(session, "_setGetPropertyFn")
	if setGetPropertyFn != lua.LNil {
		L.Push(setGetPropertyFn)
		L.Push(session)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			id := L.CheckInt64(1)
			name := L.CheckString(2)
			prop, ok := r.variableStore.GetProperty(id, name)
			if !ok {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(lua.LString(prop))
			return 1
		}))
		L.PCall(2, 0, nil)
	}

	// _setCreateFn - create variable (basic version, used by session.lua createVariable)
	setCreateFn := L.GetField(session, "_setCreateFn")
	if setCreateFn != lua.LNil {
		L.Push(setCreateFn)
		L.Push(session)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			parentID := L.CheckInt64(1)
			luaObject := L.CheckTable(2)
			propsTable := L.OptTable(3, nil)

			props := make(map[string]string)
			if propsTable != nil {
				propsTable.ForEach(func(k, v lua.LValue) {
					if ks, ok := k.(lua.LString); ok {
						props[string(ks)] = lua.LVAsString(v)
					}
				})
			}

			// Extract type from metatable (frictionless convention)
			r.extractTypeProperty(L, luaObject, props)

			id, err := r.variableStore.CreateVariable(vendedID, parentID, luaObject, props)
			if err != nil {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(lua.LNumber(id))
			return 1
		}))
		L.PCall(2, 0, nil)
	}

	// _setUpdateFn - update variable
	setUpdateFn := L.GetField(session, "_setUpdateFn")
	if setUpdateFn != lua.LNil {
		L.Push(setUpdateFn)
		L.Push(session)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			id := L.CheckInt64(1)
			value := L.Get(2)
			propsTable := L.OptTable(3, nil)

			var jsonValue json.RawMessage
			if value != lua.LNil {
				goValue := luaToGo(value)
				if goValue != nil {
					data, _ := json.Marshal(goValue)
					jsonValue = data
				}
			}

			var props map[string]string
			if propsTable != nil {
				props = make(map[string]string)
				propsTable.ForEach(func(k, v lua.LValue) {
					if ks, ok := k.(lua.LString); ok {
						props[string(ks)] = lua.LVAsString(v)
					}
				})
			}

			r.variableStore.Update(id, jsonValue, props)
			return 0
		}))
		L.PCall(2, 0, nil)
	}

	// _setDestroyFn - destroy variable
	setDestroyFn := L.GetField(session, "_setDestroyFn")
	if setDestroyFn != lua.LNil {
		L.Push(setDestroyFn)
		L.Push(session)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			id := L.CheckInt64(1)
			r.variableStore.Destroy(id)
			return 0
		}))
		L.PCall(2, 0, nil)
	}
}

// addGoSessionMethods adds Go-specific methods that need access to Go structs.
func (r *Runtime) addGoSessionMethods(L *lua.LState, session *lua.LTable, vendedID string) {
	// createAppVariable - creates variable 1 and stores reference in Go struct
	L.SetField(session, "createAppVariable", L.NewFunction(func(L *lua.LState) int {
		luaObject := L.CheckTable(2)
		propsTable := L.OptTable(3, nil)

		props := make(map[string]string)
		if propsTable != nil {
			propsTable.ForEach(func(k, v lua.LValue) {
				if ks, ok := k.(lua.LString); ok {
					props[string(ks)] = lua.LVAsString(v)
				}
			})
		}

		r.extractTypeProperty(L, luaObject, props)

		// Create app variable (parentID 0)
		id, err := r.variableStore.CreateVariable(vendedID, 0, luaObject, props)
		if err != nil {
			L.RaiseError("failed to create app variable: %v", err)
			return 0
		}

		// Store in Go struct for getApp() access
		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			luaSess.appVariableID = id
			luaSess.appObject = luaObject
		}

		// Track in session's _objectToId
		objectToId := L.GetField(session, "_objectToId")
		if objectToId != lua.LNil {
			L.SetField(objectToId.(*lua.LTable), "", lua.LNumber(id)) // weak key
			L.RawSet(objectToId.(*lua.LTable), luaObject, lua.LNumber(id))
		}

		r.Log(2, "LuaRuntime: created app variable %d for session %s", id, vendedID)

		L.Push(lua.LNumber(id))
		return 1
	}))

	// getApp - returns the Lua app object directly (not a wrapper)
	L.SetField(session, "getApp", L.NewFunction(func(L *lua.LState) int {
		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok || luaSess.appObject == nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(luaSess.appObject)
		return 1
	}))

	// Override createVariable to support parent lookup by object reference
	L.SetField(session, "createVariable", L.NewFunction(func(L *lua.LState) int {
		var parentID int64
		parentArg := L.Get(2)
		switch p := parentArg.(type) {
		case lua.LNumber:
			parentID = int64(p)
		case *lua.LTable:
			// Look up variable ID by object reference
			tracker := r.variableStore.GetTracker(vendedID)
			if tracker != nil {
				for _, v := range tracker.RootVariables() {
					if v.Value == p {
						parentID = v.ID
						break
					}
				}
			}
			if parentID == 0 {
				L.RaiseError("parent object not found in tracker")
				return 0
			}
		default:
			L.RaiseError("createVariable: parentId must be a number or table")
			return 0
		}

		luaObject := L.CheckTable(3)
		propsTable := L.OptTable(4, nil)

		props := make(map[string]string)
		if propsTable != nil {
			propsTable.ForEach(func(k, v lua.LValue) {
				if ks, ok := k.(lua.LString); ok {
					props[string(ks)] = lua.LVAsString(v)
				}
			})
		}

		r.extractTypeProperty(L, luaObject, props)

		id, err := r.variableStore.CreateVariable(vendedID, parentID, luaObject, props)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}

		// Track in session's _objectToId
		objectToId := L.GetField(session, "_objectToId")
		if objectToId != lua.LNil {
			L.RawSet(objectToId.(*lua.LTable), luaObject, lua.LNumber(id))
		}

		L.Push(lua.LNumber(id))
		return 1
	}))

	// Override destroyVariable to support object reference lookup
	L.SetField(session, "destroyVariable", L.NewFunction(func(L *lua.LState) int {
		var id int64
		arg := L.Get(2)
		switch v := arg.(type) {
		case lua.LNumber:
			id = int64(v)
		case *lua.LTable:
			tracker := r.variableStore.GetTracker(vendedID)
			if tracker != nil {
				foundID, found := tracker.LookupObject(v)
				if found {
					id = foundID
				}
			}
			if id == 0 {
				return 0 // Object not found - nothing to destroy
			}
		default:
			L.RaiseError("destroyVariable: argument must be a number or table")
			return 0
		}

		if err := r.variableStore.Destroy(id); err != nil {
			r.Log(2, "LuaRuntime: destroyVariable error: %v", err)
		}

		// Remove from session's _variables cache
		variables := L.GetField(session, "_variables")
		if variables != lua.LNil {
			L.SetField(variables.(*lua.LTable), fmt.Sprintf("%d", id), lua.LNil)
		}

		return 0
	}))
}

// extractTypeProperty extracts type from metatable or direct field (frictionless convention).
func (r *Runtime) extractTypeProperty(L *lua.LState, obj *lua.LTable, props map[string]string) {
	if props["type"] != "" {
		return
	}
	// First check metatable
	mt := L.GetMetatable(obj)
	if mt != lua.LNil {
		if mtTbl, ok := mt.(*lua.LTable); ok {
			if typeVal := L.GetField(mtTbl, "type"); typeVal != lua.LNil {
				props["type"] = lua.LVAsString(typeVal)
				return
			}
		}
	}
	// Fall back to direct "type" field
	if typeVal := L.GetField(obj, "type"); typeVal != lua.LNil {
		props["type"] = lua.LVAsString(typeVal)
	}
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
				r.Log(0, "Watcher callback error: %v", err)
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
	// Use tracker's DetectChanges
	changes := r.variableStore.DetectChanges(vendedID)
	if len(changes) == 0 {
		return nil
	}

	tracker := r.variableStore.GetTracker(vendedID)
	if tracker == nil {
		return nil
	}

	var updates []VariableUpdate
	for _, change := range changes {
		if change.ValueChanged {
			v := tracker.GetVariable(change.VariableID)
			if v == nil {
				continue
			}

			jsonBytes, err := tracker.ToValueJSONBytes(v.Value)
			if err != nil {
				r.Log(1, "AfterBatch: failed to marshal variable %d: %v", change.VariableID, err)
				continue
			}

			r.Log(2, "AfterBatch: variable %d changed", change.VariableID)

			updates = append(updates, VariableUpdate{
				VarID: change.VariableID,
				Value: json.RawMessage(jsonBytes),
			})

			// Also update the variable store so watchers get notified
			if err := r.variableStore.Update(change.VariableID, json.RawMessage(jsonBytes), nil); err != nil {
				r.Log(1, "AfterBatch: failed to update store for variable %d: %v", change.VariableID, err)
			}
		}
	}

	return updates
}

// HandleFrontendCreate handles a variable create message from the frontend.
// For path-based variables, it creates the variable in the tracker and resolves the path.
// If a wrapper property is set, the wrapper transforms the value.
// Returns the variable ID and resolved value.
func (r *Runtime) HandleFrontendCreate(sessionID string, parentID int64, properties map[string]string) (int64, json.RawMessage, error) {
	path := properties["path"]
	if path == "" {
		return 0, nil, fmt.Errorf("HandleFrontendCreate: path property required")
	}

	// Lookup session directly
	r.mu.RLock()
	session := r.sessions[sessionID]
	r.mu.RUnlock()

	if session == nil {
		return 0, nil, fmt.Errorf("session %s not found", sessionID)
	}

	tracker := r.variableStore.GetTracker(session.ID)
	if tracker == nil {
		return 0, nil, fmt.Errorf("session %s tracker not found", session.ID)
	}

	// Create the child variable in the tracker with the path
	v := tracker.CreateVariable(nil, parentID, path, properties)

	// Resolve the path to get the initial value
	resolvedValue, err := v.Get()
	if err != nil {
		r.Log(1, "HandleFrontendCreate: path resolution failed for %s: %v", path, err)
		// Return variable ID but nil value - frontend will see empty
		return v.ID, nil, nil
	}

	// Convert to JSON
	jsonValue, err := tracker.ToValueJSONBytes(resolvedValue)
	if err != nil {
		r.Log(1, "HandleFrontendCreate: JSON conversion failed: %v", err)
		return v.ID, nil, nil
	}

	// Check for wrapper and apply it
	wrapperType := properties["wrapper"]
	if wrapperType != "" {
		factory, ok := GetGlobalWrapperFactory(wrapperType)
		if ok {
			// Create a WrapperVariable adapter for the tracker variable
			wrapperVar := &trackerVariableAdapter{
				Variable: v,
			}
			wrapper := factory(session, wrapperVar)
			if wrapper != nil {
				// The wrapper itself is the new value
				jsonValue, err = tracker.ToValueJSONBytes(wrapper)
				if err != nil {
					r.Log(1, "HandleFrontendCreate: wrapper JSON conversion failed: %v", err)
				} else {
					r.Log(2, "HandleFrontendCreate: wrapper %s transformed value to %s", wrapperType, string(jsonValue))
				}
			}
		} else {
			r.Log(1, "HandleFrontendCreate: wrapper type %s not found", wrapperType)
		}
	}

	r.Log(2, "HandleFrontendCreate: created var %d for path %s, value=%s", v.ID, path, string(jsonValue))

	return v.ID, jsonValue, nil
}

// trackerVariableAdapter adapts a change-tracker Variable to WrapperVariable interface
type trackerVariableAdapter struct {
	*changetracker.Variable
}

func (a *trackerVariableAdapter) GetID() int64 {
	return a.ID
}

func (a *trackerVariableAdapter) GetValue() interface{} {
	return a.Value
}

func (a *trackerVariableAdapter) GetProperty(name string) string {
	return a.Properties[name]
}

// HandleFrontendUpdate handles an update to a path-based variable from frontend.
// Updates the backend object via the variable's path using v.Set().
// CRC: crc-LuaRuntime.md
// Sequence: seq-relay-message.md
func (r *Runtime) HandleFrontendUpdate(sessionID string, varID int64, value json.RawMessage) error {
	tracker := r.variableStore.GetTracker(sessionID)
	if tracker == nil {
		return fmt.Errorf("session %s tracker not found", sessionID)
	}

	v := tracker.GetVariable(varID)
	if v == nil {
		return fmt.Errorf("variable %d not found in tracker", varID)
	}

	// Parse the JSON value to a Go value
	var goValue interface{}
	if err := json.Unmarshal(value, &goValue); err != nil {
		return fmt.Errorf("failed to parse value: %w", err)
	}

	// Update the backend object via the variable's path
	if err := v.Set(goValue); err != nil {
		r.Log(1, "HandleFrontendUpdate: Set failed for var %d: %v", varID, err)
		return err
	}

	r.Log(2, "HandleFrontendUpdate: updated var %d with value %s", varID, string(value))

	return nil
}

// registerRequire adds a custom require() function that works with both
// filesystem (--dir mode) and embedded bundle.
func (r *Runtime) registerRequire() {
	L := r.state

	// Table to cache loaded modules (like package.loaded)
	loaded := L.NewTable()

	requireFn := L.NewFunction(func(L *lua.LState) int {
		modName := L.CheckString(1)

		// Check if already loaded
		if cached := L.GetField(loaded, modName); cached != lua.LNil {
			L.Push(cached)
			return 1
		}

		// Convert module name to filename (e.g., "foo.bar" -> "foo/bar.lua")
		filename := strings.ReplaceAll(modName, ".", string(filepath.Separator)) + ".lua"

		var code string

		// Try filesystem first (works for --dir mode)
		filePath := filepath.Join(r.luaDir, filename)
		content, fsErr := os.ReadFile(filePath)
		if fsErr == nil {
			code = string(content)
		} else {
			// Try bundle (works for bundled binaries)
			bundlePath := "lua/" + strings.ReplaceAll(modName, ".", "/") + ".lua"
			bundleContent, bundleErr := bundle.ReadFile(bundlePath)
			if bundleErr == nil {
				code = string(bundleContent)
			} else {
				L.RaiseError("module '%s' not found:\n\tno file '%s'\n\tno bundled file '%s'",
					modName, filePath, bundlePath)
				return 0
			}
		}

		// Execute the module code
		if err := L.DoString(code); err != nil {
			L.RaiseError("error loading module '%s': %v", modName, err)
			return 0
		}

		// Get the return value (module table) or true if nothing returned
		result := L.Get(-1)
		if result == lua.LNil {
			result = lua.LTrue
		}

		// Cache and return
		L.SetField(loaded, modName, result)
		L.Push(result)
		return 1
	})

	// Set as global require
	L.SetGlobal("require", requireFn)

	// Also expose package.loaded for compatibility
	pkg := L.NewTable()
	L.SetField(pkg, "loaded", loaded)
	L.SetGlobal("package", pkg)
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

		r.Log(2, "LuaRuntime: registered presenter type %s", name)

		return 0
	}))

	// ui.log([level,] message)
	L.SetField(uiMod, "log", L.NewFunction(func(L *lua.LState) int {
		top := L.GetTop()
		var level int
		var msg string

		if top == 1 {
			level = 0
			msg = L.CheckString(1)
		} else {
			level = L.CheckInt(1)
			msg = L.CheckString(2)
		}

		r.Log(level, "[lua] %s", msg)
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
		r.wrapperRegistry.Register(name, func(session *LuaSession, variable WrapperVariable) interface{} {
			return NewLuaWrapper(session, tbl, variable)
		})

		r.Log(2, "LuaRuntime: registered wrapper type %s", name)

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
	instance *lua.LTable
}

// CreateItemWrapper creates an ItemWrapper instance for a ViewListItem.
// The ItemWrapper constructor receives the ViewListItem: ItemWrapper(viewListItem).
// Returns nil if no itemType is specified or the type isn't registered.
func (r *Runtime) CreateItemWrapper(typeName string, viewItem *ViewListItem) (*ItemWrapperInstance, error) {
	if typeName == "" {
		return nil, nil
	}

	result, err := r.execute(func() (interface{}, error) {
		r.mu.RLock()
		pt, ok := r.presenterTypes[typeName]
		r.mu.RUnlock()

		if !ok {
			// Auto-discovery: Check if there's a global table with this name
			L := r.state
			val := L.GetGlobal(typeName)
			if tbl, ok := val.(*lua.LTable); ok {
				pt = &PresenterType{
					Name:    typeName,
					Methods: make(map[string]*lua.LFunction),
					Table:   tbl,
				}
				r.mu.Lock()
				r.presenterTypes[typeName] = pt
				r.mu.Unlock()

				r.Log(2, "LuaRuntime: auto-discovered presenter type %s", typeName)
			} else {
				return nil, fmt.Errorf("item wrapper type %s not found", typeName)
			}
		}

		L := r.state

		// Create new instance table
		instance := L.NewTable()

		// Set metatable to inherit from presenter type
		mt := L.NewTable()
		L.SetField(mt, "__index", pt.Table)
		L.SetMetatable(instance, mt)

		// Set ViewListItem properties on the instance
		// The presenter can access: viewListItem.item, viewListItem.list, viewListItem.index
		L.SetField(instance, "viewListItem", r.createViewListItemLuaWrapper(L, viewItem))

		// Call init method if exists, passing the viewListItem
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

		return &ItemWrapperInstance{
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

// createViewListItemLuaWrapper creates a Lua wrapper for a ViewListItem.
func (r *Runtime) createViewListItemLuaWrapper(L *lua.LState, viewItem *ViewListItem) *lua.LTable {
	wrapper := L.NewTable()

	// viewListItem.item - the domain object
	L.SetField(wrapper, "item", goToLua(L, viewItem.GetItem()))

	// viewListItem.index - position in list
	L.SetField(wrapper, "index", lua.LNumber(viewItem.GetIndex()))

	// viewListItem:remove() - removes this item from the list
	L.SetField(wrapper, "remove", L.NewFunction(func(L *lua.LState) int {
		if err := viewItem.Remove(); err != nil {
			r.Log(1, "ViewListItem.remove error: %v", err)
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
