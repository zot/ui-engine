# WidgetBinder

**Source Spec:** libraries.md, components.md

## Responsibilities

### Knows
- widgetTypes: Map of widget tag to binding handler
- shoelaceBindings: Shoelace-specific binding rules
- tabulatorBindings: Tabulator grid binding rules

### Does
- bindWidget: Apply widget-specific bindings
- bindShoelaceInput: Handle sl-input ui-value, ui-disabled
- bindShoelaceButton: Handle sl-button ui-action
- bindShoelaceSelect: Handle sl-select ui-items, ui-index, ui-namespace
- bindTabulator: Handle ui-tabulator, ui-columns
- bindDivContent: Handle ui-content HTML binding
- bindDivView: Handle ui-view, ui-namespace object binding
- bindDivViewList: Handle ui-viewlist, ui-namespace array binding
- bindDynamicViewdef: Handle ui-viewdef computed viewdef

## Collaborators

- BindingEngine: General binding coordination
- ViewRenderer: Element creation
- ValueBinding: Value updates
- EventBinding: Event handling

## Sequences

- seq-bind-element.md: Widget binding flow
- seq-render-view.md: Widget rendering
- seq-handle-event.md: Widget event handling
