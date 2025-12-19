// Package lua provides ViewListItem type for ViewList array elements.
// CRC: crc-ViewListItem.md
// Spec: viewdefs.md
// Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
package lua

import (
	"sync"
)

// ViewListItem represents an element in a ViewList.
// It provides domain object access (Item), presenter access (Item),
// list context (List), and position tracking (Index).
type ViewListItem struct {
	Item  interface{} // Domain object reference
	List  *ViewList
	Index int
	mu    sync.RWMutex
}

// NewViewListItem creates a new ViewListItem for a domain object.
func NewViewListItem(item interface{}, list *ViewList, index int) *ViewListItem {
	if list != nil && list.session != nil {
		list.session.Log(4, "NewViewListItem: index=%d", index)
	}
	return &ViewListItem{
		Item:  item,
		List:  list,
		Index: index,
	}
}

func (vli *ViewListItem) GetObjID() int64 {
	objID, _ := vli.List.Tracker().LookupObject(vli)
	return objID
}

// GetItem returns the (possibly wrapped) item reference.
func (vli *ViewListItem) GetItem() interface{} {
	vli.mu.RLock()
	defer vli.mu.RUnlock()
	return vli.Item
}

// GetList returns the owning ViewList.
func (vli *ViewListItem) GetList() *ViewList {
	vli.mu.RLock()
	defer vli.mu.RUnlock()
	return vli.List
}

// GetIndex returns the position in the list.
func (vli *ViewListItem) GetIndex() int {
	vli.mu.RLock()
	defer vli.mu.RUnlock()
	return vli.Index
}

// SetIndex updates the position (called when list reorders).
func (vli *ViewListItem) SetIndex(index int) {
	vli.mu.Lock()
	defer vli.mu.Unlock()
	vli.Index = index
}

// Remove removes this item from the list via list.RemoveAt(index).
// This is a convenience method for presenter delete actions.
func (vli *ViewListItem) Remove() error {
	vli.mu.RLock()
	list := vli.List
	index := vli.Index
	vli.mu.RUnlock()

	if list != nil {
		return list.RemoveAt(index)
	}
	return nil
}
