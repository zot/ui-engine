# Configuration and Logging

## Overview

The `Config` object is the central source of truth for application settings, including server options, session management, and logging configuration. It is responsible for loading values from CLI flags, environment variables, and configuration files, prioritizing them correctly.

## Logging Architecture

To ensure consistent output formatting and centralized verbosity control, **all logging must be performed through the `Config` object**. 

### The `Log` Method

The `Config` object exposes a `Log` method that handles verbosity filtering and formatting:

```go
func (c *Config) Log(level int, format string, args ...interface{})
```

- **level**: The verbosity level required to see this message (1-4).
- **format**: A `printf`-style format string.
- **args**: Arguments for the format string.

### Verbosity Levels

The system supports granular verbosity levels to control debug output. These levels are managed centrally by `Config`.

| Level | Flag    | Scope                                            | Description                                      |
|-------|---------|--------------------------------------------------|--------------------------------------------------|
| 0     | (none)  | Errors only                                      | Standard operation; silence is golden.           |
| 1     | `-v`    | Connection events                                | Client connect, disconnect, reconnect events.    |
| 2     | `-vv`   | Protocol messages                                | High-level protocol operations (Create, Watch).  |
| 3     | `-vvv`  | Variable operations                              | Detailed CRUD operations and property changes.   |
| 4     | `-vvvv` | Variable values                                  | Full value dumps (can be large).                 |

### Subsystem Integration and Delegation

Subsystems (e.g., `Runtime`, `Backend`, `ProtocolHandler`) **must not** maintain their own internal verbosity state or flags. Instead, they must follow this delegation pattern:

1.  **Reference**: The subsystem holds a reference to the `Config` instance.
2.  **Delegation**: The subsystem exposes its own `Log` method that delegates strictly to `Config.Log`.
3.  **No Direct Logging**: Direct use of the standard library `log` or `fmt` packages for operational output is prohibited in core components.

**Example Pattern:**

```go
type Subsystem struct {
    cfg *Config
}

func (s *Subsystem) Log(level int, format string, args ...interface{}) {
    s.cfg.Log(level, format, args...)
}
```
