# ViewList

**Source Spec:** viewdefs.md, protocol.md, libraries.md

## Responsibilities

### Knows

**Frontend (DOM management):**
- elementId: ID of container element for the list (NOT direct DOM reference)
- exemplarHtml: HTML string for cloning items (default: `<div></div>`)
- viewIds: Parallel array of View element IDs (NOT direct element references)
- delegate: Optional delegate for add/remove notifications

**Backend (Wrapper behavior):**
- variable: The Variable object (received in constructor, stored for later access)
- value: The array value (accessed via `variable:getValue()`)
- items: Array of ViewListItem objects (one per array element)
- selectionIndex: Current selection index for frontend use (default: 0, or -1 for no selection)
- itemType: Optional custom ViewListItem type name (from variable's `item` property)

### Does

**Frontend (DOM management):**
- create: Initialize from element with ui-viewlist attribute, vend element ID if needed
- setExemplarHtml: Set HTML string for cloning items (e.g., `<sl-option></sl-option>`)
- getElement: Look up DOM element by elementId (via document.getElementById)
- update: Sync viewIds array with bound variable array
- addItem: Clone exemplar HTML, create variable with inherited namespace properties, render and append
- removeItem: Destroy variable, remove element from DOM by ID
- reorder: Reorder view elements to match array order
- clear: Remove all items
- setDelegate: Set delegate for notifications
- notifyAdd: Notify delegate of item addition
- notifyRemove: Notify delegate of item removal
- parsePathProperties: Extract wrapper and item properties from path
- inheritNamespaceProperties: Copy namespace and fallbackNamespace from ViewList variable to exemplar variable

**Backend (Wrapper behavior):**
- new(variable): Constructor receives Variable, sets fallbackNamespace property, returns new or existing wrapper
- sync: Sync ViewListItems with array on wrapper reuse
- removeAt: Remove item at index (called by ViewListItem.remove())
- destroy: Clean up all ViewListItems when variable destroyed
- setFallbackNamespace: Set `fallbackNamespace: "list-item"` on the variable

## Collaborators

- ElementIdVendor: Vends unique element ID if element lacks one
- View: Individual view elements in the list (frontend)
- ViewRenderer: Creates ViewLists (frontend)
- BindingEngine: Binds list to variable, parses path properties (frontend)
- Variable: Provides getValue() and getWrapper() for wrapper reuse
- Resolver: Calls CreateWrapper(variable) on value changes
- ViewListItem: Created for each array element, holds item/list/index
- ObjectRegistry: Registers ViewList and ViewListItems for path navigation
- LuaSession: Creates ViewListItem instances (backend)

## Notes

### Default Access Property

The `ui-viewlist` binding automatically adds `access=r` (read-only) if no `access` property is specified. ViewLists are typically read-only bindings that display array contents.

### ViewList as Wrapper

ViewList operates in two contexts:

1. **Frontend**: Manages DOM elements for rendering array items
2. **Backend**: Acts as a wrapper that stands in for the variable's value

When `ui-viewlist="contacts?item=ContactPresenter"` is used:
1. Frontend creates variable with `wrapper=ViewList` and `item=ContactPresenter` properties
2. `Resolver.CreateWrapper(variable)` calls `ViewList:new(variable)`
3. ViewList stores `variable` property (accesses array via `variable:getValue()`)
4. **ViewList sets `fallbackNamespace: "list-item"` on the variable**
5. ViewList reads `item` property from variable for custom ViewListItem type
6. ViewList is registered in object registry (stands in for child path navigation)
7. ViewList maintains `items` array of ViewListItem objects
8. ViewList maintains `selectionIndex` for frontend selection state

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

### Exemplar Namespace Inheritance

ViewList exemplars follow standard namespace inheritance rules for views:
- The exemplar's variable inherits `namespace` from the ViewList's variable (unless the exemplar specifies `ui-namespace`)
- The exemplar's variable inherits `fallbackNamespace` from the ViewList's variable

This allows a ViewList with `ui-namespace="COMPACT"` to render all its items using `TYPE.COMPACT` viewdefs without requiring each exemplar to specify the namespace.

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
