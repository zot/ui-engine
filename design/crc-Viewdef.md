# Viewdef

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- type: Presenter type this viewdef applies to
- viewName: View name (e.g., "DEFAULT", "list-item")
- html: HTML template string with ui-* bindings
- bindings: Parsed list of binding directives from ui-* attributes

### Does
- getKey: Return TYPE.VIEW identifier string
- getHtml: Return raw HTML template
- parseBindings: Extract ui-* attributes and their paths
- hasBinding: Check if specific binding type exists
- clone: Create copy of viewdef for modification

## Collaborators

- ViewdefStore: Stores viewdefs by TYPE.VIEW key
- BindingEngine: Applies bindings to DOM elements
- ViewRenderer: Instantiates viewdef for presenter

## Sequences

- seq-load-viewdefs.md: Loading viewdefs from variable 1
- seq-bind-element.md: Applying bindings to elements
