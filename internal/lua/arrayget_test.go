package lua

// CRC: crc-LuaSession.md | Test: test-ArrayGetter.md | R152

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// TestArrayGetterEmptyTable — test-ArrayGetter.md "an empty table reads as an
// empty array, and only there". R152.
func TestArrayGetterEmptyTable(t *testing.T) {
	s := &LuaSession{State: lua.NewState()}
	defer s.State.Close()

	empty := s.State.NewTable()
	if _, n, err := s.ArrayGetter(empty); err != nil || n != 0 {
		t.Errorf("empty table: count=%d err=%v, want an empty array", n, err)
	}
	internal := s.State.NewTable()
	internal.RawSetString("_cache", lua.LTrue)
	if _, n, err := s.ArrayGetter(internal); err != nil || n != 0 {
		t.Errorf("table with only _ keys: count=%d err=%v, want an empty array", n, err)
	}

	one := s.State.NewTable()
	one.Append(lua.LString("x"))
	if _, n, err := s.ArrayGetter(one); err != nil || n != 1 {
		t.Errorf("one-element array: count=%d err=%v", n, err)
	}

	object := s.State.NewTable()
	object.RawSetString("name", lua.LString("x"))
	if _, _, err := s.ArrayGetter(object); err == nil {
		t.Error("a table with string keys read as an array, want an error")
	}
	typed := s.State.NewTable()
	mt := s.State.NewTable()
	mt.RawSetString("type", lua.LString("Thing"))
	s.State.SetMetatable(typed, mt)
	if _, _, err := s.ArrayGetter(typed); err == nil {
		t.Error("an empty typed object read as an array, want an error")
	}

	// isArray stays strict: serialization must keep sending {} as an object.
	if s.isArray(empty) {
		t.Error("isArray accepted an empty table; an empty object would serialize as []")
	}
}
