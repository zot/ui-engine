# Remove Prototype Feature

## Purpose

Enable removal of prototypes from LuaSession at runtime. This supports scenarios where modules are unloaded or prototypes need to be replaced entirely (not just updated).

## Feature

`LuaSession.RemovePrototype(name string, children bool)` removes a registered prototype by name.

### Behavior

- Removes the prototype with the given `name` from `prototypeRegistry`
- Removes corresponding entry from `instanceRegistry`
- If `children` is true, also removes any prototype whose name starts with `NAME.` (dot-separated children)
  - Example: `RemovePrototype("contacts", true)` removes "contacts", "contacts.Person", "contacts.Address", etc.
  - Example: `RemovePrototype("contacts", false)` removes only "contacts"
- If prototype doesn't exist, the method returns silently (no error)
- Does NOT destroy existing instances - they retain their metatables but won't receive future mutations
