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
- resolveBindingPath: Navigate path in presenter data structure

## Collaborators

- ValueBinding: Handles variable-to-element bindings
- EventBinding: Handles element-to-variable bindings
- Viewdef: Source of binding directives
- Variable: Target of bindings
- WatchManager: Subscribes to variable changes

## Sequences

- seq-bind-element.md: Element binding process
- seq-handle-event.md: Event to variable flow
- seq-render-view.md: Full view rendering
