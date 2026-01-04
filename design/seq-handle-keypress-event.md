# Sequence: Handle Keypress Event

**Source Spec:** viewdefs.md
**Use Case:** Processing keydown event for ui-event-keypress-* binding

## Participants

- Element: DOM element with keypress binding
- EventBinding: Keypress event binding handler (implemented by BindingEngine, cleanup registered with Widget)
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- Backend: External backend (if bound)

## Sequence

```
     Element            EventBinding         ProtocolHandler        VariableStore            Backend
        |                      |                      |                      |                      |
        |---keydown event----->|                      |                      |                      |
        |   (any key)          |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---matchesTargetKey-->|                      |                      |
        |                      |   (event.key vs      |                      |                      |
        |                      |    targetKey)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if NO match - ignore event]       |                      |
        |                      |<-(return)------------|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if YES match - process event]     |                      |
        |                      |                      |                      |                      |
        |                      |---update(varId,----->|                      |                      |
        |                      |    keyName)          |                      |                      |
        |                      |   (e.g., "enter")    |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---isUnbound?-------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if unbound]                       |                      |
        |                      |                      |---store()----------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if bound]  |                      |                      |
        |                      |                      |---relay()------------------------------------>|
        |                      |                      |                      |                      |
        |                      |                      |     [backend processes keypress]             |
        |                      |                      |<----------------------------------response---|
        |                      |                      |                      |                      |
        |                      |                      |---notifyWatchers---->|                      |
        |                      |                      |                      |                      |
```

## Notes

### Key Matching

The `matchesTargetKey` function performs case-insensitive comparison between the browser's `event.key` value and the normalized target key:

| Attribute Key | Normalized Target | Matches event.key |
|---------------|-------------------|-------------------|
| `enter`       | `Enter`           | `Enter`           |
| `escape`      | `Escape`          | `Escape`          |
| `left`        | `ArrowLeft`       | `ArrowLeft`       |
| `right`       | `ArrowRight`      | `ArrowRight`      |
| `up`          | `ArrowUp`         | `ArrowUp`         |
| `down`        | `ArrowDown`       | `ArrowDown`       |
| `tab`         | `Tab`             | `Tab`             |
| `space`       | ` ` (space)       | ` ` or `Spacebar` |
| `a`           | `a` (letter)      | `a` or `A`        |

### Variable Value

When the target key is matched, the variable is set to the **attribute key name** (lowercase), not the browser's event.key value:
- `ui-event-keypress-enter` -> variable value is `"enter"`
- `ui-event-keypress-escape` -> variable value is `"escape"`
- `ui-event-keypress-left` -> variable value is `"left"`

This provides a consistent, predictable value for backend handlers regardless of browser key naming.

### Multiple Keypress Bindings

An element can have multiple keypress bindings for different keys:

```html
<input ui-event-keypress-enter="onEnter" ui-event-keypress-escape="onCancel">
```

Each binding creates a separate EventBinding instance with its own targetKey. All share the same `keydown` listener attachment point but filter independently.

### Focus Requirement

For non-input elements, the element must be focusable to receive keyboard events. Add `tabindex="0"` to make an element focusable:

```html
<div ui-event-keypress-escape="closeModal" tabindex="0">...</div>
```

### Event Propagation

Keypress bindings do NOT automatically call `preventDefault()` or `stopPropagation()`. The keydown event continues normal propagation. If the backend handler needs to prevent default behavior, it should return an appropriate response.
