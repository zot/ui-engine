# PortableOS
**Requirements:** R146, R147, R148, R149, R150, R151

The `ui.*` functions that do file and process work in Go, so apps never shell out.
`mkdir -p`, `ls`, `test -d`, `base64 -d`, and POSIX shell syntax do not exist on
Windows; these behave the same on every platform the server runs on. Stateless:
each function is a thin wrapper over the Go standard library.

## Knows
- nothing (stateless)

## Does
- registerPortableOS(L, uiMod): add the functions below to the `ui` table
- ui.mkdir(path): `os.MkdirAll`; `true`, or `nil, err` (R146)
- ui.isDir(path): `os.Stat` (follows links and junctions) then `IsDir`; any error is `false` (R147)
- ui.listDir(path): `os.ReadDir` (already sorted by name, hidden entries included), each entry's `isDir` from `os.Stat` on its full path so links are followed; `nil, err` when the directory cannot be read (R148)
- ui.base64Decode(text): `base64.StdEncoding`; `nil, err` on invalid input (R149)
- ui.exec(argv): `exec.Command(argv[1], argv[2:]...)`, no shell; stdout on success; on failure `nil, err` with the exit status and trimmed stderr (R150)
- ui.platform(): `{os = runtime.GOOS, arch = runtime.GOARCH}` (R151)

## Collaborators
- LuaSession: `registerUIModule` calls `registerPortableOS` while building the `ui` table
