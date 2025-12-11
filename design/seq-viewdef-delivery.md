# Sequence: Viewdef Delivery

**Source Spec:** protocol.md, viewdefs.md
**Use Case:** Backend delivers viewdefs with priority batching

## Participants

- Backend: External backend application
- MessageBatcher: Priority-based message batching
- ProtocolHandler: Message processing
- ViewdefStore: Server-side viewdef storage
- Frontend: Browser client

## Sequence

```
     Backend            MessageBatcher         ProtocolHandler         ViewdefStore            Frontend
        |                      |                      |                      |                      |
        |---create(var,val,    |                      |                      |                      |
        |   {type:Contact})-->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---handleCreate------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---needsViewdef?----->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--Contact.DEFAULT----|                      |
        |                      |                      |   not yet sent       |                      |
        |                      |                      |                      |                      |
        |                      |<--queueViewdef-------|                      |                      |
        |                      |   (id:1, viewdefs:   |                      |                      |
        |                      |    high priority)    |                      |                      |
        |                      |                      |                      |                      |
        |                      |<--queueValue---------|                      |                      |
        |                      |   (varId, val,       |                      |                      |
        |                      |    med priority)     |                      |                      |
        |                      |                      |                      |                      |
        |     [flush pending changes]                 |                      |                      |
        |                      |                      |                      |                      |
        |                      |---buildBatch-------->|                      |                      |
        |                      |   separateByPriority |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [batch built: high first, then medium]  |                      |
        |                      |                      |                      |                      |
        |                      |---send([            -|-----------------------------------batch---->|
        |                      |   {update,id:1,      |                      |                      |
        |                      |    props:{viewdefs:  |                      |                      |
        |                      |    {Contact.DEFAULT: |                      |                      |
        |                      |    "<template>..."}}}|                      |                      |
        |                      |   {update,varId,val} |                      |                      |
        |                      |  ])                  |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |     [frontend]       |
        |                      |                      |                      |---handleBatch------->|
        |                      |                      |                      |                      |
        |                      |                      |                      |     [process high priority first]
        |                      |                      |                      |---storeViewdefs----->|
        |                      |                      |                      |                      |
        |                      |                      |                      |     [then medium priority]
        |                      |                      |                      |---handleUpdate------>|
        |                      |                      |                      |   (can now render)   |
        |                      |                      |                      |                      |
```

## Notes

- Viewdefs delivered via variable 1's `viewdefs` property with `:high` priority
- Priority batching ensures viewdefs arrive before variables that need them
- A single variable may have multiple updates in batch if value/properties differ in priority
- Backend tracks which viewdefs have been sent to avoid duplicates
- Frontend validates viewdefs (single template root) before storing
