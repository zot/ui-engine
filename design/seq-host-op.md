# Sequence: Host Operation

**Source Spec:** host-ops.md
**Requirements:** R205, R206, R207, R208, R209, R210, R211, R212, R213, R214, R215, R216

## Participants

- Host: defines the operation, on the executor
- Lua: app code calling `ui[name]`, on the executor
- HostOp: the binding
- Tailor: the call's sewing
- Worker: the goroutine running work
- Executor: runs the delivery stitch

## Sequence: one call

```
1.1. Host -> LuaRig.DefineHostOp(name, deadline, op)
  1.1.1. ui[name] = binding
1.2. Lua -> ui[name](target, field, args..., [cb])
  1.2.1. target not a table or field not a string -> Lua argument error
  1.2.2. trailing function -> cb, removed from args
  1.2.3. work, err = op(args)
  1.2.4. ExecRig.Tailor(name, 2 * deadline)
  1.2.5. err -> deliver(nil, err); return
  1.2.6. go Worker; return nothing
1.3. Worker
  1.3.1. ctx = Tailor.Context(), with the deadline when non-zero
  1.3.2. result, err = work(ctx)
  1.3.3. deliver(result, err)
1.4. deliver -> Tailor.EndUpdate(stitch)
  1.4.1. Executor: success -> target[field] = GoToLua(result); target[field.."Error"] = nil
  1.4.2. Executor: failure -> target[field] = nil; target[field.."Error"] = message
  1.4.3. Executor: cb -> cb(result, nil) or cb(nil, message); a raise fails the stitch
  1.4.4. change detection runs after the stitch
1.5. work outlives twice the deadline -> Tailor closes as failed; the late EndUpdate is dropped
```
