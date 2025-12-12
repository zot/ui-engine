# LuaVariable

**Source Spec:** libraries.md

## Responsibilities

### Knows
- id: Variable ID (integer)
- luaObject: Reference to the live Lua table this variable points to
- session: Back-reference to LuaSession for operations

### Does
- getId: Return the variable ID
- getObject: Return the live Lua object (table) this variable references
- getProperty: Return property value by name (delegates to session/store)
- updateProperties: Update only properties (not value) - for metadata like viewdefs

## Collaborators

- LuaSession: Provides backend operations, stores object references for change detection

## Sequences

- seq-lua-session-init.md: App variable created via session:createAppVariable()
- seq-viewdef-delivery.md: App variable wrapper used to set viewdefs property
- seq-lua-handle-action.md: Variable values accessed during action handling
- seq-backend-refresh.md: Change detection computes JSON from referenced object

## Notes

- **No update() method for values**: Backend code modifies Lua objects directly
- LuaSession tracks object references and auto-detects changes after batch processing
- updateProperties() still exists for metadata changes (viewdefs, type, etc.)
- The Lua object is the single source of truth; variables just reference it
