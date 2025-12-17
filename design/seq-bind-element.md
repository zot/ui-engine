# Sequence: Bind Element

**Source Spec:** viewdefs.md, libraries.md
**Use Case:** Applying ui-* bindings to a DOM element

## Participants

- ViewRenderer: View display
- BindingEngine: Binding coordinator
- VariableStore: Variable creation and watching
- Backend: Server-side path resolution

## Sequence

```
     ViewRenderer         BindingEngine         VariableStore            Backend
        |                      |                      |                      |
        |---bind(element,----->|                      |                      |
        |    contextVarId)     |                      |                      |
        |                      |                      |                      |
        |                      |---parseAttributes--->|                      |
        |                      |   (ui-*)             |                      |
        |                      |                      |                      |
        |                      |     [for each ui-value, ui-attr-*, ui-class-*, ui-style-*-*]
        |                      |                      |                      |
        |                      |---parsePath--------->|                      |
        |                      |   (extract path &    |                      |
        |                      |    options)          |                      |
        |                      |                      |                      |
        |                      |---store.create------>|                      |
        |                      |   {parentId:         |                      |
        |                      |    contextVarId,     |                      |
        |                      |    properties:       |---create msg-------->|
        |                      |    {path:"field"}}   |                      |
        |                      |                      |                      |
        |                      |<--childVarId---------|                      |
        |                      |                      |                      |
        |                      |                      |                      |---resolve path
        |                      |                      |                      |   on parent obj
        |                      |                      |                      |
        |                      |---store.watch------->|                      |
        |                      |   (childVarId)       |<--update(childVarId,-|
        |                      |                      |      resolvedValue)  |
        |                      |                      |                      |
        |                      |<--callback(value)----|                      |
        |                      |                      |                      |
        |                      |---apply(element,---->|                      |
        |                      |      value)          |                      |
        |                      |                      |                      |
        |                      |     [for each ui-event-*, ui-action]        |
        |                      |                      |                      |
        |                      |---store.create------>|                      |
        |                      |   {parentId:         |                      |
        |                      |    contextVarId,     |---create msg-------->|
        |                      |    properties:       |                      |
        |                      |    {path:"method()", |                      |
        |                      |     access:"action"}}|                      |
        |                      |                      |                      |
        |                      |<--actionVarId--------|                      |
        |                      |                      |                      |
        |                      |---addEventListener-->|                      |
        |                      |   (on click: update  |                      |
        |                      |    actionVarId)      |                      |
        |                      |                      |                      |
        |<--bound element------|                      |                      |
        |                      |                      |                      |
```

## Notes

### Child Variable Architecture (Critical)

**All path-based bindings MUST create child variables for backend path resolution.**

Variable values are **object references** (`{"obj": 1}`), not actual data. Client-side path resolution is impossible. Each binding:

1. **Creates** a child variable with `path` property under the context variable
2. **Watches** the child variable for value updates from the backend
3. **Destroys** the child variable on unbind

This applies to: `ui-value`, `ui-attr-*`, `ui-class-*`, `ui-style-*-*`, `ui-event-*`, `ui-action`

### Path Options

- Path values can include URL-style parameters (?create=Type&prop=value)
- **Properties without values default to `true`:** `name?keypress` equals `name?keypress=true`

### Input Event Selection

For input elements, the update event depends on `keypress` property:
- Default: `blur` (native) or `sl-change` (Shoelace)
- With `keypress`: `input` (native) or `sl-input` (Shoelace)
- See seq-input-value-binding.md for detailed flow

### Unbind Cleanup

When unbinding, the binding engine:
1. Stops watching the child variable
2. Removes event listeners
3. **Destroys the child variable** (`store.destroy(childVarId)`)

### Nullish Path Handling

Paths use nullish coalescing (see crc-PathNavigator.md):
- Read: Displays empty/default value when path is nullish (no error)
- Write: Sends `error(varId, 'path-failure', description)` when path is nullish (UI shows error indicator, clears on success)
