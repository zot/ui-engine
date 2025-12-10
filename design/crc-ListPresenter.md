# ListPresenter

**Source Spec:** main.md

## Responsibilities

### Knows
- items: Array of objects in the list
- selectionIndex: Index of currently selected item (-1 if none)
- disabled: Whether list interaction is disabled (default: false)

### Does
- getItems: Return items array
- setItems: Replace items array
- addItem: Append item to list
- removeItem: Remove item at index
- getSelectedItem: Return items[selectionIndex] or null
- setSelectionIndex: Update selection
- isDisabled: Return disabled state
- setDisabled: Enable/disable interaction

## Collaborators

- Presenter: Base presenter functionality
- ViewdefStore: Renders with list view
- BindingEngine: Binds to ui-viewlist elements

## Sequences

- seq-create-presenter.md: Creating list presenter
- seq-render-view.md: Rendering list items
