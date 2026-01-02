# EventBinding

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: ID of source element (NOT direct DOM reference)
- eventType: DOM event name (click, change, input, etc.)
- variableId: Target variable ID
- actionPath: Path for action triggers (e.g., "submit()")

### Does
- attach: Add event listener to element (looked up by ID)
- detach: Remove event listener
- handleEvent: Process DOM event and update variable
- extractEventValue: Get relevant value from event (input value, click data, etc.)
- isAction: Check if binding triggers an action vs. value update
- getElement: Look up DOM element by elementId (via document.getElementById)
- destroy: Clean up listener

## Collaborators

- BindingEngine: Creates and manages bindings
- Variable: Target of event updates
- ProtocolHandler: Sends update messages
- WidgetBinder: Widget-specific event handling

## Sequences

- seq-handle-event.md: Event processing flow
- seq-bind-element.md: Creating event binding
