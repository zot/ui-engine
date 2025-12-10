# WatchManager

**Source Spec:** protocol.md

## Responsibilities

### Knows
- watchCounts: Map of variable ID to observer count
- watchers: Map of variable ID to list of connection IDs
- inactiveVariables: Set of variable IDs marked inactive

### Does
- watch: Add observer for variable, increment tally
- unwatch: Remove observer, decrement tally
- shouldForwardWatch: Return true if tally changed 0->1 (bound variables)
- shouldForwardUnwatch: Return true if tally changed 1->0 (bound variables)
- notifyWatchers: Send update to all observers when variable changes
- isInactive: Check if variable or ancestor has inactive property set
- getWatcherCount: Return current observer count for variable

## Collaborators

- Variable: Tracks individual watch state
- ProtocolHandler: Receives watch/unwatch messages
- MessageRelay: Sends updates to watchers
- VariableStore: Checks inactive property

## Sequences

- seq-watch-variable.md: Watch subscription and tallying
- seq-update-variable.md: Notifying watchers of changes
