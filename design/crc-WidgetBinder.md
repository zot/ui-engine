# WidgetBinder

**Source Spec:** libraries.md, components.md

## Responsibilities

### Knows
- widgetTypes: Map of widget tag to binding handler
- shoelaceBindings: Shoelace-specific binding rules
- tabulatorBindings: Tabulator grid binding rules
- shoelaceInputTags: Set of Shoelace input tags (`sl-input`, `sl-textarea`)

### Does
- bindWidget: Apply widget-specific bindings (called by BindingEngine for widget elements)
- bindShoelaceInput: Handle sl-input ui-value, ui-disabled with event selection
- bindShoelaceTextarea: Handle sl-textarea (same as sl-input)
- bindShoelaceButton: Handle sl-button ui-action
- bindShoelaceSelect: Handle sl-select ui-items, ui-index, ui-namespace
- bindTabulator: Handle ui-tabulator, ui-columns
- bindDivContent: Handle ui-content HTML binding
- bindDivView: Handle ui-view, ui-namespace object binding
- bindDivViewList: Handle ui-viewlist, ui-namespace array binding
- bindDynamicViewdef: Handle ui-viewdef computed viewdef
- selectShoelaceEvent: Choose `sl-change` or `sl-input` based on `keypress` option

## Widget-BindingEngine Integration

BindingEngine calls `bindWidget()` for recognized widget tags, passing:
- Element reference
- Variable reference
- Bindings map (attribute name to value)
- Path options (parsed from path, including `keypress`)

For `sl-input` and `sl-textarea`, the `keypress` option determines the event:
- `keypress` absent or `false`: Listen to `sl-change` (fires on blur)
- `keypress` present or `true`: Listen to `sl-input` (fires on every keystroke)

## Collaborators

- BindingEngine: General binding coordination (calls bindWidget for Shoelace elements)
- ViewRenderer: Element creation
- ValueBinding: Value updates
- EventBinding: Event handling
- PathSyntax: Path options passed through from BindingEngine

## Sequences

- seq-bind-element.md: Widget binding flow
- seq-render-view.md: Widget rendering
- seq-handle-event.md: Widget event handling
- seq-input-value-binding.md: Shoelace input event selection
