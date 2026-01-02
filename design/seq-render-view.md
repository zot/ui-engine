# Sequence: Render View

**Source Spec:** viewdefs.md, libraries.md
**Use Case:** Rendering variable with viewdef, handling ui-view and ui-viewlist

## Participants

- ViewRenderer: View display manager
- View: Individual object reference view
- ViewList: Array of object reference views
- ViewdefStore: Viewdef storage with pending views
- BindingEngine: Binding processor

## Sequence

```
     ViewRenderer               View              ViewdefStore          BindingEngine            ViewList
        |                      |                      |                      |                      |
        |---render(element,    |                      |                      |                      |
        |    variable)-------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---checkRequirements->|                      |                      |
        |                      |   (value? type?)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if missing value or type]         |                      |
        |                      |---addPendingView---->|                      |                      |
        |                      |                      |                      |                      |
        |                      |<--false (not ready)--|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if has value and type]            |                      |
        |                      |---resolveNamespace-->|                      |                      |
        |                      |   (3-tier lookup)    |                      |                      |
        |                      |                      |                      |                      |
        |                      |          [1. try TYPE.{namespace}]          |                      |
        |                      |---get(TYPE.NS)------>|                      |                      |
        |                      |<--viewdef or null----|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [2. if null, try TYPE.{fallbackNamespace}]                |
        |                      |---get(TYPE.fallbackNS)->                    |                      |
        |                      |<--viewdef or null----|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [3. if null, try TYPE.DEFAULT]     |                      |
        |                      |---get(TYPE.DEFAULT)->|                      |                      |
        |                      |<--viewdef or null----|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if all null, add to pending]      |                      |
        |                      |---addPendingView---->|                      |                      |
        |                      |<--false (not ready)--|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if viewdef found]                 |                      |
        |                      |---clear()----------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |---cloneTemplate----->|                      |                      |
        |                      |   (deep clone)       |                      |                      |
        |                      |                      |                      |                      |
        |                      |---bind(elements)-----|--------------------->|                      |
        |                      |                      |                      |                      |
        |                      |     [for ui-view elements]                  |                      |
        |                      |---createView-------->|                      |                      |
        |                      |   (vendHtmlId,       |                      |                      |
        |                      |    inheritNamespace) |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for ui-viewlist elements]              |                      |
        |                      |---createViewList-----|-----------------------------------create--->|
        |                      |                      |                      |                      |
        |                      |                      |                      |     [set exemplar,   |
        |                      |                      |                      |<--inheritNamespace]--|
        |                      |                      |                      |                      |
        |                      |<--true (rendered)----|                      |                      |
        |                      |                      |                      |                      |
        |     [when viewdefs arrive via variable 1]   |                      |                      |
        |                      |                      |                      |                      |
        |                      |---processPending---->|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each pending view]                 |                      |
        |                      |---render()---------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if returns true, remove from pending]                    |
        |                      |---removePending----->|                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- render(element, variable) returns boolean: true if rendered, false if pending
- Requirements: variable has value, variable has type property, viewdef exists
- **3-tier namespace resolution:**
  1. Try `TYPE.{namespace}` (from variable's namespace property)
  2. If not found, try `TYPE.{fallbackNamespace}` (from variable's fallbackNamespace property)
  3. If not found, use `TYPE.DEFAULT`
- Pending views: Views that can't render added to pending list
- When viewdefs arrive, pending views re-attempt render
- ui-view creates View with unique frontend-vended HTML id
- ui-viewlist creates ViewList with exemplar element (default: div, or sl-option for selects)
- **Namespace inheritance:**
  - `ui-namespace` attribute sets variable's `namespace` property
  - If no attribute, `namespace` is inherited from parent variable
  - `fallbackNamespace` is always inherited from parent variable
  - ViewList wrapper sets `fallbackNamespace: "list-item"` on its variable
