# ChangeDetector

**Source Spec:** libraries.md

## Responsibilities

### Knows
- watchedVariables: Set of variable IDs being watched
- previousValues: Map of variable ID to last known value
- pendingRefresh: Flag indicating refresh is needed
- throttleInterval: Minimum time between background refreshes

### Does
- addWatch: Start tracking variable for changes
- removeWatch: Stop tracking variable
- refresh: Compute values for all watched variables, detect changes
- detectChange: Compare current value to previous
- scheduleRefresh: Queue background-triggered refresh with throttling
- sendUpdates: Send update messages for changed variables
- afterMessage: Trigger refresh after client message receipt

## Collaborators

- BackendConnection: Triggers refresh cycle
- PathNavigator: Computes variable values
- ProtocolHandler: Sends update messages
- WatchManager: Coordinates with frontend watches

## Sequences

- seq-backend-refresh.md: Full refresh cycle
- seq-update-variable.md: Change propagation
