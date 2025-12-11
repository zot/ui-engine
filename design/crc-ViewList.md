# ViewList

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- element: Container DOM element for the list
- variable: Variable bound to array of object references
- namespace: Viewdef namespace for child views (default: DEFAULT)
- exemplar: Element to clone for each item (default: div)
- views: Parallel array of View elements
- delegate: Optional delegate for add/remove notifications

### Does
- create: Initialize from element with ui-viewlist attribute
- setExemplar: Set element to clone for list items (e.g., sl-option)
- update: Sync views array with bound variable array
- addItem: Clone exemplar, create variable, render and append
- removeItem: Destroy variable, remove element from DOM
- reorder: Reorder view elements to match array order
- clear: Remove all items
- setDelegate: Set delegate for notifications
- notifyAdd: Notify delegate of item addition
- notifyRemove: Notify delegate of item removal

## Collaborators

- View: Individual view elements in the list
- ViewRenderer: Creates ViewLists
- BindingEngine: Binds list to variable
- Variable: Source array of object references

## Sequences

- seq-viewlist-update.md: Array change handling
- seq-render-view.md: ViewList rendering within views
