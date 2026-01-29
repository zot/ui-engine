package lua

import (
	"encoding/json"
	"fmt"
	"testing"

	golua "github.com/yuin/gopher-lua"
	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui-engine/internal/config"
)

// mockVariableStore implements VariableStore for testing
type mockVariableStore struct {
	variables map[int64]*mockVariable
	nextID    int64
	trackers  map[string]*changetracker.Tracker
}

type mockVariable struct {
	id         int64
	parentID   int64
	value      json.RawMessage
	properties map[string]string
}

func newMockStore() *mockVariableStore {
	return &mockVariableStore{
		variables: make(map[int64]*mockVariable),
		nextID:    1,
		trackers:  make(map[string]*changetracker.Tracker),
	}
}

func (s *mockVariableStore) Create(parentID int64, value json.RawMessage, properties map[string]string) (int64, error) {
	id := s.nextID
	s.nextID++
	s.variables[id] = &mockVariable{
		id:         id,
		parentID:   parentID,
		value:      value,
		properties: properties,
	}
	return id, nil
}

func (s *mockVariableStore) Get(id int64) (json.RawMessage, map[string]string, bool) {
	v, ok := s.variables[id]
	if !ok {
		return nil, nil, false
	}
	return v.value, v.properties, true
}

func (s *mockVariableStore) GetProperty(id int64, name string) (string, bool) {
	v, ok := s.variables[id]
	if !ok {
		return "", false
	}
	val, exists := v.properties[name]
	return val, exists
}

func (s *mockVariableStore) Update(id int64, value json.RawMessage, properties map[string]string) error {
	v, ok := s.variables[id]
	if !ok {
		return nil
	}
	if value != nil {
		v.value = value
	}
	if properties != nil {
		for k, val := range properties {
			v.properties[k] = val
		}
	}
	return nil
}

func (s *mockVariableStore) Destroy(id int64) error {
	delete(s.variables, id)
	return nil
}

// CreateSession creates a tracker for a session
func (s *mockVariableStore) CreateSession(sessionID string, resolver changetracker.Resolver) {
	tracker := changetracker.NewTracker()
	tracker.Resolver = resolver
	s.trackers[sessionID] = tracker
}

// DestroySession removes a session's tracker
func (s *mockVariableStore) DestroySession(sessionID string) {
	delete(s.trackers, sessionID)
}

// GetTracker returns the tracker for a session
func (s *mockVariableStore) GetTracker(sessionID string) *changetracker.Tracker {
	return s.trackers[sessionID]
}

// CreateVariable creates a variable using the tracker
func (s *mockVariableStore) CreateVariable(sessionID string, parentID int64, luaObject *golua.LTable, properties map[string]string) (int64, error) {
	tracker := s.trackers[sessionID]
	if tracker == nil {
		// Create tracker if not exists
		tracker = changetracker.NewTracker()
		s.trackers[sessionID] = tracker
	}

	// Create variable in tracker
	v := tracker.CreateVariable(luaObject, parentID, "", properties)
	id := v.ID

	// Also store in mock
	jsonValue, _ := tracker.ToValueJSONBytes(luaObject)
	s.variables[id] = &mockVariable{
		id:         id,
		parentID:   parentID,
		value:      jsonValue,
		properties: properties,
	}

	return id, nil
}

// DetectChanges returns whether there are changes for a session
func (s *mockVariableStore) DetectChanges(sessionID string) bool {
	tracker := s.trackers[sessionID]
	if tracker == nil {
		return false
	}
	return tracker.DetectChanges()
}

// GetChanges returns the detected changes for a session
func (s *mockVariableStore) GetChanges(sessionID string) []changetracker.Change {
	tracker := s.trackers[sessionID]
	if tracker == nil {
		return nil
	}
	return tracker.GetChanges()
}

// TestObjectRegistration verifies that Lua tables are registered as objects
// and their Value JSON is an object reference.
func TestObjectRegistration(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Create app variable
	rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			local App = {type = "TestApp"}
			App.__index = App
			function App:new(tbl)
				tbl = tbl or {}
				setmetatable(tbl, self)
				tbl.title = tbl.title or "Test App"
				tbl.count = tbl.count or 0
				return tbl
			end

			app = App:new({title = "My App", count = 42})
			session:createAppVariable(app)
		`
		return nil, L.DoString(code)
	})

	// Verify app variable was created
	if sess.appVariableID != 1 {
		t.Fatalf("Expected app variable ID 1, got %d", sess.appVariableID)
	}

	// Check that ValueJSON is an object reference
	tracker := store.GetTracker("1")
	if tracker == nil {
		t.Fatal("Expected tracker for session 1")
	}

	v := tracker.GetVariable(1)
	if v == nil {
		t.Fatal("Expected variable 1 in tracker")
	}

	// ValueJSON should be ObjectRef (object ID is assigned after variable ID)
	objRef, ok := v.ValueJSON.(changetracker.ObjectRef)
	if !ok {
		t.Fatalf("Expected ValueJSON to be ObjectRef, got %T: %v", v.ValueJSON, v.ValueJSON)
	}
	// Note: Object ID is 2 because variable ID (1) is assigned first, then object registration
	// uses the next available ID (2) when ToValueJSON is called
	if objRef.Obj != 2 {
		t.Errorf("Expected ObjectRef.Obj=2 (assigned after variable ID), got %d", objRef.Obj)
	}

	// Serialized form should be {"obj": 2}
	jsonBytes, err := tracker.ToValueJSONBytes(v.Value)
	if err != nil {
		t.Fatalf("ToValueJSONBytes error: %v", err)
	}
	expected := `{"obj":2}`
	if string(jsonBytes) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(jsonBytes))
	}

	t.Log("Object registration test passed!")
}

// TestCreateMultipleVariables verifies creating app and child variables
func TestCreateMultipleVariables(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Create app and child variable
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			local App = {type = "TestApp"}
			App.__index = App
			function App:new(tbl)
				tbl = tbl or {}
				setmetatable(tbl, self)
				return tbl
			end

			local Item = {type = "TestItem"}
			Item.__index = Item
			function Item:new(tbl)
				tbl = tbl or {}
				setmetatable(tbl, self)
				tbl.name = tbl.name or ""
				return tbl
			end

			app = App:new()
			appVarId = session:createAppVariable(app)

			-- Create a second tracked object (as root variable)
			-- Note: change-tracker requires child variables to use paths, not direct values
			item = Item:new({name = "Test Item"})
			itemId = session:createVariable(0, item)  -- 0 = root variable

			-- Verify item ID (3, because: app var=1, app obj=2, item var=3)
			assert(itemId == 3, "Expected item ID 3, got " .. tostring(itemId))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}

	// Check both variables exist in tracker
	tracker := store.GetTracker("1")
	if tracker == nil {
		t.Fatal("Expected tracker for session 1")
	}

	// Debug: list all variables in tracker
	allVars := tracker.Variables()
	t.Logf("Tracker has %d variables:", len(allVars))
	for _, v := range allVars {
		t.Logf("  Variable ID=%d ParentID=%d", v.ID, v.ParentID)
	}

	v1 := tracker.GetVariable(1)
	if v1 == nil {
		t.Error("Expected variable 1 (app) in tracker")
	}

	// Item variable - should be a root variable (ID=3 after app var=1, app obj=2)
	itemVar := tracker.GetVariable(3)
	if itemVar == nil {
		t.Error("Expected variable 3 (item) in tracker")
	} else {
		t.Logf("Found item variable with ID=%d, ParentID=%d", itemVar.ID, itemVar.ParentID)
	}

	// Both should have object refs as ValueJSON
	if _, ok := v1.ValueJSON.(changetracker.ObjectRef); !ok {
		t.Errorf("Variable 1 ValueJSON should be ObjectRef, got %T", v1.ValueJSON)
	}
	if itemVar != nil {
		if _, ok := itemVar.ValueJSON.(changetracker.ObjectRef); !ok {
			t.Errorf("Item variable ValueJSON should be ObjectRef, got %T", itemVar.ValueJSON)
		}
	}

	t.Log("Create multiple variables test passed!")
}

// TestLuaResolverArrayConversion verifies that Lua arrays convert properly
// with object elements becoming object refs
func TestLuaResolverArrayConversion(t *testing.T) {
	L := golua.NewState()
	defer L.Close()

	// Create a minimal session for the resolver
	sess := &LuaSession{State: L}
	resolver := &LuaResolver{Session: sess}
	tracker := changetracker.NewTracker()
	tracker.Resolver = resolver

	// Create a Lua array with mixed content: number, table (object), string
	err := L.DoString(`
		person = {name = "Alice", age = 30}
		mixedArray = {1, person, "hello"}
	`)
	if err != nil {
		t.Fatalf("Lua error: %v", err)
	}

	// Get the person table and register it
	personTbl := L.GetGlobal("person").(*golua.LTable)
	personID, _ := tracker.RegisterObject(personTbl)

	// Get the mixed array
	arrayTbl := L.GetGlobal("mixedArray").(*golua.LTable)

	// luaValueToGo should preserve *lua.LTable for navigation purposes
	// (arrays are NOT converted to []any here - that happens in ToValueJSON)
	goArray, err := resolver.luaValueToGo(arrayTbl)
	if err != nil {
		t.Fatalf("luaValueToGo error: %v", err)
	}

	// Should still be *lua.LTable (for proper path navigation in Variable.Set)
	tbl, ok := goArray.(*golua.LTable)
	if !ok {
		t.Fatalf("Expected *lua.LTable (preserved for navigation), got %T", goArray)
	}

	// Verify the table still has our data
	if tbl.Len() != 3 {
		t.Fatalf("Expected 3 elements, got %d", tbl.Len())
	}

	// Now convert to Value JSON - the resolver's ConvertToValueJSON handles this
	// and converts the array to []any with proper element conversion
	valueJSON := tracker.ToValueJSON(arrayTbl)
	jsonArr, ok := valueJSON.([]any)
	if !ok {
		t.Fatalf("Expected []any from ToValueJSON, got %T", valueJSON)
	}
	if len(jsonArr) != 3 {
		t.Fatalf("Expected 3 elements in JSON array, got %d", len(jsonArr))
	}

	// Element 0: number -> float64
	if jsonArr[0] != float64(1) {
		t.Errorf("Element 0: expected float64(1), got %T(%v)", jsonArr[0], jsonArr[0])
	}

	// Element 2: string -> string
	if jsonArr[2] != "hello" {
		t.Errorf("Element 2: expected 'hello', got %T(%v)", jsonArr[2], jsonArr[2])
	}

	// Check element 1 is now an ObjectRef with the registered ID
	objRef, ok := jsonArr[1].(changetracker.ObjectRef)
	if !ok {
		t.Errorf("Element 1 in ValueJSON: expected ObjectRef, got %T(%v)", jsonArr[1], jsonArr[1])
	} else if objRef.Obj != personID {
		t.Errorf("Expected ObjectRef.Obj=%d, got %d", personID, objRef.Obj)
	}

	// Serialize to JSON bytes
	jsonBytes, err := json.Marshal(valueJSON)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	expected := fmt.Sprintf(`[1,{"obj":%d},"hello"]`, personID)
	if string(jsonBytes) != expected {
		t.Errorf("Expected %s, got %s", expected, string(jsonBytes))
	}

	t.Log("Lua array conversion test passed!")
}

// TestLuaResolverRWMethod verifies that Call and CallWith work correctly
// with Lua varargs methods for read/write access patterns.
// Spec: viewdefs.md (Read/Write Methods section)
// Design: crc-PathSyntax.md, seq-path-resolve.md
func TestLuaResolverRWMethod(t *testing.T) {
	L := golua.NewState()
	defer L.Close()

	// Create a minimal session for the resolver
	sess := &LuaSession{State: L}
	resolver := &LuaResolver{Session: sess}

	// Create a Lua table with a varargs method that acts as getter/setter
	err := L.DoString(`
		obj = {
			_value = "initial"
		}
		function obj:value(...)
			if select('#', ...) > 0 then
				self._value = select(1, ...)  -- write
			end
			return self._value  -- read
		end
	`)
	if err != nil {
		t.Fatalf("Lua error: %v", err)
	}

	objTbl := L.GetGlobal("obj").(*golua.LTable)

	// Test 1: Call (read) - should return current value without modifying
	result, err := resolver.Call(objTbl, "value")
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if result != "initial" {
		t.Errorf("Call: expected 'initial', got %v", result)
	}

	// Verify _value unchanged
	underlyingVal := L.GetField(objTbl, "_value")
	if golua.LVAsString(underlyingVal) != "initial" {
		t.Errorf("_value should still be 'initial' after Call")
	}

	// Test 2: CallWith (write) - should set the value
	err = resolver.CallWith(objTbl, "value", "updated")
	if err != nil {
		t.Fatalf("CallWith error: %v", err)
	}

	// Verify _value changed
	underlyingVal = L.GetField(objTbl, "_value")
	if golua.LVAsString(underlyingVal) != "updated" {
		t.Errorf("_value should be 'updated' after CallWith, got %v", golua.LVAsString(underlyingVal))
	}

	// Test 3: Call again (read) - should return the updated value
	result, err = resolver.Call(objTbl, "value")
	if err != nil {
		t.Fatalf("Call error after write: %v", err)
	}
	if result != "updated" {
		t.Errorf("Call after write: expected 'updated', got %v", result)
	}

	// Test 4: CallWith with different types
	err = resolver.CallWith(objTbl, "value", float64(42))
	if err != nil {
		t.Fatalf("CallWith with number error: %v", err)
	}
	result, _ = resolver.Call(objTbl, "value")
	if result != float64(42) {
		t.Errorf("Expected float64(42), got %T(%v)", result, result)
	}

	t.Log("Lua resolver rw method test passed!")
}

// =============================================================================
// Prototype Management Tests
// Test Design: test-Lua.md (Prototype Management Tests section)
// =============================================================================

// TestPrototypeCreation verifies session:prototype creates prototypes with
// type and __index set automatically
func TestPrototypeCreation(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", {
				name = "",
				email = "",
			})

			-- Verify type is set
			assert(Person.type == "Person", "Expected Person.type='Person', got " .. tostring(Person.type))

			-- Verify __index is set to self
			assert(Person.__index == Person, "Expected Person.__index == Person")

			-- Verify init fields are on prototype
			assert(Person.name == "", "Expected Person.name=''")
			assert(Person.email == "", "Expected Person.email=''")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}

	// Verify prototype is registered
	if sess.prototypeRegistry["Person"] == nil {
		t.Error("Expected Person prototype to be registered")
	}
}

// TestPrototypeDefaultNew verifies default :new() method is provided
func TestPrototypeDefaultNew(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Thing = session:prototype("Thing", { value = 0 })

			-- Default :new() should exist
			assert(Thing.new ~= nil, "Expected default :new() method")

			-- Create instance using default :new()
			local t = Thing:new({ value = 42 })

			-- Verify instance is created correctly
			assert(t.value == 42, "Expected t.value=42, got " .. tostring(t.value))
			assert(getmetatable(t) == Thing, "Expected metatable to be Thing")
			assert(t.type == "Thing", "Expected t.type='Thing' (inherited)")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}
}

// TestPrototypeCustomNew verifies custom :new() is preserved
func TestPrototypeCustomNew(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Counter = session:prototype("Counter", { count = 0 })
			Counter.nextId = Counter.nextId or 0

			-- Custom :new() with ID assignment
			function Counter:new(instance)
				instance = session:create(Counter, instance)
				instance.id = Counter.nextId
				Counter.nextId = Counter.nextId + 1
				return instance
			end

			local c1 = Counter:new()
			local c2 = Counter:new()

			-- Verify custom :new() is used
			assert(c1.id == 0, "Expected c1.id=0, got " .. tostring(c1.id))
			assert(c2.id == 1, "Expected c2.id=1, got " .. tostring(c2.id))
			assert(Counter.nextId == 2, "Expected Counter.nextId=2")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}
}

// TestEmptyMarker verifies EMPTY marks tracked nil fields
func TestEmptyMarker(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			-- Verify EMPTY global exists
			assert(EMPTY ~= nil, "Expected EMPTY global to exist")
			assert(type(EMPTY) == "table", "Expected EMPTY to be a table")

			User = session:prototype("User", {
				name = "",
				avatar = EMPTY,  -- starts nil, tracked for mutation
			})

			-- EMPTY should be removed from prototype (field is nil)
			assert(User.avatar == nil, "Expected User.avatar=nil (EMPTY removed)")

			-- Instance should also have nil avatar
			local u = User:new()
			assert(u.avatar == nil, "Expected u.avatar=nil")
			assert(u.name == "", "Expected u.name='' (inherited)")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}

	// Verify prototype is registered
	if sess.prototypeRegistry["User"] == nil {
		t.Error("Expected User prototype to be registered")
	}
}

// TestSessionCreate verifies session:create tracks instances
func TestSessionCreate(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "" })

			-- Direct use of session:create
			local p = session:create(Person, { name = "Alice" })

			assert(p.name == "Alice", "Expected p.name='Alice'")
			assert(getmetatable(p) == Person, "Expected metatable to be Person")
			assert(p.type == "Person", "Expected p.type='Person' (inherited)")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}

	// Verify instance is tracked
	info := sess.prototypeRegistry["Person"]
	if info == nil {
		t.Fatal("Expected Person in prototype registry")
	}
	// Instance tracking is in instanceRegistry
	if len(sess.instanceRegistry) == 0 {
		t.Error("Expected at least one prototype in instance registry")
	}
}

// TestSessionCreateNil verifies nil instance becomes empty table
func TestSessionCreateNil(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "default" })

			-- Create with nil (should create empty table)
			local p = session:create(Person, nil)

			assert(p ~= nil, "Expected p to be a table, not nil")
			assert(type(p) == "table", "Expected p to be a table")
			assert(getmetatable(p) == Person, "Expected metatable to be Person")
			assert(p.name == "default", "Expected p.name='default' (inherited)")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}
}

// TestPrototypeUpdateDetectsChange verifies init change detection
func TestPrototypeUpdateDetectsChange(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Initial prototype
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "" })
			local alice = Person:new({ name = "Alice" })
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Simulate hot-reload with new field
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "", email = "" })

			-- Verify email field now exists on prototype
			assert(Person.email == "", "Expected Person.email=''")
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Reload Lua execution error: %v", err)
	}

	// Verify prototype was queued for mutation
	// (queue is cleared after processMutationQueue, so check stored init instead)
	info := sess.prototypeRegistry["Person"]
	if info == nil {
		t.Fatal("Expected Person in prototype registry")
	}
	if info.storedInit["email"] == nil {
		t.Error("Expected stored init to have email field after update")
	}
}

// TestPrototypeIdentityPreserved verifies existing instances keep working
func TestPrototypeIdentityPreserved(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "" })
			alice = Person:new({ name = "Alice" })
			originalPerson = Person  -- save reference

			-- Simulate hot-reload
			Person = session:prototype("Person", { name = "", age = 0 })

			-- Verify table identity preserved
			assert(Person == originalPerson, "Expected Person table identity to be preserved")

			-- Verify alice still valid
			assert(getmetatable(alice) == Person, "Expected alice's metatable to still be Person")

			-- Verify alice inherits new field
			assert(alice.age == 0, "Expected alice.age=0 (inherited from updated prototype)")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}
}

// TestDetectRemovedFields verifies removal detection for cleanup
func TestDetectRemovedFields(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			-- Initial prototype with oldField
			Person = session:prototype("Person", { name = "", oldField = "data" })
			p = Person:new({ name = "Bob", oldField = "mydata" })

			-- Verify initial state
			assert(p.oldField == "mydata", "Expected p.oldField='mydata'")
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload with oldField removed - this should queue mutation and nil out field
	_, err = sess.LoadCodeDirect("reload", `
		-- Hot-reload with oldField removed
		Person = session:prototype("Person", { name = "" })
	`)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify oldField was nil'd on instance
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			-- Use rawget to verify the field was removed from instance (not checking metatable)
			assert(rawget(p, "oldField") == nil, "Expected rawget(p, 'oldField')=nil after removal, got " .. tostring(rawget(p, "oldField")))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// TestNoMutationWhenUnchanged verifies identical init doesn't queue
func TestNoMutationWhenUnchanged(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "" })
			mutateCount = 0

			function Person:mutate()
				mutateCount = mutateCount + 1
			end
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload with identical init
	_, err = sess.LoadCodeDirect("reload", `
		Person = session:prototype("Person", { name = "" })
	`)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify mutate was not called (no change detected)
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			assert(mutateCount == 0, "Expected mutateCount=0 (no change), got " .. tostring(mutateCount))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// TestMutationQueueCallsMutate verifies :mutate() called on instances
func TestMutationQueueCallsMutate(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "" })
			alice = Person:new({ name = "Alice" })
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload with new field and mutate method
	// Use a flag to verify mutate() was called (avoids or-pattern issues with empty strings)
	_, err = sess.LoadCodeDirect("reload", `
		Person = session:prototype("Person", { name = "", email = "" })
		function Person:mutate()
			self.mutated = true
		end
	`)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify mutate was called
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			assert(alice.mutated == true, "Expected alice.mutated=true (mutate was called), got " .. tostring(alice.mutated))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// TestMutationQueueFIFOOrder verifies prototypes processed in declaration order
func TestMutationQueueFIFOOrder(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			mutationOrder = {}

			Address = session:prototype("Address", { city = "" })
			Person = session:prototype("Person", { name = "" })

			a = Address:new({ city = "NYC" })
			p = Person:new({ name = "Bob" })
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload both with mutate methods that track order
	_, err = sess.LoadCodeDirect("reload", `
		Address = session:prototype("Address", { city = "", zip = "" })
		function Address:mutate()
			table.insert(mutationOrder, "Address")
		end

		Person = session:prototype("Person", { name = "", age = 0 })
		function Person:mutate()
			table.insert(mutationOrder, "Person")
		end
	`)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify order
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			assert(#mutationOrder == 2, "Expected 2 mutations, got " .. #mutationOrder)
			assert(mutationOrder[1] == "Address", "Expected Address first, got " .. tostring(mutationOrder[1]))
			assert(mutationOrder[2] == "Person", "Expected Person second, got " .. tostring(mutationOrder[2]))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// TestMutateErrorsIsolated verifies one bad mutate doesn't break others
func TestMutateErrorsIsolated(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Bad = session:prototype("Bad", { x = 0 })
			Good = session:prototype("Good", { y = 0 })

			b = Bad:new()
			g = Good:new()
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload with Bad.mutate that errors and Good.mutate that works
	_, err = sess.LoadCodeDirect("reload", `
		Bad = session:prototype("Bad", { x = 1 })
		function Bad:mutate()
			error("mutation failed!")
		end

		Good = session:prototype("Good", { y = 1 })
		function Good:mutate()
			self.y = 42
		end
	`)
	// Note: reload should succeed despite mutation error (pcall isolation)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify Good.mutate still ran
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			assert(g.y == 42, "Expected g.y=42, got " .. tostring(g.y))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// TestRemovedFieldsNildAfterMutate verifies field removal happens after migration
func TestRemovedFieldsNildAfterMutate(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", { name = "", fullName = "" })
			p = Person:new({ fullName = "Alice Smith" })
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload: rename fullName to name
	_, err = sess.LoadCodeDirect("reload", `
		Person = session:prototype("Person", { name = "" })
		function Person:mutate()
			-- Migration: copy fullName to name before fullName is removed
			-- Use rawget to check for field on instance directly (not metatable)
			local instanceFullName = rawget(self, "fullName")
			if instanceFullName and instanceFullName ~= "" then
				self.name = instanceFullName
			end
		end
	`)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify migration worked and field was removed
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			assert(p.name == "Alice Smith", "Expected p.name='Alice Smith', got " .. tostring(p.name))
			-- Check rawget for fullName (removed fields are nil'd on instance)
			assert(rawget(p, "fullName") == nil, "Expected rawget(p, 'fullName')=nil (removed), got " .. tostring(rawget(p, "fullName")))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// TestPrototypeSharedStatePreserved verifies non-init fields preserved on reload
func TestPrototypeSharedStatePreserved(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Counter = session:prototype("Counter", { count = 0 })
			Counter.nextId = Counter.nextId or 0
			Counter.nextId = Counter.nextId + 1  -- nextId = 1
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Initial Lua execution error: %v", err)
	}

	// Reload with same pattern
	_, err = sess.LoadCodeDirect("reload", `
		Counter = session:prototype("Counter", { count = 0 })
		Counter.nextId = Counter.nextId or 0  -- should stay 1
	`)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}

	// Verify nextId preserved
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State

		code := `
			assert(Counter.nextId == 1, "Expected Counter.nextId=1 (preserved), got " .. tostring(Counter.nextId))
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}
}

// =============================================================================
// Original Tests
// =============================================================================

// TestUILog verifies ui.log works with 1 or 2 arguments
func TestUILog(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	// Capture log output via config verbosity?
	// The log output goes to stdout/stderr via log.Printf, which is hard to capture in test.
	// But we just want to ensure it doesn't crash (RaiseError).

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			-- Test 1 argument (should succeed with default level)
			ui.log("Message with default level")

			-- Test 2 arguments (should succeed with specific level)
			ui.log(1, "Message with level 1")
		`
		return nil, L.DoString(code)
	})

	if err != nil {
		t.Fatalf("Lua execution failed: %v", err)
	}
}

// CRC: crc-LuaSession.md
func TestRemovePrototype(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Create some prototypes
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Person = session:prototype("Person", {name = ""})
			ContactPerson = session:prototype("contacts.Person", {phone = ""})
			ContactAddress = session:prototype("contacts.Address", {street = ""})
			Animal = session:prototype("Animal", {species = ""})
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}

	// Verify all prototypes exist
	if sess.prototypeRegistry["Person"] == nil {
		t.Error("Expected Person prototype to be registered")
	}
	if sess.prototypeRegistry["contacts.Person"] == nil {
		t.Error("Expected contacts.Person prototype to be registered")
	}
	if sess.prototypeRegistry["contacts.Address"] == nil {
		t.Error("Expected contacts.Address prototype to be registered")
	}
	if sess.prototypeRegistry["Animal"] == nil {
		t.Error("Expected Animal prototype to be registered")
	}

	// Test removing single prototype (children=false)
	sess.RemovePrototype("Person", false)
	if sess.prototypeRegistry["Person"] != nil {
		t.Error("Expected Person prototype to be removed")
	}
	// Others should still exist
	if sess.prototypeRegistry["contacts.Person"] == nil {
		t.Error("Expected contacts.Person to still exist")
	}

	// Test removing with children=true
	sess.RemovePrototype("contacts", true)
	if sess.prototypeRegistry["contacts.Person"] != nil {
		t.Error("Expected contacts.Person to be removed")
	}
	if sess.prototypeRegistry["contacts.Address"] != nil {
		t.Error("Expected contacts.Address to be removed")
	}

	// Animal should still exist
	if sess.prototypeRegistry["Animal"] == nil {
		t.Error("Expected Animal prototype to still exist")
	}

	// Test removing non-existent prototype (should not error)
	sess.RemovePrototype("NonExistent", false)
	sess.RemovePrototype("AlsoNonExistent", true)
}


// TestModuleTracking verifies that module tracking works correctly
func TestModuleTracking(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)
	rt.SetWrapperRegistry(NewWrapperRegistry())

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Simulate loading a module by setting current module
	sess.SetCurrentModule("apps/contacts/app.lua", "apps/contacts")

	// Register a prototype while module is active
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)

		code := `
			Contact = session:prototype("Contact", {name = ""})
			ui.registerPresenter("ContactPresenter", {})
		`
		return nil, L.DoString(code)
	})
	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}

	sess.ClearCurrentModule()

	// Verify module tracking
	module, exists := sess.modules["apps/contacts/app.lua"]
	if !exists {
		t.Fatal("Expected module to be tracked")
	}
	if len(module.Prototypes) != 1 || module.Prototypes[0] != "Contact" {
		t.Errorf("Expected prototype 'Contact' to be tracked, got: %v", module.Prototypes)
	}
	if len(module.PresenterTypes) != 1 || module.PresenterTypes[0] != "ContactPresenter" {
		t.Errorf("Expected presenter type 'ContactPresenter' to be tracked, got: %v", module.PresenterTypes)
	}

	// Verify directory tracking
	mods, exists := sess.moduleDirectories["apps/contacts"]
	if !exists || len(mods) != 1 {
		t.Fatalf("Expected directory to have 1 module, got: %v", mods)
	}

	// Unload the module
	sess.UnloadModule("apps/contacts/app.lua")

	// Verify cleanup
	if _, exists := sess.modules["apps/contacts/app.lua"]; exists {
		t.Error("Expected module to be removed after unload")
	}
	if sess.prototypeRegistry["Contact"] != nil {
		t.Error("Expected Contact prototype to be removed after unload")
	}
	if sess.presenterTypes["ContactPresenter"] != nil {
		t.Error("Expected ContactPresenter to be removed after unload")
	}
}

// TestUnloadDirectory verifies that directory unloading works correctly
func TestUnloadDirectory(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	store := newMockStore()
	rt.SetVariableStore(store)

	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Simulate loading multiple modules from the same directory
	sess.SetCurrentModule("apps/myapp/module1.lua", "apps/myapp")
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		L.SetGlobal("session", sess.sessionTable)
		return nil, L.DoString(`M1 = session:prototype("Module1Type", {})`)
	})
	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}
	sess.ClearCurrentModule()

	sess.SetCurrentModule("apps/myapp/module2.lua", "apps/myapp")
	_, err = rt.execute(func() (interface{}, error) {
		L := rt.State
		return nil, L.DoString(`M2 = session:prototype("Module2Type", {})`)
	})
	if err != nil {
		t.Fatalf("Lua execution error: %v", err)
	}
	sess.ClearCurrentModule()

	// Verify both modules are tracked
	if len(sess.moduleDirectories["apps/myapp"]) != 2 {
		t.Fatalf("Expected 2 modules in directory, got: %d", len(sess.moduleDirectories["apps/myapp"]))
	}

	// Unload the directory
	sess.UnloadDirectory("apps/myapp")

	// Verify all modules and prototypes are removed
	if len(sess.modules) != 0 {
		t.Errorf("Expected no modules after directory unload, got: %v", sess.modules)
	}
	if sess.prototypeRegistry["Module1Type"] != nil {
		t.Error("Expected Module1Type to be removed")
	}
	if sess.prototypeRegistry["Module2Type"] != nil {
		t.Error("Expected Module2Type to be removed")
	}
	if _, exists := sess.moduleDirectories["apps/myapp"]; exists {
		t.Error("Expected directory entry to be removed")
	}
}
