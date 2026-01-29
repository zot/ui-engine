# Requirements

## Feature: Remove Prototype
**Source:** specs/remove-prototype.md

- **R1:** LuaSession must have a RemovePrototype(name string, children bool) method
- **R2:** RemovePrototype removes the named prototype from prototypeRegistry
- **R3:** RemovePrototype removes the corresponding entry from instanceRegistry
- **R4:** When children=true, RemovePrototype removes all prototypes whose name starts with "NAME." (dot-separated children)
- **R5:** When children=false, RemovePrototype removes only the exact named prototype
- **R6:** RemovePrototype returns silently if the prototype doesn't exist (no error)
- **R7:** Existing instances retain their metatables after prototype removal (no instance destruction)

## Feature: Module Tracking and Unloading
**Source:** specs/module-tracking.md

- **R8:** A Module struct must track prototypes, presenterTypes, and wrappers registered by a single module
- **R9:** LuaSession must have a moduleDirectories field mapping directory paths to their modules
- **R10:** LuaSession must have an unloadDirectory(name) method exposed to Lua as session:unloadDirectory(name)
- **R11:** unloadDirectory must call unloadModule for each module in the directory
- **R12:** unloadDirectory must remove the directory entry from moduleDirectories after unloading modules
- **R13:** unloadDirectory must clean up HotLoader state: watchers, symlinkTargets, pendingReloads for the directory
- **R14:** LuaSession must have an unloadModule(moduleName) method exposed to Lua as session:unloadModule(name)
- **R15:** unloadModule must remove the module's entry from loadedModules
- **R16:** unloadModule must remove prototypes registered by the module from prototypeRegistry (using RemovePrototype)
- **R17:** unloadModule must remove presenter types registered by the module from presenterTypes
- **R18:** unloadModule must remove wrappers registered by the module from wrapperRegistry
- **R19:** unloadModule must remove viewdefs registered by the module from viewdefManager
- **R20:** unloadModule must clean up HotLoader state for the module file: watches, symlinkTargets, pendingReloads
- **R21:** (inferred) Module registration must happen during require() or RequireLuaFile() calls to track resources
- **R22:** (inferred) A directory can contain multiple modules
