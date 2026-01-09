# Sequence: Viewdef Hot-Reload

**Source Spec:** viewdefs.md
**Use Case:** Viewdef file changes trigger push to frontend and view re-rendering

## Participants

- FileWatcher: Monitors viewdef directory for changes
- ViewdefStore: Backend viewdef storage with session tracking
- Session: Connected browser session
- MessageBatcher: Priority-based message batching
- Frontend: Browser client
- ViewdefStore (FE): Frontend viewdef cache
- View: Rendered view with data-ui-viewdef attribute

## Sequence

```
FileWatcher      ViewdefStore       Session        MessageBatcher        Frontend       ViewdefStore(FE)      View
    |                 |                |                 |                   |                 |               |
    |--fileChanged--->|                |                 |                   |                 |               |
    |  Contact.COMPACT|                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |--reload------->|                 |                   |                 |               |
    |                 |  file content  |                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |--getSessions-->|                 |                   |                 |               |
    |                 |  that received |                 |                   |                 |               |
    |                 |  Contact.COMPACT                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |<--[s1, s2]----|                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |       [for each session]         |                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |----------------|-queueViewdef--->|                   |                 |               |
    |                 |                |  (id:1,viewdefs:|                   |                 |               |
    |                 |                |   high priority)|                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |                |---afterBatch--->|                   |                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |---send batch----->|                 |               |
    |                 |                |                 |  [{update,id:1,   |                 |               |
    |                 |                |                 |   props:viewdefs}]|                 |               |
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |--storeViewdefs->|               |
    |                 |                |                 |                   |  Contact.COMPACT|               |
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |--rerenderViews->|               |
    |                 |                |                 |                   |  ForKey         |               |
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |--querySelector|
    |                 |                |                 |                   |                 |  [data-ui-    |
    |                 |                |                 |                   |                 |  viewdef=     |
    |                 |                |                 |                   |                 |  Contact.     |
    |                 |                |                 |                   |                 |  COMPACT]     |
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |<--[view1,v2]--|
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |    [for each] |
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |--rerender---->|
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |--clear()
    |                 |                |                 |                   |                 |               |  unbind
    |                 |                |                 |                   |                 |               |  widgets
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |--render()
    |                 |                |                 |                   |                 |               |  new
    |                 |                |                 |                   |                 |               |  content
    |                 |                |                 |                   |                 |               |
```

## Notes

- Backend file watcher uses same pattern as LuaHotLoader
- Only sessions that have received the viewdef get the update (tracked in sentViewdefs)
- Push uses variable 1's `viewdefs` property with `:high` priority
- afterBatch triggers immediate flush to connected clients
- Frontend queries DOM by `data-ui-viewdef` attribute to find views
- Each matching view is re-rendered with same variable, unbinding old widgets
- ViewList item views also have `data-ui-viewdef` and are included in re-render
