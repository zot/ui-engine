# Sequence: ViewList Presenter Sync

**Source Spec:** viewdefs.md, protocol.md, libraries.md
**Use Case:** ViewList wrapper syncs ViewListItem objects when source array changes

## Participants

- Variable: Array variable with wrapper=ViewList
- Resolver: Calls CreateWrapper on value changes
- ViewList: Wrapper stored in variable, manages ViewListItem objects
- ViewListItem: Holds item, list, index for each array element
- ObjectRegistry: Registers ViewList and ViewListItems for navigation
- LuaRuntime: Creates ViewListItem instances

## Sequence

```
     Variable              Resolver               ViewList           ViewListItem         ObjectRegistry
        |                      |                      |                      |                      |
        |   [on variable create with wrapper=ViewList]                                              |
        |                      |                      |                      |                      |
        |---CreateWrapper----->|                      |                      |                      |
        |   (variable)         |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---new(variable)----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getWrapper()------>|                      |
        |                      |                      |   (returns nil)      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getValue()-------->|                      |
        |                      |                      |   (get array)        |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getProperty------->|                      |
        |                      |                      |   ("item" type)      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---sync()------------>|                      |
        |                      |                      |   (initial sync)     |                      |
        |                      |                      |                      |                      |
        |                      |                      |     [for each array item, create ViewListItem]
        |                      |                      |                      |                      |
        |                      |                      |---new(list, idx)--->|                      |
        |                      |                      |                      |                      |
        |                      |                      |---item = array[i]-->|                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---register---------->|
        |                      |                      |                      |   (ViewListItem)     |
        |                      |                      |                      |                      |
        |                      |<--viewList-----------|                      |                      |
        |                      |   (new instance)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |---register-----------|--------------------------------------------->|
        |                      |   (ViewList as       |                      |                      |
        |                      |    navigation value) |                      |                      |
        |                      |                      |                      |                      |
        |---storeWrapper------>|                      |                      |                      |
        |   (internal field)   |                      |                      |                      |
        |                      |                      |                      |                      |
        |   [on value change - wrapper reuse and sync]                                              |
        |                      |                      |                      |                      |
        |---CreateWrapper----->|                      |                      |                      |
        |   (variable)         |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---new(variable)----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getWrapper()------>|                      |
        |                      |                      |   (returns existing) |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getValue()-------->|                      |
        |                      |                      |   (get new array)    |                      |
        |                      |                      |                      |                      |
        |                      |                      |---sync()------------>|                      |
        |                      |                      |                      |                      |
        |                      |                      |     [1. assign items to existing ViewListItems]
        |                      |                      |                      |                      |
        |                      |                      |---item = array[i]-->|                      |
        |                      |                      |---index = i-------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |     [2. trim excess ViewListItems if array shrunk]
        |                      |                      |                      |                      |
        |                      |                      |---remove(items[n])-->|                      |
        |                      |                      |                      |---unregister-------->|
        |                      |                      |                      |                      |
        |                      |                      |     [3. add new ViewListItems if array grew]
        |                      |                      |                      |                      |
        |                      |                      |---new(list, idx)--->|                      |
        |                      |                      |---item = array[i]-->|                      |
        |                      |                      |                      |---register---------->|
        |                      |                      |                      |                      |
        |                      |<--viewList-----------|                      |                      |
        |                      |   (same instance)    |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

### Wrapper Reuse Pattern

ViewList constructor checks for existing wrapper:
1. `variable:getWrapper()` returns existing wrapper if any
2. If existing, update `value` reference and call `sync()`
3. If none, create new wrapper and call `sync()`
4. Return wrapper (existing or new)

This preserves internal state like `selectionIndex` across array changes.

### ViewListItem Sync Algorithm

ViewList.sync() performs three steps:

1. **Assign items**: For each item in the array, assign it to the corresponding ViewListItem's `item` property
2. **Trim excess**: If the ViewListItem array is longer than the item array, remove excess ViewListItems
3. **Add new**: If the ViewListItem array is shorter, create new ViewListItems for additional items

```lua
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
        local vli = ViewListItem:new(self, idx)
        vli.item = array[idx + 1]
        table.insert(self.items, vli)
    end
end
```

### ViewListItem Object Structure

Each ViewListItem created by ViewList has:
- `item`: Reference to the domain object (`{obj: ID}`)
- `list`: Reference to the ViewList wrapper object
- `index`: Position in the list (0-based)

### Custom ViewListItems

When `item=PresenterType` is specified in path properties, ViewList creates instances of that custom type instead of plain ViewListItems:
- `PresenterType:new(viewList, index)` is called
- `item` property is assigned after construction

Custom ViewListItems can have UI-specific methods like `delete()` that access `self.item` and `self.list`.

### State Preservation

On wrapper reuse:
- `selectionIndex` is preserved (not reset)
- ViewListItem objects may be reused (same object, different `item`)
- ViewListItem indices are updated to match array positions

### Frontend Rendering

Frontend receives ViewListItem refs and renders using ViewListItem's type viewdef.
ViewListItem viewdef uses `ui-view="item"` to display the domain object.
The item's viewdef paths like `ui-value="name"` work directly on the domain object.
