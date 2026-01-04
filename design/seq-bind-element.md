# Sequence: Bind Element

**Source Spec:** viewdefs.md, libraries.md
**Use Case:** Applying ui-* bindings to a DOM element

## Participants

- ViewRenderer: View display
- BindingEngine: Binding coordinator
- Widget: Binding context for element (element ID, variable map)
- VariableStore: Variable creation and watching
- Backend: Server-side path resolution

## Sequence

```
     ViewRenderer         BindingEngine            Widget           VariableStore            Backend
        |                      |                      |                      |                      |
        |---bind(element,----->|                      |                      |                      |
        |    contextVarId)     |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---getOrCreateWidget->|                      |                      |
        |                      |   (element)          |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---vendElementId----->|                      |
        |                      |                      |   (via ElementIdVendor|                     |
        |                      |                      |    if no id: "ui-{n}")|                     |
        |                      |                      |                      |                      |
        |                      |<--widget-------------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseAttributes--->|                      |                      |
        |                      |   (ui-*)             |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each ui-value, ui-attr-*, ui-class-*, ui-style-*, ui-code]
        |                      |                      |                      |                      |
        |                      |---parsePath--------->|                      |                      |
        |                      |   (extract path &    |                      |                      |
        |                      |    options)          |                      |                      |
        |                      |                      |                      |                      |
        |                      |---store.create------>|                      |                      |
        |                      |   {parentId:         |                      |                      |
        |                      |    contextVarId,     |                      |                      |
        |                      |    properties:       |                      |---create msg-------->|
        |                      |    {path:"field",    |                      |                      |
        |                      |     elementId:id}}   |                      |                      |
        |                      |                      |                      |                      |
        |                      |<--childVarId---------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---widget.register--->|                      |                      |
        |                      |   Binding(name,      |                      |                      |
        |                      |    childVarId)       |                      |                      |
        |                      |                      |                      |                      |
        |                      |---widget.addUnbind-->|                      |                      |
        |                      |   Handler(name,      |                      |                      |
        |                      |    cleanupFn)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |                      |---resolve path
        |                      |                      |                      |                      |   on parent obj
        |                      |                      |                      |                      |
        |                      |---store.watch------->|                      |                      |
        |                      |   (childVarId)       |                      |<--update(childVarId,-|
        |                      |                      |                      |      resolvedValue)  |
        |                      |                      |                      |                      |
        |                      |<--callback(value)----|                      |                      |
        |                      |                      |                      |                      |
        |                      |---apply(element,---->|                      |                      |
        |                      |      value)          |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each ui-event-*, ui-action]        |                      |
        |                      |                      |                      |                      |
        |                      |---store.create------>|                      |                      |
        |                      |   {parentId:         |                      |                      |
        |                      |    contextVarId,     |                      |---create msg-------->|
        |                      |    properties:       |                      |                      |
        |                      |    {path:"method()", |                      |                      |
        |                      |     access:"action"}}|                      |                      |
        |                      |                      |                      |                      |
        |                      |<--actionVarId--------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---addEventListener-->|                      |                      |
        |                      |   (on click: update  |                      |                      |
        |                      |    actionVarId)      |                      |                      |
        |                      |                      |                      |                      |
        |<--bound element------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

### Widget Creation

When binding an element:
1. BindingEngine calls `getOrCreateWidget(element)`
2. If element has no `id` attribute, Widget requests one from global ElementIdVendor: `ui-{counter}`
3. Widget tracks mapping of binding names to variable IDs
4. Variables store `elementId` property (not direct DOM reference) for Widget access

### Child Variable Architecture (Critical)

**All path-based bindings MUST create child variables for backend path resolution.**

Variable values are **object references** (`{"obj": 1}`), not actual data. Client-side path resolution is impossible. Each binding:

1. **Creates** a child variable with `path` property under the context variable
2. **Registers** the binding with the Widget (name -> variable ID mapping)
3. **Watches** the child variable for value updates from the backend
4. **Destroys** the child variable on unbind (and unregisters from Widget)

This applies to: `ui-value`, `ui-attr-*`, `ui-class-*`, `ui-style-*`, `ui-code`, `ui-event-*`, `ui-action`

### Variable-Widget Relationship

Variables created by bindings do NOT store direct DOM references:
- Variable stores `elementId` property (the Widget's element ID)
- Element is looked up via `document.getElementById(elementId)` when needed
- This avoids memory leaks and enables serialization

### Path Options

- Path values can include URL-style parameters (?create=Type&prop=value)
- **Properties without values default to `true`:** `name?keypress` equals `name?keypress=true`

### Default Access Property

When creating child variables, the binding engine sets a default `access` property if not explicitly specified:
- `ui-value` on interactive elements (input, textarea, select, sl-*): `access=rw`
- `ui-value` on non-interactive elements (div, span, etc.): `access=r`
- `ui-attr-*`, `ui-class-*`, `ui-style-*`, `ui-code`: `access=r`
- `ui-view`, `ui-viewlist`: `access=r`

### Input Event Selection

For input elements, the update event depends on `keypress` property:
- Default: `blur` (native) or `sl-change` (Shoelace)
- With `keypress`: `input` (native) or `sl-input` (Shoelace)
- See seq-input-value-binding.md for detailed flow

### Unbind Cleanup (Widget-Based)

When unbinding an element:
1. BindingEngine calls `widget.unbindAll()`
2. Widget iterates `unbindHandlers` map and calls each cleanup function
3. Each cleanup function: stops watching, removes listeners, destroys child variable
4. Widget clears `unbindHandlers` map
5. BindingEngine removes Widget from `widgets` map
6. Widget removes auto-vended element ID if applicable

### Nullish Path Handling

Paths use nullish coalescing (see crc-PathNavigator.md):
- Read: Displays empty/default value when path is nullish (no error)
- Write: Sends `error(varId, 'path-failure', description)` when path is nullish (UI shows error indicator, clears on success)
