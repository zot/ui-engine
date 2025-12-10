# ValueBinding

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- element: Target DOM element
- variableId: Bound variable ID
- bindingType: One of value, attr, class, style
- attributeName: For attr/class/style, the specific attribute
- path: Path property value for backend use

### Does
- apply: Set element property from variable value
- update: Refresh element when variable changes
- getTargetProperty: Determine which element property to set
- transformValue: Apply any value transformations
- destroy: Clean up binding and unwatch variable

## Collaborators

- BindingEngine: Creates and manages bindings
- Variable: Source of bound value
- WatchManager: Notifies of value changes
- WidgetBinder: Handles widget-specific bindings

## Sequences

- seq-bind-element.md: Creating value binding
- seq-update-variable.md: Propagating value changes
