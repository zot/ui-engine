# HostOp
**Requirements:** R205, R206, R207, R208, R209, R210, R211, R212, R213, R214, R215, R216

An asynchronous `ui.*` binding a host defines. Each call is one Tailor: the binding
reads its arguments on the executor, the work runs on its own goroutine, and one
terminal active stitch delivers the result into Lua state.

## Knows
- HostOp: the host's function type, `func(args []LValue) (work, error)` (R209)
- per definition: name, deadline, op, and the SessionRef of the LuaRig it was defined on
- per call (captured by the call's closures, touched only on the executor): target,
  field, callback (R206, R208)

## Does
- define (`LuaRig.DefineHostOp`): on the executor, set `ui[name]` to the binding (R205).
- binding (on the executor): check `target` is a table and `field` a string, raising a
  Lua argument error otherwise (R206); take a trailing function as the callback (R208);
  call op with the rest (R209); create a Tailor named after the operation with twice the
  deadline, or none (R211, R216); with op's error, deliver it at once (R210); otherwise
  start work on a goroutine; return nothing (R207).
- run work (off the executor): give it the Tailor's context, narrowed to the deadline when
  there is one (R211, R216); deliver what it returns.
- deliver: post the Tailor's terminal active stitch (`EndUpdate`) (R212), which writes
  `target[field]` and `target[field .. "Error"]` (R213, R214), then calls the callback
  with the result or the message (R215). A raising callback fails the stitch, which the
  Tailor logs with its name.

## Collaborators
- LuaRig: the session's Lua state for the definition and the `GoToLua` conversion
- ExecRig: creating the Tailor
- Tailor: the sewing, its context and deadline backstop, the delivery stitch

## Sequences
- seq-host-op.md
