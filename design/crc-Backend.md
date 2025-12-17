# Backend

**Source Spec:** main.md (UI Server Architecture section)

## Responsibilities

### Knows
- session: Reference to owning Session

### Does
- Watch: Subscribe connection to variable changes (LuaBackend manages tally; ProxiedBackend relays)
- Unwatch: Unsubscribe connection from variable (LuaBackend manages tally; ProxiedBackend relays)
- UnwatchAll: Remove all watches for a connection (on disconnect)
- HandleMessage: Process protocol message batch (LuaBackend processes locally; ProxiedBackend relays to external)
- Shutdown: Clean up backend resources

## Collaborators

- Session: Owns this backend, delegates protocol messages
- LuaBackend: Concrete implementation for hosted Lua (processes messages, detects changes)
- ProxiedBackend: Concrete implementation for external backends (relays messages)

## Sequences

- seq-session-create-backend.md: Creating backend when session starts
- seq-backend-watch.md: Watch subscription flow through Backend interface

## Notes

- **Interface**: Backend is an interface, not a concrete class
- **Per-session scope**: Each session has exactly one Backend instance
- **Asymmetric implementations**:
  - LuaBackend: Processes messages locally, owns per-session change-tracker, manages watch tallies
  - ProxiedBackend: Pure relay to external backend, no local processing
- **No DetectChanges in interface**: Only LuaBackend needs change detection; ProxiedBackend external backend handles its own
