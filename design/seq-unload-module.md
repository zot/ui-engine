# Sequence: Unload Module

**Source Spec:** module-tracking.md
**Requirements:** R14, R15, R16, R17, R18, R19, R20

## Participants
- LuaCode: Lua script calling session:unloadModule
- LuaSession: Session managing module state
- Module: Tracks resources for the module
- HotLoader: File watcher (via cleanup callback)

## Flow: session:unloadModule(moduleName)

```
LuaCode                    LuaSession                      Module              HotLoader
   |                           |                              |                     |
   |--unloadModule(name)------>|                              |                     |
   |                           |                              |                     |
   |                           |--lookup modules[name]------->|                     |
   |                           |<---module or nil-------------|                     |
   |                           |                              |                     |
   |                           | [if module exists]           |                     |
   |                           |                              |                     |
   |                           | for each module.prototypes:  |                     |
   |                           |--RemovePrototype(protoName)--|                     |
   |                           |                              |                     |
   |                           | for each module.presenterTypes:|                   |
   |                           |--delete presenterTypes[name]-|                     |
   |                           |                              |                     |
   |                           | for each module.wrappers:    |                     |
   |                           |--wrapperRegistry.Remove(name)|                     |
   |                           |                              |                     |
   |                           |--viewdefManager.UnloadModule(name)                 |
   |                           |                              |                     |
   |                           |--loadedModules[name] = nil---|                     |
   |                           |                              |                     |
   |                           |--hotLoaderCleanup(name)------|----------------->   |
   |                           |                              |     CleanupModule   |
   |                           |                              |     - remove watch  |
   |                           |                              |     - del symlink   |
   |                           |                              |     - del pending   |
   |                           |                              |                     |
   |                           |--remove from moduleDirectories                     |
   |                           |--delete modules[name]--------|                     |
   |<----(returns)-------------|                              |                     |
```

## Flow: session:unloadDirectory(dirPath)

```
LuaCode                    LuaSession                      Module              HotLoader
   |                           |                              |                     |
   |--unloadDirectory(path)--->|                              |                     |
   |                           |                              |                     |
   |                           |--lookup moduleDirectories[path]                    |
   |                           |<---modules list or nil-------|                     |
   |                           |                              |                     |
   |                           | [if modules exist]           |                     |
   |                           |                              |                     |
   |                           | for each module in list:     |                     |
   |                           |--unloadModule(module.name)-->|  (see above flow)   |
   |                           |                              |                     |
   |                           |--hotLoaderCleanup(dirPath)---|----------------->   |
   |                           |                              |   CleanupDirectory  |
   |                           |                              |   - remove dir watch|
   |                           |                              |   - del all symlinks|
   |                           |                              |   - del all pending |
   |                           |                              |                     |
   |                           |--delete moduleDirectories[path]                    |
   |<----(returns)-------------|                              |                     |
```

## Notes

- unloadModule is idempotent: calling with unknown module name is a no-op
- unloadDirectory calls unloadModule for each module, then cleans up directory-level state
- HotLoader cleanup is done via callback to avoid circular dependency
- Existing instances of removed prototypes keep their metatables (per R7)
