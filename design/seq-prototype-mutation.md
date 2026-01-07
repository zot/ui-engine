# Sequence: Prototype Mutation Processing

**Source Spec:** libraries.md (Prototype Management, Post-load mutation processing)
**Use Case:** Process queued prototypes after Lua file load to migrate live instances

## Participants

- LuaSession: Per-session Lua environment (owns mutation queue)
- MutationQueue: FIFO queue of (prototype, removedKeys) pairs
- InstanceRegistry: Weak set of instances per prototype
- Prototype: Lua prototype table (may have :mutate method)
- Instance: Live instance of a prototype

## Sequence

```
     LuaSession          MutationQueue        InstanceRegistry       Prototype         Instance
        |                     |                     |                   |                 |
        |--processMutationQueue()                   |                   |                 |
        |                     |                     |                   |                 |
        |--[for each entry in FIFO order]          |                   |                 |
        |                     |                     |                   |                 |
        |--dequeue()--------->|                     |                   |                 |
        |                     |                     |                   |                 |
        |<--(proto, removed)--|                     |                   |                 |
        |                     |                     |                   |                 |
        |--getLiveInstances(proto)---------------->|                   |                 |
        |                     |                     |                   |                 |
        |<--[weak refs, skip dead]-----------------|                   |                 |
        |                     |                     |                   |                 |
        |--[for each live instance]                |                   |                 |
        |                     |                     |                   |                 |
        |--[if proto has :mutate]                  |                   |                 |
        |                     |                     |                   |                 |
        |--pcall(proto.mutate, instance)-------------------------------->|                 |
        |                     |                     |                   |                 |
        |                     |                     |                   |--[migrate]----->|
        |                     |                     |                   |                 |
        |<--ok/err (logged)-------------------------|-----------------------------------------|
        |                     |                     |                   |                 |
        |--[for each key in removed]               |                   |                 |
        |                     |                     |                   |                 |
        |--instance[key] = nil---------------------------------------------------------->|
        |                     |                     |                   |                 |
        |--[clear queue]----->|                     |                   |                 |
        |                     |                     |                   |                 |
```

## Prototype Change Detection

```
     LuaSession             prototypeRegistry
        |                        |
        |--prototype(name, init)-|
        |                        |
        |--[get stored init]---->|
        |                        |
        |<--storedInit-----------|
        |                        |
        |--[compare init to storedInit]
        |                        |
        |--[if different]        |
        |  - copy init values into existing prototype
        |  - compute removedKeys = storedInit.keys - init.keys
        |  - store new shallow copy
        |  - queue (prototype, removedKeys)
        |                        |
```

## EMPTY Marker Handling

```lua
-- EMPTY is a global sentinel: {}
-- Marks fields that start nil but should be tracked for mutation

Person = session:prototype("Person", {
    name = "",
    avatar = EMPTY,  -- starts nil, tracked for mutation
})
```

Processing:
1. `session:prototype` copies init including EMPTY markers
2. After copying, removes EMPTY values from prototype (field defaults to nil)
3. Stored copy preserves EMPTY markers for change detection
4. On hot-reload, EMPTY fields are compared correctly

## Notes

- **FIFO Order**: Prototypes are processed in declaration order (dependencies first)
- **Weak References**: Dead instances are skipped during iteration (GC collected)
- **Error Isolation**: pcall prevents one bad :mutate() from breaking others
- **Field Removal**: Removed fields are nil'd out AFTER :mutate() runs
- **Idempotent Mutations**: :mutate() methods should be idempotent (may run multiple times)
- **No Instance Copies**: Instances inherit defaults via metatable, not field copies

## Example Flow

**Initial load:**
```lua
Person = session:prototype("Person", { name = "" })
local alice = Person:new({name = "Alice"})
```

**Hot-reload with new field:**
```lua
Person = session:prototype("Person", {
    name = "",
    email = "",  -- NEW
})

function Person:mutate()
    self.email = self.email or "unknown@example.com"
end
```

Processing:
1. `session:prototype` detects init changed (email added)
2. Updates Person prototype in place, queues (Person, [])
3. After file load, `processMutationQueue()` runs
4. For alice: calls `Person:mutate(alice)`, alice.email = "unknown@example.com"

**Hot-reload with removed field:**
```lua
Person = session:prototype("Person", {
    name = "",
    -- email removed
})
```

Processing:
1. `session:prototype` detects init changed (email removed)
2. Updates Person prototype, queues (Person, ["email"])
3. After file load, `processMutationQueue()` runs
4. For alice: calls `Person:mutate(alice)` if exists, then alice.email = nil
