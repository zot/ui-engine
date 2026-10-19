// CRC: crc-PortableOS.md | R146-R151
// Portable OS primitives for Lua apps, so they never shell out: mkdir -p, ls,
// test -d, base64 -d and POSIX shell syntax do not exist on Windows.
package lua

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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

// ui.exec(argv) -> stdout | nil, err
// R150: no shell, so arguments pass verbatim; err carries exit status and stderr
func luaExec(L *lua.LState) int {
	tbl := L.CheckTable(1)
	n := tbl.Len()
	if n == 0 {
		L.ArgError(1, "argv is empty")
	}
	argv := make([]string, n)
	for i := range argv {
		argv[i] = tbl.RawGetInt(i + 1).String()
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return pushErr(L, err)
	}
	L.Push(lua.LString(stdout.String()))
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
