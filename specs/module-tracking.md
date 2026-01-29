# Module Tracking and Unloading

## Purpose

Enable tracking of resources per-module and support module/directory unloading at runtime. This supports scenarios where application modules are dynamically loaded/unloaded, such as during development hot-reloading or runtime plugin management.

## Definitions

- **Module**: A Lua file that registers resources (prototypes, presenter types, wrappers) with the session
- **Directory**: A folder containing one or more modules (e.g., `apps/contacts/`)

## Features

### Module Class (Go)

A new struct that tracks resources registered by a single module.

**Tracks:**
- `prototypes`: List of prototype names registered by this module
- `presenterTypes`: List of presenter type names registered by this module
- `wrappers`: List of wrapper names registered by this module

### LuaSession.moduleDirectories Field

```go
moduleDirectories map[string][]*Module // directory path -> modules in that directory
```

Tracks the relationship between directories and the modules they contain. A directory can contain multiple modules.

### LuaSession.unloadDirectory(name) Method

Unloads all modules in a directory. Exposed to Lua via `session:unloadDirectory(name)`.

**Behavior:**
- Looks up all modules in `moduleDirectories[name]`
- Calls `unloadModule` for each module
- Removes the directory entry from `moduleDirectories`
- Cleans up hotloader state for the directory:
  - Removes watches for the directory
  - Clears `symlinkTargets` entries for files in the directory
  - Clears `pendingReloads` entries for files in the directory

### LuaSession.unloadModule(moduleName) Method

Removes all tracking related to a module. Exposed to Lua via `session:unloadModule(name)`.

**Removes from LuaSession:**
- Entry from `loadedModules`
- Prototypes from `prototypeRegistry` (using `RemovePrototype`)
- Presenter types from `presenterTypes`
- Wrappers from `wrapperRegistry`
- Viewdefs from `viewdefManager` (if any registered by module)

**Removes from HotLoader (via callback/interface):**
- Watches for the module file
- `symlinkTargets` entry for the module file
- `pendingReloads` entry for the module file

## Integration Points

- Module registration happens during `require()` or `RequireLuaFile()` calls
- Tracking requires the module to identify itself (e.g., via file path)
- HotLoader cleanup requires coordination between LuaSession and HotLoader
