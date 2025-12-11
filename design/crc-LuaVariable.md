# LuaVariable

**Source Spec:** libraries.md

## Responsibilities

### Knows
- id: Variable ID (integer)
- session: Back-reference to LuaSession for operations

### Does
- getId: Return the variable ID
- getValue: Return current value (delegates to session/store)
- getProperty: Return property value by name (delegates to session/store)
- update: Update value (and optionally properties)
- updateProperties: Update only properties (not value)

## Collaborators

- LuaSession: Provides backend operations and caching

## Sequences

- seq-lua-session-init.md: App variable created via session:createAppVariable()
- seq-viewdef-delivery.md: App variable wrapper used to set viewdefs property
- seq-lua-handle-action.md: Variable values accessed during action handling
