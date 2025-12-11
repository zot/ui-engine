# Viewdef

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- type: Type name this viewdef applies to (e.g., Contact)
- namespace: Namespace name (e.g., DEFAULT, COMPACT, OPTION)
- template: Parsed template element
- bindings: Parsed list of binding directives from ui-* attributes

### Does
- getKey: Return TYPE.NAMESPACE identifier string
- getTemplate: Return template element for cloning
- parseBindings: Extract ui-* attributes and their paths
- hasBinding: Check if specific binding type exists
- clone: Deep clone template contents for rendering

## Collaborators

- ViewdefStore: Stores viewdefs by TYPE.NAMESPACE key
- BindingEngine: Applies bindings to DOM elements
- ViewRenderer: Instantiates viewdef for rendering
- View: Uses viewdef to render object reference

## Sequences

- seq-load-viewdefs.md: Loading and validating viewdefs from variable 1
- seq-render-view.md: Rendering views with viewdefs
- seq-bind-element.md: Applying bindings to elements
