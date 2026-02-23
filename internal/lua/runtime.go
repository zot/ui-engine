// CRC: crc-LuaSession.md, crc-LuaRuntime.md (alias), crc-LuaVariable.md, crc-Module.md
// Spec: interfaces.md, deployment.md, libraries.md, protocol.md, module-tracking.md
// Sequence: seq-lua-executor-init.md, seq-lua-execute.md, seq-lua-handle-action.md, seq-lua-session-init.md, seq-session-create-backend.md, seq-unload-module.md, seq-require-lua-file.md
//
// LuaSession provides per-session Lua isolation. Each frontend session gets its own
// LuaSession with a separate Lua VM state. Server owns luaSessions map and creates
// sessions via callbacks from SessionManager.
package lua

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"weak"

	lua "github.com/yuin/gopher-lua"
	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui-engine/internal/bundle"
	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/viewdef"
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

// LuaSession represents a Lua runtime environment for a frontend session.
// Each session has its own Lua state for complete isolation.
// ID is the vended session ID (compact integer string like "1", "2") for backend communication.
type LuaSession struct {
	// Lua VM state and execution
	State          *lua.LState
	loadedModules  *lua.LTable // Unified load tracker, keyed by baseDir-relative paths
	presenterTypes map[string]*PresenterType
	luaDir         string
	executorChan   chan WorkItem
	done           chan struct{}
	config         *config.Config
	mu             sync.RWMutex
	batchTriggered bool

	// Variable management
	variableStore   VariableStore
	mainLuaCode     string
	wrapperRegistry *WrapperRegistry
	viewdefManager  *viewdef.ViewdefManager

	// Session identity and state
	ID              string      // Vended session ID (e.g., "1", "2", "3")
	sessionTable    *lua.LTable // The session object exposed to Lua
	appVariableID   int64       // Variable 1 for this session (set by Lua code)
	appObject       *lua.LTable // Reference to the app Lua object
	McpState        *lua.LTable // Logical state root for MCP (defaults to appObject)
	McpStateID      int64       // Variable ID of mcpState (if tracked)
	mutationVersion int64       // Hot-loading mutation version for schema migrations (deprecated)

	// Prototype management for hot-loading
	prototypeRegistry map[string]*prototypeInfo      // name -> stored init copy for change detection
	instanceRegistry  map[*lua.LTable][]weakInstance // prototype -> weak list of instances
	mutationQueue     []mutationEntry                // FIFO queue of prototypes pending mutation

	// Module tracking for unloading
	modules           map[string]*Module   // tracking key -> Module instance
	moduleDirectories map[string][]*Module // directory path -> modules in that directory
	currentModule     *Module              // module being loaded (for resource tracking)
	hotLoaderCleanup  func(path string)    // callback to clean up HotLoader state
}

// prototypeInfo stores information about a registered prototype for change detection.
type prototypeInfo struct {
	prototype  *lua.LTable           // The prototype table
	storedInit map[string]lua.LValue // Shallow copy of init for change detection
}

// weakInstance holds a weak reference to a Lua table instance.
// The weak reference allows the instance to be garbage collected.
type weakInstance struct {
	ptr weak.Pointer[lua.LTable]
}

// mutationEntry represents a prototype queued for mutation processing.
type mutationEntry struct {
	prototype   *lua.LTable
	removedKeys []string
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
	DetectChanges(sessionID string) bool
	GetChanges(sessionID string) []changetracker.Change
}

// NewRuntime creates a new LuaSession with executor goroutine.
func NewRuntime(cfg *config.Config, luaDir string, vdm *viewdef.ViewdefManager) (*LuaSession, error) {
	L := lua.NewState()

	s := &LuaSession{
		config:            cfg,
		State:             L,
		loadedModules:     L.NewTable(), // Unified load tracker, keyed by baseDir-relative paths
		presenterTypes:    make(map[string]*PresenterType),
		luaDir:            luaDir,
		executorChan:      make(chan WorkItem, 100),
		done:              make(chan struct{}),
		viewdefManager:    vdm,
		prototypeRegistry: make(map[string]*prototypeInfo),
		instanceRegistry:  make(map[*lua.LTable][]weakInstance),
		modules:           make(map[string]*Module),
		moduleDirectories: make(map[string][]*Module),
	}

	// Load standard libraries
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)
	lua.OpenOs(L)

	// Register custom require() that works with both filesystem and bundle
	s.registerRequire()

	// Register UI module
	s.registerUIModule()

	// Register EMPTY global for declaring nil fields tracked for mutation
	s.registerEmptyGlobal()

	// Try to load session module (lib/lua/session.lua) - optional for testing
	s.loadSessionModule()

	// Start executor goroutine
	s.startExecutor()

	return s, nil
}

// loadSessionModule tries to load lib/lua/session.lua and stores the module globally.
// Returns silently if module not found (allows tests to work without it).
func (r *LuaSession) loadSessionModule() {
	L := r.State

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

// registerEmptyGlobal creates the EMPTY global used for declaring nil fields tracked for mutation.
// EMPTY is an empty table that acts as a marker in prototype init tables.
func (r *LuaSession) registerEmptyGlobal() {
	L := r.State
	empty := L.NewTable()
	L.SetGlobal("EMPTY", empty)
}

// Log logs a message via the config.
func (r *LuaSession) Log(level int, format string, args ...interface{}) {
	r.config.Log(level, format, args...)
}

// SetVariableStore sets the variable store for session operations.
func (r *LuaSession) SetVariableStore(store VariableStore) {
	r.variableStore = store
}

// SetWrapperRegistry sets the wrapper registry for registering Lua wrappers.
func (r *LuaSession) SetWrapperRegistry(registry *WrapperRegistry) {
	r.wrapperRegistry = registry
}

// GetGlobalTable looks up a Lua global by name and returns it if it's a table.
// Used for auto-discovery of Lua-defined wrappers.
// Returns nil if the global doesn't exist or isn't a table.
func (r *LuaSession) GetGlobalTable(name string) interface{} {
	var result interface{}
	r.execute(func() (interface{}, error) {
		L := r.State
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
func (r *LuaSession) SetMainLuaCode(code string) {
	r.mainLuaCode = code
}

// CreateLuaSession initializes this LuaSession for a frontend session.
// vendedID is the compact session ID (e.g., "1", "2") for backend communication.
// Loads and executes main.lua with a session global.
// Must be called after SetVariableStore.
// Returns self after initialization.
func (s *LuaSession) CreateLuaSession(vendedID string) (*LuaSession, error) {
	if s.variableStore == nil {
		return nil, fmt.Errorf("variable store not set")
	}

	// Create resolver linked to this session
	resolver := &LuaResolver{Session: s}

	// Create session in variable store with LuaResolver
	s.variableStore.CreateSession(vendedID, resolver)

	_, err := s.execute(func() (interface{}, error) {
		// Create session table for this frontend session
		sessionTable := s.createSessionTable(vendedID)

		// Initialize session-specific fields on self
		s.ID = vendedID
		s.sessionTable = sessionTable

		// Set session global
		s.State.SetGlobal("session", sessionTable)

		// Load main.lua for this session
		if err := s.loadMainLua(); err != nil {
			s.variableStore.DestroySession(vendedID)
			return nil, err
		}
		s.Log(2, "LuaRuntime: created Lua session %s", vendedID)

		return nil, nil
	})

	if err != nil {
		return nil, err
	}
	return s, nil
}

// loadMainLua loads main.lua from filesystem or cached bundle code.
// Registers the file in loadedModules for hot-reload tracking.
// Uses ComputeTrackingKey to handle symlinks correctly.
func (r *LuaSession) loadMainLua() error {
	// Try cached bundle code first
	if r.mainLuaCode != "" {
		// Mark as loaded for hot-reload tracking
		r.State.SetField(r.loadedModules, "main.lua", lua.LTrue)
		if err := r.State.DoString(r.mainLuaCode); err != nil {
			r.State.SetField(r.loadedModules, "main.lua", lua.LNil) // Unmark on error
			return fmt.Errorf("failed to execute main.lua: %w", err)
		}
		return nil
	}

	// Try filesystem
	mainPath := filepath.Join(r.luaDir, "main.lua")
	if _, err := os.Stat(mainPath); err == nil {
		// Compute tracking key for hot-reload (resolves symlinks)
		trackingKey := "main.lua" // fallback
		if r.config != nil && r.config.Server.Dir != "" {
			if key, err := ComputeTrackingKey(r.config.Server.Dir, mainPath); err == nil {
				trackingKey = key
			}
		}
		// Mark as loaded for hot-reload tracking
		r.State.SetField(r.loadedModules, trackingKey, lua.LTrue)
		if err := r.State.DoFile(mainPath); err != nil {
			r.State.SetField(r.loadedModules, trackingKey, lua.LNil) // Unmark on error
			return fmt.Errorf("failed to load main.lua: %w", err)
		}
		return nil
	}

	// No main.lua found - this is OK for hybrid mode where backend creates variable 1
	r.Log(2, "LuaRuntime: no main.lua found (hybrid mode or backend-only)")
	return nil
}

// DestroyLuaSession cleans up a Lua session.
// Note: The actual Lua state cleanup is handled by Server calling Shutdown().
func (r *LuaSession) DestroyLuaSession(vendedID string) {
	r.Log(2, "LuaRuntime: destroyed Lua session %s", vendedID)
}

// GetLuaSession returns this session if the vendedID matches.
// With per-session isolation, each LuaSession IS the session.
func (r *LuaSession) GetLuaSession(vendedID string) (*LuaSession, bool) {
	if r.ID == vendedID {
		return r, true
	}
	return nil, false
}

// createSessionTable creates the session object using lib/lua/session.lua module.
// Falls back to inline creation if module not loaded (for testing).
// vendedID is the compact session ID (e.g., "1", "2") exposed to Lua code.
func (r *LuaSession) createSessionTable(vendedID string) *lua.LTable {
	// Get Session class from loaded module
	sessionModule := r.State.GetGlobal("_SessionModule")
	var session *lua.LTable

	if sessionModule != lua.LNil {
		// Use session module
		sessionModTbl := sessionModule.(*lua.LTable)
		SessionClass := r.State.GetField(sessionModTbl, "Session").(*lua.LTable)

		// Call Session.new() to create session instance (no backend = embedded mode)
		r.State.Push(r.State.GetField(SessionClass, "new"))
		r.State.Push(SessionClass)
		if err := r.State.PCall(1, 1, nil); err != nil {
			r.Log(0, "Session.new() failed: %v", err)
			session = r.createFallbackSessionTable(vendedID)
		} else {
			session = r.State.Get(-1).(*lua.LTable)
			r.State.Pop(1)
		}
	} else {
		// Fallback for tests - create minimal session table
		session = r.createFallbackSessionTable(vendedID)
	}

	// Store session ID
	r.State.SetField(session, "_sessionID", lua.LString(vendedID))

	// Initialize reloading flag (used by hot-loader)
	r.State.SetField(session, "reloading", lua.LFalse)

	// Inject Go functions (only if module-based session)
	if sessionModule != lua.LNil {
		r.injectSessionFunctions(session, vendedID)
	}

	// Add Go-specific methods that need access to Go structs
	r.addGoSessionMethods(session, vendedID)

	return session
}

// createFallbackSessionTable creates a minimal session table for testing when module not loaded.
func (r *LuaSession) createFallbackSessionTable(vendedID string) *lua.LTable {
	session := r.State.NewTable()
	r.State.SetField(session, "_variables", r.State.NewTable())
	r.State.SetField(session, "_watchers", r.State.NewTable())
	// Create weak-keyed _objectToId table
	objectToId := r.State.NewTable()
	mt := r.State.NewTable()
	r.State.SetField(mt, "__mode", lua.LString("k"))
	r.State.SetMetatable(objectToId, mt)
	r.State.SetField(session, "_objectToId", objectToId)
	return session
}

// injectSessionFunctions injects Go backend functions into a Lua session.
func (r *LuaSession) injectSessionFunctions(session *lua.LTable, vendedID string) {
	// _setGetValueFn - get variable value
	setGetValueFn := r.State.GetField(session, "_setGetValueFn")
	if setGetValueFn != lua.LNil {
		r.State.Push(setGetValueFn)
		r.State.Push(session)
		r.State.Push(r.State.NewFunction(func(L *lua.LState) int {
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
			L.Push(r.GoToLua(goVal))
			return 1
		}))
		r.State.PCall(2, 0, nil)
	}

	// _setGetPropertyFn - get variable property
	setGetPropertyFn := r.State.GetField(session, "_setGetPropertyFn")
	if setGetPropertyFn != lua.LNil {
		r.State.Push(setGetPropertyFn)
		r.State.Push(session)
		r.State.Push(r.State.NewFunction(func(L *lua.LState) int {
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
		r.State.PCall(2, 0, nil)
	}

	// _setCreateFn - create variable (basic version, used by session.lua createVariable)
	setCreateFn := r.State.GetField(session, "_setCreateFn")
	if setCreateFn != lua.LNil {
		r.State.Push(setCreateFn)
		r.State.Push(session)
		r.State.Push(r.State.NewFunction(func(L *lua.LState) int {
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
			r.extractTypeProperty(luaObject, props)

			id, err := r.variableStore.CreateVariable(vendedID, parentID, luaObject, props)
			if err != nil {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(lua.LNumber(id))
			return 1
		}))
		r.State.PCall(2, 0, nil)
	}

	// _setUpdateFn - update variable
	setUpdateFn := r.State.GetField(session, "_setUpdateFn")
	if setUpdateFn != lua.LNil {
		r.State.Push(setUpdateFn)
		r.State.Push(session)
		r.State.Push(r.State.NewFunction(func(L *lua.LState) int {
			id := L.CheckInt64(1)
			value := L.Get(2)
			propsTable := L.OptTable(3, nil)

			var jsonValue json.RawMessage
			if value != lua.LNil {
				goValue := LuaToGo(value)
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
		r.State.PCall(2, 0, nil)
	}

	// _setDestroyFn - destroy variable
	setDestroyFn := r.State.GetField(session, "_setDestroyFn")
	if setDestroyFn != lua.LNil {
		r.State.Push(setDestroyFn)
		r.State.Push(session)
		r.State.Push(r.State.NewFunction(func(L *lua.LState) int {
			id := L.CheckInt64(1)
			r.variableStore.Destroy(id)
			return 0
		}))
		r.State.PCall(2, 0, nil)
	}
}

// addGoSessionMethods adds Go-specific methods that need access to Go structs.
func (r *LuaSession) addGoSessionMethods(session *lua.LTable, vendedID string) {
	// createAppVariable - creates variable 1 and stores reference in Go struct
	r.State.SetField(session, "createAppVariable", r.State.NewFunction(func(L *lua.LState) int {
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

		r.extractTypeProperty(luaObject, props)

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
			// Default MCP state to app object
			if luaSess.McpState == nil {
				luaSess.McpState = luaObject
				luaSess.McpStateID = id
			}
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
	r.State.SetField(session, "getApp", r.State.NewFunction(func(L *lua.LState) int {
		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok || luaSess.appObject == nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(luaSess.appObject)
		return 1
	}))

	// Override createVariable to support parent lookup by object reference
	r.State.SetField(session, "createVariable", r.State.NewFunction(func(L *lua.LState) int {
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

		r.extractTypeProperty(luaObject, props)

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
	r.State.SetField(session, "destroyVariable", r.State.NewFunction(func(L *lua.LState) int {
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

	// newVersion - increment mutation version for hot-loading schema migrations
	r.State.SetField(session, "newVersion", r.State.NewFunction(func(L *lua.LState) int {
		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok {
			L.Push(lua.LNumber(0))
			return 1
		}
		luaSess.mutationVersion++
		L.Push(lua.LNumber(luaSess.mutationVersion))
		return 1
	}))

	// getVersion - get current mutation version
	r.State.SetField(session, "getVersion", r.State.NewFunction(func(L *lua.LState) int {
		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok {
			L.Push(lua.LNumber(0))
			return 1
		}
		L.Push(lua.LNumber(luaSess.mutationVersion))
		return 1
	}))

	// needsMutation - check if object needs migration (obj._mutationVersion < session version)
	// DEPRECATED: Use session:prototype() for automatic mutation instead
	r.State.SetField(session, "needsMutation", r.State.NewFunction(func(L *lua.LState) int {
		obj := L.CheckTable(2)

		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok {
			L.Push(lua.LFalse)
			return 1
		}

		// Get obj._mutationVersion (defaults to 0 if not set)
		objVersion := L.GetField(obj, "_mutationVersion")
		var objVersionNum int64
		if num, ok := objVersion.(lua.LNumber); ok {
			objVersionNum = int64(num)
		}

		// Return true if object version is less than session version
		L.Push(lua.LBool(objVersionNum < luaSess.mutationVersion))
		return 1
	}))

	// prototype - declare/update a prototype with instance field tracking
	// session:prototype(name, init, base) -> prototype table
	r.State.SetField(session, "prototype", r.State.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(2)
		var init *lua.LTable
		if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
			init = L.CheckTable(3)
		}
		var base *lua.LTable
		if L.GetTop() >= 4 && L.Get(4) != lua.LNil {
			base = L.CheckTable(4)
		}

		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}

		prototype := luaSess.prototypeImpl(name, init, base)
		L.Push(prototype)
		return 1
	}))

	// create - create a tracked instance with weak reference
	// session:create(prototype, instance) -> instance table
	r.State.SetField(session, "create", r.State.NewFunction(func(L *lua.LState) int {
		prototype := L.CheckTable(2)
		var instance *lua.LTable
		if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
			instance = L.CheckTable(3)
		} else {
			instance = L.NewTable()
		}

		luaSess, ok := r.GetLuaSession(vendedID)
		if !ok {
			L.Push(instance)
			return 1
		}

		luaSess.createInstance(prototype, instance)
		L.Push(instance)
		return 1
	}))

	// removePrototype - remove a prototype from the registry
	// session:removePrototype(name, children) -> nil
	// CRC: crc-LuaSession.md
	r.State.SetField(session, "removePrototype", r.State.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(2)
		children := false
		if L.GetTop() >= 3 {
			children = L.ToBool(3)
		}

		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			luaSess.RemovePrototype(name, children)
		}
		return 0
	}))

	// unloadModule - remove all tracking related to a module
	// session:unloadModule(moduleName) -> nil
	// Seq: seq-unload-module.md
	r.State.SetField(session, "unloadModule", r.State.NewFunction(func(L *lua.LState) int {
		moduleName := L.CheckString(2)

		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			luaSess.UnloadModule(moduleName)
		}
		return 0
	}))

	// unloadDirectory - unload all modules in a directory
	// session:unloadDirectory(dirPath) -> nil
	// Seq: seq-unload-module.md
	r.State.SetField(session, "unloadDirectory", r.State.NewFunction(func(L *lua.LState) int {
		dirPath := L.CheckString(2)

		luaSess, ok := r.GetLuaSession(vendedID)
		if ok {
			luaSess.UnloadDirectory(dirPath)
		}
		return 0
	}))
}

// prototypeImpl implements session:prototype(name, init, base).
// Creates or updates a prototype with automatic change detection and mutation queueing.
// Uses prototypeRegistry for lookup (not Lua globals), enabling dotted names like "contacts.Contact".
// If base is nil, defaults to registered "Object" prototype (if exists).
// Returns the prototype for the caller to assign to a global.
func (r *LuaSession) prototypeImpl(name string, init *lua.LTable, base *lua.LTable) *lua.LTable {
	empty := r.State.GetGlobal("EMPTY")

	// Resolve base prototype: use provided base, or default to "Object" if registered
	if base == nil {
		if objectInfo := r.prototypeRegistry["Object"]; objectInfo != nil {
			base = objectInfo.prototype
		}
	}

	// Look up in prototype registry (NOT Lua globals)
	info := r.prototypeRegistry[name]

	if info == nil {
		// Create new prototype
		var prototype *lua.LTable
		if init != nil {
			prototype = init
		} else {
			prototype = r.State.NewTable()
		}

		// Set type and __index for instance method lookup
		r.State.SetField(prototype, "type", lua.LString(name))
		r.State.SetField(prototype, "__index", prototype)

		// Set up inheritance chain if base exists
		if base != nil {
			r.State.SetMetatable(prototype, base)
		} else {
			// Add default new method only if no base (base provides :new() via inheritance)
			if r.State.GetField(prototype, "new") == lua.LNil {
				r.State.SetField(prototype, "new", r.State.NewFunction(func(L *lua.LState) int {
					var instance *lua.LTable
					if L.GetTop() >= 2 && L.Get(2) != lua.LNil {
						instance = L.CheckTable(2)
					} else {
						instance = L.NewTable()
					}
					r.createInstance(prototype, instance)
					L.Push(instance)
					return 1
				}))
			}
		}

		// Store shallow copy of init for change detection
		storedInit := r.copyInitTable(init, empty)

		// Remove EMPTY values from prototype (they should default to nil)
		r.removeEmptyValues(prototype, empty)

		// Register prototype in registry
		r.prototypeRegistry[name] = &prototypeInfo{
			prototype:  prototype,
			storedInit: storedInit,
		}

		// Initialize instance tracking for this prototype
		r.instanceRegistry[prototype] = nil

		// Track prototype in current module
		if r.currentModule != nil {
			r.currentModule.AddPrototype(name)
		}

		return prototype
	}

	// Prototype already exists in registry
	prototype := info.prototype

	// If init is nil, just return existing prototype (no update)
	if init == nil {
		return prototype
	}

	// Check if init differs from stored copy
	newInit := r.copyInitTable(init, empty)
	if r.initTablesEqual(info.storedInit, newInit) {
		return prototype
	}

	// Init changed - compute removed keys
	removedKeys := r.computeRemovedKeys(info.storedInit, newInit)

	// Update prototype with new init values
	r.updatePrototype(prototype, init, empty)

	// Store new init copy
	info.storedInit = newInit

	// Queue for mutation
	r.mutationQueue = append(r.mutationQueue, mutationEntry{
		prototype:   prototype,
		removedKeys: removedKeys,
	})

	return prototype
}

// createInstance implements session:create(prototype, instance).
// Sets metatable and registers instance for tracking with weak reference.
func (r *LuaSession) createInstance(prototype, instance *lua.LTable) {
	// Set metatable to prototype
	r.State.SetMetatable(instance, prototype)

	// Register instance for tracking with weak reference
	weakInst := weakInstance{ptr: weak.Make(instance)}
	r.instanceRegistry[prototype] = append(r.instanceRegistry[prototype], weakInst)
}

// copyInitTable creates a shallow copy of init table for change detection.
// Preserves EMPTY markers in the copy.
func (r *LuaSession) copyInitTable(init *lua.LTable, empty lua.LValue) map[string]lua.LValue {
	result := make(map[string]lua.LValue)
	if init == nil {
		return result
	}

	init.ForEach(func(key, value lua.LValue) {
		if keyStr, ok := key.(lua.LString); ok {
			keyName := string(keyStr)
			// Skip special fields
			if keyName == "type" || keyName == "__index" || keyName == "new" {
				return
			}
			result[keyName] = value
		}
	})
	return result
}

// removeEmptyValues removes EMPTY marker values from a table (they should default to nil).
func (r *LuaSession) removeEmptyValues(tbl *lua.LTable, empty lua.LValue) {
	var keysToRemove []lua.LValue
	tbl.ForEach(func(key, value lua.LValue) {
		if value == empty {
			keysToRemove = append(keysToRemove, key)
		}
	})
	for _, key := range keysToRemove {
		r.State.SetTable(tbl, key, lua.LNil)
	}
}

// initTablesEqual compares two init table copies for equality.
func (r *LuaSession) initTablesEqual(a, b map[string]lua.LValue) bool {
	if len(a) != len(b) {
		return false
	}
	for key, valA := range a {
		valB, exists := b[key]
		if !exists {
			return false
		}
		// Compare by identity for tables, by value for primitives
		if valA != valB {
			return false
		}
	}
	return true
}

// computeRemovedKeys returns keys present in old but not in new.
func (r *LuaSession) computeRemovedKeys(old, new map[string]lua.LValue) []string {
	var removed []string
	for key := range old {
		if _, exists := new[key]; !exists {
			removed = append(removed, key)
		}
	}
	return removed
}

// updatePrototype updates an existing prototype with new init values.
func (r *LuaSession) updatePrototype(prototype, init *lua.LTable, empty lua.LValue) {
	init.ForEach(func(key, value lua.LValue) {
		if keyStr, ok := key.(lua.LString); ok {
			keyName := string(keyStr)
			// Skip special fields
			if keyName == "type" || keyName == "__index" || keyName == "new" {
				return
			}
			// Set value (or nil if EMPTY)
			if value == empty {
				r.State.SetField(prototype, keyName, lua.LNil)
			} else {
				r.State.SetField(prototype, keyName, value)
			}
		}
	})
}

// ProcessMutationQueue processes queued prototypes after file load.
// Called after LoadCode completes. Uses executor for thread safety.
func (r *LuaSession) ProcessMutationQueue() {
	if len(r.mutationQueue) == 0 {
		return
	}

	r.execute(func() (interface{}, error) {
		r.processMutationQueueDirect()
		return nil, nil
	})
}

// processMutationQueueDirect processes mutations without executor wrapping.
// MUST only be called from within an execute() context.
func (r *LuaSession) processMutationQueueDirect() {
	if len(r.mutationQueue) == 0 {
		return
	}

	for _, entry := range r.mutationQueue {
		r.processMutationEntry(entry)
	}

	// Clear the queue
	r.mutationQueue = nil
}

// processMutationEntry processes a single mutation entry.
func (r *LuaSession) processMutationEntry(entry mutationEntry) {
	prototype := entry.prototype
	removedKeys := entry.removedKeys

	// Get instances and filter out dead weak refs (tortoise and hare compaction)
	weakInstances := r.instanceRegistry[prototype]
	n := 0
	for i := range weakInstances {
		if weakInstances[i].ptr.Value() != nil {
			weakInstances[n] = weakInstances[i]
			n++
		}
	}
	weakInstances = weakInstances[:n]
	r.instanceRegistry[prototype] = weakInstances

	// Check if prototype has a mutate method
	mutateMethod := r.State.GetField(prototype, "mutate")
	hasMutate := mutateMethod != lua.LNil && mutateMethod.Type() == lua.LTFunction

	// Process each live instance
	for i := range weakInstances {
		instance := weakInstances[i].ptr.Value()
		// Call mutate method if exists (wrapped in pcall for error isolation)
		if hasMutate {
			err := r.State.CallByParam(lua.P{
				Fn:      mutateMethod,
				NRet:    0,
				Protect: true,
			}, instance)
			if err != nil {
				r.Log(1, "LuaRuntime: mutate error for %v: %v", GetType(r.State, prototype), err)
			}
		}

		// Nil out removed fields
		for _, key := range removedKeys {
			r.State.SetField(instance, key, lua.LNil)
		}
	}
}

// RemovePrototype removes a prototype from the registry.
// If children is true, also removes prototypes whose name starts with "name." (dot-separated children).
// CRC: crc-LuaSession.md
func (r *LuaSession) RemovePrototype(name string, children bool) {
	if info, exists := r.prototypeRegistry[name]; exists {
		delete(r.instanceRegistry, info.prototype)
		delete(r.prototypeRegistry, name)
	}

	if !children {
		return
	}

	prefix := name + "."
	for childName, info := range r.prototypeRegistry {
		if strings.HasPrefix(childName, prefix) {
			delete(r.instanceRegistry, info.prototype)
			delete(r.prototypeRegistry, childName)
		}
	}
}

// SetHotLoaderCleanup sets the callback for cleaning up HotLoader state during unload.
func (r *LuaSession) SetHotLoaderCleanup(cleanup func(path string)) {
	r.hotLoaderCleanup = cleanup
}

// UnloadModule removes all tracking related to a module.
// This removes prototypes, presenter types, wrappers, and loadedModules entry.
// It also cleans up HotLoader state via the cleanup callback.
func (r *LuaSession) UnloadModule(moduleName string) {
	module, exists := r.modules[moduleName]
	if !exists {
		return
	}

	// Remove prototypes registered by this module
	for _, protoName := range module.Prototypes {
		r.RemovePrototype(protoName, false)
	}

	// Remove presenter types registered by this module
	for _, ptName := range module.PresenterTypes {
		delete(r.presenterTypes, ptName)
	}

	// Remove wrappers registered by this module
	if r.wrapperRegistry != nil {
		for _, wrapperName := range module.Wrappers {
			r.wrapperRegistry.Remove(wrapperName)
		}
	}

	// Remove from loadedModules
	r.loadedModules.RawSetString(moduleName, lua.LNil)

	// Clean up HotLoader state
	if r.hotLoaderCleanup != nil {
		r.hotLoaderCleanup(moduleName)
	}

	// Remove from moduleDirectories
	if mods, ok := r.moduleDirectories[module.Directory]; ok {
		for i, m := range mods {
			if m.Name == moduleName {
				r.moduleDirectories[module.Directory] = slices.Delete(mods, i, i+1)
				break
			}
		}
		// Clean up empty directory entry
		if len(r.moduleDirectories[module.Directory]) == 0 {
			delete(r.moduleDirectories, module.Directory)
		}
	}

	// Remove module entry
	delete(r.modules, moduleName)
}

// UnloadDirectory unloads all modules in a directory and cleans up HotLoader state.
func (r *LuaSession) UnloadDirectory(dirPath string) {
	modules, exists := r.moduleDirectories[dirPath]
	if !exists {
		return
	}

	// Make a copy of module names to avoid modifying slice while iterating
	moduleNames := make([]string, len(modules))
	for i, m := range modules {
		moduleNames[i] = m.Name
	}

	// Unload each module
	for _, name := range moduleNames {
		r.UnloadModule(name)
	}

	// Clean up HotLoader directory state
	if r.hotLoaderCleanup != nil {
		r.hotLoaderCleanup(dirPath)
	}

	// Remove directory entry (should already be gone, but be safe)
	delete(r.moduleDirectories, dirPath)
}

// SetCurrentModule sets the module being loaded for resource tracking.
func (r *LuaSession) SetCurrentModule(trackingKey, directory string) {
	module := NewModule(trackingKey, directory)
	r.modules[trackingKey] = module
	r.moduleDirectories[directory] = append(r.moduleDirectories[directory], module)
	r.currentModule = module
}

// ClearCurrentModule clears the current module after load completes.
func (r *LuaSession) ClearCurrentModule() {
	r.currentModule = nil
}

// GetCurrentModule returns the module currently being loaded, if any.
func (r *LuaSession) GetCurrentModule() *Module {
	return r.currentModule
}

// extractTypeProperty extracts type from metatable or direct field (frictionless convention).
func (r *LuaSession) extractTypeProperty(obj any, props map[string]string) {
	if props["type"] != "" {
		return
	} else if typ := GetType(r.State, obj); typ != "" {
		props["type"] = typ
	}
}

// NotifyPropertyChange notifies Lua watchers of a property change for a session.
// Called by external code when a variable property changes.
// vendedID is the compact session ID (e.g., "1", "2").
func (r *LuaSession) NotifyPropertyChange(vendedID string, varID int64, property string, value interface{}) {
	if r.ID != vendedID || r.sessionTable == nil {
		return
	}

	r.execute(func() (interface{}, error) {
		r.notifyPropertyChangeInternal(varID, property, value)
		return nil, nil
	})
}

// notifyPropertyChangeInternal notifies watchers (must be called from executor).
func (r *LuaSession) notifyPropertyChangeInternal(varID int64, property string, value interface{}) {
	watchers := r.State.GetField(r.sessionTable, "_watchers").(*lua.LTable)
	key := fmt.Sprintf("%d", varID)

	varWatchers := r.State.GetField(watchers, key)
	if varWatchers == lua.LNil {
		return
	}

	luaValue := r.GoToLua(value)

	// Call property-specific watchers
	propWatchers := r.State.GetField(varWatchers.(*lua.LTable), property)
	if propWatchers != lua.LNil {
		r.callWatchers(propWatchers.(*lua.LTable), luaValue)
	}

	// Call wildcard watchers
	wildcardWatchers := r.State.GetField(varWatchers.(*lua.LTable), "*")
	if wildcardWatchers != lua.LNil {
		r.callWatchers(wildcardWatchers.(*lua.LTable), luaValue, lua.LString(property))
	}
}

// callWatchers calls all watcher callbacks in a table.
func (r *LuaSession) callWatchers(watchers *lua.LTable, args ...lua.LValue) {
	watchers.ForEach(func(_, cb lua.LValue) {
		if fn, ok := cb.(*lua.LFunction); ok {
			r.State.Push(fn)
			for _, arg := range args {
				r.State.Push(arg)
			}
			if err := r.State.PCall(len(args), 0, nil); err != nil {
				r.Log(0, "Watcher callback error: %v", err)
			}
		}
	})
}

// startExecutor creates the goroutine that processes work items.
func (r *LuaSession) startExecutor() {
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
func (r *LuaSession) execute(fn func() (interface{}, error)) (interface{}, error) {
	result := make(chan WorkResult, 1)
	r.executorChan <- WorkItem{fn: fn, result: result}
	res := <-result
	return res.Value, res.Err
}

// ExecuteInSession executes a function within the context of this session.
// It sets the global 'session' variable to this session's table before execution.
// Spec: mcp.md
// CRC: crc-LuaRuntime.md
// Sequence: seq-mcp-run.md
func (r *LuaSession) ExecuteInSession(sessionID string, fn func() (interface{}, error)) (interface{}, error) {
	if r.ID != sessionID {
		return nil, fmt.Errorf("session %s not found (this session is %s)", sessionID, r.ID)
	}

	return r.execute(func() (interface{}, error) {
		L := r.State

		// Set the global 'session' variable
		L.SetGlobal("session", r.sessionTable)

		// Execute the function
		return fn()
	})
}

// RedirectOutput redirects Lua's print function and standard streams to log files.
// It is used by the MCP server in Configured state.
// Spec: mcp.md
// CRC: crc-LuaRuntime.md
func (r *LuaSession) RedirectOutput(logPath, errPath string) error {
	// Create/Open log files
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	errFile, err := os.OpenFile(errPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("failed to open error log file: %v", err)
	}
	// Keep file open for the process duration
	_ = errFile

	// We don't close these files here; they remain open for the process lifetime
	// or until reconfigured. In a robust system we might want to manage them better.

	r.execute(func() (interface{}, error) {
		L := r.State

		// Override print
		L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
			top := L.GetTop()
			for i := 1; i <= top; i++ {
				str := L.ToStringMeta(L.Get(i)).String()
				if i > 1 {
					logFile.WriteString("\t")
				}
				logFile.WriteString(str)
			}
			logFile.WriteString("\n")
			// Sync/Flush to ensure output is visible immediately (e.g. to tail -f)
			logFile.Sync()
			return 0
		}))

		// Redirect io.stdout / io.stderr (if your Lua environment uses the io library)
		// Note: gopher-lua's io library writes to os.Stdout/Stderr by default.
		// Changing that requires hacking the library or standard library options,
		// which is complex. For now, replacing 'print' covers most user code.
		// System-level fmt.Println from Go code will still go to process Stdout/Stderr.
		// We've already redirected global Go logging to Stderr in main.go.

		return nil, nil
	})

	return nil
}

// VariableUpdate represents a detected change to be sent to the frontend.
type VariableUpdate struct {
	VarID      int64
	Value      json.RawMessage
	Properties map[string]string
}

func (r *LuaSession) TriggerBatch() {
	r.batchTriggered = true
}

// AfterBatch triggers change detection for a session after processing a message batch.
// Returns a list of variable updates that need to be sent to the frontend.
// vendedID is the compact session ID (e.g., "1", "2").
func (r *LuaSession) AfterBatch(vendedID string) []VariableUpdate {
	// Use tracker's DetectChanges
	for range 4 {
		if !r.variableStore.DetectChanges(vendedID) || !r.batchTriggered {
			break
		}
		r.batchTriggered = false
	}
	changes := r.variableStore.GetChanges(vendedID)

	// Check for viewdef changes even if no variable changes (e.g., hot-reload)
	// NOTE: GetChangedViewdefsForSession marks viewdefs as sent, so only call once
	defs := r.viewdefManager.GetChangedViewdefsForSession(vendedID)
	if len(changes) == 0 && len(defs) == 0 {
		return nil
	}

	tracker := r.variableStore.GetTracker(vendedID)
	if tracker == nil {
		return nil
	}

	v1 := tracker.GetVariable(1)
	if v1 == nil {
		return nil
	}
	var sending changetracker.Change

	// Load viewdefs for any new types encountered
	for _, change := range changes {
		if slices.Contains(change.PropertiesChanged, "type") {
			if v := tracker.GetVariable(change.VariableID); v != nil {
				typ := v.Properties["type"]
				r.viewdefManager.LoadViewdefsForType(typ)
			}
		}
		if change.VariableID == 1 && slices.Contains(change.PropertiesChanged, "viewdefs") {
			sending = change
		}
	}

	// Handle viewdef changes (defs already loaded above)
	if len(defs) > 0 {
		if defBytes, err := json.Marshal(defs); err != nil {
			r.Log(0, "Error serializing viewdefs: %s", err.Error())
		} else {
			v1.Properties["viewdefs"] = string(defBytes)
			r.Log(4, "SENDING VIEWDEFS: %s", v1.Properties["viewdefs"])
			if sending.VariableID == 0 {
				// need to insert a change for the viewdefs
				new := make([]changetracker.Change, len(changes)+1)
				new[0] = changetracker.Change{
					VariableID:        1,
					Priority:          changetracker.PriorityHigh,
					ValueChanged:      false,
					PropertiesChanged: []string{"viewdefs"},
				}
				copy(new[1:], changes)
				changes = new
			}
		}
	}

	var updates []VariableUpdate
	for _, change := range changes {
		v := tracker.GetVariable(change.VariableID)
		if v == nil {
			continue
		}
		var value json.RawMessage
		var props map[string]string
		if change.ValueChanged {
			// Use wrapped value if present
			val := v.NavigationValue()
			jsonBytes, err := tracker.ToValueJSONBytes(val)
			if err != nil {
				r.Log(1, "ERROR: AfterBatch failed to marshal variable %d: %v", change.VariableID, err)
				continue
			}
			value = json.RawMessage(jsonBytes)
		}
		if len(change.PropertiesChanged) > 0 {
			props = make(map[string]string, len(change.PropertiesChanged))
			for _, prop := range change.PropertiesChanged {
				props[prop] = v.Properties[prop]
			}
			if props["viewdefs"] != "" {
				r.Log(4, "ADDING VIEWDEFS TO UPDATES: %s", v1.Properties["viewdefs"])
			}
		}
		r.Log(2, "AfterBatch: variable %d changed", change.VariableID)
		updates = append(updates, VariableUpdate{
			VarID:      change.VariableID,
			Value:      value,
			Properties: props,
		})

		// Also update the variable store so watchers get notified
		if err := r.variableStore.Update(change.VariableID, value, props); err != nil {
			r.Log(1, "AfterBatch: failed to update store for variable %d: %v", change.VariableID, err)
		}
	}
	// clear sent viewdefs
	v1.Properties["viewdefs"] = ""
	return updates
}

// HandleFrontendCreate handles a variable create message from the frontend.
// For path-based variables, it creates the variable in the tracker and resolves the path.
// If a wrapper property is set, the tracker automatically creates it via the resolver.
// Spec: protocol.md - create(id, parentId, value, properties, nowatch?, unbound?)
// The id is provided by the frontend (frontend-vended IDs).
// Returns the resolved value (wrapped if applicable) and properties.
func (r *LuaSession) HandleFrontendCreate(sessionID string, id int64, parentID int64, properties map[string]string) error {
	path := properties["path"]
	if path == "" {
		return fmt.Errorf("HandleFrontendCreate: path property required")
	}

	if r.ID != sessionID {
		return fmt.Errorf("session %s not found (this session is %s)", sessionID, r.ID)
	}

	tracker := r.variableStore.GetTracker(r.ID)
	if tracker == nil {
		return fmt.Errorf("session %s tracker not found", r.ID)
	}
	// Create the child variable in the tracker with the frontend-provided ID.
	// This automatically triggers Resolver.CreateWrapper if the property is set.
	v := tracker.CreateVariableWithId(id, nil, parentID, path, properties)
	if v == nil {
		return fmt.Errorf("HandleFrontendCreate: variable ID %d already in use", id)
	}

	// Nil out cached JSON so that when the auto-watch triggers ChangeAll,
	// DetectChanges will see the value as changed (from nil to the actual value).
	// Without this, the value would already match the cached JSON and no update would be sent.
	v.ValueJSON = nil
	v.WrapperJSON = nil
	return nil
}

// TrackerVariableAdapter adapts a change-tracker Variable to WrapperVariable interface
type TrackerVariableAdapter struct {
	*changetracker.Variable
	Session *LuaSession
}

func WrapTrackerVariable(session *LuaSession, v *changetracker.Variable) *TrackerVariableAdapter {
	return &TrackerVariableAdapter{
		Variable: v,
		Session:  session,
	}
}

// HandleFrontendUpdate handles an update to a path-based variable from frontend.
// Updates the backend object via the variable's path using v.Set().
// CRC: crc-LuaRuntime.md
// Sequence: seq-relay-message.md
func (r *LuaSession) HandleFrontendUpdate(sessionID string, varID int64, value json.RawMessage, properties map[string]string) error {
	tracker := r.variableStore.GetTracker(sessionID)
	if tracker == nil {
		return fmt.Errorf("session %s tracker not found", sessionID)
	}

	v := tracker.GetVariable(varID)
	if v == nil {
		return fmt.Errorf("variable %d not found in tracker", varID)
	}

	// Apply frontend-sent properties to tracker variable
	for k, val := range properties {
		v.SetProperty(k, val)
	}

	// Skip value update if no value sent (properties-only update)
	if len(value) == 0 {
		return nil
	}

	// Parse the JSON value to a Go value
	var goValue interface{}
	if err := json.Unmarshal(value, &goValue); err != nil {
		return fmt.Errorf("failed to parse value: %w", err)
	}

	// Update the backend object via the variable's path
	if err := v.Set(goValue); err != nil {
		r.Log(0, "HandleFrontendUpdate: Set failed for var %d: %v", varID, err)
		return err
	}

	r.Log(2, "HandleFrontendUpdate: updated var %d with value %s", varID, string(value))

	return nil
}

// registerRequire adds a custom require() function that works with both
// filesystem (--dir mode) and embedded bundle.
// Uses unified loadedModules table shared with hot-loader.
// Handles circular dependencies by marking as loaded before executing.
// Stores under both module name (for require lookups) and tracking key (for hot-reload).
func (r *LuaSession) registerRequire() {
	L := r.State
	loaded := r.loadedModules // Use unified load tracker

	requireFn := L.NewFunction(func(L *lua.LState) int {
		modName := L.CheckString(1)

		// Check if already loaded by module name (handles circularity)
		if cached := L.GetField(loaded, modName); cached != lua.LNil {
			L.Push(cached)
			return 1
		}

		// Convert module name to filename (e.g., "foo.bar" -> "foo/bar.lua")
		filename := strings.ReplaceAll(modName, ".", string(filepath.Separator)) + ".lua"

		// Mark as loaded BEFORE executing (handles circular dependencies)
		L.SetField(loaded, modName, lua.LTrue)

		// Delegate to DirectRequireLuaFile for actual loading
		result, err := r.DirectRequireLuaFile(filename)
		if err != nil {
			// Unmark on error (allows retry)
			L.SetField(loaded, modName, lua.LNil)
			L.RaiseError("error loading module '%s': %v", modName, err)
			return 0
		}

		// Also cache under module name for require("foo.bar") lookups
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
func (r *LuaSession) registerUIModule() {
	L := r.State

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
		// Track presenter type in current module
		if r.currentModule != nil {
			r.currentModule.AddPresenterType(name)
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
		goVal := LuaToGo(val)
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
		L.Push(r.GoToLua(val))
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
		r.wrapperRegistry.Register(name, func(session *LuaSession, variable *TrackerVariableAdapter) interface{} {
			return NewLuaWrapper(session, tbl, variable)
		})

		// Track wrapper in current module
		if r.currentModule != nil {
			r.currentModule.AddWrapper(name)
		}

		r.Log(2, "LuaRuntime: registered wrapper type %s", name)

		return 0
	}))

	L.SetGlobal("ui", uiMod)
}

// LoadFile loads and executes a Lua file via executor (relative to luaDir).
// Deprecated: Use RequireLuaFile for hot-reload compatible loading.
func (r *LuaSession) LoadFile(filename string) error {
	path := filepath.Join(r.luaDir, filename)
	return r.LoadFileAbsolute(path)
}

// LoadFileAbsolute loads and executes a Lua file via executor (absolute path).
// Deprecated: Use RequireLuaFile for hot-reload compatible loading.
func (r *LuaSession) LoadFileAbsolute(path string) error {
	_, err := r.execute(func() (interface{}, error) {
		return nil, r.loadFileInternal(path)
	})
	return err
}

// loadFileInternal is the non-executor version used internally.
// Uses unified load tracker with circularity handling.
func (r *LuaSession) loadFileInternal(path string) error {
	L := r.State
	loaded := r.loadedModules

	// Check if already loaded
	if L.GetField(loaded, path) != lua.LNil {
		return nil // Already loaded
	}

	// Mark as loaded BEFORE executing (handles circular dependencies)
	L.SetField(loaded, path, lua.LTrue)

	// Execute the file
	if err := L.DoFile(path); err != nil {
		// Unmark on error (allows retry)
		L.SetField(loaded, path, lua.LNil)
		return fmt.Errorf("failed to load %s: %w", path, err)
	}

	return nil
}

// IsFileLoaded checks if a file has been loaded by this session.
// The trackingKey should be a baseDir-relative path (e.g., "apps/myapp/app.lua").
// Used by hot-loader to skip files not yet loaded.
// CRC: crc-LuaSession.md
func (r *LuaSession) IsFileLoaded(trackingKey string) bool {
	var loaded bool
	r.execute(func() (interface{}, error) {
		loaded = r.State.GetField(r.loadedModules, trackingKey) != lua.LNil
		return nil, nil
	})
	return loaded
}

// BaseDir returns the site root directory from config.
func (r *LuaSession) BaseDir() string {
	return r.config.Server.Dir
}

// RequireLuaFile loads a Lua file using the unified load tracker.
// Skips if already loaded (like require()). Returns error if file not found or execution fails.
// This is the preferred method for hot-reload compatible file loading.
func (r *LuaSession) RequireLuaFile(filename string) error {
	_, err := r.execute(func() (any, error) {
		return r.DirectRequireLuaFile(filename)
	})
	return err
}

// DirectRequireLuaFile loads a Lua file and tracks it by baseDir-relative path.
// The filename can be relative to luaDir (e.g., "mcp.lua") or relative to baseDir
// (e.g., "apps/myapp/init.lua"). Symlinks are resolved to compute the tracking key.
// CRC: crc-LuaSession.md
func (r *LuaSession) DirectRequireLuaFile(filename string) (lua.LValue, error) {
	L := r.State
	loaded := r.loadedModules

	// Determine the absolute file path
	var absPath string
	if filepath.IsAbs(filename) {
		absPath = filename
	} else {
		// Try relative to luaDir first (backward compatible)
		absPath = filepath.Join(r.luaDir, filename)
		if _, err := os.Stat(absPath); err != nil {
			// Try relative to baseDir
			absPath = filepath.Join(r.config.Server.Dir, filename)
		}
	}

	var code string
	var trackingKey string

	// Try filesystem first
	content, fsErr := os.ReadFile(absPath)
	if fsErr == nil {
		code = string(content)
		// Compute tracking key for hot-reload (baseDir-relative path)
		if r.config != nil && r.config.Server.Dir != "" {
			if key, err := ComputeTrackingKey(r.config.Server.Dir, absPath); err == nil {
				trackingKey = key
			}
		}
	} else {
		// Try bundle (works for bundled binaries)
		bundlePath := "lua/" + strings.ReplaceAll(filename, string(filepath.Separator), "/")
		bundleContent, bundleErr := bundle.ReadFile(bundlePath)
		if bundleErr == nil {
			code = string(bundleContent)
			trackingKey = filename // Use filename as key for bundle
		} else {
			return lua.LNil, fmt.Errorf("file not found: %s (also tried bundle: %s)", absPath, bundlePath)
		}
	}

	// Check if already loaded by tracking key
	if cached := L.GetField(loaded, trackingKey); cached != lua.LNil {
		return cached, nil // Already loaded
	}

	// Mark as loaded BEFORE executing (handles circular dependencies)
	L.SetField(loaded, trackingKey, lua.LTrue)

	// Set current module for resource tracking
	directory := filepath.Dir(trackingKey)
	r.SetCurrentModule(trackingKey, directory)
	defer r.ClearCurrentModule()

	// Execute the code
	if err := L.DoString(code); err != nil {
		// Unmark on error (allows retry)
		L.SetField(loaded, trackingKey, lua.LNil)
		// Clean up module tracking on error
		delete(r.modules, trackingKey)
		if mods, ok := r.moduleDirectories[directory]; ok {
			for i, m := range mods {
				if m.Name == trackingKey {
					r.moduleDirectories[directory] = slices.Delete(mods, i, i+1)
					break
				}
			}
		}
		return lua.LNil, fmt.Errorf("failed to load %s: %w", filename, err)
	}

	// Get the return value (module table) or use true marker
	result := L.Get(-1)
	if result == lua.LNil {
		result = lua.LTrue
	}

	// Update cache with actual result
	L.SetField(loaded, trackingKey, result)

	return result, nil
}

// ComputeTrackingKey computes a baseDir-relative tracking key for a file.
// Symlinks are resolved to get the actual target path.
// This is a package-level function used by both LuaSession and HotLoader.
// CRC: crc-LuaSession.md
func ComputeTrackingKey(baseDir, absPath string) (string, error) {
	// Resolve symlinks to get the actual target path
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		resolved = absPath
	}

	relPath, err := filepath.Rel(baseDir, resolved)
	if err != nil {
		return "", fmt.Errorf("path %s is not under baseDir %s", resolved, baseDir)
	}

	return filepath.Clean(relPath), nil
}

// SetReloading sets the session.reloading flag.
// Called by hot-loader before/after reloading files.
func (r *LuaSession) SetReloading(reloading bool) {
	r.execute(func() (interface{}, error) {
		if r.sessionTable != nil {
			if reloading {
				r.State.SetField(r.sessionTable, "reloading", lua.LTrue)
			} else {
				r.State.SetField(r.sessionTable, "reloading", lua.LFalse)
			}
		}
		return nil, nil
	})
}

// LoadCode loads and executes Lua code string via executor.
// It returns the result of the execution (if any).
// After execution, processes any queued prototype mutations.
// Spec: mcp.md, libraries.md
// CRC: crc-LuaRuntime.md, crc-LuaSession.md
func (r *LuaSession) LoadCode(name, code string) (interface{}, error) {
	return r.execute(func() (interface{}, error) {
		// Load the string into a function
		fn, err := r.State.LoadString(code)
		if err != nil {
			return nil, fmt.Errorf("failed to load code %s: %w", name, err)
		}

		// Push function
		r.State.Push(fn)

		// Call it (0 arguments, 1 result)
		if err := r.State.PCall(0, 1, nil); err != nil {
			return nil, fmt.Errorf("failed to execute code %s: %w", name, err)
		}

		// Get result
		ret := r.State.Get(-1) // Get top
		r.State.Pop(1)         // Pop it

		// Process mutation queue after code execution
		r.processMutationQueueDirect()

		if ret == lua.LNil {
			return nil, nil
		}

		// Convert to Go
		return LuaToGo(ret), nil
	})
}

// LoadCodeDirect executes Lua code without executor wrapping.
// Use this when already inside ExecuteInSession to avoid deadlock.
// After execution, processes any queued prototype mutations.
// MUST only be called from within an execute() context.
func (r *LuaSession) LoadCodeDirect(name, code string) (interface{}, error) {
	// Load the string into a function
	fn, err := r.State.LoadString(code)
	if err != nil {
		return nil, fmt.Errorf("failed to load code %s: %w", name, err)
	}

	// Push function
	r.State.Push(fn)

	// Call it (0 arguments, 1 result)
	if err := r.State.PCall(0, 1, nil); err != nil {
		return nil, fmt.Errorf("failed to execute code %s: %w", name, err)
	}

	// Get result
	ret := r.State.Get(-1) // Get top
	r.State.Pop(1)         // Pop it

	// Process mutation queue after code execution
	r.processMutationQueueDirect()

	if ret == lua.LNil {
		return nil, nil
	}

	// Convert to Go
	return LuaToGo(ret), nil
}

// GetPresenterType returns a registered presenter type.
func (r *LuaSession) GetPresenterType(name string) (*PresenterType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pt, ok := r.presenterTypes[name]
	return pt, ok
}

// ListPresenterTypes returns all registered presenter type names.
func (r *LuaSession) ListPresenterTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.presenterTypes))
	for name := range r.presenterTypes {
		names = append(names, name)
	}
	return names
}

// CallMethod invokes a method on a Lua presenter instance via executor.
func (r *LuaSession) CallMethod(instance *lua.LTable, method string, args ...interface{}) (interface{}, error) {
	return r.execute(func() (interface{}, error) {
		L := r.State

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
			L.Push(r.GoToLua(arg))
		}

		// Call method (self + args)
		if err := L.PCall(len(args)+1, 1, nil); err != nil {
			return nil, err
		}

		// Get result
		result := L.Get(-1)
		L.Pop(1)

		return LuaToGo(result), nil
	})
}

// CallLuaWrapperMethod invokes a method on a Lua wrapper table via executor.
// Used by LuaWrapper to call computeValue and destroy methods.
// The instance can be any interface{} but must be a *lua.LTable at runtime.
func (r *LuaSession) CallLuaWrapperMethod(instance interface{}, method string, args ...interface{}) (interface{}, error) {
	tbl, ok := instance.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("wrapper instance is not a Lua table")
	}

	return r.execute(func() (interface{}, error) {
		L := r.State

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
					L.Push(r.GoToLua(val))
				}
			} else {
				L.Push(r.GoToLua(arg))
			}
		}

		// Call method (self + args)
		if err := L.PCall(len(args)+1, 1, nil); err != nil {
			return nil, err
		}

		// Get result
		result := L.Get(-1)
		L.Pop(1)

		return LuaToGo(result), nil
	})
}

// CreateInstance creates a new instance of a presenter type via executor.
func (r *LuaSession) CreateInstance(typeName string, props map[string]interface{}) (*lua.LTable, error) {
	result, err := r.execute(func() (interface{}, error) {
		r.mu.RLock()
		pt, ok := r.presenterTypes[typeName]
		r.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("presenter type %s not found", typeName)
		}

		L := r.State

		// Create new instance table
		instance := L.NewTable()

		// Set metatable to inherit from presenter type
		mt := L.NewTable()
		L.SetField(mt, "__index", pt.Table)
		L.SetMetatable(instance, mt)

		// Set initial properties
		for k, v := range props {
			L.SetField(instance, k, r.GoToLua(v))
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
func (r *LuaSession) CreateItemWrapper(typeName string, viewItem *ViewListItem) (*ItemWrapperInstance, error) {
	if typeName == "" {
		return nil, nil
	}

	result, err := r.execute(func() (interface{}, error) {
		r.mu.RLock()
		pt, ok := r.presenterTypes[typeName]
		r.mu.RUnlock()

		if !ok {
			// Auto-discovery: Check if there's a global table with this name
			L := r.State
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

		L := r.State

		// Create new instance table
		instance := L.NewTable()

		// Set metatable to inherit from presenter type
		mt := L.NewTable()
		L.SetField(mt, "__index", pt.Table)
		L.SetMetatable(instance, mt)

		// Set ViewListItem properties on the instance
		// The presenter can access: viewListItem.item, viewListItem.list, viewListItem.index
		L.SetField(instance, "viewListItem", r.createViewListItemLuaWrapper(viewItem))

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

func (s *LuaSession) createLuaViewListItem(viewItem *ViewListItem) *lua.LTable {
	return s.createViewListItemLuaWrapper(viewItem)
}

// createViewListItemLuaWrapper creates a Lua wrapper for a ViewListItem.
func (r *LuaSession) createViewListItemLuaWrapper(viewItem *ViewListItem) *lua.LTable {
	wrapper := r.State.NewTable()

	// viewListItem.item - the domain object
	r.State.SetField(wrapper, "item", r.GoToLua(viewItem.GetItem()))

	// viewListItem.item - the domain object
	r.State.SetField(wrapper, "baseItem", r.GoToLua(viewItem.GetBaseItem()))

	// viewListItem.index - position in list
	r.State.SetField(wrapper, "index", lua.LNumber(viewItem.GetIndex()))

	return wrapper
}

// GetValue gets a value from a Lua table via executor.
func (r *LuaSession) GetValue(tbl *lua.LTable, key string) interface{} {
	result, _ := r.execute(func() (interface{}, error) {
		val := r.State.GetField(tbl, key)
		return LuaToGo(val), nil
	})
	return result
}

// SetValue sets a value on a Lua table via executor.
func (r *LuaSession) SetValue(tbl *lua.LTable, key string, value interface{}) {
	r.execute(func() (interface{}, error) {
		r.State.SetField(tbl, key, r.GoToLua(value))
		return nil, nil
	})
}

// Shutdown cleans up the Lua VM and stops executor.
func (r *LuaSession) Shutdown() {
	close(r.done)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.State.Close()
}

// GoToLua converts a Go value to Lua.
func (r *LuaSession) GoToLua(val any) lua.LValue {
	if val == nil {
		return lua.LNil
	}

	_, isTable := val.(*lua.LTable)
	switch v := val.(type) {
	case lua.LBool, lua.LNumber, lua.LString, *lua.LTable, *lua.LNilType:
		return val.(lua.LValue)
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
	case *ViewListItem:
		return r.createViewListItemLuaWrapper(v)
	case []any:
		tbl := r.State.NewTable()
		for i, item := range v {
			r.State.RawSetInt(tbl, i+1, r.GoToLua(item))
		}
		return tbl
	case map[string]interface{}:
		tbl := r.State.NewTable()
		for k, item := range v {
			r.State.SetField(tbl, k, r.GoToLua(item))
		}
		return tbl
	default:
		if isTable {
			panic("TYPE SWITCH")
		}
		r.Log(4, "VALUE %#v TYPE: %v", val, reflect.ValueOf(val).Type())
		return lua.LString(fmt.Sprintf("%v", v))
	}
}

// isArray checks if a Lua table is an array (sequential integer keys starting at 1).
// Returns true if the table has only numeric keys with no string keys (excluding _ prefixed).
func (s *LuaSession) isArray(tbl *lua.LTable) bool {
	hasNumericKeys := false
	hasStringKeys := false

	if GetType(s.State, tbl) != "" {
		return false
	}
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

func (s *LuaSession) GetTracker() *changetracker.Tracker {
	return s.variableStore.GetTracker(s.ID)
}

// GetAppVariableID returns the app variable ID (variable 1) for this session.
func (s *LuaSession) GetAppVariableID() int64 {
	return s.appVariableID
}

func (s *LuaSession) Set(varID int64, value any) error {
	return s.GetTracker().GetVariable(varID).Set(value)
}

func (s *LuaSession) ArrayGet(array any, index int) (any, error) {
	if f, _, err := s.ArrayGetter(array); err != nil {
		return nil, err
	} else {
		return f(index)
	}
}

func (s *LuaSession) ArrayGetter(array any) (func(int) (any, error), int, error) {
	if la, ok := (array).(*lua.LTable); ok {
		if !s.isArray(la) {
			return nil, 0, fmt.Errorf("Attempt to index Lua table that is not an array")
		}
		return func(index int) (any, error) {
			index += 1
			if index < 1 || index > la.Len() {
				return nil, fmt.Errorf("Bad index %d for Lua table of length %d", index-1, la.Len())
			}
			return la.RawGetInt(index), nil
		}, la.Len(), nil
	}
	v := reflect.ValueOf(array)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, 0, fmt.Errorf("Attempt to index object that is not a slice: %#v", array)
	}
	return func(index int) (any, error) {
		if index < 0 || v.Len() <= index {
			return nil, fmt.Errorf("Bad index %d for slice of length %d", index-1, v.Len())
		}
		return v.Index(index).Interface(), nil
	}, v.Len(), nil
}

// LuaToGo converts a Lua value to Go.
// Fields prefixed with "_" are skipped (internal/private fields).
func LuaToGo(val lua.LValue) interface{} {
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
				arr[i-1] = LuaToGo(v.RawGetInt(i))
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
					m[keyStr] = LuaToGo(value)
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
