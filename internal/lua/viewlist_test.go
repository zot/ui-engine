package lua

import (
	"testing"
)

// TestViewListInitialization tests that ViewList initializes correctly.
// Note: Full SyncViewItems testing requires a complete session setup.
// These tests verify basic struct initialization.
func TestViewListInitialization(t *testing.T) {
	vl := &ViewList{
		Items:          make([]*ViewListItem, 0),
		SelectionIndex: -1,
		nextObjID:      -1,
	}

	if vl.SelectionIndex != -1 {
		t.Errorf("Expected initial SelectionIndex to be -1, got %d", vl.SelectionIndex)
	}

	if len(vl.Items) != 0 {
		t.Errorf("Expected Items to be empty, got %d items", len(vl.Items))
	}

	if vl.nextObjID != -1 {
		t.Errorf("Expected nextObjID to be -1, got %d", vl.nextObjID)
	}
}

func TestViewListItemCreation(t *testing.T) {
	vl := &ViewList{
		Items:          make([]*ViewListItem, 0),
		SelectionIndex: -1,
		nextObjID:      -1,
	}

	// Create a ViewListItem directly
	item := NewViewListItem(nil, vl, 0)

	if item == nil {
		t.Fatal("Expected NewViewListItem to return non-nil")
	}

	if item.Index != 0 {
		t.Errorf("Expected Index to be 0, got %d", item.Index)
	}

	if item.List != vl {
		t.Errorf("Expected List to be the parent ViewList")
	}
}
