# Sequence: Bind Element

**Source Spec:** viewdefs.md, libraries.md
**Use Case:** Applying ui-* bindings to a DOM element

## Participants

- ViewRenderer: View display
- BindingEngine: Binding coordinator
- ValueBinding: Value binding handler
- EventBinding: Event binding handler
- WatchManager: Variable watching

## Sequence

```
     ViewRenderer         BindingEngine          ValueBinding          EventBinding          WatchManager
        |                      |                      |                      |                      |
        |---bind(element,----->|                      |                      |                      |
        |    viewdef)          |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseAttributes--->|                      |                      |
        |                      |   (ui-*)             |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each ui-value, ui-attr-*, ui-class-*, ui-style-*-*]      |
        |                      |---createValueBinding>|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---parsePath--------->|                      |
        |                      |                      |   (with ?params)     |                      |
        |                      |                      |                      |                      |
        |                      |                      |---createVariable---->|                      |
        |                      |                      |   (with path prop)   |                      |
        |                      |                      |                      |                      |
        |                      |                      |---watch(varId)------>|                      |
        |                      |                      |                      |---addWatch---------->|
        |                      |                      |                      |                      |
        |                      |                      |<--initial value------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---apply(element)---->|                      |
        |                      |                      |                      |                      |
        |                      |     [for each ui-event-*]                   |                      |
        |                      |---createEventBinding>|                      |                      |
        |                      |                      |---create()---------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---addEventListener-->|
        |                      |                      |                      |                      |
        |<--bound element------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Path values can include URL-style parameters (?create=Type&prop=value)
- Variables created for each binding with path property
- Value bindings watch variables and apply to element
- Event bindings attach DOM listeners
- Bindings cleaned up when element is removed
- **Nullish path handling:** Paths use nullish coalescing (see crc-PathNavigator.md):
  - Read: Displays empty/default value when path is nullish (no error)
  - Write: Sends `error(varId, 'path-failure', description)` when path is nullish (UI shows error indicator, clears on success)
