package lua

import (
	"testing"

	"github.com/zot/ui/internal/config"
)

// mockWrapperVariable implements WrapperVariable for testing
type mockWrapperVariable struct {
	id         int64
	value      interface{}
	properties map[string]string
}

func (m *mockWrapperVariable) GetID() int64 {
	return m.id
}

func (m *mockWrapperVariable) GetValue() interface{} {
	return m.value
}

func (m *mockWrapperVariable) GetProperty(name string) string {
	return m.properties[name]
}

func TestNewViewListWithInitialValue(t *testing.T) {
	runtime, err := NewRuntime(config.DefaultConfig(), "/tmp")
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Shutdown()

	rawValue := []interface{}{
		&mockDomainObject{Name: "A"},
		&mockDomainObject{Name: "B"},
	}

	variable := &mockWrapperVariable{id: 1, value: rawValue}
	wrapper := NewViewList(runtime, variable)

	vl, ok := wrapper.(*ViewList)
	if !ok {
		t.Fatalf("Expected wrapper to be of type *ViewList, got %T", wrapper)
	}

	if vl.SelectionIndex != -1 {
		t.Errorf("Expected SelectionIndex to be -1, got %d", vl.SelectionIndex)
	}

	if len(vl.Items) != 2 {
		t.Fatalf("Expected Items slice to have length 2, got %d", len(vl.Items))
	}
	if vl.Items[0].Index != 0 || vl.Items[1].Index != 1 {
		t.Errorf("Expected indices to be 0 and 1, got %d and %d", vl.Items[0].Index, vl.Items[1].Index)
	}
}

type mockDomainObject struct {
	Name string
}

func TestSyncViewItems(t *testing.T) {
	runtime, err := NewRuntime(config.DefaultConfig(), "/tmp")
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Shutdown()

	variable := &mockWrapperVariable{id: 1}
	wrapper := NewViewList(runtime, variable)
	vl := wrapper.(*ViewList)

	// Test Growth
	vl.value = []interface{}{
		map[string]interface{}{"Name": "A"},
		map[string]interface{}{"Name": "B"},
	}
	vl.SyncViewItems()

	if len(vl.Items) != 2 {
		t.Fatalf("Growth: Expected Items slice to have length 2, got %d", len(vl.Items))
	}
	if vl.Items[0].Index != 0 || vl.Items[1].Index != 1 {
		t.Errorf("Growth: Expected indices to be 0 and 1, got %d and %d", vl.Items[0].Index, vl.Items[1].Index)
	}
	if vl.Items[0].Item.(map[string]interface{})["Name"] != "A" {
		t.Errorf("Growth: Expected item 0 to have name 'A', got %s", vl.Items[0].Item.(map[string]interface{})["Name"])
	}

	// Test Shrink
	vl.value = []interface{}{
		map[string]interface{}{"Name": "C"},
	}
	vl.SyncViewItems()

	if len(vl.Items) != 1 {
		t.Fatalf("Shrink: Expected Items slice to have length 1, got %d", len(vl.Items))
	}
	if vl.Items[0].Item.(map[string]interface{})["Name"] != "C" {
		t.Errorf("Shrink: Expected item 0 to have name 'C', got %s", vl.Items[0].Item.(map[string]interface{})["Name"])
	}

	// Test Reorder
	objA := map[string]interface{}{"Name": "A"}
	objB := map[string]interface{}{"Name": "B"}
	vl.value = []interface{}{objA, objB}
	vl.SyncViewItems()
	vl.value = []interface{}{objB, objA}
	vl.SyncViewItems()

	if len(vl.Items) != 2 {
		t.Fatalf("Reorder: Expected Items slice to have length 2, got %d", len(vl.Items))
	}
	if vl.Items[0].Item.(map[string]interface{})["Name"] != "B" {
		t.Errorf("Reorder: Expected item 0 to have name 'B', got %s", vl.Items[0].Item.(map[string]interface{})["Name"])
	}
	if vl.Items[1].Item.(map[string]interface{})["Name"] != "A" {
		t.Errorf("Reorder: Expected item 1 to have name 'A', got %s", vl.Items[1].Item.(map[string]interface{})["Name"])
	}
}
