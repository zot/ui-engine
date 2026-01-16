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

## Symlink Resolution

Symlink handling follows the same pattern as LuaHotLoader (see seq-lua-hotload.md "Symlink Resolution Sequence"):
- If a viewdef file is a symlink, the server also watches the target directory
- Changes to symlink targets reload as if the symlink file changed
- See cross-cutting concern "Hot-Loading Symlink Tracking" in design.md

## Notes

- Backend file watcher uses same pattern as LuaHotLoader
- **Symlink tracking**: All hot-loading follows the same symlink resolution pattern
- Only sessions that have received the viewdef get the update (tracked in sentViewdefs)
- Push uses variable 1's `viewdefs` property with `:high` priority
- afterBatch triggers immediate flush to connected clients
- Frontend queries DOM by `data-ui-viewdef` attribute to find views
- Each matching view is re-rendered with same variable, unbinding old widgets
- ViewList item views also have `data-ui-viewdef` and are included in re-render
- **Variable destruction**: When child views/viewlists are destroyed during re-render, they destroy their associated variables to prevent leaks
