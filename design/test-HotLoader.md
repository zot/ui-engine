# Test Design: LuaHotLoader

**Source Specs**: deployment.md (Lua Hot-Loading section)
**CRC Cards**: crc-LuaHotLoader.md
**Sequences**: seq-lua-hotload.md, seq-server-startup.md

## Overview

Tests for the Lua hot-loading component that watches lua directory for file changes and reloads modified files in all active sessions.

## Test Cases

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

### Test: Scan for existing symlinks on start

**Purpose**: Verify symlinks are detected during initialization

**Input**:
- lua/app.lua -> ../apps/myapp/app.lua (symlink)
- hotLoader.Start()

**References**:
- CRC: crc-LuaHotLoader.md - "Does: resolveSymlinks"
- CRC: crc-LuaHotLoader.md - "Knows: symlinkTargets"

**Expected Results**:
- Symlink resolved to target
- Target directory (../apps/myapp/) added to watch list
- symlinkTargets map updated

---

### Test: Watch symlink target directory

**Purpose**: Verify changes to symlink targets are detected

**Input**:
- lua/app.lua -> /real/path/app.lua
- Modify /real/path/app.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: handleFileChange"
- Sequence: seq-lua-hotload.md (Symlink Resolution Sequence)

**Expected Results**:
- Change detected in target directory
- Resolved back to lua/app.lua
- Reloaded as lua/app.lua (not /real/path/app.lua)

---

### Test: Add symlink watch on create

**Purpose**: Verify new symlinks get their targets watched

**Input**:
- Create symlink lua/new.lua -> /other/path/new.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: updateSymlinkWatches"

**Expected Results**:
- New symlink detected
- Target directory resolved
- Watch added for /other/path/

---

### Test: Remove symlink watch on delete

**Purpose**: Verify symlink removal cleans up watches

**Input**:
- Remove symlink lua/app.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: updateSymlinkWatches"
- CRC: crc-LuaHotLoader.md - "Knows: watchedDirs"

**Expected Results**:
- Remove event detected
- symlinkTargets entry removed
- Target directory watch removed (if refcount reaches 0)

---

### Test: Update symlink watch on rename

**Purpose**: Verify renamed symlinks update watches

**Input**:
- Rename lua/old.lua to lua/new.lua (both symlinks)

**References**:
- CRC: crc-LuaHotLoader.md - "Does: updateSymlinkWatches"

**Expected Results**:
- Old symlink watch removed
- New symlink watch added
- No orphaned watches

---

### Test: Reference counting for shared target directories

**Purpose**: Verify multiple symlinks to same directory share watch

**Input**:
- lua/a.lua -> /shared/a.lua
- lua/b.lua -> /shared/b.lua
- Remove lua/a.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Knows: watchedDirs"

**Expected Results**:
- /shared/ watched once (refcount=2)
- After removing lua/a.lua, refcount=1
- /shared/ still watched (b.lua needs it)

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

### Test: Resolve reload path for symlink target

**Purpose**: Verify symlink target changes resolve to symlink path

**Input**:
- lua/app.lua -> /real/path/app.lua
- Modify /real/path/app.lua

**References**:
- CRC: crc-LuaHotLoader.md - "Does: reloadFile"
- Sequence: seq-lua-hotload.md (Symlink Resolution)

**Expected Results**:
- Changed path is /real/path/app.lua
- Resolved to lua/app.lua
- Content read from lua/app.lua (via symlink)
- Code name is "app.lua"

---

## Coverage Summary

**Responsibilities Covered:**
- LuaHotLoader Knows: luaDir, watcher, symlinkTargets, watchedDirs
- LuaHotLoader Does: Start, Stop, handleFileChange, resolveSymlinks, updateSymlinkWatches, reloadFile

**Scenarios Covered:**
- seq-lua-hotload.md: File change detection, debouncing, all-session reload
- seq-lua-hotload.md (Symlink Resolution): Symlink tracking, target watching

**Collaborators Tested:**
- Server: GetLuaSessions() integration
- LuaSession: LoadCode() calls
- Config: lua.hotload setting, verbosity logging
- fsnotify: Event handling, error handling

**Gaps**: None identified
