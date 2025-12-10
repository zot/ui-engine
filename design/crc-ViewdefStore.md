# ViewdefStore

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- viewdefs: Map of TYPE.VIEW to Viewdef
- pendingUpdates: Batched viewdef updates awaiting delivery

### Does
- store: Add or replace viewdef by TYPE.VIEW key
- get: Retrieve viewdef by TYPE.VIEW key
- getForType: Get all viewdefs for a presenter type
- has: Check if viewdef exists
- batchUpdate: Queue viewdef update for batching
- flushUpdates: Send batched updates to frontend via variable 1
- remove: Delete viewdef

## Collaborators

- Viewdef: Individual viewdef instances
- Variable: Variable 1 holds viewdefs property
- ProtocolHandler: Delivers viewdef updates
- MCPTool: Creates viewdefs via MCP

## Sequences

- seq-load-viewdefs.md: Initial viewdef loading
- seq-mcp-create-presenter.md: AI creating viewdefs
