# Test Design: ArrayGetter
**Source:** crc-LuaSession.md

## Test: an empty table reads as an empty array, and only there
**Purpose:** R152 — a ViewList bound to `{}` must sync to no items without an error, while a table with keys, or a typed object, is still refused, and `isArray` stays strict so serialization keeps an empty object an object.
**Input:** an empty table; a table with only a `_` key; a one-element array; a table with a string key; an empty table whose metatable names a type; then `isArray` on the empty table.
**Expected:** empty arrays (count 0, no error) for the first two, count 1 for the array, errors for the keyed table and the typed object, and `isArray` false for the empty table.
**Refs:** crc-LuaSession.md
**Code:** internal/lua/arrayget_test.go
**Alarm:** 1
**Fire alarm:** delete the `isEmptyTable` branch in `ArrayGetter`, the state that logged 11,484 errors in ark in a day; red is the empty table reporting "Attempt to index Lua table that is not an array".
**Inject:** internal/lua/runtime.go:LuaSession.ArrayGetter
@alarm-hash: fcac49419e1b42cb
**Pulled:** 2026-11-12 — rang: the empty table and the `_`-only table both reported "Attempt to index Lua table that is not an array".
