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

// ViewListWrapper transforms an array of domain object refs into ViewItem refs.
// It creates ViewItem objects for each item in the source array.
//
// The wrapper constructor receives the variable, allowing access to properties
// like "item" (ItemWrapper type name). The wrapper instance is stored internally
// in the variable.
//
// ViewList uses the "list-item" namespace by default for its ViewItems.
type ViewListWrapper struct {
	runtime      *Runtime
	variable     WrapperVariable       // The variable being wrapped (for property access)
	itemType     string                // ItemWrapper type name from "item" property
	viewItems    map[int64]*ViewItem   // sourceObjID -> ViewItem
	order        []int64               // ordered list of source obj IDs
	nextObjID    int64                 // counter for generating ViewItem object IDs
	mu           sync.RWMutex
}

// NewViewListWrapper creates a new ViewList wrapper for a variable.
// The constructor receives the variable to access properties like "item".
func NewViewListWrapper(runtime *Runtime, variable WrapperVariable) Wrapper {
	// Read item type from variable properties during construction
	itemType := variable.GetProperty("item")

	if runtime != nil && runtime.verbosity >= 2 {
		log.Printf("[v2] ViewListWrapper: created for variable %d with item type %q", variable.GetID(), itemType)
	}

	return &ViewListWrapper{
		runtime:   runtime,
		variable:  variable,
		itemType:  itemType,
		viewItems: make(map[int64]*ViewItem),
		order:     make([]int64, 0),
		nextObjID: -1, // Start negative IDs for UI server managed objects
	}
}

// ComputeValue transforms the raw array value into an array of ViewItem refs.
// Called when the monitored value changes.
func (w *ViewListWrapper) ComputeValue(rawValue json.RawMessage) (json.RawMessage, error) {
	if len(rawValue) == 0 {
		return json.Marshal([]interface{}{})
	}

	// Parse source array
	var sourceRefs []map[string]interface{}
	if err := json.Unmarshal(rawValue, &sourceRefs); err != nil {
		// Not an array or invalid format - return as-is
		return rawValue, nil
	}

	// Extract source object IDs
	sourceIDs := make([]int64, 0, len(sourceRefs))
	for _, ref := range sourceRefs {
		if objID, ok := ref["obj"].(float64); ok {
			sourceIDs = append(sourceIDs, int64(objID))
		}
	}

	// Sync ViewItems with source array
	if err := w.syncViewItems(sourceIDs); err != nil {
		return nil, err
	}

	// Build stored value as array of ViewItem refs
	storedValue := make([]map[string]int64, 0, len(w.order))
	w.mu.RLock()
	for _, sourceID := range w.order {
		if viewItem, ok := w.viewItems[sourceID]; ok {
			storedValue = append(storedValue, map[string]int64{"obj": viewItem.ObjID})
		}
	}
	w.mu.RUnlock()

	return json.Marshal(storedValue)
}

// syncViewItems creates/removes/reorders ViewItems to match source array.
func (w *ViewListWrapper) syncViewItems(sourceIDs []int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Build set of current source IDs
	newSourceSet := make(map[int64]int) // sourceID -> index
	for i, id := range sourceIDs {
		newSourceSet[id] = i
	}

	// Remove ViewItems for items no longer in source
	for sourceID, viewItem := range w.viewItems {
		if _, exists := newSourceSet[sourceID]; !exists {
			w.destroyViewItem(viewItem)
			delete(w.viewItems, sourceID)
		}
	}

	// Add ViewItems for new items, update indices for existing
	for i, sourceID := range sourceIDs {
		if viewItem, exists := w.viewItems[sourceID]; exists {
			// Update index if changed
			if viewItem.Index != i {
				viewItem.SetIndex(i)
			}
		} else {
			// Create new ViewItem
			viewItem, err := w.createViewItem(sourceID, i)
			if err != nil {
				if w.runtime != nil && w.runtime.verbosity >= 1 {
					log.Printf("[v1] ViewListWrapper: failed to create ViewItem: %v", err)
				}
				continue
			}
			w.viewItems[sourceID] = viewItem
		}
	}

	// Update order
	w.order = sourceIDs

	return nil
}

// createViewItem creates a ViewItem for a source domain object.
func (w *ViewListWrapper) createViewItem(sourceID int64, index int) (*ViewItem, error) {
	// Generate a negative object ID for UI server managed objects
	objID := w.nextObjID
	w.nextObjID--

	viewItem := NewViewItem(objID, sourceID, w, index)

	// If item type specified, create ItemWrapper and set on viewItem.item
	if w.itemType != "" && w.runtime != nil {
		// ItemWrapper is constructed with the ViewItem: ItemWrapper(viewItem)
		itemInstance, err := w.runtime.CreateItemWrapper(w.itemType, viewItem)
		if err != nil {
			// Log but continue - ViewItem can work without ItemWrapper
			if w.runtime.verbosity >= 2 {
				log.Printf("[v2] ViewListWrapper: could not create %s instance: %v", w.itemType, err)
			}
		} else if itemInstance != nil {
			// Set the wrapped item reference on the ViewItem
			viewItem.SetItem(itemInstance.ObjRef())
		}
	}

	if w.runtime != nil && w.runtime.verbosity >= 2 {
		log.Printf("[v2] ViewListWrapper: created ViewItem %d for source %d at index %d", objID, sourceID, index)
	}

	return viewItem, nil
}

// destroyViewItem cleans up a ViewItem.
func (w *ViewListWrapper) destroyViewItem(viewItem *ViewItem) {
	if w.runtime != nil && w.runtime.verbosity >= 2 {
		log.Printf("[v2] ViewListWrapper: destroying ViewItem %d for source %d", viewItem.ObjID, viewItem.GetBaseItemID())
	}
	// Cleanup would happen here if needed
}

// Destroy cleans up all ViewItems when the variable is destroyed.
func (w *ViewListWrapper) Destroy() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for sourceID, viewItem := range w.viewItems {
		w.destroyViewItem(viewItem)
		delete(w.viewItems, sourceID)
	}
	w.order = nil

	return nil
}

// RemoveAt removes the item at the given index from the source array.
// This is called by ViewItem.Remove() for delete actions.
func (w *ViewListWrapper) RemoveAt(index int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if index < 0 || index >= len(w.order) {
		return fmt.Errorf("index %d out of range", index)
	}

	// TODO: This needs to notify the backend to actually remove the item
	// from the source array. For now, just log.
	if w.runtime != nil && w.runtime.verbosity >= 1 {
		log.Printf("[v1] ViewListWrapper: RemoveAt(%d) called - needs backend integration", index)
	}

	return nil
}

// GetViewItemForSource returns the ViewItem for a source object ID.
func (w *ViewListWrapper) GetViewItemForSource(sourceID int64) (*ViewItem, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	viewItem, ok := w.viewItems[sourceID]
	return viewItem, ok
}

// GetAllViewItems returns all ViewItems in order.
func (w *ViewListWrapper) GetAllViewItems() []*ViewItem {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make([]*ViewItem, 0, len(w.order))
	for _, sourceID := range w.order {
		if viewItem, ok := w.viewItems[sourceID]; ok {
			result = append(result, viewItem)
		}
	}
	return result
}

// init auto-registers the ViewList wrapper when package is imported.
// This enables frictionless development - just use wrapper=ViewList in a path.
func init() {
	RegisterWrapperType("ViewList", func(runtime *Runtime, variable WrapperVariable) Wrapper {
		return NewViewListWrapper(runtime, variable)
	})
}

// Ensure ViewListWrapper implements Wrapper interface.
var _ Wrapper = (*ViewListWrapper)(nil)

// PresenterRef creates an object reference JSON for a presenter.
func PresenterRef(objID int64) json.RawMessage {
	data, _ := json.Marshal(map[string]int64{"obj": objID})
	return data
}

// ParseObjectRef extracts the object ID from a JSON object reference.
func ParseObjectRef(data json.RawMessage) (int64, error) {
	var ref map[string]interface{}
	if err := json.Unmarshal(data, &ref); err != nil {
		return 0, err
	}
	if obj, ok := ref["obj"].(float64); ok {
		return int64(obj), nil
	}
	return 0, fmt.Errorf("not an object reference")
}
