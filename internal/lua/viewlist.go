// Package lua provides ViewList wrapper for array value transformation.
// CRC: crc-ViewList.md, crc-Wrapper.md
// Spec: viewdefs.md, protocol.md
// Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
package lua

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// ViewList transforms an array of domain object refs into ViewListItem refs.
// It creates ViewListItem objects for each item in the source array.
type ViewList struct {
	runtime        *Runtime
	variable       WrapperVariable       // The variable being wrapped (for property access)
	value          []interface{}         // The raw array of domain objects
	Items          []*ViewListItem       // The list of ViewListItem objects
	SelectionIndex int                   // The current selection index
	itemType       string                // ItemWrapper type name from "item" property
	nextObjID      int64                 // counter for generating ViewListItem object IDs
	mu             sync.RWMutex
}

// NewViewList creates a new ViewList wrapper for a variable.
func NewViewList(runtime *Runtime, variable WrapperVariable) interface{} {
	itemType := variable.GetProperty("item")

	if runtime != nil && runtime.verbosity >= 2 {
		log.Printf("[v2] ViewList: created for variable %d with item type %q", variable.GetID(), itemType)
	}

	vl := &ViewList{
		runtime:        runtime,
		variable:       variable,
		itemType:       itemType,
		Items:          make([]*ViewListItem, 0),
		value:          make([]interface{}, 0),
		SelectionIndex: -1, // Default to no selection
		nextObjID:      -1, // Start negative IDs for UI server managed objects
	}

	// The initial value is now processed here instead of ComputeValue
	rawValue := variable.GetValue()
	if rawValue != nil {
		if rawBytes, ok := rawValue.([]byte); ok {
			if err := json.Unmarshal(rawBytes, &vl.value); err != nil {
				log.Printf("[v1] ViewList: could not unmarshal initial value: %v", err)
			}
		} else if rawJSON, ok := rawValue.(json.RawMessage); ok {
			if err := json.Unmarshal(rawJSON, &vl.value); err != nil {
				log.Printf("[v1] ViewList: could not unmarshal initial value: %v", err)
			}
		}
	}

	vl.SyncViewItems()
	return vl
}

// SyncViewItems synchronizes the `Items` slice with the `value` slice.
func (vl *ViewList) SyncViewItems() {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	// Grow: If len(value) > len(Items), append new ViewListItems
	for len(vl.value) > len(vl.Items) {
		newItem, err := vl.createListItem()
		if err != nil {
			log.Printf("[v1] ViewList: failed to create ViewListItem: %v", err)
			break
		}
		vl.Items = append(vl.Items, newItem)
	}

	// Shrink: If len(value) < len(Items), remove ViewListItems
	for len(vl.value) < len(vl.Items) {
		lastIndex := len(vl.Items) - 1
		vl.destroyListItem(vl.Items[lastIndex])
		vl.Items = vl.Items[:lastIndex]
	}

	// Update: Iterate and update Item and Index for each ViewListItem
	for i, item := range vl.value {
		vl.Items[i].Item = item
		vl.Items[i].Index = i
	}
}

// createListItem creates a new ViewListItem.
func (vl *ViewList) createListItem() (*ViewListItem, error) {
	objID := vl.nextObjID
	vl.nextObjID--

	listItem := NewViewListItem(objID, nil, vl, 0)

	if vl.itemType != "" && vl.runtime != nil {
		itemInstance, err := vl.runtime.CreateItemWrapper(vl.itemType, listItem)
		if err != nil {
			if vl.runtime.verbosity >= 2 {
				log.Printf("[v2] ViewList: could not create %s instance: %v", vl.itemType, err)
			}
		} else if itemInstance != nil {
			listItem.Item = itemInstance
		}
	}

	if vl.runtime != nil && vl.runtime.verbosity >= 2 {
		log.Printf("[v2] ViewList: created ViewListItem %d", objID)
	}

	return listItem, nil
}

// destroyListItem cleans up a ViewListItem.
func (vl *ViewList) destroyListItem(listItem *ViewListItem) {
	if vl.runtime != nil && vl.runtime.verbosity >= 2 {
		log.Printf("[v2] ViewList: destroying ViewListItem %d", listItem.GetObjID())
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

	if index < 0 || index >= len(vl.value) {
		return fmt.Errorf("index %d out of range", index)
	}

	// TODO: This needs to notify the backend to actually remove the item
	if vl.runtime != nil && vl.runtime.verbosity >= 1 {
		log.Printf("[v1] ViewList: RemoveAt(%d) called - needs backend integration", index)
	}

	return nil
}

// init auto-registers the ViewList wrapper when package is imported.
func init() {
	RegisterWrapperType("ViewList", func(runtime *Runtime, variable WrapperVariable) interface{} {
		return NewViewList(runtime, variable)
	})
}
