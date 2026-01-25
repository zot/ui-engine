# ViewList

**Source Spec:** viewdefs.md, protocol.md, libraries.md

## Responsibilities

### Knows

**Frontend (DOM management):**
- elementId: ID of container element for the list (NOT direct DOM reference)
- exemplarHtml: HTML string for cloning items (default: `<div></div>`)
- itemViews: Array of View instances for child items (enables multi-element cleanup)
- delegate: Optional delegate for add/remove notifications

**Backend (Wrapper behavior):**
- variable: The Variable object (received in constructor, stored for later access)
- value: The array value (accessed via `variable:getValue()`)
- items: Array of ViewListItem objects (one per array element)
- selectionIndex: Current selection index for frontend use (default: 0, or -1 for no selection)
- itemType: Optional custom ViewListItem type name (from variable's `itemWrapper` property)

### Does

**Frontend (DOM management):**
- create: Initialize from element with ui-viewlist attribute, vend element ID if needed, register widget
- setExemplarHtml: Set HTML string for cloning items (e.g., `<sl-option></sl-option>`)
- getElement: Look up DOM element by elementId (via document.getElementById)
- update: Sync itemViews array with bound variable array
- addItem: Clone exemplar HTML, create variable with inherited namespace properties, render and append
- removeItem: Destroy variable, remove element from DOM by ID
- reorder: Reorder view elements to match array order
- clear: Remove all items
- destroy: Cleanup viewlist - unwatch, clear items, destroy associated variable
- setDelegate: Set delegate for notifications
- notifyAdd: Notify delegate of item addition
- notifyRemove: Notify delegate of item removal
- parsePathProperties: Extract wrapper and item properties from path
- inheritNamespaceProperties: Copy namespace and fallbackNamespace from ViewList variable to exemplar variable
- notifyParentRendered: After adding items, add parent variable ID to BindingEngine's pendingScrollNotifications set

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

When `ui-viewlist="contacts?itemWrapper=ContactPresenter"` is used:
1. Frontend creates variable with `wrapper=lua.ViewList` and `itemWrapper=ContactPresenter` properties
2. `Resolver.CreateWrapper(variable)` calls `ViewList:new(variable)`
3. ViewList stores `variable` property (accesses array via `variable:getValue()`)
4. **ViewList sets `fallbackNamespace: "list-item"` on the variable**
5. ViewList reads `itemWrapper` property from variable for custom ViewListItem type
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
- `contacts?itemWrapper=ContactPresenter` - Custom ViewListItem type for each item
- `messages?scrollOnOutput` - Auto-scroll when items are added

### Widget Registration

ViewLists register themselves with the BindingEngine's widgets map. This enables:
- Consistent cleanup and lifecycle management
- `scrollOnOutput` support (set on the widget, not the ViewList)

When `scrollOnOutput` is specified in the path (e.g., `ui-viewlist="messages?scrollOnOutput"`), it is set on the element's widget, not on the ViewList itself. See crc-Widget.md for details.

### Render Notifications

When ViewList items render, they notify their parent so ancestor widgets with `scrollOnOutput` can scroll:

1. After adding items, call `notifyParentRendered()` which adds the parent variable ID to BindingEngine's `pendingScrollNotifications` set
2. The BindingEngine processes these notifications after the batch completes (see crc-BindingEngine.md)
3. If an ancestor widget has `scrollOnOutput`, it scrolls to bottom

This batched approach ensures multiple item additions cause only one scroll.

### ViewListItem Objects

ViewList creates a ViewListItem object for each array element. Each ViewListItem has:
- `item`: The actual backend object from the array (from variable's Value)
- `list`: Reference to the ViewList object
- `index`: Position in the list (0-based)

See crc-ViewListItem.md for full documentation.

### Custom ViewListItems

When `itemWrapper=PresenterType` is specified in path properties, ViewList creates instances of that type instead of plain ViewListItems. The custom type is constructed with the ViewList and index: `PresenterType:new(viewList, index)`.

Custom ViewListItems can have UI-specific methods like `delete()` that can:
- Access the domain object via `self.item`
- Remove itself via `self.list:removeAt(self.index)`

## Variable Destruction

When a ViewList is destroyed, it must destroy its associated variable:
1. The variable was created when the ViewList was set up (via `setupViewList`)
2. `destroy()` calls `VariableStore.destroy(varId)` to notify the backend
3. Backend destruction is recursive - destroys all child variables (including item views)
4. This prevents variable leaks during hot-reload re-render cycles

## Sequences

- seq-viewlist-update.md: Array change handling (frontend DOM sync)
- seq-render-view.md: ViewList rendering within views
- seq-wrapper-transform.md: ViewList wrapper creation and reuse (backend)
- seq-viewlist-presenter-sync.md: ViewListItem sync on array changes
