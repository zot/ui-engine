// Package lua provides ViewList wrapper for array value transformation.
// CRC: crc-ViewList.md, crc-Wrapper.md
// Spec: viewdefs.md, protocol.md
// Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
package lua

import (
	"fmt"
	"sync"

	changetracker "github.com/zot/change-tracker"
)

// ViewList transforms an array of domain object refs into ViewListItem refs.
// It creates ViewListItem objects for each item in the source array.
type ViewList struct {
	session        *LuaSession
	variable       *TrackerVariableAdapter // The variable being wrapped (for property access)
	value          interface{}             // The raw array of domain objects (slice or array)
	Items          []*ViewListItem         // The actual list of ViewListItem objects
	SelectionIndex int                     // The current selection index
	itemType       string                  // ItemWrapper type name from "item" property
	nextObjID      int64                   // counter for generating ViewListItem object IDs
	mu             sync.RWMutex
}

// NewViewList creates a new ViewList wrapper for a variable.
func NewViewList(sess *LuaSession, variable *TrackerVariableAdapter) interface{} {
	var vl *ViewList
	var ok bool
	if vl, ok = variable.NavigationValue().(*ViewList); !ok {
		itemType := variable.Properties["itemWrapper"]
		if sess != nil {
			sess.Log(2, "ViewList: created for variable %d with item type %q", variable.ID, itemType)
			sess.Log(4, "ViewList created: varID=%d itemType=%q", variable.ID, itemType)
		}
		sess.Log(4, "CREATING LIST ON %#v", variable.NavigationValue())
		vl = &ViewList{
			session:        sess,
			variable:       variable,
			itemType:       itemType,
			Items:          make([]*ViewListItem, 0),
			value:          nil,
			SelectionIndex: -1, // Default to no selection
			nextObjID:      -1, // Start negative IDs for UI server managed objects
		}
		// Set fallbackNamespace for namespace resolution cascade (high priority)
		variable.SetProperty("fallbackNamespace:high", "list-item")
	}
	vl.Update(variable.NavigationValue())
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
// ArrayGetter in SyncViewItems handles both Go slices and Lua tables.
func (vl *ViewList) Update(newValue interface{}) {
	vl.mu.Lock()
	vl.session.Log(4, "ViewList\n  variable %d\n  value: %v", vl.variable.ID, vl.variable.Value)
	vl.value = newValue
	vl.mu.Unlock()

	// Sync items (acquires its own lock)
	vl.SyncViewItems()
}

// SyncViewItems synchronizes the `Items` slice with the `value` slice.
func (vl *ViewList) SyncViewItems() {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	get, count, err := vl.session.ArrayGetter(vl.value)

	vl.session.Log(4, "VIEWLIST GETTER ON VALUE %#v", vl.value)
	if err != nil {
		vl.session.Log(0, "Error synchronizing view list: %s", err.Error())
		get = func(i int) (any, error) { return nil, fmt.Errorf("No items") }
		count = 0
	}

	// Grow: If len(value) > len(Items), append new ViewListItems
	for count > len(vl.Items) {
		newItem := NewViewListItem(nil, vl, 0)
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
	for i, view := range vl.Items {
		if item, err := get(i); err != nil {
			vl.session.Log(0, "Error synchronizing item %d of view list for %#v", vl.value)
		} else if view.BaseItem != item || view.Index != i {
			vl.session.TriggerBatch()
			vl.session.Log(4, "VIEWLIST VIEW %d CHANGED: %#v", i, item)
			view.Index = i
			view.BaseItem = item
			vl.session.Log(4, "NEW ITEM %v", item)
			if vl.itemType != "" {
				if wrapper := vl.session.GetTracker().Resolver.CreateValue(nil, vl.itemType, view); wrapper == nil {
					vl.session.Log(0, "Error, ViewList could not create instance of %s", vl.itemType)
				} else {
					vl.session.Log(4, "CREATED VIEWLIST %s WRAPPER", vl.itemType)
					view.Item = wrapper
					continue
				}
			}
			view.Item = item
		} else {
			vl.session.Log(4, "VIEWLIST VIEW %d DID NOT CHANGE", i, item)
		}
	}
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

// init auto-registers the ViewList wrapper when package is imported.
func init() {
	RegisterWrapperType("lua.ViewList", func(sess *LuaSession, variable *TrackerVariableAdapter) interface{} {
		return NewViewList(sess, variable)
	})
}
