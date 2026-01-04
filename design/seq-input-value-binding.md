# Sequence: Input Value Binding

**Source Spec:** libraries.md, viewdefs.md
**Use Case:** Two-way binding for input elements with blur/keypress event selection

## Participants

- ViewRenderer: View display
- BindingEngine: Binding coordinator
- PathSyntax: Path parser
- ValueBinding: Value binding handler
- WidgetBinder: Widget-specific bindings (for Shoelace)
- WatchManager: Variable watching
- ProtocolHandler: Message processor

## Sequence

```
     ViewRenderer         BindingEngine           PathSyntax          ValueBinding          WidgetBinder         WatchManager       ProtocolHandler
        |                      |                      |                      |                      |                      |                      |
        |---bind(element,----->|                      |                      |                      |                      |                      |
        |    contextVarId)     |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |---getAttribute------>|                      |                      |                      |                      |
        |                      |   ("ui-value")       |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |---parsePath--------->|                      |                      |                      |                      |
        |                      |   ("name?keypress")  |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |<--{segments, --------|                      |                      |                      |                      |
        |                      |    options: {        |                      |                      |                      |                      |
        |                      |      keypress: true  |                      |                      |                      |                      |
        |                      |    }}                |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |---isWidgetElement?-->|                      |                      |                      |                      |
        |                      |   (sl-input)         |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |          [Native input: <input>, <textarea>]                       |                      |                      |
        |                      |---createValueBinding>|                      |                      |                      |                      |
        |                      |   (element, options) |---create()---------->|                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---selectEvent------->|                      |                      |
        |                      |                      |                      |   (keypress=true?)   |                      |                      |
        |                      |                      |                      |<--"input"------------|                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---addEventListener-->|                      |                      |
        |                      |                      |                      |   ("input" | "blur") |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |          [Shoelace input: <sl-input>, <sl-textarea>]               |                      |                      |
        |                      |---bindWidget-------->|                      |                      |                      |                      |
        |                      |   (element, var,     |                      |---bind()------------>|                      |                      |
        |                      |    bindings, options)|                      |                      |                      |                      |
        |                      |                      |                      |                      |---selectEvent------->|                      |
        |                      |                      |                      |                      |   (keypress=true?)   |                      |
        |                      |                      |                      |                      |<--"sl-input"---------|                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |---addEventListener-->|                      |
        |                      |                      |                      |                      |   ("sl-input" |      |                      |
        |                      |                      |                      |                      |    "sl-change")      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |          [User types in input]              |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |<--event fired--------|                      |                      |
        |                      |                      |                      |   (input | blur |    |                      |                      |
        |                      |                      |                      |    sl-input |        |                      |                      |
        |                      |                      |                      |    sl-change)        |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---extractValue------>|                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---shouldSuppress?--->|                      |                      |
        |                      |                      |                      |   (compare values)   |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |          [if NOT suppressed (value changed OR access=action/w)]   |                      |
        |                      |                      |                      |---update(varId,----->|                      |----------------->|
        |                      |                      |                      |    value)            |                      |                      |
        |                      |                      |                      |                      |                      |                      |
```

## Notes

- Path properties without values default to `true`: `name?keypress` equals `name?keypress=true`
- Default event for all input elements is blur-based (network-efficient)
- `keypress` property switches to immediate updates (every keystroke)
- Native elements use DOM events: `blur` vs `input`
- Shoelace elements use custom events: `sl-change` vs `sl-input`
- BindingEngine detects widget tags and delegates to WidgetBinder
- WidgetBinder receives path options from BindingEngine
- Both paths share the same update flow: extract value, check suppression, update variable
- **Duplicate suppression**: If new value equals cached value AND access is not `action` or `w`, skip the update
