# Test Design: LuaHotLoader

**Source Specs**: deployment.md (Lua Hot-Loading section), libraries.md (Prototype Management)
**CRC Cards**: crc-LuaHotLoader.md, crc-LuaSession.md
**Sequences**: seq-lua-hotload.md, seq-server-startup.md, seq-prototype-mutation.md

## Overview

Tests for the Lua hot-loading component that watches lua directory for file changes and reloads modified files in all active sessions, including prototype mutation processing.

## Test Cases

> Symlink-tracking tests retired 2026-10-07 (T1, T2): apps are watched through their `lua/<app>` entry. Per-app watching, re-discovery and collision tests live in [test-AppDiscovery.md](test-AppDiscovery.md).


### Test: Initialize hot loader

**Purpose**: Verify hot loader creation with watcher

**Input**:
- NewHotLoader(config, luaDir, getSessions)

**References**:
- CRC: crc-LuaHotLoader.md - "Knows: luaDir, watcher"

**Expected Results**:
- HotLoader instance created
- fsnotify watcher initialized
- luaDir stored correctly
- getSessions callback stored

---

### Test: Start watching lua directory

**Purpose**: Verify watcher starts monitoring lua directory

**Input**:
- hotLoader.Start()

**References**:
- CRC: crc-LuaHotLoader.md - "Does: Start"
- Sequence: seq-server-startup.md

**Expected Results**:
- Lua directory added to watch list
- Event loop started
- Debounce loop started
- Log message indicates watching started

---

### Test: Detect .lua file modification

**Purpose**: Verify write events trigger reload

**Input**:
- Modify existing lua/app.lua file

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"
- Sequence: seq-lua-hotload.md

**Expected Results**:
- Write event detected
- File queued for reload
- Only .lua files processed

---

### Test: Detect .lua file creation

**Purpose**: Verify create events trigger reload

**Input**:
- Create new lua/new.lua file

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"

**Expected Results**:
- Create event detected
- File queued for reload
- Symlink check performed for new file

---

### Test: Ignore non-lua files

**Purpose**: Verify only .lua files are processed

**Input**:
- Modify lua/notes.txt
- Modify lua/data.json

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"

**Expected Results**:
- Events ignored
- No reload triggered
- No errors logged

---

### Test: Debounce rapid file changes

**Purpose**: Verify multiple rapid changes result in single reload

**Input**:
- Modify lua/app.lua 5 times within 50ms

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"
- Sequence: seq-lua-hotload.md - "debounce 100ms"

**Expected Results**:
- Only one reload triggered
- Reload occurs after debounce delay (100ms)
- No duplicate reloads

---

### Test: Debounce delay per file

**Purpose**: Verify debouncing is per-file, not global

**Input**:
- Modify lua/app.lua
- Wait 50ms
- Modify lua/utils.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"

**Expected Results**:
- Both files reloaded
- app.lua reloaded ~100ms after its last change
- utils.lua reloaded ~100ms after its last change

---

### Test: Reload in all active sessions

**Purpose**: Verify modified file is reloaded in every session

**Input**:
- 3 active LuaSessions
- Modify lua/app.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"
- CRC: crc-LuaHotLoader.md - "Collaborators: Server"
- Sequence: seq-lua-hotload.md

**Expected Results**:
- getSessions() called
- LoadCode called on all 3 sessions
- Each session receives updated code

---

### Test: Reload with no active sessions

**Purpose**: Verify reload handles empty session list gracefully

**Input**:
- 0 active sessions
- Modify lua/app.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"

**Expected Results**:
- No error
- Reload detected and logged
- No LoadCode calls (empty list)

---

### Test: Reload error in one session

**Purpose**: Verify errors in one session don't affect others

**Input**:
- 2 active sessions
- Session 1 has Lua error on reload
- Session 2 is healthy

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"
- Sequence: seq-lua-hotload.md - "Error Handling"

**Expected Results**:
- Session 1 error logged
- Session 2 still reloaded successfully
- No crash or panic

---

### Test: Config option enables hot loading

**Purpose**: Verify --hotload flag controls feature

**Input**:
- Config with lua.hotload = true
- Server startup

**References**:
- CRC: crc-LuaHotLoader.md - "Collaborators: Config"
- Sequence: seq-server-startup.md

**Expected Results**:
- HotLoader created and started
- Watching active

---

### Test: Config option disables hot loading

**Purpose**: Verify hot loading disabled when not configured

**Input**:
- Config with lua.hotload = false (default)
- Server startup

**References**:
- CRC: crc-LuaHotLoader.md - "Collaborators: Config"

**Expected Results**:
- HotLoader not created
- No file watching
- No performance overhead

---

### Test: Graceful shutdown stops watcher

**Purpose**: Verify Stop cleans up resources

**Input**:
- hotLoader.Stop()

**References**:
- CRC: crc-LuaHotLoader.md - "Does: Stop"

**Expected Results**:
- done channel closed
- Event loop exits
- Debounce loop exits
- Watcher closed
- No goroutine leaks

---

### Test: Shutdown with pending reloads

**Purpose**: Verify pending reloads are discarded on shutdown

**Input**:
- Modify lua/app.lua
- Immediately call Stop() (before debounce fires)

**References**:
- CRC: crc-LuaHotLoader.md - "Does: Stop"

**Expected Results**:
- Stop completes cleanly
- Pending reload discarded (not an error)
- No race conditions

---

### Test: Handle deleted file gracefully

**Purpose**: Verify deleted files don't cause errors

**Input**:
- Modify lua/app.lua
- Delete lua/app.lua before debounce fires

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"

**Expected Results**:
- File read fails
- Error logged
- No crash
- Continue processing other files

---

### Test: Handle watcher errors

**Purpose**: Verify watcher errors are logged

**Input**:
- Watcher encounters filesystem error

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"
- CRC: crc-LuaHotLoader.md - "Collaborators: fsnotify"

**Expected Results**:
- Error logged
- Watcher continues running
- No crash

---

### Test: Resolve reload path for direct file

**Purpose**: Verify direct lua files use their path

**Input**:
- Modify lua/app.lua (not a symlink)

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"

**Expected Results**:
- Reload path is lua/app.lua
- File content read from lua/app.lua
- Code name is "app.lua"

---

## Prototype Mutation Integration Tests

### Test: Process mutation queue after file reload

**Purpose**: Verify mutation queue processed after LoadFileAbsolute

**Input**:
```lua
-- Initial app.lua
Person = session:prototype("Person", { name = "" })
local p = Person:new({ name = "Alice" })
```
- Modify app.lua to add email field
- Hot-reload triggered

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"
- CRC: crc-LuaSession.md - "Does: processMutationQueue"
- Sequence: seq-lua-hotload.md

**Expected Results**:
- LoadFileAbsolute called
- session:prototype queues Person for mutation
- processMutationQueue called after file execution
- Instance p updated

---

### Test: Multiple prototypes mutated in order

**Purpose**: Verify FIFO order across hot-reload

**Input**:
```lua
-- app.lua declares Address then Person
Address = session:prototype("Address", { city = "" })
Person = session:prototype("Person", { name = "" })
```
- Hot-reload adds fields to both

**References**:
- Sequence: seq-lua-hotload.md
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Address processed before Person
- Dependency order maintained
- All instances updated

---

### Test: Mutation errors don't break reload

**Purpose**: Verify reload completes despite mutation errors

**Input**:
```lua
Bad = session:prototype("Bad", { x = 0 })
function Bad:mutate() error("fail") end
local b = Bad:new()
```
- Hot-reload modifies Bad

**References**:
- Sequence: seq-lua-hotload.md
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Error logged
- Reload considered successful
- Other sessions not affected

---

### Test: Empty mutation queue after reload

**Purpose**: Verify queue cleared even when empty

**Input**:
- Hot-reload file with no prototype changes

**References**:
- CRC: crc-LuaSession.md - "Does: processMutationQueue"

**Expected Results**:
- processMutationQueue called
- No-op when queue empty
- No error

---

### Test: Multiple files hot-reloaded sequentially

**Purpose**: Verify each file gets its own mutation processing

**Input**:
- Modify models.lua (defines Person)
- Modify views.lua (defines PersonView)
- Both reloaded in sequence

**References**:
- Sequence: seq-lua-hotload.md

**Expected Results**:
- models.lua loaded, mutation queue processed
- views.lua loaded, mutation queue processed
- Queue cleared between files

---

## Coverage Summary

**Responsibilities Covered:**
- LuaHotLoader Knows: luaDir, watcher, appWatches, pendingReloads
- LuaHotLoader Does: Start, Stop, handleFileChange, resolveSymlinks, updateSymlinkWatches, reloadFile
- LuaSession: processMutationQueue integration

**Scenarios Covered:**
- seq-lua-hotload.md: File change detection, debouncing, all-session reload, prototype management
- seq-lua-hotload.md (Symlink Resolution): Symlink tracking, target watching
- seq-prototype-mutation.md: Post-load processing integration

**Collaborators Tested:**
- Server: GetLuaSessions() integration
- LuaSession: LoadCode() calls, processMutationQueue() calls
- Config: lua.hotload setting, verbosity logging
- fsnotify: Event handling, error handling

**Gaps**: None identified

---

## Viewdef HotLoader Tests

**Source Specs**: viewdefs.md (Hot-reload re-rendering), main.md (Hot-Loading System)
**CRC Cards**: crc-ViewdefStore.md
**Sequences**: seq-viewdef-hotload.md
**Implementation**: `internal/viewdef/hotloader_test.go`

### Test: Initialize viewdef hot loader

**Purpose**: Verify viewdef hot loader creation with watcher

**Input**:
- NewHotLoader(config, viewdefDir, manager, sessions)

**References**:
- CRC: crc-ViewdefStore.md - "Knows: fileWatcher"

**Expected Results**:
- HotLoader instance created
- fsnotify watcher initialized
- viewdefDir stored correctly
- manager and sessions callbacks stored

---

### Test: Start watching viewdef directory

**Purpose**: Verify watcher starts monitoring viewdef directory

**Input**:
- hotLoader.Start()

**References**:
- CRC: crc-ViewdefStore.md - "Does: startWatching"
- Sequence: seq-viewdef-hotload.md

**Expected Results**:
- Viewdef directory added to watch list
- Event loop started
- Debounce loop started
- Log message indicates watching started

---

### Test: Detect .html file modification

**Purpose**: Verify write events trigger reload

**Input**:
- Modify existing viewdefs/Contact.html file

**References**:
- CRC: crc-ViewdefStore.md - "Does: handleFileChange"
- Sequence: seq-viewdef-hotload.md

**Expected Results**:
- Write event detected
- File queued for reload
- Only .html files processed

---

### Test: Ignore non-html files

**Purpose**: Verify only .html files are processed

**Input**:
- Modify viewdefs/styles.css
- Modify viewdefs/data.json

**References**:
- CRC: crc-ViewdefStore.md - "Does: handleFileChange"

**Expected Results**:
- Events ignored
- No reload triggered
- No errors logged

---

### Test: Debounce rapid viewdef changes

**Purpose**: Verify multiple rapid changes result in single reload

**Input**:
- Modify viewdefs/Contact.html 5 times within 50ms

**References**:
- CRC: crc-ViewdefStore.md - "Does: handleFileChange"

**Expected Results**:
- Only one reload triggered
- Reload occurs after debounce delay (100ms)
- No duplicate reloads

---

### Test: Push only to sessions that received viewdef

**Purpose**: Verify updates only go to affected sessions

**Input**:
- 3 active sessions
- Session 1 and 2 have received Contact viewdef
- Session 3 has not
- Modify viewdefs/Contact.html

**References**:
- CRC: crc-ViewdefStore.md - "Knows: sentViewdefs"
- CRC: crc-ViewdefStore.md - "Does: handleFileChange"
- Sequence: seq-viewdef-hotload.md

**Expected Results**:
- Sessions 1 and 2 receive push
- Session 3 does not receive push
- Push includes updated viewdef content

---

### Test: Update viewdef manager on reload

**Purpose**: Verify viewdef manager gets updated content

**Input**:
- Modify viewdefs/Contact.html

**References**:
- CRC: crc-ViewdefStore.md - "Does: handleFileChange"
- CRC: crc-ViewdefStore.md - "Collaborators: ViewdefManager"

**Expected Results**:
- manager.updateViewdef() called
- Viewdef key derived from filename (Contact)
- New content stored in manager

---

### Test: Viewdef graceful shutdown

**Purpose**: Verify Stop cleans up resources

**Input**:
- hotLoader.Stop()

**References**:
- CRC: crc-ViewdefStore.md - "Does: stopWatching"

**Expected Results**:
- done channel closed
- Event loop exits
- Debounce loop exits
- Watcher closed
- No goroutine leaks

---

### Test: Viewdef shutdown with pending reloads

**Purpose**: Verify pending reloads are discarded on shutdown

**Input**:
- Modify viewdefs/Contact.html
- Immediately call Stop() (before debounce fires)

**References**:
- CRC: crc-ViewdefStore.md - "Does: stopWatching"

**Expected Results**:
- Stop completes cleanly
- Pending reload discarded
- No race conditions

---

### Test: Handle deleted viewdef file

**Purpose**: Verify deleted files don't cause errors

**Input**:
- Modify viewdefs/Contact.html
- Delete viewdefs/Contact.html before debounce fires

**References**:
- CRC: crc-ViewdefStore.md - "Does: handleFileChange"

**Expected Results**:
- File read fails
- Error logged
- No crash
- Continue processing other files

---

## Viewdef Coverage Summary

**Responsibilities Covered:**
- ViewdefStore Knows: fileWatcher, sentViewdefs, watchedSources
- ViewdefStore Does: startWatching, stopWatching, handleFileChange, resolveSymlinks, updateSymlinkWatches

**Scenarios Covered:**
- seq-viewdef-hotload.md: File change detection, debouncing, selective session push

**Collaborators Tested:**
- ViewdefManager: updateViewdef() integration
- SessionPusher: PushViewdefs() calls, GetSessionIDs() calls
- Config: verbosity logging
- fsnotify: Event handling, error handling

**Gaps**: None identified
