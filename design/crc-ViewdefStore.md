# ViewdefStore

**Source Spec:** viewdefs.md, main.md "Hot-Loading System"

## Responsibilities

### Knows
- viewdefs: Map of TYPE.NAMESPACE to template element
- pendingUpdates: Batched viewdef updates awaiting delivery
- pendingViews: List of Views waiting for viewdefs to render
- fileWatcher: (backend) File watcher for viewdef directory (like LuaHotLoader)
- sentViewdefs: (backend) Map of session ID to set of sent viewdef keys
- symlinkTargets: (backend) Map of symlink paths to their resolved target directories
- watchedDirs: (backend) Set of directories currently being watched

### Does
- store: Add or replace viewdef by TYPE.NAMESPACE key
- get: Retrieve viewdef by TYPE.NAMESPACE, falls back to TYPE.DEFAULT
- getForType: Get all viewdefs for a type
- has: Check if viewdef exists for TYPE.NAMESPACE
- validate: Parse HTML string, verify single template root element
- batchUpdate: Queue viewdef update for batching
- flushUpdates: Send batched updates to frontend via variable 1 with :high priority
- remove: Delete viewdef
- addPendingView: Add view to pending list (missing type or viewdef)
- processPendingViews: Re-render pending views when viewdefs arrive
- removePendingView: Remove view from pending list after successful render
- startWatching: (backend) Start file watcher for viewdef directory
- stopWatching: (backend) Stop file watcher
- handleFileChange: (backend) Reload viewdef, queue re-push for sessions that received it
- resolveSymlinks: (backend) Scan viewdef directory for symlinks, resolve and watch target directories
- updateSymlinkWatches: (backend) When symlinks change, update watched directories accordingly
- rerenderViewsForKey: (frontend) Query `[data-ui-viewdef="KEY"]`, call rerender() on each

## Collaborators

- Viewdef: Individual viewdef instances (template elements)
- Variable: Variable 1 holds viewdefs property
- ProtocolHandler: Delivers viewdef updates
- View: Views waiting for viewdefs, views to re-render on hot-reload
- MessageBatcher: Queues viewdef updates with :high priority
- LuaHotLoader: (backend) Similar file watching pattern

## Notes

### Backend Hot-Reload

When hot-loading is enabled:
1. File watcher monitors viewdef directory (like LuaHotLoader)
2. **Symlink tracking**: See cross-cutting concern "Hot-Loading Symlink Tracking"
3. On file change, reload content and update viewdefs map
4. For each session that has received the changed viewdef (tracked in sentViewdefs):
   - Queue a re-push via variable 1's `viewdefs` property with :high priority
5. This triggers `ws.afterBatch` on connected clients

### Frontend Hot-Reload

When updated viewdefs arrive via variable 1:
1. Store the updated viewdefs
2. For each updated viewdef key:
   - Call `rerenderViewsForKey(key)` to find and re-render matching views
3. Re-rendering unbinds old widgets and creates new ones

## Sequences

- seq-load-viewdefs.md: Initial viewdef loading and validation
- seq-viewdef-delivery.md: Priority-based viewdef delivery
- seq-viewdef-hotload.md: Hot-reload flow (file change → push → re-render)
