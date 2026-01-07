# Test Design: Lua Runtime

**Source Specs**: interfaces.md, deployment.md, libraries.md (Prototype Management)
**CRC Cards**: crc-LuaSession.md, crc-LuaPresenterLogic.md
**Sequences**: seq-load-lua-code.md, seq-lua-handle-action.md, seq-prototype-mutation.md

## Overview

Tests for embedded Lua backend supporting presentation logic and prototype management for hot-loading.

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

## Prototype Management Tests

### Test: Create new prototype with session:prototype

**Purpose**: Verify new prototype creation with init fields

**Input**:
```lua
Person = session:prototype("Person", {
    name = "",
    email = "",
})
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Global `Person` created
- Person.type = "Person"
- Person.__index = Person
- Default :new() method provided
- Init copy stored in prototypeRegistry
- Instance tracking set up (empty weak set)

---

### Test: Prototype sets type and __index automatically

**Purpose**: Verify prototype metadata is set

**Input**:
```lua
Item = session:prototype("Item", { name = "" })
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"

**Expected Results**:
- Item.type == "Item"
- Item.__index == Item
- Instances inherit via metatable

---

### Test: Default :new() method provided

**Purpose**: Verify default :new() calls session:create

**Input**:
```lua
Thing = session:prototype("Thing", { value = 0 })
local t = Thing:new({value = 42})
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"

**Expected Results**:
- Default :new() exists
- Calls session:create internally
- Instance tracked in weak set
- Returns instance with metatable set

---

### Test: Custom :new() method preserved

**Purpose**: Verify custom :new() not overwritten

**Input**:
```lua
Counter = session:prototype("Counter", { count = 0 })
Counter.nextId = Counter.nextId or 0
function Counter:new(instance)
    instance = session:create(Counter, instance)
    instance.id = Counter.nextId
    Counter.nextId = Counter.nextId + 1
    return instance
end
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"

**Expected Results**:
- Custom :new() preserved on reload
- nextId increments correctly
- Instances have unique IDs

---

### Test: EMPTY marker for nil fields

**Purpose**: Verify EMPTY marks tracked nil fields

**Input**:
```lua
User = session:prototype("User", {
    name = "",
    avatar = EMPTY,  -- starts nil, tracked for mutation
})
local u = User:new()
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- User.avatar is nil (EMPTY removed after copy)
- Stored init copy has EMPTY marker preserved
- u.avatar == nil (not EMPTY)
- avatar field tracked for mutation detection

---

### Test: Create instance with session:create

**Purpose**: Verify instance creation and tracking

**Input**:
```lua
Person = session:prototype("Person", { name = "" })
local p = session:create(Person, { name = "Alice" })
```

**References**:
- CRC: crc-LuaSession.md - "Does: create(prototype, instance)"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Metatable set to Person
- p.name == "Alice"
- p.type == "Person" (via metatable)
- Instance added to Person's weak set

---

### Test: Create instance with nil creates empty table

**Purpose**: Verify nil instance becomes empty table

**Input**:
```lua
Person = session:prototype("Person", { name = "" })
local p = session:create(Person, nil)
```

**References**:
- CRC: crc-LuaSession.md - "Does: create(prototype, instance)"

**Expected Results**:
- p is a table (not nil)
- Metatable set to Person
- p.name == "" (inherited from prototype)

---

### Test: Weak reference allows GC

**Purpose**: Verify tracked instances can be garbage collected

**Input**:
```lua
Person = session:prototype("Person", { name = "" })
local p = Person:new({ name = "Temp" })
p = nil
collectgarbage()
```

**References**:
- CRC: crc-LuaSession.md - "Knows: instanceRegistry"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Instance collected by GC
- Weak set automatically cleans up
- No memory leak
- Iteration skips dead references

---

### Test: Update prototype on reload detects change

**Purpose**: Verify init change detection

**Input**:
```lua
-- Initial load
Person = session:prototype("Person", { name = "" })

-- Hot-reload with new field
Person = session:prototype("Person", { name = "", email = "" })
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Same Person table (identity preserved)
- email field added to prototype
- New init copy stored
- Prototype queued for mutation

---

### Test: Update prototype preserves table identity

**Purpose**: Verify existing instances keep working

**Input**:
```lua
Person = session:prototype("Person", { name = "" })
local alice = Person:new({ name = "Alice" })
-- Hot-reload
Person = session:prototype("Person", { name = "", age = 0 })
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- alice still valid
- getmetatable(alice) == Person (same table)
- alice.age == 0 (inherited from updated prototype)

---

### Test: Detect removed fields

**Purpose**: Verify removal detection for cleanup

**Input**:
```lua
-- Initial
Person = session:prototype("Person", { name = "", oldField = "" })
local p = Person:new({ name = "Bob", oldField = "data" })

-- Hot-reload (oldField removed)
Person = session:prototype("Person", { name = "" })
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Prototype queued with removedKeys = ["oldField"]
- After processMutationQueue: p.oldField == nil

---

### Test: No mutation queue when init unchanged

**Purpose**: Verify identical init doesn't queue

**Input**:
```lua
-- Initial
Person = session:prototype("Person", { name = "" })

-- Hot-reload with same init
Person = session:prototype("Person", { name = "" })
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"

**Expected Results**:
- Init comparison shows no change
- Prototype not queued
- processMutationQueue has nothing to do

---

### Test: Process mutation queue calls :mutate()

**Purpose**: Verify :mutate() called on instances

**Input**:
```lua
Person = session:prototype("Person", { name = "" })
local alice = Person:new({ name = "Alice" })

-- Hot-reload with mutate method
Person = session:prototype("Person", { name = "", email = "" })
function Person:mutate()
    self.email = self.email or "default@example.com"
end
-- processMutationQueue() called after load
```

**References**:
- CRC: crc-LuaSession.md - "Does: processMutationQueue"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Person:mutate(alice) called
- alice.email == "default@example.com"
- Queue cleared after processing

---

### Test: Mutation queue FIFO order

**Purpose**: Verify prototypes processed in declaration order

**Input**:
```lua
-- Dependencies: Address before Person
Address = session:prototype("Address", { city = "" })
Person = session:prototype("Person", { name = "", address = EMPTY })
```

**References**:
- CRC: crc-LuaSession.md - "Knows: mutationQueue"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Address queued first
- Person queued second
- Address processed before Person
- Dependency order maintained

---

### Test: :mutate() errors isolated with pcall

**Purpose**: Verify one bad mutate doesn't break others

**Input**:
```lua
Bad = session:prototype("Bad", { x = 0 })
function Bad:mutate()
    error("mutation failed!")
end

Good = session:prototype("Good", { y = 0 })
function Good:mutate()
    self.y = 42
end

local b = Bad:new()
local g = Good:new()
-- Hot-reload both
```

**References**:
- CRC: crc-LuaSession.md - "Does: processMutationQueue"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Bad:mutate() error logged
- Good:mutate() still called
- g.y == 42
- No crash

---

### Test: Removed fields nil'd after :mutate()

**Purpose**: Verify field removal happens after migration

**Input**:
```lua
Person = session:prototype("Person", { name = "", fullName = "" })
local p = Person:new({ fullName = "Alice Smith" })

-- Hot-reload: rename fullName to name
Person = session:prototype("Person", { name = "" })
function Person:mutate()
    self.name = self.name or self.fullName
end
```

**References**:
- CRC: crc-LuaSession.md - "Does: processMutationQueue"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- :mutate() runs first (can read fullName)
- p.name == "Alice Smith" (migrated)
- p.fullName == nil (removed after mutate)

---

### Test: Skip dead instances during mutation

**Purpose**: Verify GC'd instances not processed

**Input**:
```lua
Person = session:prototype("Person", { name = "" })
local p = Person:new({ name = "Temp" })
p = nil
collectgarbage()
-- Hot-reload
Person = session:prototype("Person", { name = "", age = 0 })
-- processMutationQueue()
```

**References**:
- CRC: crc-LuaSession.md - "Does: processMutationQueue"
- Sequence: seq-prototype-mutation.md

**Expected Results**:
- Dead instance skipped
- No error
- Live instances processed normally

---

### Test: Prototype shared state preserved

**Purpose**: Verify non-init fields preserved on reload

**Input**:
```lua
Counter = session:prototype("Counter", { count = 0 })
Counter.nextId = Counter.nextId or 0
Counter.nextId = Counter.nextId + 1  -- nextId = 1

-- Hot-reload
Counter = session:prototype("Counter", { count = 0 })
Counter.nextId = Counter.nextId or 0  -- nextId stays 1
```

**References**:
- CRC: crc-LuaSession.md - "Does: prototype(name, init)"

**Expected Results**:
- Counter.nextId == 1 (preserved)
- Guarded assignment pattern works
- Only init fields updated

---

## Coverage Summary

**Responsibilities Covered:**
- LuaSession: CreateLuaSession, ExecuteInSession, getApp, HandleFrontendUpdate, AfterBatch, Shutdown
- LuaSession: prototype(name, init), create(prototype, instance), processMutationQueue
- LuaPresenterLogic: defineType, defineMethod, defineProperty, instantiate, handleAction, updateProperty, notifyChange

**Scenarios Covered:**
- seq-load-lua-code.md: All paths
- seq-lua-handle-action.md: All paths
- seq-prototype-mutation.md: All paths (new prototype, update, removal, mutate, FIFO order, error isolation)

**Gaps**: None identified
