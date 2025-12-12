package lua

import (
	"encoding/json"
	"testing"

	golua "github.com/yuin/gopher-lua"
)

// mockVariableStore implements VariableStore for testing
type mockVariableStore struct {
	variables map[int64]*mockVariable
	nextID    int64
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

func TestChangeDetection(t *testing.T) {
	// Create runtime
	rt, err := NewRuntime("/tmp")
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer rt.Shutdown()

	// Set up mock store
	store := newMockStore()
	rt.SetVariableStore(store)
	rt.SetVerbosity(2)

	// Create a Lua session
	sess, err := rt.CreateLuaSession("1")
	if err != nil {
		t.Fatalf("Failed to create Lua session: %v", err)
	}

	// Execute Lua code to create app variable
	rt.execute(func() (interface{}, error) {
		L := rt.state

		// Set session global
		L.SetGlobal("session", sess.sessionTable)

		// Create app object and variable
		code := `
			local App = {type = "TestApp"}
			App.__index = App
			function App:new(tbl)
				tbl = tbl or {}
				setmetatable(tbl, self)
				tbl.title = tbl.title or "Initial Title"
				tbl.count = tbl.count or 0
				return tbl
			end

			app = App:new({title = "Test App", count = 0})
			session:createAppVariable(app)
		`
		if err := L.DoString(code); err != nil {
			return nil, err
		}
		return nil, nil
	})

	// Verify app variable was created
	if sess.appVariableID != 1 {
		t.Fatalf("Expected app variable ID 1, got %d", sess.appVariableID)
	}

	// Check initial value in store
	val, _, _ := store.Get(1)
	var initial map[string]interface{}
	json.Unmarshal(val, &initial)
	if initial["title"] != "Test App" {
		t.Errorf("Expected title 'Test App', got %v", initial["title"])
	}

	// Modify the app object directly (simulate what a method would do)
	rt.execute(func() (interface{}, error) {
		L := rt.state
		appObj := L.GetGlobal("app").(*golua.LTable)
		L.SetField(appObj, "title", golua.LString("Modified Title"))
		L.SetField(appObj, "count", golua.LNumber(42))
		return nil, nil
	})

	// Run change detection
	updates := rt.AfterBatch("1")

	// Should detect the change
	if len(updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(updates))
	}

	// Check the update contains the new value
	var updated map[string]interface{}
	json.Unmarshal(updates[0].Value, &updated)
	if updated["title"] != "Modified Title" {
		t.Errorf("Expected title 'Modified Title', got %v", updated["title"])
	}
	if updated["count"] != float64(42) {
		t.Errorf("Expected count 42, got %v", updated["count"])
	}

	// Store should also be updated
	val2, _, _ := store.Get(1)
	var storeVal map[string]interface{}
	json.Unmarshal(val2, &storeVal)
	if storeVal["title"] != "Modified Title" {
		t.Errorf("Store should have updated value, got %v", storeVal["title"])
	}

	// Run change detection again - no changes this time
	updates2 := rt.AfterBatch("1")
	if len(updates2) != 0 {
		t.Errorf("Expected 0 updates on second call, got %d", len(updates2))
	}

	t.Log("Change detection test passed!")
}

func TestPrivateFieldsNotSerialized(t *testing.T) {
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

	// Create app with public and private fields
	rt.execute(func() (interface{}, error) {
		L := rt.state
		L.SetGlobal("session", sess.sessionTable)

		code := `
			local App = {type = "TestApp"}
			App.__index = App
			function App:new(tbl)
				tbl = tbl or {}
				setmetatable(tbl, self)
				tbl.title = tbl.title or "Public Title"
				tbl.count = tbl.count or 0
				tbl._privateData = tbl._privateData or "secret"
				tbl._nextId = tbl._nextId or 1
				return tbl
			end

			app = App:new({
				title = "My App",
				count = 5,
				_privateData = "internal data",
				_nextId = 100
			})
			session:createAppVariable(app)
		`
		return nil, L.DoString(code)
	})

	// Check the stored value
	val, _, _ := store.Get(1)
	var data map[string]interface{}
	json.Unmarshal(val, &data)

	// Public fields should be present
	if data["title"] != "My App" {
		t.Errorf("Expected title 'My App', got %v", data["title"])
	}
	if data["count"] != float64(5) {
		t.Errorf("Expected count 5, got %v", data["count"])
	}

	// Private fields (prefixed with _) should NOT be present
	if _, ok := data["_privateData"]; ok {
		t.Error("Private field _privateData should not be serialized")
	}
	if _, ok := data["_nextId"]; ok {
		t.Error("Private field _nextId should not be serialized")
	}

	t.Log("Private fields test passed!")
}

func TestCreateVariable(t *testing.T) {
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
				tbl.items = tbl.items or {}
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
			assert(itemId == 2, "Expected item ID 2")
		`
		return nil, L.DoString(code)
	})

	// Check both variables exist
	if len(store.variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(store.variables))
	}

	// Check watched variables
	if len(sess.watchedVariables) != 2 {
		t.Errorf("Expected 2 watched variables, got %d", len(sess.watchedVariables))
	}

	// Modify the child item and detect changes
	rt.execute(func() (interface{}, error) {
		L := rt.state
		itemObj := L.GetGlobal("item").(*golua.LTable)
		L.SetField(itemObj, "name", golua.LString("Modified Item"))
		return nil, nil
	})

	updates := rt.AfterBatch("1")
	if len(updates) != 1 {
		t.Errorf("Expected 1 update for item, got %d", len(updates))
	}
	if len(updates) > 0 && updates[0].VarID != 2 {
		t.Errorf("Expected update for variable 2, got %d", updates[0].VarID)
	}

	t.Log("CreateVariable test passed!")
}
