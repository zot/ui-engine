# Variable Protocol

The variable protocol is the core of the platform. The backend maintains a hierarchical variable tree that:
- Maps to paths within presenter JSON objects
- Triggers automatic sync on changes
- Enables fine-grained reactivity

Each variable has
- an optional parent id, see "Variable Identity"
- a value, see "Variable Values"
- optional properties which are strings. Empty string is equivalent to unset

## Variable Identity

- Each variable has a unique **id** (integer, counting up from 1)
- ID of 0 or absent indicates "no variable" / null reference
- **Variable IDs are scoped to a session** - multiple sessions can each have their own variable 1
- **Standard variables** are registered with `@NAME` ids (e.g., `@app`, `@customers`)
  - These provide well-known entry points into the variable tree

## Variable Values

Variable values are JSON, interpreted as follows:
- Strings, numbers, booleans, and null are interpreted normally
- Arrays contain only variable values (no nested objects, only references)
- Objects of the form `{obj: ID}` are **object references**:
  - Positive IDs (1+): Objects managed by the backend
  - Negative IDs (-1 and lower): Objects managed by the UI server itself
  - Object properties contain only variable values (arrays allowed, but nested objects must be references)

The `get` command automatically resolves object references, returning the actual object data rather than IDs. Use `getObjects([objId, ...])` to retrieve objects directly by ID.

## Standard Variable Properties

Variable metadata properties with special meaning:

| Property   | Values                                   | Description                                                           |
|------------|------------------------------------------|-----------------------------------------------------------------------|
| `create`   | Type name (e.g., `MyModule.MyClass`)     | Instantiates an object of this type as the variable's value           |
| `path`     | Dot-separated path (e.g., `father.name`) | Path to bound data (see syntax below)                                 |
| `access`   | `r`, `w`, `rw`, `action`                 | Read/write permissions. `action` = write-only trigger (like a button) |
| `type`     | Type name string                         | Auto-set by backend to the runtime type name of the variable's value  |
| `inactive` | any or unset                             | if set, variable updates will not be relayed for this or its children |
| `wrapper`  | Type name (e.g., `ViewList`)             | Instantiates a wrapper object that becomes the variable's value. |

**Access modes:**
- `r` = readable only
- `w` = writeable only
- `rw` = readable and writeable
- `action` = writeable, triggers a function call (like a button click)

**Method path constraints:**
- Paths ending in `()` (no argument) must have access `r` or `action`
- Paths ending in `(_)` (with argument) must have access `w` or `action`

**Path syntax:**
- Property access: `name`
- Array indexing: `1`, `2` (1-based)
- Parent traversal: `..`
- Method calls: `getName()`
- Standard variable prefix: `@customers.2.name` (starts from a well-known registered variable)
- Path properties: `contacts?wrapper=ViewList&item=ContactPresenter`
  - Properties after `?` are set on the created variable
  - Uses URL query string syntax: `key=value&key2=value2`
  - Common properties: `wrapper`, `item`, `create`

## Variable Wrappers

The `wrapper` property specifies a **wrapper type name** (e.g., `ViewList`). When set, the backend instantiates a wrapper object that becomes the variable's value for the purposes of path navigation.

**Factory Registries:**

There are two types of factory registries for creating objects:

1.  **Create Factory:** Used for the `create` property. It creates a new object from a value.
2.  **Wrapper Factory:** Used for the `wrapper` property. It creates a new wrapper instance from a variable.

Both registries are populated automatically by `init()` functions in the Go code, following a frictionless development principle.

**Wrapper Lifecycle:**
1. When a variable is created with `wrapper=TypeName` in path properties, the `WrapperFactory` for that type is called.
2. The factory receives the variable: `Factory(runtime, variable)`.
3. The factory returns a new wrapper object (`interface{}`), which is stored internally in the variable's `wrapperInstance` field.
4. This wrapper object becomes the `storedValue` of the variable.
5. If the wrapper object has a `Destroy()` method, it will be called when the variable is destroyed.

**Wrapper Behavior:**

The wrapper object itself stands in for the variable's value when child variables navigate paths. The wrapper is registered in the object registry and becomes the variable's navigation value.

**Wrapper Creation and Reuse:**

The `WrapperFactory` is called whenever the variable's value changes. The factory can:
- Return a **new wrapper** if none exists yet.
- Return the **existing wrapper** if it should be reused (to preserve internal state like selection). The wrapper is responsible for syncing its internal state with the new value.
- Return `nil` if no wrapper is needed.

This allows stateful wrappers like `ViewList` to update their internal state when the underlying array changes, rather than being replaced and losing state (e.g., selection index, scroll position).


## Variable Value Processing

Change detection is handled by the `change-tracker` package. The UI platform provides a `Resolver` implementation and calls `DetectChanges()`, which:

1. Computes current values for all watched variables via path resolution
2. Detects changes and queues updates
3. Creates/destroys wrappers as needed via `Resolver.CreateWrapper(variable)`

Variable values are sent to the frontend in "value JSON" form (objects as `{obj: ID}` refs).

**Property priority:**

In `create` and `update` messages, property names can be suffixed with `:high`, `:med`, or `:low` to set processing priority:
- `:high` - Processed first
- `:med` - Processed after high priority
- `:low` - Processed last
- No suffix - Priority unchanged from previous value

High priority properties are handled before low priority ones. This allows control over processing order when property handling has dependencies (e.g., `viewdefs:high` ensures viewdefs are available before rendering).

## Variable Protocol Messages

The UI server relays protocol messages bidirectionally between frontend and backend. The same messages flow in both directions.

**Push-only model:** Protocol messages are push-only, not request-response. Senders do not wait for acknowledgment; they assume success unless an `error` message is received.

**Relayed messages** (frontend ↔ UI server ↔ backend):
- `create(parentId, value, properties, nowatch?, unbound?)` - Create a new variable
  - if properties contains a value for `create`, the `value` is ignored because the backend / UI server will create the object
  - `nowatch` indicates that the variable should not be watched
  - `unbound` indicates that the variable's storage is in the UI server itself and not managed by an external app
  - Property names can have priority suffixes (`:high`, `:med`, `:low`, omitting a suffix leaves the priority unchanged)
- `destroy(varId)` - Destroy a variable and all its children
- `update(varId, value?, properties?)` - Update the variable's value and/or properties
  - Property names can have priority suffixes (`:high`, `:med`, `:low`), omitting a suffix leaves the priority unchanged
- `watch(varId)` - Subscribe to value changes; immediately sends an update message
  - `unwatch(varId)` - Unsubscribe from value changes

**Server-response messages** (only sent from UI server)
- `error(varId, code, description)` - indicates an error condition on a variable
  - `code` - One-word error code (e.g., `path-failure`, `not-found`, `unauthorized`)
  - `description` - Human-readable error description
  - Error conditions persist until cleared by a successful operation on the same variable

**UI server-handled messages** (not relayed):
- `get([varId, ...])` - Retrieve variable values from UI server storage
  - Used by apps that don't bind their own data to the variables
  - For objects, returns `{obj: ID, value: JSON}`
- `getObjects([objId, ...])` - Retrieve UI server objects by ID

**Source of truth responsibilities:**
- For **unbound** variables: The UI server is the source of truth - it stores state changes (`create`, `update`, `destroy`) AND forwards messages
- For **bound** variables: The backend is the source of truth - it stores state changes, the UI server only forwards

The holder of the source of truth is responsible for properly reflecting state change messages in storage.

**Watch tallying:**

The UI server maintains a count of observers for each variable. For bound variables:
- `watch` is only forwarded to the backend when the tally changes from 0 → 1
- `unwatch` is only forwarded to the backend when the tally changes from 1 → 0

This allows multiple frontend observers without redundant backend notifications.

## Message Batching

Messages can be sent individually as JSON objects or batched as JSON arrays:

```json
// Single message
{"type": "update", "id": 5, "value": "hello"}

// Batched messages
[
  {"type": "update", "id": 5, "properties": {"viewdefs": {...}}},
  {"type": "update", "id": 10, "value": "world"},
  {"type": "update", "id": 5, "value": {"name": "Alice"}}
]
```

## Session-Based Communication

Protocol batches between UI server and backend include a session ID. This allows the backend to maintain per-session state.

**Session ID vending:**

The UI server maintains a mapping between internal session IDs (UUIDs) and compact vended IDs for backend communication. Vended IDs are sequential integers starting from 1, saving bandwidth compared to sending full UUID strings.

- Internal session ID: `df785bf8982879c0a582560990dbeae3` (32 chars)
- Vended session ID: `1`, `2`, `3`, ... (1-3 chars typically)

The UI server tracks `internalID ↔ vendedID` mappings. Backend only sees vended IDs.

**Batch format (server ↔ backend):**
```json
{"session": 1, "messages": [
  {"type": "watch", "id": 1},
  {"type": "update", "id": 5, "value": "hello"}
]}
```

**Session lifecycle:**
- When the UI server receives a batch with a new session ID, the backend creates a corresponding session
- For embedded Lua: A new Lua session is created and `main.lua` is executed with the `session` global
- Executing `main.lua` serves as the notification that a new session has started
- The Lua code is responsible for creating variable 1 (the app variable) and sending its initial state

**Priority-based batching:**

Both values and properties have priority (`high`, `medium` (default), `low`). When sending batched updates:

1. Queue all pending changes (values and properties)
2. Separate by priority - a single variable may have changes at different priorities
3. Order the batch: all high-priority updates first, then medium, then low
4. A variable may appear multiple times in a batch if its value and properties have different priorities

**Example:** If variable 5 has a high-priority `viewdefs` property update and a medium-priority value update, the batch contains two separate update messages for variable 5.

**Viewdef delivery:**

When a variable is created or its value changes, the backend sets the `type` property based on the value's type. If viewdefs for that type haven't been sent to the frontend, the backend queues them. Before sending updates, pending viewdefs are set on the app variable (ID 1) as a `viewdefs` property:

```json
{"type": "update", "id": 1, "properties": {
  "viewdefs:high": {"Contact.DEFAULT": "<template>...</template>", "Contact.COMPACT": "<template>...</template>"}
}}
```

Viewdefs use `:high` priority to ensure they're processed before the variables that need them.
