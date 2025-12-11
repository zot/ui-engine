# BindingEngine

**Source Spec:** viewdefs.md, libraries.md

## Responsibilities

### Knows
- activeBindings: Map of element to list of active bindings
- pathCache: Cache of resolved paths for performance

### Does
- bind: Apply all ui-* bindings to an element
- unbind: Remove all bindings from an element
- createValueBinding: Create ui-value, ui-attr-*, ui-class-*, ui-style-*-* binding
- createEventBinding: Create ui-event-* binding
- parsePath: Parse path with optional URL-style properties (?create=Type&prop=value)
- updateBinding: Refresh binding when variable changes
- resolveBindingPath: Navigate path in presenter data structure (uses nullish coalescing)

## Nullish Path Handling

Bindings gracefully handle nullish paths (via PathNavigator):
- **Read direction:** Display empty/default value when path segment is null/undefined (no error)
- **Write direction:** Issue `error` message with code `path-failure` when intermediate path segment is nullish, allowing UI to show error state (e.g., red border). Error clears on successful update.

Example: `ui-value="selectedContact.firstName"` works when `selectedContact` is null (shows empty).
When user attempts to edit a field with a nullish path, the field shows an error indicator until the path becomes valid.

## Collaborators

- ValueBinding: Handles variable-to-element bindings
- EventBinding: Handles element-to-variable bindings
- Viewdef: Source of binding directives
- Variable: Target of bindings
- WatchManager: Subscribes to variable changes
- View: Handles ui-view bindings
- ViewList: Handles ui-viewlist bindings

## Sequences

- seq-bind-element.md: Element binding process
- seq-handle-event.md: Event to variable flow
- seq-render-view.md: Full view rendering
- seq-viewlist-update.md: ViewList array updates
