# ViewListItem

**Source Spec:** viewdefs.md

## Responsibilities

### Knows

- item: The actual backend object from the array (taken from variable's Value, not ValueJSON)
- list: Reference to the ViewList object that owns this item
- index: Position in the list (0-based)

### Does

- new(list, index): Constructor receives ViewList and index
- getItem: Return the domain object reference
- getList: Return the owning ViewList
- getIndex: Return position in list
- setIndex: Update position when list reorders
- remove: Convenience method to call `list:removeAt(index)`

## Collaborators

- ViewList: Creates and manages ViewListItem lifecycle, updates index on sync
- ObjectRegistry: Registers ViewListItem for path navigation
- LuaSession: Creates ViewListItem instances

## Notes

### ViewListItem Purpose

ViewListItem is an intermediate object that ViewList creates for each array element. It provides:

1. **Domain object access** via `item` - the actual backend object (not JSON)
2. **List context** via `list` - enables operations like `list:removeAt(index)`
3. **Position tracking** via `index` - allows index-based operations

### ViewListItem Structure

Each ViewListItem has three properties:
- `item`: The actual backend object from variable's Value array (not ValueJSON)
- `list`: Reference to the owning ViewList wrapper
- `index`: Position in the list (0-based)

```lua
local ViewListItem = {type = "ViewListItem"}
ViewListItem.__index = ViewListItem

function ViewListItem:new(list, index)
    local vli = {
        item = nil,  -- Set by ViewList during sync
        list = list,
        index = index
    }
    setmetatable(vli, self)
    return vli
end

function ViewListItem:remove()
    self.list:removeAt(self.index)
end
```

### ViewListItem vs ViewItem

**Note:** This class was renamed from ViewItem to ViewListItem to better reflect its purpose as an item within a ViewList wrapper. The spec uses "ViewListItem" consistently.

### ViewListItem Lifecycle

ViewListItems are managed by ViewList:

1. **Creation**: ViewList creates ViewListItem when array grows
2. **Update**: ViewList assigns `item` property during sync
3. **Reorder**: ViewList updates `index` property during sync
4. **Destruction**: ViewList removes ViewListItem when array shrinks

ViewListItems are NOT destroyed when the domain object changes - only when the array shrinks. This allows the same ViewListItem to track different domain objects over time.

### Custom ViewListItems

When `item=PresenterType` is specified on the ViewList variable, ViewList creates instances of that custom type instead of plain ViewListItems. The custom type must:

1. Accept `(list, index)` in its constructor
2. Have an `item` property that ViewList can assign

```lua
local ContactPresenter = {type = "ContactPresenter"}
ContactPresenter.__index = ContactPresenter

function ContactPresenter:new(list, index)
    local presenter = {
        item = nil,  -- Set by ViewList during sync
        list = list,
        index = index
    }
    setmetatable(presenter, self)
    return presenter
end

function ContactPresenter:delete()
    -- Can access domain object via self.item
    -- Can remove from list via self.list:removeAt(self.index)
    self.list:removeAt(self.index)
end
```

### ViewListItem Viewdef

ViewList uses the `list-item` namespace by default for rendering ViewListItems. The viewdef displays the `item` (domain object) with optional controls:

```html
<!-- ViewListItem.list-item viewdef -->
<template>
  <div style="display: flex; align-items: center;">
    <div ui-view="item" ui-namespace="list-item" style="flex: 1;"></div>
    <sl-icon-button name="x" ui-action="remove()"></sl-icon-button>
  </div>
</template>
```

This indirection allows:
1. ViewListItem to be the variable type bound to the ViewList
2. The nested view to render the domain object using its own viewdef
3. Each item to have a delete button that calls `remove()` on the ViewListItem

### Data Flow Example

```
Backend: contacts array in variable.Value: [Contact1, Contact2]  (actual objects)

ViewList wrapper (stands in for navigation):
  |- variable: Variable object
  |- items: [ViewListItem1, ViewListItem2]
  |- selectionIndex: 0

ViewListItem1:
  |- item: Contact1  (actual backend object)
  |- list: ViewList ref
  |- index: 0

ViewListItem2:
  |- item: Contact2  (actual backend object)
  |- list: ViewList ref
  |- index: 1

Frontend receives ViewListItem refs for rendering.
ViewListItem viewdef uses ui-view="item" to display the Contact.
Path navigation from ViewListItem.item accesses the actual Contact object.
```

## Sequences

- seq-viewlist-presenter-sync.md: ViewListItem creation and sync
- seq-wrapper-transform.md: ViewList returns ViewListItem refs
