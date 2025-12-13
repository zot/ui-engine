# ChangeDetector

**Source Spec:** libraries.md

> **Package Reference:** ChangeDetector is provided by the `change-tracker` package
> (`github.com/zot/change-tracker`). This CRC documents what the package provides,
> not a component to be implemented from scratch. The package handles variable
> tracking, change detection, and update dispatch. See libraries.md for package details.

## Responsibilities

### Knows
- watchedVariables: Map of variable ID to path being watched
- objectReferences: Map of variable ID to backend object reference (Go pointer)
- cachedJSON: Map of variable ID to last JSON value sent
- pendingRefresh: Flag indicating refresh is needed
- throttleInterval: Minimum time between background refreshes
- lastRefresh: Timestamp of last refresh
- registry: ObjectRegistry for identity-based serialization (Go only)

### Does
- addWatch: Start tracking variable for changes, register object in ObjectRegistry
- removeWatch: Stop tracking variable, unregister object from ObjectRegistry
- refresh: Compute values for all watched variables, detect changes
- computeValue: Serialize backend object to JSON using ObjectRegistry for object refs
- detectChange: Compare current JSON to cached value
- scheduleRefresh: Queue background-triggered refresh with throttling
- sendUpdates: Send update messages for changed variables
- afterBatch: Trigger refresh after processing client message batch

## Collaborators

- ObjectRegistry: Maps objects to variable IDs for identity-based serialization (Go only)
- PathNavigator: Resolves paths to current object values
- BackendConnection: External backends trigger refresh cycle
- ProtocolHandler: Sends update messages

## Sequences

- seq-backend-refresh.md: Full refresh cycle
- seq-object-registry.md: Registration and serialization with object refs
- seq-update-variable.md: Change propagation

## Notes

- **Key insight**: Variables store references to backend objects, not copies
- Change detection works by computing current JSON from object and comparing to cached JSON
- **Go backend**: Uses ObjectRegistry for identity-based serialization
- **Lua backend**: LuaSession implements change detection internally (tables have identity)
- Refresh is automatic after message batch processing (no manual update calls needed)
- During serialization, objects found in ObjectRegistry emit `{"obj": id}` instead of inline values
- This allows the same object appearing in multiple locations to serialize identically
