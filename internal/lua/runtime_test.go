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
