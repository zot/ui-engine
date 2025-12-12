# Sequence: Wrapper Transform

**Source Spec:** protocol.md
**Use Case:** Variable with wrapper computes stored value from raw value

## Participants

- Variable: Has wrapperInstance stored internally
- WrapperRegistry: Creates wrapper instances by type name
- Wrapper: Stored in variable, transforms raw value to stored value
- WatchManager: Notifies watchers with stored value

## Sequence

```
     Variable           WrapperRegistry            Wrapper              WatchManager
        |                      |                      |                      |
        |   [on create with wrapper property]        |                      |
        |                      |                      |                      |
        |---getWrapperFactory->|                      |                      |
        |   ("ViewList")       |                      |                      |
        |                      |                      |                      |
        |<--factory------------|                      |                      |
        |                      |                      |                      |
        |---create(variable)---|--------------------->|                      |
        |                      |                      |                      |
        |                      |                      |---getProperty------->|
        |                      |                      |   ("item" type)      |
        |                      |                      |                      |
        |---storeWrapper-------|--------------------->|                      |
        |   (internal field)   |                      |                      |
        |                      |                      |                      |
        |   [on value change]                         |                      |
        |                      |                      |                      |
        |---detectChanges----->|                      |                      |
        |   (compare to        |                      |                      |
        |    monitored value)  |                      |                      |
        |                      |                      |                      |
        |   [if changed]                              |                      |
        |                      |                      |                      |
        |---updateMonitored--->|                      |                      |
        |   (shallow copy      |                      |                      |
        |    for arrays)       |                      |                      |
        |                      |                      |                      |
        |---computeValue-------|--------------------->|                      |
        |   (rawValue)         |                      |                      |
        |                      |                      |                      |
        |                      |                      |---syncPresenters---->|
        |                      |                      |   (for ViewList)     |
        |                      |                      |                      |
        |<--storedValue--------|----------------------|                      |
        |                      |                      |                      |
        |---storeValue-------->|                      |                      |
        |   (stored value)     |                      |                      |
        |                      |                      |                      |
        |---notifyWatchers-----|------------------------------------->|
        |   (stored value)     |                      |                      |
        |                      |                      |                      |
```

## Notes

- Wrapper constructor receives the variable, stores reference internally
- Wrapper can access variable properties (e.g., `item=ContactPresenter`)
- Wrapper instance is stored internally in variable (not as a property)
- On value changes: `wrapper.computeValue(rawValue)` returns stored value
- Without wrapper: stored value = raw value in "value JSON" form
- With wrapper: stored value = result of `wrapper.computeValue(rawValue)`
