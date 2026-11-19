# ExecAsync
**Requirements:** R250, R251, R252

`ui.execAsync(target, field, argv [, cb])`: the one host operation ui-engine defines
for every session. It runs a program the way `ui.exec` does, but off the executor, so
the session keeps working while the program runs (`specs/interfaces.md`,
`specs/host-ops.md`).

## Knows
- defaultExecTimeout: 60 seconds (R252)

## Does
- defineExecAsync(ref): `ref.Lua().DefineHostOp("execAsync", 0, op)`. The host-op
  deadline is 0 because the limit is per call; the work applies it (R250).
- op(args): on the executor, `lua.ReadExec(args[1])` reads `argv` and its `timeout`
  (default 60 seconds); a bad `argv` is an op error, delivered at once (R251, R252).
- work(ctx): with a limit, wrap ctx in `context.WithTimeout`; run `lua.ExecArgv(ctx,
  argv)`; return its stdout. When the limit expired, the error says the program timed
  out after the limit, instead of the bare kill status (R251, R252).

## Collaborators
- Server: `CreateLuaBackendForSession` calls defineExecAsync after `attachLua` and before
  `CreateLuaSession`, so `main.lua` can use it
- LuaRig: `DefineHostOp`, which delivers the stdout or the error (`crc-HostOp.md`)
- PortableOS: `ReadExec` and `ExecArgv`, shared with `ui.exec`

## Sequences
- seq-host-op.md (each call is an ordinary host-operation call)
- seq-session-create-backend.md (where it is defined)
