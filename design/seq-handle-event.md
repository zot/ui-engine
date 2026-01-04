# Sequence: Handle Event

**Source Spec:** viewdefs.md
**Use Case:** Processing DOM event and updating variable (with value sync)

## Participants

- Element: DOM element with event binding
- EventBinding: Event binding handler (implemented by BindingEngine, cleanup registered with Widget)
- Widget: Binding context for the element (owns all bindings via unbindHandlers)
- Variable: Target variable (value set locally before sending)
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- Backend: External backend (if bound)

## Sequence

```
     Element        EventBinding          Widget           Variable       ProtocolHandler         Backend
        |                |                   |                 |                 |                   |
        |--DOM event---->|                   |                 |                 |                   |
        | (click/change) |                   |                 |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---syncValueBinding----------------->|                 |                   |
        |                |   (check for ui-value binding)      |                 |                   |
        |                |                   |                 |                 |                   |
        |                |          [if ui-value binding exists on widget]       |                   |
        |                |---getVariable---->|                 |                 |                   |
        |                |   ("ui-value")    |                 |                 |                   |
        |                |<--valueVarId------|                 |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---getElementValue-|---------------->|                 |                   |
        |                |   (current value) |                 |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---compare values--|---------------->|                 |                   |
        |                |   (duplicate suppression check)     |                 |                   |
        |                |                   |                 |                 |                   |
        |                |          [if values differ AND NOT (access=action OR access=w)]           |
        |                |---setLocalValue---|---------------->|                 |                   |
        |                |   (optimistic)    |                 |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---sendUpdate------|-----------------|---------------->|                   |
        |                |   (value var)     |                 |                 |                   |
        |                |                   |                 |                 |---relay---------->|
        |                |                   |                 |                 |                   |
        |                |          [end value sync]           |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---extractValue--->|                 |                 |                   |
        |                |   (event value)   |                 |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---setLocalValue---|---------------->|                 |                   |
        |                |   (event var)     |                 |                 |                   |
        |                |                   |                 |                 |                   |
        |                |---sendUpdate------|-----------------|---------------->|                   |
        |                |   (event var)     |                 |                 |---relay---------->|
        |                |                   |                 |                 |                   |
        |                |                   |                 |     [backend processes event]       |
        |                |                   |                 |                 |<------response----|
        |                |                   |                 |                 |                   |
```

## Notes

**Spec References:**
- **Frontend Update Behavior** (viewdefs.md): Whenever sending a variable update, MUST first set the value in local variable cache
- **Duplicate Update Suppression** (viewdefs.md): Bindings without `access=action` or `access=w` MUST NOT send update if value unchanged
- **Event Bindings** (viewdefs.md): When ui-event-* fires on element with ui-value binding, check if element's current value differs from variable's cached value; if different, send value update first

**Key behaviors:**
- **Value sync first**: Before sending event update, sync ui-value binding if present on same widget
- **Duplicate suppression**: Skip value sync update if values are equal (unless access=action or access=w)
- **Local value setting**: Always set variable.value locally BEFORE sending update to backend
- Event type determines value extraction (input value, click coords, etc.)
- Actions trigger method calls on presenter (always sent, never suppressed)
- Value updates change bound variable value
- Backend processes event after receiving synced value
- Optimistic updates provide immediate UI feedback
