# Host Operations — asynchronous `ui.*` bindings

**Language:** Go, Lua
**Environment:** ui-engine backend (`internal/server`, re-exported through `cli`)

## Purpose

A host (Frictionless, ark) often needs to give Lua apps an operation that does slow work
off the session executor: run fossil, read many files, call the network. Lua must never
spawn the host to get it, and the work must not block the executor. A **host operation**
is such an operation, exposed as a `ui.*` function: Lua calls it, it returns at once, the
work runs on a goroutine, and the result lands in Lua state, where change detection
pushes it to the browser.

Host operations are built on Tailor (`specs/tailor.md`): each call is one sewing.

## Lua

```lua
ui.op(target, field, args...)        -- result lands in target[field]
ui.op(target, field, args..., cb)    -- and then cb(result, err) is called
```

- `target` is a table and `field` a string; anything else is a Lua argument error, raised
  at the call.
- The call returns nothing, at once.
- When the work finishes, one active stitch writes both fields and runs change detection:
  - on success, `target[field]` is the result and `target[field .. "Error"]` is nil;
  - on failure, `target[field]` is nil and `target[field .. "Error"]` is the error
    message.
- A trailing function argument is a callback, called after the fields are written as
  `cb(result, nil)` on success and `cb(nil, message)` on failure. It is not passed to the
  operation as an argument.
- Lua code normally reacts to the fields changing, not to a callback.

## Go

```go
// HostOp reads the call's arguments on the executor and returns the work to run off it.
type HostOp func(args []lua.LValue) (work func(ctx context.Context) (any, error), err error)

// DefineHostOp installs ui[name] in this session's Lua state. Call it on the executor.
func (v LuaView) DefineHostOp(name string, deadline time.Duration, op HostOp)
```

- When Lua calls `ui[name]`, the binding checks `target` and `field`, removes a trailing
  callback, and calls `op` with the remaining arguments, still on the executor, so `op`
  may read Lua values. `op` returns a `work` function that captures plain Go values.
- If `op` returns an error, it is delivered to the fields and callback as a failure, and
  no work runs.
- Each call creates a Tailor named after the operation. `work` runs on its own goroutine,
  given a context that is cancelled when the deadline passes or the session ends. The
  result is delivered by the Tailor's terminal active stitch.
- The result is converted to Lua the same way other Go values are (`GoToLua`).
- **Deadline:** a deadline of 0 means none. With a deadline, `work`'s context expires
  after it, and a `work` that returns the context's error delivers it as a failure
  ("context deadline exceeded"). A `work` that ignores its context is a leak the Tailor
  bounds: the Tailor closes as failed after twice the deadline, and nothing is delivered.
- A callback that raises an error is logged with the operation's name, like any failing
  stitch.

## Non-goals

- No pending marker field. Add one when a caller needs it.
- No specific operations. Hosts define them; Frictionless's checkpoint status is the
  first.
