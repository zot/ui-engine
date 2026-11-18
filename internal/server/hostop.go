package server

import (
	"context"
	"time"

	gopher "github.com/yuin/gopher-lua"
)

// HostOp reads a host operation's arguments on the executor and returns the work
// to run off it. work must capture plain Go values, never Lua ones.
// CRC: crc-HostOp.md | R209
type HostOp func(args []gopher.LValue) (work func(ctx context.Context) (any, error), err error)

// hostOpCall is one call of a host operation: its Tailor and where the result
// lands. target, field and cb are Lua values, touched only on the executor.
// CRC: crc-HostOp.md | R211
type hostOpCall struct {
	t        *Tailor
	deadline time.Duration
	target   *gopher.LTable
	field    string
	cb       *gopher.LFunction
}

// DefineHostOp installs ui[name] in this session's Lua state. Call it on the
// executor. A deadline of 0 means none.
// CRC: crc-HostOp.md | Seq: seq-host-op.md#1.1 | R205
func (v LuaView) DefineHostOp(name string, deadline time.Duration, op HostOp) {
	ls := v.Session()
	if ls == nil {
		return
	}
	L := ls.State
	ui, ok := L.GetGlobal("ui").(*gopher.LTable)
	if !ok {
		return
	}
	L.SetField(ui, name, L.NewFunction(func(L *gopher.LState) int {
		return v.callHostOp(L, name, deadline, op)
	}))
}

// callHostOp is the binding: check the target and field, take a trailing
// callback, let op read the arguments, then run the work off the executor.
// CRC: crc-HostOp.md | Seq: seq-host-op.md#1.2 | R206, R207, R208, R209, R210, R211, R216
func (v LuaView) callHostOp(L *gopher.LState, name string, deadline time.Duration, op HostOp) int {
	field, ok := L.Get(2).(gopher.LString)
	if !ok {
		L.ArgError(2, "field must be a string")
	}
	c := &hostOpCall{deadline: deadline, target: L.CheckTable(1), field: string(field)}
	top := L.GetTop()
	if fn, ok := L.Get(top).(*gopher.LFunction); ok && top > 2 {
		c.cb = fn
		top--
	}
	args := make([]gopher.LValue, 0, top-2)
	for i := 3; i <= top; i++ {
		args = append(args, L.Get(i))
	}
	work, err := op(args)
	c.t = v.ref.Exec().Tailor(name, 2*deadline)
	if err != nil {
		c.deliver(nil, err)
		return 0
	}
	go c.run(work)
	return 0
}

// run executes work off the executor with the Tailor's context, narrowed to the
// deadline when there is one, and delivers what it returns.
// CRC: crc-HostOp.md | Seq: seq-host-op.md#1.3 | R211, R216
func (c *hostOpCall) run(work func(context.Context) (any, error)) {
	ctx := c.t.Context()
	if c.deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.deadline)
		defer cancel()
	}
	result, err := work(ctx)
	c.deliver(result, err)
}

// deliver posts the terminal active stitch that writes both fields and then calls
// the callback. A raising callback fails the stitch, which the Tailor logs.
// CRC: crc-HostOp.md | Seq: seq-host-op.md#1.4 | R212, R213, R214, R215
func (c *hostOpCall) deliver(result any, err error) {
	c.t.EndUpdate(func() error {
		L := c.t.State()
		var value, message gopher.LValue = gopher.LNil, gopher.LNil
		if err != nil {
			message = gopher.LString(err.Error())
		} else {
			value = c.t.Ref().Lua().Session().GoToLua(result)
		}
		L.SetField(c.target, c.field, value)
		L.SetField(c.target, c.field+"Error", message)
		if c.cb == nil {
			return nil
		}
		return L.CallByParam(gopher.P{Fn: c.cb, NRet: 0, Protect: true}, value, message)
	})
}
