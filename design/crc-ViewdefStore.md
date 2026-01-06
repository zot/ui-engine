# ViewdefStore

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- viewdefs: Map of TYPE.NAMESPACE to template element
- pendingUpdates: Batched viewdef updates awaiting delivery
- pendingViews: List of Views waiting for viewdefs to render

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

## Collaborators

- Viewdef: Individual viewdef instances (template elements)
- Variable: Variable 1 holds viewdefs property
- ProtocolHandler: Delivers viewdef updates
- View: Views waiting for viewdefs
- MessageBatcher: Queues viewdef updates with :high priority

## Sequences

- seq-load-viewdefs.md: Initial viewdef loading and validation
- seq-viewdef-delivery.md: Priority-based viewdef delivery
