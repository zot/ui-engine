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
        |    variable,         |                      |                      |                      |
        |    namespace)------->|                      |                      |                      |
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
        |                      |---get(TYPE.NS)------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [fallback to TYPE.DEFAULT if not found]                   |
        |                      |<--viewdef or null----|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if viewdef not found]             |                      |
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
        |                      |   (vendHtmlId)       |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for ui-viewlist elements]              |                      |
        |                      |---createViewList-----|-----------------------------------create--->|
        |                      |                      |                      |                      |
        |                      |                      |                      |     [set exemplar]   |
        |                      |                      |                      |<--setExemplar--------|
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

- render(element, variable, namespace) returns boolean: true if rendered, false if pending
- Requirements: variable has value, variable has type property, viewdef exists
- Fallback: If TYPE.NAMESPACE not found, tries TYPE.DEFAULT
- Pending views: Views that can't render added to pending list
- When viewdefs arrive, pending views re-attempt render
- ui-view creates View with unique frontend-vended HTML id
- ui-viewlist creates ViewList with exemplar element (default: div, or sl-option for selects)
- ui-namespace attribute specifies namespace for nested views
