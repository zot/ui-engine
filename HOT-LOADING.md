# Hot-Loading Design Details

> For developer usage, see "Hot-Reloading Lua Code" in `USAGE.md`.

## How It Works

### Prototype Declaration: `session:prototype(name, init)`

```
if global[name] is nil:
    if init is nil: init = {}
    init.type = name
    init.__index = init
    if init.new is nil: init.new = default_new  -- function(self, instance) return session:create(self, instance) end
    global[name] = init
    store shallow copy of init for change detection
    create instance tracking for this prototype
else if init is non-nil:
    storedInit = get stored init copy for this prototype
    if init differs from storedInit:
        removedKeys = keys in storedInit but not in init
        copy init values into existing prototype (preserves table identity)
        store new shallow copy of init
        queue prototype for mutation (with removedKeys)
```

**Key:** Existing prototype table is preserved—only values are updated. Instance metatables remain valid.

### Instance Creation: `session:create(prototype, instance)`

```
if instance is nil: instance = {}
setmetatable(instance, prototype)
add instance to prototype's instance collection (weak reference)
return instance
```

### Instance Tracking

- Map: `prototype -> weak set of instances`
- Weak references allow GC to collect unused instances
- Dead references cleaned up during iteration

### Post-Load Mutation Processing

After loading a Lua file:

```
for each (prototype, removedKeys) in mutation queue (FIFO order):
    for each live instance of prototype:
        if prototype has :mutate() method:
            pcall(prototype.mutate, instance)  -- catch and log errors
        for each key in removedKeys:
            instance[key] = nil  -- clean up removed fields
clear mutation queue
```

## Example Flow

**Initial load:**
```lua
Person = session:prototype("Person", {
    name = "",
})
-- default :new() provided automatically

local alice = Person:new({name = "Alice"})
local bob = Person:new({name = "Bob"})
```

State: `Person` prototype exists, 2 tracked instances.

**Hot-reload with schema change:**
```lua
Person = session:prototype("Person", {
    name = "",
    email = "",  -- NEW FIELD
})

function Person:mutate()
    self.email = self.email or ""
end
```

Post-load processing:
1. `Person` is in mutation queue (init changed)
2. `Person:mutate()` exists
3. Call `alice:mutate()`, `bob:mutate()`
4. Both now have `email = ""`

## Field Removal

When `session:prototype()` detects a field was removed:

```lua
-- Original
Person = session:prototype("Person", {
    name = "",
    oldField = "",  -- Will be removed
})

-- After edit (oldField removed)
Person = session:prototype("Person", {
    name = "",
})
```

Post-load processing:
1. Detect `oldField` was in prototype but not in new init
2. Call `person:mutate()` on each instance (if exists)
3. Set `instance.oldField = nil` for each instance

## Design Decisions

1. **Init values go directly into prototype** — instances inherit via metatable lookup. No copying defaults to each instance.

2. **Field removal detection** — fields in current prototype missing from init indicate removal. Removed fields are nil'd out after `mutate()`.

3. **Mutation queue is FIFO** — prototype declaration order controls mutation order. Declare dependencies first (Address before Person).

4. **No auto-defaults in create()** — the developer's `:new()` method initializes what it needs.

## Benefits

1. **No manual mutate() calls** — framework handles it after hot-reload
2. **Instance tracking is automatic** — `session:create` registers instances
3. **Prototype identity preserved** — existing instances keep working
4. **Mutations run once per reload** — not on every method call
5. **Error isolation** — pcall prevents one bad migration from breaking others
6. **Weak references** — no memory leaks from tracking

## API Summary

| Function | Purpose |
|----------|---------|
| `session:prototype(name, init)` | Declare/update prototype with default fields and `:new()` |
| `session:create(prototype, instance)` | Create tracked instance |
| `Prototype:new(instance)` | Default provided; override for custom init |
| `Prototype:mutate()` | Optional migration method, called automatically |
