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

**Access modes:**
- `r` = readable only
- `w` = writeable only
- `rw` = readable and writeable
- `action` = writeable, triggers a function call (like a button click)

**Path syntax:**
- Property access: `name`
- Array indexing: `1`, `2` (1-based)
- Parent traversal: `..`
- Method calls: `getName()`
- Standard variable prefix: `@customers.2.name` (starts from a well-known registered variable)

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
- `error(varId, description)` - indicates that a variable could not be created

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
