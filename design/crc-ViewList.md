# ViewList

**Source Spec:** viewdefs.md, protocol.md, libraries.md

## Responsibilities

### Knows

**Frontend (DOM management):**
- element: Container DOM element for the list
- namespace: Viewdef namespace for child views (default: `list-item`)
- exemplar: Element to clone for each item (default: div)
- views: Parallel array of View elements
- delegate: Optional delegate for add/remove notifications

**Backend (Wrapper behavior):**
- variable: The Variable object (received in constructor, stored for later access)
- value: The array value (accessed via `variable:getValue()`)
- items: Array of ViewListItem objects (one per array element)
- selectionIndex: Current selection index for frontend use (default: 0, or -1 for no selection)
- itemType: Optional custom ViewListItem type name (from variable's `item` property)

### Does

**Frontend (DOM management):**
- create: Initialize from element with ui-viewlist attribute
- setExemplar: Set element to clone for list items (e.g., sl-option)
- update: Sync views array with bound variable array
- addItem: Clone exemplar, create variable, render and append
- removeItem: Destroy variable, remove element from DOM
- reorder: Reorder view elements to match array order
- clear: Remove all items
- setDelegate: Set delegate for notifications
- notifyAdd: Notify delegate of item addition
- notifyRemove: Notify delegate of item removal
- parsePathProperties: Extract wrapper and item properties from path

**Backend (Wrapper behavior):**
- new(variable): Constructor receives Variable, returns new or existing wrapper
- sync: Sync ViewListItems with array on wrapper reuse
- removeAt: Remove item at index (called by ViewListItem.remove())
- destroy: Clean up all ViewListItems when variable destroyed

## Collaborators

- View: Individual view elements in the list (frontend)
- ViewRenderer: Creates ViewLists (frontend)
- BindingEngine: Binds list to variable, parses path properties (frontend)
- Variable: Provides getValue() and getWrapper() for wrapper reuse
- Resolver: Calls CreateWrapper(variable) on value changes
- ViewListItem: Created for each array element, holds item/list/index
- ObjectRegistry: Registers ViewList and ViewListItems for path navigation
- LuaSession: Creates ViewListItem instances (backend)

## Notes

### ViewList as Wrapper

ViewList operates in two contexts:

1. **Frontend**: Manages DOM elements for rendering array items
2. **Backend**: Acts as a wrapper that stands in for the variable's value

When `ui-viewlist="contacts?item=ContactPresenter"` is used:
1. Frontend creates variable with `wrapper=ViewList` and `item=ContactPresenter` properties
2. `Resolver.CreateWrapper(variable)` calls `ViewList:new(variable)`
3. ViewList stores `variable` property (accesses array via `variable:getValue()`)
4. ViewList reads `item` property from variable for custom ViewListItem type
5. ViewList is registered in object registry (stands in for child path navigation)
6. ViewList maintains `items` array of ViewListItem objects
7. ViewList maintains `selectionIndex` for frontend selection state

### Wrapper Reuse and Sync

When the bound array changes, `CreateWrapper` is called again. ViewList returns the existing wrapper and syncs its ViewListItems:

1. **Assign items**: For each item in the array, assign it to the corresponding ViewListItem's `item` property
2. **Trim excess**: If the ViewListItem array is longer than the item array, remove excess ViewListItems
3. **Add new**: If the ViewListItem array is shorter, create new ViewListItems for additional items

This preserves internal state (like `selectionIndex`) while keeping ViewListItems in sync with the array.

```lua
function ViewList:new(variable)
    local existing = variable:getWrapper()
    if existing then
        existing.value = variable:getValue()
        existing:sync()  -- Sync ViewListItems with new array
        return existing
    end

    local wrapper = {
        variable = variable,
        value = variable:getValue(),
        items = {},
        selectionIndex = 0
    }
    setmetatable(wrapper, self)
    wrapper:sync()  -- Initial sync
    return wrapper
end

function ViewList:sync()
    local array = self.value or {}
    -- 1. Assign items to existing ViewListItems
    for i, item in ipairs(array) do
        if self.items[i] then
            self.items[i].item = item
            self.items[i].index = i - 1  -- 0-based
        end
    end
    -- 2. Trim excess ViewListItems
    while #self.items > #array do
        table.remove(self.items)
    end
    -- 3. Add new ViewListItems
    while #self.items < #array do
        local idx = #self.items
        local viewListItem = ViewListItem:new(self, idx)
        viewListItem.item = array[idx + 1]
        table.insert(self.items, viewListItem)
    end
end
```

### Path Property Syntax

ViewList configuration via path properties:
- `contacts` - Basic ViewList with default ViewListItem
- `contacts?item=ContactPresenter` - Custom ViewListItem type for each item

### ViewListItem Objects

ViewList creates a ViewListItem object for each array element. Each ViewListItem has:
- `item`: The actual backend object from the array (from variable's Value)
- `list`: Reference to the ViewList object
- `index`: Position in the list (0-based)

See crc-ViewListItem.md for full documentation.

### Custom ViewListItems

When `item=PresenterType` is specified in path properties, ViewList creates instances of that type instead of plain ViewListItems. The custom type is constructed with the ViewList and index: `PresenterType:new(viewList, index)`.

Custom ViewListItems can have UI-specific methods like `delete()` that can:
- Access the domain object via `self.item`
- Remove itself via `self.list:removeAt(self.index)`

## Sequences

- seq-viewlist-update.md: Array change handling (frontend DOM sync)
- seq-render-view.md: ViewList rendering within views
- seq-wrapper-transform.md: ViewList wrapper creation and reuse (backend)
- seq-viewlist-presenter-sync.md: ViewListItem sync on array changes
