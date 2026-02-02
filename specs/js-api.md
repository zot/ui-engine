# JavaScript API

The frontend exposes a JavaScript API for programmatic interaction with the UI system.

## Global Access

The `UIApp` instance is exposed as `window.uiApp` after initialization. This allows:
- Console debugging during development
- External scripts to interact with the UI system
- Custom widgets to send updates programmatically

## UIApp.updateValue(elementId, value?)

Updates the `ui-value` binding variable for an element.

**Parameters:**
- `elementId`: The DOM element's ID
- `value` (optional): The new value. If undefined, uses the element's current `value` property.

**Behavior:**
- Looks up the widget for the element
- Gets the variable ID for the `ui-value` binding
- Sends an update with the provided value (or element's value if undefined)
- No-op if element has no `ui-value` binding

**Use cases:**
- Custom components that need to notify the backend of value changes
- Integration with third-party libraries that don't dispatch standard events
- Programmatic value updates from custom JavaScript code

**Example:**
```javascript
// Update from element's current value
window.uiApp.updateValue('my-input')

// Update with specific value
window.uiApp.updateValue('my-input', 'new value')
```
