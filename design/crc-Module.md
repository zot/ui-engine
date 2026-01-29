# Module

**Source Spec:** module-tracking.md
**Requirements:** R8, R21, R22

## Responsibilities

### Knows
- name: The module's tracking key (baseDir-relative file path)
- directory: The directory containing this module
- prototypes: List of prototype names registered by this module
- presenterTypes: List of presenter type names registered by this module
- wrappers: List of wrapper names registered by this module

### Does
- AddPrototype(name): Track a prototype registered by this module
- AddPresenterType(name): Track a presenter type registered by this module
- AddWrapper(name): Track a wrapper registered by this module

## Collaborators

- LuaSession: Creates and owns Module instances; uses Module data during unload

## Notes

- Module instances are created during `require()` or `RequireLuaFile()` execution
- The module's tracking key is the baseDir-relative resolved path (e.g., `apps/contacts/app.lua`)
- Module is a simple data holder; LuaSession performs the actual registration and cleanup
