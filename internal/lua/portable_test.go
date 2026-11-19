// CRC: crc-PortableOS.md | Test: test-PortableOS.md
package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/site/sitetest"
)

const helperEnv = "UI_ENGINE_TEST_HELPER"

// TestHelperProcess is the child for the ui.exec cases, not a test: with
// helperEnv set it acts on the arguments after "--" and exits.
func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	switch args[0] {
	case "echo":
		for _, a := range args[1:] {
			fmt.Println(a)
		}
		os.Exit(0)
	case "fail":
		fmt.Fprint(os.Stderr, "boom")
		os.Exit(3)
	}
	os.Exit(99)
}

// runPortable runs code in a fresh runtime with the given string globals set,
// then returns the named globals.
func runPortable(t *testing.T, globals map[string]string, code string, results ...string) []lua.LValue {
	t.Helper()
	rt, err := NewRuntime(config.DefaultConfig(), t.TempDir(), nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Shutdown()
	var out []lua.LValue
	rt.execute(func() (interface{}, error) {
		L := rt.State
		for k, v := range globals {
			L.SetGlobal(k, lua.LString(v))
		}
		if err := L.DoString(code); err != nil {
			t.Fatalf("lua: %v", err)
		}
		for _, name := range results {
			out = append(out, L.GetGlobal(name))
		}
		return nil, nil
	})
	return out
}

// Test: mkdir creates parents and accepts an existing directory (R146)
func TestUIMkdir(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "file")
	os.WriteFile(file, nil, 0o644)
	deep := filepath.Join(tmp, "a", "b", "c")
	r := runPortable(t, map[string]string{"deep": deep, "under": filepath.Join(file, "x")}, `
		first = ui.mkdir(deep)
		second = ui.mkdir(deep)
		bad, badErr = ui.mkdir(under)
	`, "first", "second", "bad", "badErr")
	if r[0] != lua.LTrue || r[1] != lua.LTrue {
		t.Errorf("mkdir = %v, %v; want true, true", r[0], r[1])
	}
	if info, err := os.Stat(deep); err != nil || !info.IsDir() {
		t.Errorf("%s is not a directory: %v", deep, err)
	}
	if r[2] != lua.LNil || r[3].String() == "" {
		t.Errorf("mkdir under a file = %v, %q; want nil, error", r[2], r[3])
	}
}

// Test: isDir follows links and never fails (R147)
func TestUIIsDir(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "dir")
	os.Mkdir(dir, 0o755)
	file := filepath.Join(tmp, "file")
	os.WriteFile(file, nil, 0o644)
	link := filepath.Join(tmp, "link")
	sitetest.LinkDir(t, dir, link)
	r := runPortable(t, map[string]string{"dir": dir, "file": file, "link": link, "missing": filepath.Join(tmp, "nope")}, `
		d, f, l, m = ui.isDir(dir), ui.isDir(file), ui.isDir(link), ui.isDir(missing)
	`, "d", "f", "l", "m")
	want := []lua.LValue{lua.LTrue, lua.LFalse, lua.LTrue, lua.LFalse}
	for i, name := range []string{"dir", "file", "link", "missing"} {
		if r[i] != want[i] {
			t.Errorf("isDir(%s) = %v, want %v", name, r[i], want[i])
		}
	}
}

// Test: listDir includes hidden entries, sorted, with linked isDir (R148)
func TestUIListDir(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, ".hidden"), nil, 0o644)
	os.WriteFile(filepath.Join(tmp, "a"), nil, 0o644)
	os.Mkdir(filepath.Join(tmp, "b"), 0o755)
	sitetest.LinkDir(t, filepath.Join(tmp, "b"), filepath.Join(tmp, "link"))
	r := runPortable(t, map[string]string{"tmp": tmp, "missing": filepath.Join(tmp, "nope")}, `
		entries = ui.listDir(tmp)
		gone, goneErr = ui.listDir(missing)
	`, "entries", "gone", "goneErr")
	list, ok := r[0].(*lua.LTable)
	if !ok {
		t.Fatalf("listDir = %v, want a table", r[0])
	}
	var got []string
	for i := 1; i <= list.Len(); i++ {
		row := list.RawGetInt(i).(*lua.LTable)
		got = append(got, fmt.Sprintf("%s:%v", row.RawGetString("name"), row.RawGetString("isDir")))
	}
	if want := ".hidden:false a:false b:true link:true"; strings.Join(got, " ") != want {
		t.Errorf("listDir = %s, want %s", strings.Join(got, " "), want)
	}
	if r[1] != lua.LNil || r[2].String() == "" {
		t.Errorf("listDir(missing) = %v, %q; want nil, error", r[1], r[2])
	}
}

// Test: base64Decode returns raw bytes (R149)
func TestUIBase64Decode(t *testing.T) {
	r := runPortable(t, nil, `
		bytes = ui.base64Decode("AP8QgAo=")
		bad, badErr = ui.base64Decode("not base64!")
	`, "bytes", "bad", "badErr")
	if got := r[0].String(); got != "\x00\xff\x10\x80\n" {
		t.Errorf("base64Decode = %q, want %q", got, "\x00\xff\x10\x80\n")
	}
	if r[1] != lua.LNil || r[2].String() == "" {
		t.Errorf("base64Decode(invalid) = %v, %q; want nil, error", r[1], r[2])
	}
}

// Test: exec passes arguments verbatim with no shell (R150)
func TestUIExecVerbatim(t *testing.T) {
	t.Setenv(helperEnv, "1")
	r := runPortable(t, map[string]string{"bin": os.Args[0]}, `
		out, err = ui.exec({bin, "-test.run=^TestHelperProcess$", "--", "echo", "a b", "$HOME", "x>y"})
	`, "out", "err")
	if r[1] != lua.LNil {
		t.Fatalf("exec failed: %s", r[1])
	}
	if got := strings.ReplaceAll(r[0].String(), "\r\n", "\n"); got != "a b\n$HOME\nx>y\n" {
		t.Errorf("exec stdout = %q, want %q", got, "a b\n$HOME\nx>y\n")
	}
}

// Test: exec failure carries the exit status and stderr (R150)
func TestUIExecFailure(t *testing.T) {
	t.Setenv(helperEnv, "1")
	r := runPortable(t, map[string]string{"bin": os.Args[0], "nobin": filepath.Join(t.TempDir(), "no-such-program")}, `
		out, err = ui.exec({bin, "-test.run=^TestHelperProcess$", "--", "fail"})
		out2, err2 = ui.exec({nobin})
	`, "out", "err", "out2", "err2")
	if r[0] != lua.LNil || !strings.Contains(r[1].String(), "3") || !strings.Contains(r[1].String(), "boom") {
		t.Errorf("exec(fail) = %v, %q; want nil and an error naming exit 3 and boom", r[0], r[1])
	}
	if r[2] != lua.LNil || r[3].String() == "" {
		t.Errorf("exec(missing) = %v, %q; want nil, error", r[2], r[3])
	}
}

// Test: platform reports GOOS and GOARCH (R151)
func TestUIPlatform(t *testing.T) {
	r := runPortable(t, nil, `p = ui.platform(); os_, arch = p.os, p.arch`, "os_", "arch")
	if r[0].String() != runtime.GOOS || r[1].String() != runtime.GOARCH {
		t.Errorf("platform = %s/%s, want %s/%s", r[0], r[1], runtime.GOOS, runtime.GOARCH)
	}
}

// Test: the limit's default and its off switch (R252)
func TestReadExecTimeout(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(`a = {"x"} b = {"x", timeout = 0} c = {} d = "x" e = {"x", timeout = "soon"}`); err != nil {
		t.Fatal(err)
	}
	if _, to, err := ReadExec(L.GetGlobal("a")); err != nil || to != DefaultExecTimeout {
		t.Errorf("ReadExec({x}) timeout = %v, %v; want %v", to, err, DefaultExecTimeout)
	}
	if argv, to, err := ReadExec(L.GetGlobal("b")); err != nil || to != 0 || len(argv) != 1 {
		t.Errorf("ReadExec({x, timeout=0}) = %v, %v, %v; want [x], 0, nil", argv, to, err)
	}
	for _, name := range []string{"c", "d", "e"} {
		if _, _, err := ReadExec(L.GetGlobal(name)); err == nil {
			t.Errorf("ReadExec(%s) succeeded, want an error", name)
		}
	}
}
