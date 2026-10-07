# Sequence: Viewdef Hot-Reload

**Source Spec:** viewdefs.md
**Use Case:** Viewdef file changes trigger push to frontend and view re-rendering

## Participants

- FileWatcher: Monitors each viewdef source directory for changes
- ViewdefStore: Backend viewdef storage with session tracking
- Session: Connected browser session
- MessageBatcher: Priority-based message batching
- Frontend: Browser client
- ViewdefStore (FE): Frontend viewdef cache
- View: Rendered view with ui-viewdef attribute

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
    |                 |                |                 |                   |                 |               |--clearChildren()
    |                 |                |                 |                   |                 |               |  destroy child
    |                 |                |                 |                   |                 |               |  views/viewlists
    |                 |                |                 |                   |                 |               |  (each destroys
    |                 |                |                 |                   |                 |               |  its variable)
    |                 |                |                 |                   |                 |               |
    |                 |                |                 |                   |                 |               |--render()
    |                 |                |                 |                   |                 |               |  new content
    |                 |                |                 |                   |                 |               |  (creates new
    |                 |                |                 |                   |                 |               |  child variables)
    |                 |                |                 |                   |                 |               |
```

## Links

The watcher watches each viewdef source through its `lua/<app>` entry, so a
changed file is reported under the path it was loaded from; no reverse lookup
is needed. If the changed file's key is owned by a higher-ranked source, the
change is ignored (seq-app-discovery.md, collisions). See cross-cutting concern
"Hot-Loading Through Activation Entries" in design.md.

## Notes

- Backend file watcher uses same pattern as LuaHotLoader
- **Links**: sources are watched through their activation entries
- Only sessions that have received the viewdef get the update (tracked in sentViewdefs)
- Push uses variable 1's `viewdefs` property with `:high` priority
- afterBatch triggers immediate flush to connected clients
- Frontend queries DOM by `ui-viewdef` attribute to find views
- Each matching view is re-rendered with same variable, unbinding old widgets
- ViewList item views also have `ui-viewdef` and are included in re-render
- **Variable destruction**: When child views/viewlists are destroyed during re-render, they destroy their associated variables to prevent leaks
