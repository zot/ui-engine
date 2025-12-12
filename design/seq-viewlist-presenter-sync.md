# Sequence: ViewList Presenter Sync

**Source Spec:** viewdefs.md, protocol.md
**Use Case:** ViewList wrapper syncs ViewItem objects when source array changes

## Participants

- Variable: Array variable with wrapperInstance=ViewList
- ViewList: Wrapper stored in variable, manages ViewItem objects
- ViewItem: Holds baseItem, item, list, index for each array element
- LuaRuntime: Creates ViewItem and ItemWrapper instances
- VariableStore: Stores ViewItem and presenter objects

## Sequence

```
     Variable              ViewList              ViewItem              LuaRuntime           VariableStore
        |                      |                      |                      |                      |
        |   [on variable create with wrapper=ViewList]                                              |
        |                      |                      |                      |                      |
        |---new ViewList------>|                      |                      |                      |
        |   (variable)         |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---getProperty------->|                      |                      |
        |                      |   ("item" type)      |                      |                      |
        |                      |                      |                      |                      |
        |---storeWrapper------>|                      |                      |                      |
        |   (internal field)   |                      |                      |                      |
        |                      |                      |                      |                      |
        |   [on value change, Variable calls computeValue]                                          |
        |                      |                      |                      |                      |
        |---computeValue------>|                      |                      |                      |
        |   (rawArray)         |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---compareArrays----->|                      |                      |
        |                      |   (old vs new refs)  |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each added item]                                          |
        |                      |                      |                      |                      |
        |                      |---createViewItem---->|                      |                      |
        |                      |   ({obj:101}, 0)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---new ViewItem------>|                      |
        |                      |                      |   (baseItem, list,   |                      |
        |                      |                      |    index)            |                      |
        |                      |                      |                      |                      |
        |                      |                      |   [if item property set]                    |
        |                      |                      |                      |                      |
        |                      |                      |---ItemWrapper------->|                      |
        |                      |                      |   (viewItem)         |                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---CreateInstance---->|
        |                      |                      |                      |   (itemType, viewItem)
        |                      |                      |                      |                      |
        |                      |                      |<--presenter ref------|                      |
        |                      |                      |   (viewItem.item =   |                      |
        |                      |                      |    {obj: P1})        |                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---registerObject---->|
        |                      |<--viewItem ref-------|                      |   (ViewItem obj ID)  |
        |                      |   ({obj: VI1})       |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each removed item]                                        |
        |                      |                      |                      |                      |
        |                      |---destroyViewItem--->|                      |                      |
        |                      |   (viewItemRef)      |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---destroy presenter->|                      |
        |                      |                      |   (if item wrapped)  |                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---unregisterObject-->|
        |                      |                      |                      |                      |
        |                      |     [for reordered items]                                          |
        |                      |                      |                      |                      |
        |                      |---updateIndex------->|                      |                      |
        |                      |   (viewItem, newIdx) |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---setIndex---------->|                      |
        |                      |                      |   (newIdx)           |                      |
        |                      |                      |                      |                      |
        |                      |---buildStoredValue-->|                      |                      |
        |                      |   (viewItem refs)    |                      |                      |
        |                      |                      |                      |                      |
        |<--[{obj:VI1},--------|                      |                      |                      |
        |    {obj:VI2},        |                      |                      |                      |
        |    {obj:VI3}]        |                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

### Wrapper Initialization

ViewList constructor:
1. Receives variable reference
2. Reads `item` property from variable to get presenter type
3. Stores variable reference for later property access
4. Wrapper instance is stored internally in the variable

### ViewItem Object Structure

Each ViewItem created by ViewList has:
- `baseItem`: Reference to the domain object (`{obj: ID}`)
- `item`: Either same as baseItem, or `ItemWrapper(viewItem)` result
- `list`: Reference to the ViewList wrapper object
- `index`: Position in the list (0-based)

### Item Wrapping

When `item=PresenterType` is specified in path properties:
1. ViewItem is created with `baseItem` = domain ref
2. ItemWrapper is constructed: `ItemWrapper(viewItem)`
3. ItemWrapper receives the ViewItem, can access baseItem, list, index
4. `viewItem.item` is set to the presenter ref

The presenter can then:
- Access domain object via `viewItem.baseItem`
- Remove itself via `viewItem.list.removeAt(viewItem.index)`

### Change Detection

ViewList compares old and new arrays by object reference IDs:
- New refs: Create ViewItem
- Missing refs: Destroy ViewItem (and presenter if wrapped)
- Same refs, different order: Update ViewItem index property

### Stored Value

The stored value is an array of ViewItem refs, not domain refs:
- Raw: `[{obj:101}, {obj:102}, {obj:103}]` (domain objects)
- Stored: `[{obj:VI1}, {obj:VI2}, {obj:VI3}]` (ViewItem objects)

Each ViewItem contains the domain ref in `baseItem` and optionally a presenter in `item`.

### Frontend Rendering

Frontend receives ViewItem refs and renders using ViewItem's type viewdef.
ViewItem viewdef uses `ui-view="item"` to display the (possibly wrapped) item.
The item's viewdef paths like `ui-value="name"` work directly on the presenter or domain object.
