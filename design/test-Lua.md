# Test Design: Lua Runtime

**Source Specs**: interfaces.md, deployment.md
**CRC Cards**: crc-LuaSession.md, crc-LuaPresenterLogic.md
**Sequences**: seq-load-lua-code.md, seq-lua-handle-action.md

## Overview

Tests for embedded Lua backend supporting presentation logic.

## Test Cases

### Test: Initialize Lua runtime

**Purpose**: Verify Lua VM creation

**Input**:
- initialize()

**References**:
- CRC: crc-LuaSession.md - "Does: CreateLuaSession"

**Expected Results**:
- Lua VM state created
- Standard libraries loaded
- Ready for code execution

---

### Test: Load Lua file

**Purpose**: Verify file loading from lua/ directory

**Input**:
- loadFile("presenters.lua")

**References**:
- CRC: crc-LuaSession.md - "Does: ExecuteInSession"
- Sequence: seq-load-lua-code.md

**Expected Results**:
- File found in lua/ directory
- Code parsed and executed
- No syntax errors

---

### Test: Load Lua code string

**Purpose**: Verify dynamic code loading

**Input**:
- loadCode("function Test:hello() return 'hi' end")

**References**:
- CRC: crc-LuaSession.md - "Does: ExecuteInSession"
- Sequence: seq-load-lua-code.md

**Expected Results**:
- Code parsed and executed
- Function defined
- Callable from runtime

---

### Test: Register presenter type

**Purpose**: Verify type registration

**Input**:
- Lua code defines StockTicker type
- registerPresenterType("StockTicker")

**References**:
- CRC: crc-LuaSession.md - "Knows: presenterTypes"

**Expected Results**:
- Type available for create property
- Methods accessible
- Properties bindable

---

### Test: Call method on Lua presenter

**Purpose**: Verify method invocation

**Input**:
- Presenter with update(symbol, price) method
- callMethod(presenter, "update", ["ACME", 142.50])

**References**:
- CRC: crc-LuaSession.md - "Does: ExecuteInSession"
- Sequence: seq-lua-handle-action.md

**Expected Results**:
- Method invoked
- Arguments passed correctly
- Return value available

---

### Test: Get presenter value

**Purpose**: Verify value retrieval

**Input**:
- Presenter with symbol property = "ACME"
- getPresenterValue(presenter, "symbol")

**References**:
- CRC: crc-LuaSession.md - "Does: getApp"

**Expected Results**:
- Returns "ACME"
- Correct type (string)
- Nil for missing properties

---

### Test: Set presenter value

**Purpose**: Verify value setting

**Input**:
- setPresenterValue(presenter, "price", 150.00)

**References**:
- CRC: crc-LuaSession.md - "Does: HandleFrontendUpdate"

**Expected Results**:
- Property updated
- Change detectable
- Watchers notified

---

### Test: Define presenter type in Lua

**Purpose**: Verify type definition API

**Input**:
- Lua code: defineType("Chat", {...})

**References**:
- CRC: crc-LuaPresenterLogic.md - "Does: defineType"

**Expected Results**:
- Type created
- Methods defined
- Properties defined

---

### Test: Define method on presenter type

**Purpose**: Verify method definition

**Input**:
- defineMethod("Chat", "send", function(...) ... end)

**References**:
- CRC: crc-LuaPresenterLogic.md - "Does: defineMethod"

**Expected Results**:
- Method added to type
- Callable via callMethod
- Receives arguments

---

### Test: Define property with getter/setter

**Purpose**: Verify property definition

**Input**:
- defineProperty("Chat", "messageCount", getter, setter)

**References**:
- CRC: crc-LuaPresenterLogic.md - "Does: defineProperty"

**Expected Results**:
- Getter called on read
- Setter called on write
- Computed properties work

---

### Test: Instantiate Lua presenter

**Purpose**: Verify instance creation

**Input**:
- create property with type "Chat"

**References**:
- CRC: crc-LuaPresenterLogic.md - "Does: instantiate"

**Expected Results**:
- New instance created
- Constructor called if defined
- Initial state set

---

### Test: Handle ui-action in Lua

**Purpose**: Verify action handling

**Input**:
- Button with ui-action="send()"
- User clicks button

**References**:
- CRC: crc-LuaPresenterLogic.md - "Does: handleAction"
- Sequence: seq-lua-handle-action.md

**Expected Results**:
- send() method called
- Form values passed
- State updates reflected

---

### Test: Notify change from Lua

**Purpose**: Verify change notification

**Input**:
- Lua modifies presenter state
- notifyChange()

**References**:
- CRC: crc-LuaPresenterLogic.md - "Does: notifyChange"

**Expected Results**:
- Changed variables detected
- Updates sent to watchers
- UI reflects changes

---

### Test: Lua runtime shutdown

**Purpose**: Verify cleanup

**Input**:
- shutdown()

**References**:
- CRC: crc-LuaSession.md - "Does: Shutdown"

**Expected Results**:
- Lua VM destroyed
- Resources freed
- No memory leaks

---

### Test: Lua error handling

**Purpose**: Verify error propagation

**Input**:
- Lua code with runtime error

**References**:
- CRC: crc-LuaSession.md - "Does: ExecuteInSession"

**Expected Results**:
- Error caught
- Error message available
- Server not crashed

---

### Test: Load from --dir directory

**Purpose**: Verify custom lua directory

**Input**:
- Server with --dir /custom
- loadFile from /custom/lua/

**References**:
- CRC: crc-LuaSession.md - "Does: CreateLuaSession"

**Expected Results**:
- Files loaded from custom path
- Default lua/ overridden
- Both locations searchable

---

## Coverage Summary

**Responsibilities Covered:**
- LuaSession: CreateLuaSession, ExecuteInSession, getApp, HandleFrontendUpdate, AfterBatch, Shutdown
- LuaPresenterLogic: defineType, defineMethod, defineProperty, instantiate, handleAction, updateProperty, notifyChange

**Scenarios Covered:**
- seq-load-lua-code.md: All paths
- seq-lua-handle-action.md: All paths

**Gaps**: None identified
