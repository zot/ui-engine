# ViewList

**Source Spec:** viewdefs.md, protocol.md

## Responsibilities

### Knows

**Frontend (DOM management):**
- element: Container DOM element for the list
- namespace: Viewdef namespace for child views (default: `list-item`)
- exemplar: Element to clone for each item (default: div)
- views: Parallel array of View elements
- delegate: Optional delegate for add/remove notifications

**Backend (Wrapper behavior):**
- variable: The variable being wrapped (received in constructor)
- itemType: Presenter type name for wrapping items (from variable's `item` property)
- viewItems: Parallel array of ViewItem objects (one per source item)

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
- Constructor: Receive variable, read `item` property for presenter type
- computeValue: Transform raw array into array of ViewItem refs
- syncViewItems: Create/remove ViewItem objects to match source array
- createViewItem: Create ViewItem for a domain item, optionally wrap with ItemWrapper
- destroyViewItem: Clean up ViewItem and its wrapped presenter (if any)
- removeAt: Remove item at index (called by presenter.delete())
- destroy: Clean up all ViewItems when variable destroyed

## Collaborators

- View: Individual view elements in the list (frontend)
- ViewRenderer: Creates ViewLists (frontend)
- BindingEngine: Binds list to variable, parses path properties (frontend)
- Variable: Stores wrapper internally, calls computeValue on changes
- ViewItem: Created for each domain object, holds baseItem/item/list/index
- ItemWrapper: Optional presenter type that wraps ViewItem
- LuaRuntime: Creates ViewItem and ItemWrapper instances via CreateInstance (backend)
- VariableStore: Stores ViewItem and presenter objects

## Notes

### ViewList as Wrapper

ViewList operates in two contexts:

1. **Frontend**: Manages DOM elements for rendering array items
2. **Backend**: Acts as a wrapper that transforms domain objects to presenters

When `ui-viewlist="contacts?item=ContactPresenter"` is used:
1. Frontend creates variable with `wrapper=ViewList` and `item=ContactPresenter` properties
2. Backend instantiates ViewList wrapper: `ViewList(variable)`
3. ViewList reads `item` property from variable to get presenter type
4. Wrapper is stored internally in the variable
5. When monitored value changes, `viewList.computeValue(rawArray)` is called
6. For each domain item, ViewList creates a ViewItem object with:
   - `baseItem`: Reference to the domain object (`{obj: ID}`)
   - `item`: Either same as baseItem, or `ItemWrapper(viewItem)` if item property set
   - `list`: Reference to the ViewList object
   - `index`: Position in the list (0-based)
7. computeValue returns ViewItem refs as stored value

### Path Property Syntax

ViewList configuration via path properties:
- `contacts` - Basic ViewList with no item presenter
- `contacts?item=ContactPresenter` - Each item wrapped in ContactPresenter
- `contacts?item=ContactPresenter&editable=true` - Additional presenter properties

### ViewItem Lifecycle

ViewList manages ViewItem lifecycle:
- Creates ViewItem when item added to source array
- Destroys ViewItem (and its presenter if any) when item removed from source array
- Updates ViewItem index when array reordered

### Item Wrapping

When `item=PresenterType` is specified, each ViewItem's `item` property holds a wrapped presenter:
- `ItemWrapper(viewItem)` is called, receiving the ViewItem
- Presenter can access `viewItem.baseItem` for the domain object
- Presenter can call `viewItem.list.removeAt(viewItem.index)` for self-removal

## Sequences

- seq-viewlist-update.md: Array change handling (frontend DOM sync)
- seq-render-view.md: ViewList rendering within views
- seq-wrapper-transform.md: ViewList wrapper transforms value (backend)
- seq-viewlist-presenter-sync.md: Presenter creation/destruction on array changes
