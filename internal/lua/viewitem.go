// Package lua provides ViewItem type for ViewList array elements.
// CRC: crc-ViewItem.md
// Spec: viewdefs.md
// Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
package lua

import (
	"encoding/json"
	"sync"
)

// ViewItem represents an element in a ViewList.
// It provides domain object access (baseItem), presenter access (item),
// list context (list), and position tracking (index).
//
// When ViewList has item=PresenterType property:
// - baseItem = domain object ref (e.g., {obj: 101})
// - item = ItemWrapper(viewItem) result (presenter ref)
//
// Without item property:
// - baseItem = domain object ref
// - item = same as baseItem
type ViewItem struct {
	ObjID    int64           // ViewItem's own object ID (negative, UI server managed)
	BaseItem json.RawMessage // Domain object reference ({obj: ID})
	Item     json.RawMessage // Either baseItem or wrapped presenter ref
	List     *ViewListWrapper
	Index    int
	mu       sync.RWMutex
}

// NewViewItem creates a new ViewItem for a domain object.
// baseItemID is the domain object's ID, index is the position in the list.
func NewViewItem(viewItemObjID int64, baseItemID int64, list *ViewListWrapper, index int) *ViewItem {
	baseItemRef, _ := json.Marshal(map[string]int64{"obj": baseItemID})
	return &ViewItem{
		ObjID:    viewItemObjID,
		BaseItem: baseItemRef,
		Item:     baseItemRef, // Default: item = baseItem
		List:     list,
		Index:    index,
	}
}

// GetObjID returns the ViewItem's object ID.
func (vi *ViewItem) GetObjID() int64 {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return vi.ObjID
}

// GetBaseItem returns the domain object reference.
func (vi *ViewItem) GetBaseItem() json.RawMessage {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return vi.BaseItem
}

// GetItem returns the (possibly wrapped) item reference.
func (vi *ViewItem) GetItem() json.RawMessage {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return vi.Item
}

// SetItem sets the item reference (used when wrapping with ItemWrapper).
func (vi *ViewItem) SetItem(item json.RawMessage) {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.Item = item
}

// GetList returns the owning ViewList.
func (vi *ViewItem) GetList() *ViewListWrapper {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return vi.List
}

// GetIndex returns the position in the list.
func (vi *ViewItem) GetIndex() int {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return vi.Index
}

// SetIndex updates the position (called when list reorders).
func (vi *ViewItem) SetIndex(index int) {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.Index = index
}

// Remove removes this item from the list via list.RemoveAt(index).
// This is a convenience method for presenter delete actions.
func (vi *ViewItem) Remove() error {
	vi.mu.RLock()
	list := vi.List
	index := vi.Index
	vi.mu.RUnlock()

	if list != nil {
		return list.RemoveAt(index)
	}
	return nil
}

// ToJSON returns the ViewItem's data as JSON for sending to frontend.
// The frontend receives this as the ViewItem object's value.
func (vi *ViewItem) ToJSON() json.RawMessage {
	vi.mu.RLock()
	defer vi.mu.RUnlock()

	// Parse baseItem to get obj ID
	var baseItemRef map[string]int64
	json.Unmarshal(vi.BaseItem, &baseItemRef)

	// Parse item to get obj ID
	var itemRef map[string]int64
	json.Unmarshal(vi.Item, &itemRef)

	data := map[string]interface{}{
		"baseItem": baseItemRef,
		"item":     itemRef,
		"index":    vi.Index,
		// list is the ViewList obj ref, but ViewItem doesn't store it as ref
		// The frontend ViewItem viewdef uses ui-action="remove()" which
		// calls the remove method on the ViewItem object itself
	}

	result, _ := json.Marshal(data)
	return result
}

// ObjRef returns an object reference to this ViewItem ({obj: ObjID}).
func (vi *ViewItem) ObjRef() json.RawMessage {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	ref, _ := json.Marshal(map[string]int64{"obj": vi.ObjID})
	return ref
}

// GetBaseItemID extracts the domain object ID from baseItem.
func (vi *ViewItem) GetBaseItemID() int64 {
	vi.mu.RLock()
	defer vi.mu.RUnlock()

	var ref map[string]interface{}
	if err := json.Unmarshal(vi.BaseItem, &ref); err != nil {
		return 0
	}
	if obj, ok := ref["obj"].(float64); ok {
		return int64(obj)
	}
	return 0
}
