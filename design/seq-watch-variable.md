# Sequence: Watch Variable

**Source Spec:** protocol.md
**Use Case:** Subscribing to variable change notifications

## Participants

- Frontend: Browser client watching variable
- ProtocolHandler: Message processor
- WatchManager: Watch subscription manager
- VariableStore: Variable storage
- Backend: External backend (for bound variables)

## Sequence

```
     Frontend          ProtocolHandler         WatchManager          VariableStore            Backend
        |                      |                      |                      |                      |
        |---watch(varId)------>|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---watch(varId)------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---incrementTally---->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--newTally-----------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---addWatcher-------->|                      |
        |                      |                      |   (frontendId)       |                      |
        |                      |                      |                      |                      |
        |                      |          [if bound and tally was 0]         |                      |
        |                      |                      |---forwardWatch------>|                      |
        |                      |                      |                      |---watch(varId)------>|
        |                      |                      |                      |                      |
        |                      |---get(varId)-------->|                      |                      |
        |                      |                      |---get(varId)-------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--variable-----------|                      |
        |                      |                      |                      |                      |
        |                      |<--variable-----------|                      |                      |
        |                      |                      |                      |                      |
        |<--update(varId,val)--|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Watch immediately returns current value via update message
- Tally tracks number of observers per variable
- For bound variables, watch only forwarded to backend on 0->1 transition
- Multiple frontend observers share single backend subscription
- Unwatch reverses process; forwarded on 1->0 transition
