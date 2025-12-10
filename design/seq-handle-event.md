# Sequence: Handle Event

**Source Spec:** viewdefs.md
**Use Case:** Processing DOM event and updating variable

## Participants

- Element: DOM element with event binding
- EventBinding: Event binding handler
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- Backend: External backend (if bound)

## Sequence

```
     Element            EventBinding         ProtocolHandler        VariableStore            Backend
        |                      |                      |                      |                      |
        |---DOM event--------->|                      |                      |                      |
        |   (click/change/etc) |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---extractValue------>|                      |                      |
        |                      |   (from event)       |                      |                      |
        |                      |                      |                      |                      |
        |                      |---isAction?--------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if action (e.g., ui-action="submit()")]                 |
        |                      |---update(varId,----->|                      |                      |
        |                      |    {action:name})    |                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if value (e.g., ui-value="name")]                       |
        |                      |---update(varId,----->|                      |                      |
        |                      |    newValue)         |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---isUnbound?-------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if unbound]                       |                      |
        |                      |                      |---store()----------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if bound]  |                      |                      |
        |                      |                      |---relay()------------------------------------>|
        |                      |                      |                      |                      |
        |                      |                      |     [backend processes action/value]        |
        |                      |                      |<----------------------------------response---|
        |                      |                      |                      |                      |
        |                      |                      |---notifyWatchers---->|                      |
        |                      |                      |                      |                      |
```

## Notes

- Event type determines value extraction (input value, click coords, etc.)
- Actions trigger method calls on presenter
- Value updates change bound variable value
- Unbound variables stored in UI server
- Bound variables forwarded to backend
- Backend can respond with UI updates
