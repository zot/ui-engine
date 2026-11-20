package server

import (
	"context"
	"errors"
	"fmt"

	gopher "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/lua"
)

// defineExecAsync installs ui.execAsync(target, field, argv [, cb]) in the
// session's Lua state: it runs a program with no shell, off the executor. Its
// host-op deadline is 0 because each call carries its own limit, applied by the work.
// CRC: crc-ExecAsync.md | Seq: seq-session-create-backend.md | R250, R252, R266
func defineExecAsync(ref SessionRef) {
	ref.Lua().DefineHostOp("execAsync", 0, func(args []gopher.LValue) (func(context.Context) (any, error), error) {
		if len(args) == 0 {
			return nil, errors.New("execAsync: argv is required")
		}
		argv, timeout, err := lua.ReadExec(args[0])
		if err != nil {
			return nil, fmt.Errorf("execAsync: %w", err)
		}
		return func(ctx context.Context) (any, error) {
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			out, err := lua.ExecArgv(ctx, argv)
			if err != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return nil, fmt.Errorf("%s timed out after %s", argv[0], timeout)
				}
				return nil, err
			}
			return out, nil
		}, nil
	})
}
