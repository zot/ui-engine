// Package lua provides ViewList wrapper for array value transformation.
// CRC: crc-ViewList.md, crc-Wrapper.md
// Spec: viewdefs.md, protocol.md
// Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
package lua

import (
	"fmt"
	"reflect"
	"sync"

	changetracker "github.com/zot/change-tracker"
)

// ViewList transforms an array of domain object refs into ViewListItem refs.
// It creates ViewListItem objects for each item in the source array.
type ViewList struct {
	session        *LuaSession
	variable       WrapperVariable // The variable being wrapped (for property access)
	value          interface{}     // The raw array of domain objects (slice or array)
	Items          []*ViewListItem // The actual list of ViewListItem objects
	SelectionIndex int             // The current selection index
	itemType       string          // ItemWrapper type name from "item" property
	nextObjID      int64           // counter for generating ViewListItem object IDs
	mu             sync.RWMutex
}

// NewViewList creates a new ViewList wrapper for a variable.
func NewViewList(sess *LuaSession, variable WrapperVariable) interface{} {
	itemType := variable.GetProperty("item")

	if sess != nil {
		sess.Log(2, "ViewList: created for variable %d with item type %q", variable.GetID(), itemType)
		sess.Log(4, "ViewList created: varID=%d itemType=%q", variable.GetID(), itemType)
	}

	vl := &ViewList{
		session:        sess,
		variable:       variable,
		itemType:       itemType,
		Items:          make([]*ViewListItem, 0),
		value:          nil,
		SelectionIndex: -1, // Default to no selection
		nextObjID:      -1, // Start negative IDs for UI server managed objects
	}

	// Initial update
	vl.Update(variable.GetValue())
	return vl
}

func (vl *ViewList) Tracker() *changetracker.Tracker {
	return vl.session.variableStore.GetTracker(vl.session.ID)
}

// Value returns the list of ViewListItems.
func (vl *ViewList) Value() interface{} {
	vl.mu.RLock()
	defer vl.mu.RUnlock()
	return vl.Items
}

// Update updates the ViewList with a new raw value from the backend.
func (vl *ViewList) Update(newValue interface{}) {
	vl.mu.Lock()

	vl.session.Log(4, "ViewList\n  variable %d\n  value: %v", vl.variable.GetID(), vl.variable.GetValue())

	// Update raw value
	if newValue != nil {
		val := reflect.ValueOf(newValue)
		kind := val.Kind()
		if kind == reflect.Slice || kind == reflect.Array {
			vl.value = newValue
		} else {
			// Not a slice/array
			vl.value = nil
			if vl.session != nil {
				vl.session.Log(1, "ViewList: expected slice or array, got %T", newValue)
			}
		}
	} else {
		vl.value = nil
	}

	vl.mu.Unlock()

	// Sync items (acquires its own lock)
	vl.SyncViewItems()
}

// SyncViewItems synchronizes the `Items` slice with the `value` slice.
func (vl *ViewList) SyncViewItems() {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	var count int
	var get func(int) interface{}

	if vl.value != nil {
		val := reflect.ValueOf(vl.value)
		count = val.Len()
		get = func(i int) interface{} { return val.Index(i).Interface() }
	} else {
		count = 0
		get = func(i int) interface{} { return nil }
	}

	// Grow: If len(value) > len(Items), append new ViewListItems
	for count > len(vl.Items) {
		newItem, err := vl.createListItem()
		if err != nil {
			if vl.session != nil {
				vl.session.Log(1, "ViewList: failed to create ViewListItem: %v", err)
			}
			break
		}
		vl.Items = append(vl.Items, newItem)
	}

	// Shrink: If len(value) < len(Items), remove ViewListItems
	for count < len(vl.Items) {
		lastIndex := len(vl.Items) - 1
		vl.destroyListItem(vl.Items[lastIndex])
		vl.Items = vl.Items[:lastIndex]
	}

	// Update: Iterate and update Item and Index for each ViewListItem
	for i := 0; i < count; i++ {
		vl.Items[i].Item = get(i)
		vl.Items[i].Index = i
	}
}

// createListItem creates a new ViewListItem.
func (vl *ViewList) createListItem() (*ViewListItem, error) {
	listItem := NewViewListItem(nil, vl, 0)

	if vl.itemType != "" && vl.session != nil {
		itemInstance, err := vl.session.CreateItemWrapper(vl.itemType, listItem)
		if err != nil {
			vl.session.Log(2, "ViewList: could not create %s instance: %v", vl.itemType, err)
		} else if itemInstance != nil {
			listItem.Item = itemInstance
		}
	}

	if vl.session != nil {
		listItem.List.Tracker().ToValueJSON(listItem)
		objID := listItem.GetObjID()
		vl.session.Log(2, "ViewList: created ViewListItem %d", objID)
		vl.session.Log(4, "ViewListItem created: objID=%d listVarID=%d", objID, vl.variable.GetID())
	}

	return listItem, nil
}

// destroyListItem cleans up a ViewListItem.
func (vl *ViewList) destroyListItem(listItem *ViewListItem) {
	if vl.session != nil {
		vl.session.Log(2, "ViewList: destroying ViewListItem %d", listItem.GetObjID())
	}
}

// Destroy cleans up all ViewListItems when the variable is destroyed.
func (vl *ViewList) Destroy() error {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	for _, listItem := range vl.Items {
		vl.destroyListItem(listItem)
	}
	vl.Items = nil
	vl.value = nil

	return nil
}

// RemoveAt removes the item at the given index from the source array.
func (vl *ViewList) RemoveAt(index int) error {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	count := 0
	if vl.value != nil {
		count = reflect.ValueOf(vl.value).Len()
	}

	if index < 0 || index >= count {
		return fmt.Errorf("index %d out of range", index)
	}

	// TODO: This needs to notify the backend to actually remove the item
	if vl.session != nil {
		vl.session.Log(1, "ViewList: RemoveAt(%d) called - needs backend integration", index)
	}

	return nil
}

// init auto-registers the ViewList wrapper when package is imported.
func init() {
	RegisterWrapperType("lua.ViewList", func(sess *LuaSession, variable WrapperVariable) interface{} {
		return NewViewList(sess, variable)
	})
}
