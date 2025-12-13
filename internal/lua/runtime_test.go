package lua

import (
	"encoding/json"
	"testing"

	changetracker "github.com/zot/change-tracker"
	golua "github.com/yuin/gopher-lua"
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

// DetectChanges returns changes for a session
func (s *mockVariableStore) DetectChanges(sessionID string) []changetracker.Change {
	tracker := s.trackers[sessionID]
	if tracker == nil {
		return nil
	}
	return tracker.DetectChanges()
}

// TestObjectRegistration verifies that Lua tables are registered as objects
// and their Value JSON is an object reference.
func TestObjectRegistration(t *testing.T) {
	rt, err := NewRuntime("/tmp")
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
		L := rt.state
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

	// ValueJSON should be ObjectRef{Obj: 1}
	objRef, ok := v.ValueJSON.(changetracker.ObjectRef)
	if !ok {
		t.Fatalf("Expected ValueJSON to be ObjectRef, got %T: %v", v.ValueJSON, v.ValueJSON)
	}
	if objRef.Obj != 1 {
		t.Errorf("Expected ObjectRef.Obj=1, got %d", objRef.Obj)
	}

	// Serialized form should be {"obj": 1}
	jsonBytes, err := tracker.ToValueJSONBytes(v.Value)
	if err != nil {
		t.Fatalf("ToValueJSONBytes error: %v", err)
	}
	expected := `{"obj":1}`
	if string(jsonBytes) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(jsonBytes))
	}

	t.Log("Object registration test passed!")
}

// TestCreateMultipleVariables verifies creating app and child variables
func TestCreateMultipleVariables(t *testing.T) {
	rt, err := NewRuntime("/tmp")
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
	rt.execute(func() (interface{}, error) {
		L := rt.state
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
			session:createAppVariable(app)

			-- Create a child item
			item = Item:new({name = "Test Item"})
			itemId = session:createVariable(app, item)

			-- Verify item ID
			assert(itemId == 2, "Expected item ID 2, got " .. tostring(itemId))
		`
		return nil, L.DoString(code)
	})

	// Check both variables exist in tracker
	tracker := store.GetTracker("1")
	if tracker == nil {
		t.Fatal("Expected tracker for session 1")
	}

	v1 := tracker.GetVariable(1)
	if v1 == nil {
		t.Error("Expected variable 1 (app) in tracker")
	}

	v2 := tracker.GetVariable(2)
	if v2 == nil {
		t.Error("Expected variable 2 (item) in tracker")
	}

	// Both should have object refs as ValueJSON
	if _, ok := v1.ValueJSON.(changetracker.ObjectRef); !ok {
		t.Errorf("Variable 1 ValueJSON should be ObjectRef, got %T", v1.ValueJSON)
	}
	if _, ok := v2.ValueJSON.(changetracker.ObjectRef); !ok {
		t.Errorf("Variable 2 ValueJSON should be ObjectRef, got %T", v2.ValueJSON)
	}

	// Variable 2 should have parent ID 1
	if v2.ParentID != 1 {
		t.Errorf("Expected variable 2 parent ID 1, got %d", v2.ParentID)
	}

	t.Log("Create multiple variables test passed!")
}

// TestLuaResolverArrayConversion verifies that Lua arrays convert properly
// with object elements becoming object refs
func TestLuaResolverArrayConversion(t *testing.T) {
	L := golua.NewState()
	defer L.Close()

	resolver := &LuaResolver{L: L}
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
	tracker.RegisterObject(personTbl, 42) // Register with ID 42

	// Get the mixed array
	arrayTbl := L.GetGlobal("mixedArray").(*golua.LTable)

	// Convert array to Go using resolver
	goArray, err := resolver.luaValueToGo(arrayTbl)
	if err != nil {
		t.Fatalf("luaValueToGo error: %v", err)
	}

	arr, ok := goArray.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", goArray)
	}
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}

	// Element 0: number -> float64
	if arr[0] != float64(1) {
		t.Errorf("Element 0: expected float64(1), got %T(%v)", arr[0], arr[0])
	}

	// Element 1: table -> *lua.LTable (kept as ref)
	if _, ok := arr[1].(*golua.LTable); !ok {
		t.Errorf("Element 1: expected *lua.LTable, got %T", arr[1])
	}

	// Element 2: string -> string
	if arr[2] != "hello" {
		t.Errorf("Element 2: expected 'hello', got %T(%v)", arr[2], arr[2])
	}

	// Now convert to Value JSON - the table should become {"obj": 42}
	valueJSON := tracker.ToValueJSON(arr)
	jsonArr, ok := valueJSON.([]any)
	if !ok {
		t.Fatalf("Expected []any from ToValueJSON, got %T", valueJSON)
	}

	// Check element 1 is now an ObjectRef
	objRef, ok := jsonArr[1].(changetracker.ObjectRef)
	if !ok {
		t.Errorf("Element 1 in ValueJSON: expected ObjectRef, got %T(%v)", jsonArr[1], jsonArr[1])
	} else if objRef.Obj != 42 {
		t.Errorf("Expected ObjectRef.Obj=42, got %d", objRef.Obj)
	}

	// Serialize to JSON bytes
	jsonBytes, err := json.Marshal(valueJSON)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	expected := `[1,{"obj":42},"hello"]`
	if string(jsonBytes) != expected {
		t.Errorf("Expected %s, got %s", expected, string(jsonBytes))
	}

	t.Log("Lua array conversion test passed!")
}
