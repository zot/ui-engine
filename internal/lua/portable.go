// CRC: crc-PortableOS.md | R146-R151
// Portable OS primitives for Lua apps, so they never shell out: mkdir -p, ls,
// test -d, base64 -d and POSIX shell syntax do not exist on Windows.
package lua

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// registerPortableOS adds the portable OS functions to the ui table.
func registerPortableOS(L *lua.LState, uiMod *lua.LTable) {
	L.SetField(uiMod, "mkdir", L.NewFunction(luaMkdir))
	L.SetField(uiMod, "isDir", L.NewFunction(luaIsDir))
	L.SetField(uiMod, "listDir", L.NewFunction(luaListDir))
	L.SetField(uiMod, "base64Decode", L.NewFunction(luaBase64Decode))
	L.SetField(uiMod, "exec", L.NewFunction(luaExec))
	L.SetField(uiMod, "platform", L.NewFunction(luaPlatform))
}

// pushErr pushes the (nil, message) failure pair.
func pushErr(L *lua.LState, err error) int {
	L.Push(lua.LNil)
	L.Push(lua.LString(err.Error()))
	return 2
}

// isDir reports whether path resolves to a directory, following links.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ui.mkdir(path) -> true | nil, err
// R146
func luaMkdir(L *lua.LState) int {
	if err := os.MkdirAll(L.CheckString(1), 0o755); err != nil {
		return pushErr(L, err)
	}
	L.Push(lua.LTrue)
	return 1
}

// ui.isDir(path) -> bool
// R147
func luaIsDir(L *lua.LState) int {
	L.Push(lua.LBool(isDir(L.CheckString(1))))
	return 1
}

// ui.listDir(path) -> {{name, isDir}, ...} | nil, err
// R148: os.ReadDir is sorted by name; isDir comes from Stat so links are followed
func luaListDir(L *lua.LState) int {
	dir := L.CheckString(1)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return pushErr(L, err)
	}
	list := L.NewTable()
	for _, e := range entries {
		row := L.NewTable()
		row.RawSetString("name", lua.LString(e.Name()))
		row.RawSetString("isDir", lua.LBool(isDir(filepath.Join(dir, e.Name()))))
		list.Append(row)
	}
	L.Push(list)
	return 1
}

// ui.base64Decode(text) -> bytes | nil, err
// R149
func luaBase64Decode(L *lua.LState) int {
	data, err := base64.StdEncoding.DecodeString(L.CheckString(1))
	if err != nil {
		return pushErr(L, err)
	}
	L.Push(lua.LString(data))
	return 1
}

// DefaultExecTimeout is ui.execAsync's limit when argv has no timeout.
// CRC: crc-ExecAsync.md | R252
const DefaultExecTimeout = 60 * time.Second

// ReadExec reads a Lua argv table: its array part as strings, and its timeout
// field in seconds (absent is DefaultExecTimeout, 0 is none). Call it on the
// executor.
// CRC: crc-PortableOS.md | R150, R252
func ReadExec(v lua.LValue) (argv []string, timeout time.Duration, err error) {
	tbl, ok := v.(*lua.LTable)
	if !ok {
		return nil, 0, errors.New("argv must be a table")
	}
	n := tbl.Len()
	if n == 0 {
		return nil, 0, errors.New("argv is empty")
	}
	argv = make([]string, n)
	for i := range argv {
		argv[i] = tbl.RawGetInt(i + 1).String()
	}
	switch t := tbl.RawGetString("timeout").(type) {
	case *lua.LNilType:
		timeout = DefaultExecTimeout
	case lua.LNumber:
		if t < 0 {
			return nil, 0, errors.New("argv.timeout must not be negative")
		}
		timeout = time.Duration(float64(t) * float64(time.Second))
	default:
		return nil, 0, errors.New("argv.timeout must be a number of seconds")
	}
	return argv, timeout, nil
}

// ExecArgv runs argv[0] with the rest as its arguments, directly with no shell,
// and returns its stdout. A failure carries the exit status and trimmed stderr.
// Cancelling ctx kills the program; WaitDelay keeps a grandchild that holds the
// pipes from stalling the return after a kill.
// CRC: crc-PortableOS.md | R150, R251
func ExecArgv(ctx context.Context, argv []string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return stdout.String(), nil
}

// ui.exec(argv) -> stdout | nil, err
// R150: no shell, so arguments pass verbatim; err carries exit status and stderr.
// R253: deprecated in favor of ui.execAsync; it blocks the executor and has no limit.
func luaExec(L *lua.LState) int {
	argv, _, err := ReadExec(L.CheckTable(1))
	if err != nil {
		L.ArgError(1, err.Error())
	}
	out, err := ExecArgv(context.Background(), argv)
	if err != nil {
		return pushErr(L, err)
	}
	L.Push(lua.LString(out))
	return 1
}

// ui.platform() -> {os, arch}
// R151
func luaPlatform(L *lua.LState) int {
	t := L.NewTable()
	t.RawSetString("os", lua.LString(runtime.GOOS))
	t.RawSetString("arch", lua.LString(runtime.GOARCH))
	L.Push(t)
	return 1
}
