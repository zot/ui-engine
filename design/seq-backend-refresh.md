# Sequence: Backend Refresh

**Source Spec:** libraries.md
**Use Case:** Backend detecting and propagating data changes

## Participants

- Backend: External backend program
- ChangeDetector: Change detection manager
- PathNavigator: Path resolution
- ObjectRegistry: Weak reference registry for object identity (Go only)
- BackendConnection: Connection to UI server
- ProtocolHandler: Message processor

## Sequence

```
   Backend         ChangeDetector        PathNavigator       ObjectRegistry      BackendConnection    ProtocolHandler
      |                   |                    |                    |                    |                    |
      |     [after client message or background trigger]            |                    |                    |
      |--refresh--------->|                    |                    |                    |                    |
      |                   |                    |                    |                    |                    |
      |                   |     [for each watched variable]         |                    |                    |
      |                   |--resolve---------->|                    |                    |                    |
      |                   |  (root, path)      |                    |                    |                    |
      |                   |<--currentValue-----|                    |                    |                    |
      |                   |                    |                    |                    |                    |
      |                   |--serializeWithRefs-|-------------------->|                    |                    |
      |                   |  (object)          |                    |                    |                    |
      |                   |                    |  [walk graph]      |                    |                    |
      |                   |                    |  lookup each ptr   |                    |                    |
      |                   |                    |  emit obj refs     |                    |                    |
      |                   |<--JSON-------------|<-------------------|                    |                    |
      |                   |                    |                    |                    |                    |
      |                   |--compare---------->|                    |                    |                    |
      |                   |  (cached vs new)   |                    |                    |                    |
      |                   |                    |                    |                    |                    |
      |                   |     [if changed]   |                    |                    |                    |
      |                   |--queueUpdate------>|                    |                    |                    |
      |                   |  (varID, newValue) |                    |                    |                    |
      |                   |                    |                    |                    |                    |
      |                   |--sendUpdates-------|--------------------|------------------->|                    |
      |                   |                    |                    |--send()----------->|                    |
      |                   |                    |                    |                    |--update(var,val)-->|
      |                   |                    |                    |                    |                    |
      |                   |--storeCached------>|                    |                    |                    |
      |                   |  (varID, newJSON)  |                    |                    |                    |
      |                   |                    |                    |                    |                    |
      |<--complete--------|                    |                    |                    |                    |
      |                   |                    |                    |                    |                    |
```

## Notes

- Refresh triggered after client messages automatically
- Background changes throttled to prevent flooding
- Uses reflection to compute values (no observer pattern needed)
- Only changed values sent as updates
- Previous JSON values stored for next comparison
- Thread-safe interaction with refresh logic
- **ObjectRegistry integration**: Serialization uses registry to emit `{"obj": id}` for registered objects
- Same object appearing in multiple locations will serialize identically
