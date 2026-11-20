# Test Design: Portable OS Primitives
**Source:** crc-PortableOS.md

Every case runs Lua through a real runtime (`NewRuntime`) against a `t.TempDir()`
tree, so the contract under test is the one apps see. Nothing shells out, so the
suite runs unchanged on the Windows VM; links come from `sitetest.LinkDir` (a
symlink on Unix, a junction on Windows). The `ExecArgv` cases re-run the test binary
as the child (the standard Go helper-process pattern), so they need no Unix tool.

Measured on Windows 2026-10-19: the whole file passed on the `windows-gpu` VM
(build 26200) as the standard user `arktest`, run from a scheduled task (ssh
refuses to traverse a non-admin's junction, so it would give the wrong answer).
There `LinkDir` makes a real junction, so the `isDir` and `listDir` link cases
prove junctions are followed.

## Test: mkdir creates parents and accepts an existing directory
**Purpose:** `mkdir -p` semantics, and the failure shape (R146)
**Input:** `ui.mkdir("<tmp>/a/b/c")` twice; then `ui.mkdir` on a path under a regular file
**Expected:** `true` both times and `a/b/c` is a directory; the third returns `nil` and a non-empty error string
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go

## Test: isDir follows links and never fails
**Purpose:** a linked directory counts; files and missing paths are `false`, not errors (R147)
**Input:** a directory, a file, a link to the directory, a missing path
**Expected:** `true`, `false`, `true`, `false`
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go
**Alarm:** 1
**Fire alarm:** test with `os.Lstat` instead of `os.Stat`. On Linux the plain directory still passes, so the linked case is what objects. Red: the link reports `false`.
**Inject:** internal/lua/portable.go:luaIsDir
**Pulled:** 2026-10-19 — rang: `isDir(link) = false, want true`

## Test: listDir includes hidden entries, sorted, with linked isDir
**Purpose:** the `ls -1A` replacement keeps dot-entries and follows links (R148)
**Input:** `.hidden` (file), `b` (dir), `a` (file), `link` → `b`; then a missing directory
**Expected:** names `.hidden, a, b, link` in that order with `isDir` `false, false, true, true`; the missing directory returns `nil, err`
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go
**Alarm:** 2
**Fire alarm:** take `isDir` from the `DirEntry` (which does not follow links) instead of `os.Stat`. Red: `link` reports `isDir = false`.
**Inject:** internal/lua/portable.go:luaListDir
**Pulled:** 2026-10-19 — rang: `listDir = … link:false, want … link:true`

## Test: base64Decode returns raw bytes
**Purpose:** binary-safe decode for writing images; invalid input is an error (R149)
**Input:** the encoding of bytes `00 ff 10 80 0a`; then `"not base64!"`
**Expected:** a 5-byte Lua string equal to the original bytes; then `nil, err`
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go

## Test: running a program passes arguments verbatim with no shell
**Purpose:** the property that makes running a program portable: no quoting, splitting, or expansion. `ExecArgv` is what `ui.execAsync` runs (R266)
**Input:** `ExecArgv(ctx, [<test binary>, "-test.run=^TestHelperProcess$", "--", "echo", "a b", "$HOME", "x>y"])` with the helper env set; the helper prints each argument on its own line
**Expected:** stdout is exactly `a b\n$HOME\nx>y\n`
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go
**Alarm:** 3
**Fire alarm:** run the argv through a shell as one joined string (`sh -c` on Unix). Red: the shell takes `x>y` as a redirect, so stdout comes back empty (and `a b` would split, `$HOME` expand). The redirect writes a file `y` in the package directory; delete it after pulling.
**Inject:** internal/lua/portable.go:ExecArgv
@alarm-hash: e2d92d2befdf6fc1
**Pulled:** 2026-11-20 — re-pulled after the test moved from `ui.exec` to `ExecArgv` (ui.exec removed): shell join, rang `exec stdout = "", want "a b\n$HOME\nx>y\n"` *Earlier —* 2026-11-19 — re-pulled after 11a moved the run into ExecArgv: shell injection, rang `exec stdout = "", want "a b\n$HOME\nx>y\n"` *Earlier —* 2026-10-19 — rang: `exec stdout = "", want "a b\n$HOME\nx>y\n"`
## Test: a failing program's error carries the exit status and stderr
**Purpose:** a failing command returns an error that says why (R266)
**Input:** `ExecArgv` with the helper told to write `boom` to stderr and exit 3; then with a program that does not exist
**Expected:** first: no output and an error containing `3` and `boom`; second: a non-empty error
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go
**Alarm:** 4
**Fire alarm:** drop the stderr from the error message (return `err.Error()` only). Red: the error lacks `boom`.
**Inject:** internal/lua/portable.go:ExecArgv
@alarm-hash: e2d92d2befdf6fc1
**Pulled:** 2026-11-20 — re-pulled after the test moved from `ui.exec` to `ExecArgv` (ui.exec removed): stderr dropped, rang `exec(fail) = "", exit status 3; want no output and an error naming exit 3 and boom` *Earlier —* 2026-11-19 — re-pulled after 11a moved the run into ExecArgv: stderr dropped, rang `exec(fail) = nil, "exit status 3"; want nil and an error naming exit 3 and boom` *Earlier —* 2026-10-19 — rang: `exec(fail) = nil, "exit status 3"; want … boom`
## Test: there is no blocking way to run a program
**Purpose:** `ui.exec` is gone, so no Lua call can block the session on a program (R267)
**Input:** a runtime; Lua `type(ui.exec)`
**Expected:** `nil`
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go
**Alarm:** 5
**Fire alarm:** register an `exec` function in the `ui` table again; the type is `function`
**Inject:** internal/lua/portable.go:registerPortableOS
@alarm-hash: 756cc38092f772e6
**Pulled:** 2026-11-20 — `ui.exec` registered again, rang `type(ui.exec) = function, want nil: Lua must have no blocking way to run a program (R267)`

## Test: platform reports GOOS and GOARCH
**Purpose:** apps branch on the server's platform without `uname` (R151)
**Input:** `ui.platform()`
**Expected:** `os == runtime.GOOS`, `arch == runtime.GOARCH`
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go
