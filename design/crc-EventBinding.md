# EventBinding

**Source Spec:** viewdefs.md

Event bindings connect DOM events to variable updates. They are created by BindingEngine and registered with the Widget via unbind handlers (no separate Binding interface).

## Responsibilities

### Knows (per binding)
- elementId: ID of source element (NOT direct DOM reference)
- eventType: DOM event name (click, change, input, keydown, etc.)
- variableId: Target variable ID
- actionPath: Path for action triggers (e.g., "submit()")
- targetKey: For keypress bindings, the specific key to listen for (e.g., "enter", "escape")
- widget: Reference to the Widget for this element (for accessing sibling bindings)

### Does (implemented by BindingEngine)
- createEventBinding: Add event listener to element (looked up by ID), register unbind handler with Widget
- handleEvent: Process DOM event and update variable (with value sync)
- handleKeypressEvent: Process keydown event, filter by targetKey, update variable with key name
- extractEventValue: Get relevant value from event (input value, click data, etc.)
- matchesTargetKey: Check if keyboard event matches targetKey (case-insensitive)
- isAction: Check if binding triggers an action vs. value update
- isKeypressBinding: Check if this is a keypress-specific binding (ui-event-keypress-*)
- getElement: Look up DOM element by elementId (via document.getElementById)
- syncValueBinding: Check for ui-value binding on same widget and sync value if changed

## Unbind Handler

Each event binding registers an unbind handler with the Widget that:
1. Removes the event listener from the element
2. Destroys the child variable (if any)

Called automatically when `widget.unbindAll()` is invoked.

## Keypress Binding

The `ui-event-keypress-*` attribute creates a specialized event binding for specific key presses.

**Attribute format:**
```html
<input ui-event-keypress-enter="onEnter">
<div ui-event-keypress-escape="onCancel" tabindex="0">
<input ui-event-keypress-a="onLetterA">
```

**Behavior:**
1. Listens on the `keydown` event of the element
2. Filters events by the specified key (case-insensitive comparison with `event.key`)
3. When the target key is pressed, updates the variable based on path type:
   - **Non-action path** (e.g., `lastKey`): Sets variable to key name (e.g., `"enter"`)
   - **No-arg action** (e.g., `selectFirst()`): Updates with `null` (invokes action for side-effect)
   - **1-arg action** (e.g., `handleKey(_)`): Updates with key name as argument
4. Non-matching keydown events are ignored (no variable update)

**Supported keys:**
- `enter` - Enter/Return key (matches "Enter")
- `escape` - Escape key (matches "Escape")
- `left` - Left arrow (matches "ArrowLeft")
- `right` - Right arrow (matches "ArrowRight")
- `up` - Up arrow (matches "ArrowUp")
- `down` - Down arrow (matches "ArrowDown")
- `tab` - Tab key (matches "Tab")
- `space` - Space bar (matches " " or "Spacebar")
- Single letters (`a`-`z`) - Any letter key (case-insensitive)

**Key name normalization:**
The binding normalizes the attribute key name to match the browser's `event.key` value:
- `enter` -> matches "Enter"
- `escape` -> matches "Escape"
- `left` -> matches "ArrowLeft"
- `right` -> matches "ArrowRight"
- `up` -> matches "ArrowUp"
- `down` -> matches "ArrowDown"
- `tab` -> matches "Tab"
- `space` -> matches " " (space character)
- Letters -> case-insensitive match (e.g., `a` matches "a" or "A")

## Event Update Behavior (Value Sync with ui-value)

**Spec: viewdefs.md "Event Bindings"**

When an event fires on an element that also has a `ui-value` binding, the binding performs value synchronization:

1. Check if the element's current value differs from the variable's cached value
2. If different, send a variable update with the new value first (following "Local Value Setting" principle and "Duplicate Update Suppression")
3. Then send the event update

This ensures value changes are synchronized before the event is processed.

**Note:** The value comparison in step 1 implements the duplicate update suppression requirement - if values are equal, no update is sent for the value binding.

**Why value sync is needed:**
- User may type in a field (changing element value) then press Enter before blur fires
- Without sync, the event would fire with stale variable value
- Value sync ensures backend receives current UI state before processing the event

**Example scenario:**
```
User types in input field, then presses Enter (ui-event-keypress-enter)
1. handleEvent detects keypress-enter event
2. syncValueBinding checks widget for ui-value binding
3. Found: ui-value="name" with variableId 5
4. Get current input value: "John"
5. Get variable 5's current value: "Jo" (not yet synced)
6. Values differ -> send update for variable 5 with "John" (set local value first)
7. Then send event update for the keypress-enter binding
```

## Local Value Setting (Universal Principle)

**Spec: viewdefs.md "Frontend Update Behavior"**

**Whenever the frontend sends a variable update to the backend, it MUST first set the value in the local variable cache.** This ensures the frontend's cached variable state accurately reflects the UI state being sent to the backend.

1. Set `variable.value` locally (MUST do this first)
2. Send update message to backend
3. Backend processes and may send back additional updates

This pattern enables optimistic UI updates while maintaining eventual consistency with the backend. The value sync in `syncValueBinding` follows this same principle.

## Collaborators

- BindingEngine: Creates and manages bindings
- Variable: Target of event updates (value set locally before sending)
- ProtocolHandler: Sends update messages
- Widget: Provides access to sibling bindings (e.g., ui-value on same element)
- WidgetBinder: Widget-specific event handling

## Sequences

- seq-handle-event.md: Event processing flow (with value sync)
- seq-handle-keypress-event.md: Keypress-specific event processing
- seq-bind-element.md: Creating event binding
