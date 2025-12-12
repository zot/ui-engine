# Sequence: Object Registry

**Source Spec:** libraries.md
**Use Case:** Object registration, identity-based serialization, and cleanup

## Participants

- Backend: External backend program (Go)
- ChangeDetector: Change detection manager
- PathNavigator: Path resolution
- ObjectRegistry: Weak reference registry for object identity
- Connection: Connection to UI server

## Sequence

```
                                         Object Registry Workflow
    Backend         ChangeDetector        PathNavigator       ObjectRegistry        Connection
       |                   |                    |                    |                    |
       |                   |                    |                    |                    |
       |  ========== Path Watch - Object Registration ==========    |                    |
       |                   |                    |                    |                    |
       |--addWatch-------->|                    |                    |                    |
       |  (varID, path)    |                    |                    |                    |
       |                   |--resolve---------->|                    |                    |
       |                   |  (root, path)      |                    |                    |
       |                   |<--object-----------|                    |                    |
       |                   |                    |                    |                    |
       |                   |--register--------->|-------------------->|                    |
       |                   |  (object, varID)   | weak.Make(object)  |                    |
       |                   |                    | store weakRef->varID                    |
       |                   |<--ok---------------|<-------------------|                    |
       |                   |                    |                    |                    |
       |                   |--cacheJSON-------->|                    |                    |
       |                   |  (serialize obj)   |                    |                    |
       |                   |                    |                    |                    |
       |                   |                    |                    |                    |
       |  ========== Serialization with Object References ==========|                    |
       |                   |                    |                    |                    |
       |--refresh--------->|                    |                    |                    |
       |                   |--resolve---------->|                    |                    |
       |                   |  (root, path)      |                    |                    |
       |                   |<--object-----------|                    |                    |
       |                   |  (has nested objs) |                    |                    |
       |                   |                    |                    |                    |
       |                   |--serializeWithRefs-|-------------------->|                    |
       |                   |  (object)          |                    |                    |
       |                   |                    | [walk object graph]|                    |
       |                   |                    | for each nested:   |                    |
       |                   |                    |   lookup(ptr)      |                    |
       |                   |                    |   if found:        |                    |
       |                   |                    |     emit {"obj":id}|                    |
       |                   |                    |   else:            |                    |
       |                   |                    |     emit inline    |                    |
       |                   |<--JSON w/ obj refs-|<-------------------|                    |
       |                   |                    |                    |                    |
       |                   |--compare---------->|                    |                    |
       |                   |  (cached, new)     |                    |                    |
       |                   |                    |                    |                    |
       |                   |     [if changed]   |                    |                    |
       |                   |--update------------|--------------------|------------------->|
       |                   |  (varID, json)     |                    |                    |
       |                   |--cacheJSON-------->|                    |                    |
       |                   |  (varID, newJSON)  |                    |                    |
       |                   |                    |                    |                    |
       |                   |                    |                    |                    |
       |  ========== Cleanup After GC Collection ===========        |                    |
       |                   |                    |                    |                    |
       |  [app drops reference to object]       |                    |                    |
       |                   |                    |  [GC runs]         |                    |
       |                   |                    |  weak ptr -> nil   |                    |
       |                   |                    |                    |                    |
       |                   |                    |--cleanup---------->|                    |
       |                   |                    |  (periodic)        |                    |
       |                   |                    |  scan weakRefs     |                    |
       |                   |                    |  remove nil entries|                    |
       |                   |                    |                    |                    |
```

## Notes

- **Registration timing**: Objects registered when paths are watched (addWatch)
- **Weak references**: Uses Go 1.25+ `weak` package for weak pointers
- **Identity preservation**: Same object at multiple paths serializes to same `{"obj": id}`
- **Automatic cleanup**: Periodic goroutine removes entries where weak pointer returns nil
- **Frictionless**: Domain objects require no modification - no interfaces or embedded IDs
- **Serialization**: Custom JSON marshaler walks object graph, emits refs for registered objects

## Object Identity Example

```
// Domain objects - no UI-specific code
type Contact struct {
    Name  string
    Email string
}

type App struct {
    Contacts    []*Contact  // Contact appears here
    Selected    *Contact    // Same contact may appear here
}

// When serialized with ObjectRegistry:
// - If contacts[0] is registered as varID 5
// - And selected points to same object
// - Both serialize as {"obj": 5}
```
