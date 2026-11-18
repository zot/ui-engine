package server

import (
	"weak"

	"github.com/zot/ui-engine/internal/lua"
)

// SessionRef is a handle to one session. It is cheap to copy, safe to hold
// after the session ends, and never keeps the session alive. In this step it
// resolves through the server's existing maps; later steps move session state
// behind it without changing callers.
// CRC: crc-SessionRef.md | R154, R156, R173
type SessionRef struct {
	srv      *Server
	vendedID string
	// lua is the identity check as well as the route to the Lua session:
	// vended IDs are reused (the counter resets when the last session goes),
	// so the ID alone could resolve to a newer session.
	lua weak.Pointer[lua.LuaSession]
}

// Station holds every view of one session.
// CRC: crc-SessionRef.md | R157
type Station struct {
	Lua     LuaView
	Browser BrowserView
	Exec    ExecView
}

// LuaView reaches the session's Lua state.
// CRC: crc-SessionRef.md | R157
type LuaView struct{ ref SessionRef }

// BrowserView reaches the browser side: connection state and change pushes.
// CRC: crc-SessionRef.md | R157
type BrowserView struct{ ref SessionRef }

// There is no current-session accessor: ui-engine serves many sessions, and a
// host that has a current one supplies the reference.
// CRC: crc-SessionRef.md | R167

// ExecView posts work to the session's executor.
// CRC: crc-SessionRef.md | R157
type ExecView struct{ ref SessionRef }

// SessionRef returns a reference to the session with this vended ID, or false
// when no Lua session is registered under it.
// CRC: crc-SessionRef.md | Seq: seq-session-ref.md#1.1 | R153
func (s *Server) SessionRef(vendedID string) (SessionRef, bool) {
	s.luaSessionsMu.RLock()
	ls := s.luaSessions[vendedID]
	s.luaSessionsMu.RUnlock()
	if ls == nil {
		return SessionRef{}, false
	}
	return SessionRef{srv: s, vendedID: vendedID, lua: weak.Make(ls)}, true
}

// resolve returns the live Lua session and the internal session ID, or false
// when the session has ended, been replaced under the same vended ID, or begun
// shutting down.
// CRC: crc-SessionRef.md | Seq: seq-session-ref.md#1.2.1 | R155, R156
func (r SessionRef) resolve() (*lua.LuaSession, string, bool) {
	if r.srv == nil {
		return nil, "", false
	}
	ls := r.lua.Value()
	if ls == nil {
		return nil, "", false
	}
	r.srv.luaSessionsMu.RLock()
	current := r.srv.luaSessions[r.vendedID]
	r.srv.luaSessionsMu.RUnlock()
	if current != ls || ls.IsShutdown() {
		return nil, "", false
	}
	internalID := r.srv.sessions.GetInternalID(r.vendedID)
	if internalID == "" {
		return nil, "", false
	}
	return ls, internalID, true
}

// ID returns the vended session ID.
// CRC: crc-SessionRef.md | R154
func (r SessionRef) ID() string { return r.vendedID }

// Valid reports whether the session is still running.
// CRC: crc-SessionRef.md | R155
func (r SessionRef) Valid() bool {
	_, _, ok := r.resolve()
	return ok
}

// CRC: crc-SessionRef.md | R157, R165
func (r SessionRef) Lua() LuaView         { return LuaView{r} }
func (r SessionRef) Browser() BrowserView { return BrowserView{r} }
func (r SessionRef) Exec() ExecView       { return ExecView{r} }
func (r SessionRef) Station() Station     { return Station{LuaView{r}, BrowserView{r}, ExecView{r}} }

// CRC: crc-SessionRef.md | R158
func (v LuaView) Ref() SessionRef      { return v.ref }
func (v LuaView) Station() Station     { return v.ref.Station() }
func (v BrowserView) Ref() SessionRef  { return v.ref }
func (v BrowserView) Station() Station { return v.ref.Station() }
func (v ExecView) Ref() SessionRef     { return v.ref }
func (v ExecView) Station() Station    { return v.ref.Station() }

// Post queues fn on the session's executor with the session's Lua context set,
// without waiting and without change detection afterward. It returns false,
// queueing nothing, when the reference is invalid.
// CRC: crc-SessionRef.md | Seq: seq-session-ref.md#1.2 | R159, R161
func (v ExecView) Post(fn func()) bool {
	ls, internalID, ok := v.ref.resolve()
	if !ok {
		return false
	}
	vendedID := v.ref.vendedID
	v.ref.srv.wsEndpoint.ExecuteInSessionPassive(internalID, func() {
		ls.ExecuteInSession(vendedID, func() (interface{}, error) {
			fn()
			return nil, nil
		})
	})
	return true
}

// PostUpdate is Post followed by change detection, which pushes changes when a
// browser is connected.
// CRC: crc-SessionRef.md | Seq: seq-session-ref.md#1.2.3 | R160, R161
func (v ExecView) PostUpdate(fn func()) bool {
	ls, internalID, ok := v.ref.resolve()
	if !ok {
		return false
	}
	vendedID := v.ref.vendedID
	v.ref.srv.wsEndpoint.ExecuteInSessionAsync(internalID, func() (interface{}, error) {
		return ls.ExecuteInSession(vendedID, func() (interface{}, error) {
			fn()
			return nil, nil
		})
	})
	return true
}

// Connected reports whether a browser connection is attached to the session.
// CRC: crc-SessionRef.md | R162
func (v BrowserView) Connected() bool {
	_, internalID, ok := v.ref.resolve()
	return ok && v.ref.srv.wsEndpoint.HasConnectionsForSession(internalID)
}

// Update runs change detection and pushes changes now. Call it only on the
// session's executor.
// CRC: crc-SessionRef.md | R163
func (v BrowserView) Update() {
	if _, internalID, ok := v.ref.resolve(); ok {
		v.ref.srv.wsEndpoint.RunAfterBatch(internalID)
	}
}

// Session returns the session's LuaSession, or nil when the reference is
// invalid. Use it only on the session's executor.
// CRC: crc-SessionRef.md | R164
func (v LuaView) Session() *lua.LuaSession {
	ls, _, ok := v.ref.resolve()
	if !ok {
		return nil
	}
	return ls
}
