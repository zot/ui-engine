# Presenter

**Source Spec:** main.md

## Responsibilities

### Knows
- type: Presenter type name (e.g., "form", "table", "chart")
- data: JSON object holding presenter state
- viewName: Currently active view (default: "DEFAULT")
- variableId: Associated variable ID

### Does
- getData: Return presenter state object
- setData: Update presenter state
- getType: Return presenter type
- getViewName: Return active view name
- setViewName: Switch to different view
- toVariable: Convert to variable with type property

## Collaborators

- Variable: Stored as variable value
- VariableStore: Manages persistence
- Viewdef: Renders presenter with appropriate view
- ViewdefStore: Looks up view by TYPE.VIEW

## Sequences

- seq-create-presenter.md: Creating presenter instance
- seq-render-view.md: Rendering presenter with viewdef
