# Sequence: ViewList Update

**Source Spec:** viewdefs.md
**Use Case:** ViewList responds to bound array changes

## Participants

- Variable: Array variable with object references
- ViewList: List of views for array items
- View: Individual item view
- ViewRenderer: Creates views
- BindingEngine: Applies bindings

## Sequence

```
     Variable              ViewList                  View              ViewRenderer          BindingEngine
        |                      |                      |                      |                      |
        |---update(array)----->|                      |                      |                      |
        |   [{obj:5},{obj:7},  |                      |                      |                      |
        |    {obj:9}]          |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---compareArrays----->|                      |                      |
        |                      |   (old vs new)       |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [detect added: obj:9]                   |                      |
        |                      |                      |                      |                      |
        |                      |---cloneExemplar----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |---createVariable-----|--------------------->|                      |
        |                      |   (for obj:9)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |---createView---------|--------------------->|                      |
        |                      |                      |---vendHtmlId-------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--view(htmlId)-------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---render()---------->|                      |
        |                      |                      |                      |---lookupViewdef----->|
        |                      |                      |                      |                      |
        |                      |                      |                      |---cloneTemplate----->|
        |                      |                      |                      |                      |
        |                      |                      |                      |---bind(elements)---->|
        |                      |                      |                      |                      |
        |                      |                      |<--rendered-----------|                      |
        |                      |                      |                      |                      |
        |                      |---appendChild------->|                      |                      |
        |                      |   (to container)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |---notifyDelegate---->|                      |                      |
        |                      |   (itemAdded)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [detect removed: obj:3]                 |                      |
        |                      |                      |                      |                      |
        |                      |---destroyVariable--->|                      |                      |
        |                      |                      |                      |                      |
        |                      |---removeChild------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |---notifyDelegate---->|                      |                      |
        |                      |   (itemRemoved)      |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [detect reorder]                        |                      |
        |                      |                      |                      |                      |
        |                      |---reorderChildren--->|                      |                      |
        |                      |   (match array order)|                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- ViewList maintains parallel array of View elements
- Exemplar element cloned for each new item (default: div)
- Select Views use sl-option as exemplar
- Child views use ViewList's namespace (default: DEFAULT)
- Variables created for each item enable binding within item view
- Delegate notified of add/remove for custom handling
- Reordering moves existing DOM elements without recreating

### Wrapper Integration

When the ViewList variable has `wrapper=ViewList` property (set via path properties like `contacts?item=ContactPresenter`):

1. Backend uses ViewList wrapper to transform array value
2. Source array refs are transformed to presenter refs
3. Frontend receives presenter refs instead of domain refs
4. Views render using presenter type's viewdef

See seq-viewlist-presenter-sync.md for backend presenter management.
See seq-wrapper-transform.md for wrapper value transformation.
