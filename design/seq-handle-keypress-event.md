# Sequence: Handle Keypress Event

**Source Spec:** viewdefs.md
**Use Case:** Processing keydown event for ui-event-keypress-* binding (with optional modifiers)

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
        |   (any key+mods)     |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---matchesTargetKey-->|                      |                      |
        |                      |   (event.key vs      |                      |                      |
        |                      |    targetKey)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if key NO match - ignore event]   |                      |
        |                      |<-(return)------------|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if key YES match - check modifiers]                      |
        |                      |                      |                      |                      |
        |                      |---matchesModifiers-->|                      |                      |
        |                      |   (event.ctrlKey,    |                      |                      |
        |                      |    event.shiftKey,   |                      |                      |
        |                      |    event.altKey,     |                      |                      |
        |                      |    event.metaKey     |                      |                      |
        |                      |    vs modifierKeys)  |                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if modifiers NO match - ignore]   |                      |
        |                      |<-(return)------------|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if modifiers YES match - process] |                      |
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

### Modifier Matching (Exact)

The `matchesModifiers` function performs **exact** comparison between the event's modifier state and the required modifiers:

| Required Modifiers | Event State | Match? |
|-------------------|-------------|--------|
| `{}` (none)       | No modifiers pressed | Yes |
| `{}` (none)       | Ctrl pressed | No |
| `{ctrl}`          | Ctrl pressed | Yes |
| `{ctrl}`          | Ctrl+Shift pressed | No |
| `{ctrl, shift}`   | Ctrl+Shift pressed | Yes |
| `{ctrl, shift}`   | Ctrl pressed | No |

**Implementation:**
```
matchesModifiers(event, requiredModifiers):
  return (event.ctrlKey  == requiredModifiers.has("ctrl"))
     AND (event.shiftKey == requiredModifiers.has("shift"))
     AND (event.altKey   == requiredModifiers.has("alt"))
     AND (event.metaKey  == requiredModifiers.has("meta"))
```

### Variable Value

When the target key is matched, the variable is set to the **attribute key name** (lowercase), not the browser's event.key value:
- `ui-event-keypress-enter` -> variable value is `"enter"`
- `ui-event-keypress-ctrl-enter` -> variable value is `"enter"`
- `ui-event-keypress-escape` -> variable value is `"escape"`
- `ui-event-keypress-left` -> variable value is `"left"`

This provides a consistent, predictable value for backend handlers regardless of browser key naming. Note that modifiers do not affect the variable value - only the key name is sent.

### Multiple Keypress Bindings

An element can have multiple keypress bindings for different key/modifier combinations:

```html
<input ui-event-keypress-enter="onEnter"
       ui-event-keypress-ctrl-enter="onCtrlEnter"
       ui-event-keypress-escape="onCancel">
```

Each binding creates a separate EventBinding instance with its own targetKey and modifierKeys. All share the same `keydown` listener attachment point but filter independently. When Ctrl+Enter is pressed, only the `ctrl-enter` binding fires (due to exact modifier matching), not the plain `enter` binding.

### Focus Requirement

For non-input elements, the element must be focusable to receive keyboard events. Add `tabindex="0"` to make an element focusable:

```html
<div ui-event-keypress-escape="closeModal" tabindex="0">...</div>
```

### Event Propagation

Keypress bindings do NOT automatically call `preventDefault()` or `stopPropagation()`. The keydown event continues normal propagation. If the backend handler needs to prevent default behavior, it should return an appropriate response.
