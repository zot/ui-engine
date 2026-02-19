// Package lua provides ViewListItem type for ViewList array elements.
// CRC: crc-ViewListItem.md
// Spec: viewdefs.md
// Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
package lua

import (
	"reflect"
	"sync"
)

// ViewListItem represents an element in a ViewList.
// It provides domain object access (Item), presenter access (Item),
// list context (List), and position tracking (Index).
type ViewListItem struct {
	Item     any // BaseItem or a wrapper, if ViewList.itemType is set
	BaseItem any // Domain object reference
	List     *ViewList
	Index    int
	mu       sync.RWMutex
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

func (vli *ViewListItem) GetItemObjID() int64 {
	objID, _ := vli.List.Tracker().LookupObject(vli.Item)
	return objID
}

// GetItem returns the (possibly wrapped) item reference.
func (vli *ViewListItem) GetItem() interface{} {
	vli.mu.RLock()
	defer vli.mu.RUnlock()
	return vli.Item
}

// GetItem returns the (possibly wrapped) item reference.
func (vli *ViewListItem) GetBaseItem() interface{} {
	vli.mu.RLock()
	defer vli.mu.RUnlock()
	return vli.BaseItem
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

// init auto-registers the ViewList wrapper when package is imported.
func init() {
	RegisterCreateFactory("lua.ViewListItem", reflect.TypeFor[ViewListItem](), func(sess *LuaSession, value any) interface{} {
		//return NewViewList(sess, variable)
		// can't create these from the front end
		return nil
	})
}
