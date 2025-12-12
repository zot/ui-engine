# ViewItem

**Source Spec:** viewdefs.md

## Responsibilities

### Knows

- baseItem: Reference to the domain object (`{obj: ID}`)
- item: Either same as baseItem, or `ItemWrapper(viewItem)` result if `item` property set
- list: Reference to the ViewList object that owns this item
- index: Position in the list (0-based)

### Does

- getBaseItem: Return the domain object reference
- getItem: Return the (possibly wrapped) item for rendering
- getList: Return the owning ViewList
- getIndex: Return position in list
- setIndex: Update position when list reorders
- remove: Convenience method to call `list.removeAt(index)`
- wrapItem: Create `ItemWrapper(viewItem)` if item property is set

## Collaborators

- ViewList: Creates and manages ViewItem lifecycle, updates index on reorder
- ItemWrapper: Optional wrapper type for presenter creation (e.g., ContactPresenter)
- LuaRuntime: Creates ItemWrapper instances via CreateInstance
- VariableStore: Stores ViewItem as a managed object

## Notes

### ViewItem Purpose

ViewItem is an intermediate object that ViewList creates for each array element. It provides:

1. **Domain object access** via `baseItem` - the original `{obj: ID}` reference
2. **Presenter access** via `item` - either the same as baseItem, or a wrapped presenter
3. **List context** via `list` - enables operations like `list.removeAt(index)`
4. **Position tracking** via `index` - allows index-based operations

### Item Wrapping Flow

When ViewList has `item=PresenterType` property:

1. ViewList creates ViewItem for each domain object
2. ViewItem.baseItem = domain object ref
3. ItemWrapper constructor called: `ItemWrapper(viewItem)`
4. ViewItem.item = wrapped presenter ref

The ItemWrapper receives the ViewItem, so the presenter can:
- Access the domain object via `viewItem.baseItem`
- Remove itself via `viewItem.list.removeAt(viewItem.index)`

### Presenter Access Pattern

A presenter created via item wrapping can:
```lua
function ContactPresenter:delete()
    -- Access domain object via viewItem.baseItem
    local contact = self.viewItem.baseItem
    -- Remove from list via viewItem.list.removeAt()
    self.viewItem.list:removeAt(self.viewItem.index)
end
```

### ViewItem viewdef

ViewList uses the `list-item` namespace by default for its ViewItems. The `list-item` viewdef displays the item with a delete button:

```html
<!-- ViewItem.list-item viewdef -->
<template>
  <div style="display: flex; align-items: center;">
    <div ui-view="item" ui-namespace="list-item" style="flex: 1;"></div>
    <sl-icon-button name="x" ui-action="remove()"></sl-icon-button>
  </div>
</template>
```

This indirection allows:
1. ViewItem to be the variable type bound to the ViewList
2. The nested view to also use `list-item` namespace for consistent rendering
3. Each item to have a delete button that calls `remove()` on the ViewItem

### Data Flow Example

```
Backend: contacts: [{obj: 101}, {obj: 102}]  (Contact refs)

ViewList.computeValue(rawArray) returns:
  [{obj: VI1}, {obj: VI2}]  (ViewItem refs)

Each ViewItem:
  |- baseItem: {obj: 101}    (Contact)
  |- item: {obj: P1}         (ContactPresenter, if item=ContactPresenter)
  |- list: ViewList ref
  |- index: 0

ContactPresenter (created via ItemWrapper(viewItem)):
  - Can access viewItem.baseItem (Contact)
  - Can call viewItem.list.removeAt(viewItem.index) for delete
```

## Sequences

- seq-viewlist-presenter-sync.md: ViewItem creation and wrapping
- seq-wrapper-transform.md: ViewList returns ViewItem refs
