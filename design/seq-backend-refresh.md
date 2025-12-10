# Sequence: Backend Refresh

**Source Spec:** libraries.md
**Use Case:** Backend detecting and propagating data changes

## Participants

- Backend: External backend program
- ChangeDetector: Change detection manager
- PathNavigator: Path resolution
- BackendConnection: Connection to UI server
- ProtocolHandler: Message processor

## Sequence

```
     Backend           ChangeDetector          PathNavigator       BackendConnection      ProtocolHandler
        |                      |                      |                      |                      |
        |     [after client message or background trigger]                   |                      |
        |---refresh()--------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---getWatched()------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each watched variable]             |                      |
        |                      |---computeValue------>|                      |                      |
        |                      |   (varId)            |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---resolve(path)----->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--currentValue-------|                      |
        |                      |                      |                      |                      |
        |                      |<--value--------------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---compare()--------->|                      |                      |
        |                      |   (prev vs current)  |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [if changed]     |                      |                      |
        |                      |---queueUpdate------->|                      |                      |
        |                      |   (varId, newValue)  |                      |                      |
        |                      |                      |                      |                      |
        |                      |---sendUpdates------->|                      |                      |
        |                      |                      |---send()------------>|                      |
        |                      |                      |                      |---update(var,val)--->|
        |                      |                      |                      |                      |
        |                      |---storePrevious()--->|                      |                      |
        |                      |                      |                      |                      |
        |<--complete-----------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Refresh triggered after client messages automatically
- Background changes throttled to prevent flooding
- Uses reflection to compute values (no observer pattern needed)
- Only changed values sent as updates
- Previous values stored for next comparison
- Thread-safe interaction with refresh logic
