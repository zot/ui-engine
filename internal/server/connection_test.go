// Tests for the connection-to-session relationship (test-WebSocketEndpoint.md, Item 9b of
// the coherency carve): each connection holds its Session, each Session lists its
// connections, and nothing else records either direction. A nil socket is enough here;
// any write to it would fail the test.
// CRC: crc-WebSocketEndpoint.md, crc-Session.md | Test: test-WebSocketEndpoint.md | R244, R245, R246, R247
package server

import (
	"testing"
	"time"

	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/protocol"
)

// Test: connect and disconnect link and unlink both directions (R244, R245)
func TestConnectionLinksBothDirections(t *testing.T) {
	m := NewSessionManager(time.Hour)
	ws := NewWebSocketEndpoint(config.DefaultConfig(), m, nil)
	sess, _, err := m.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	ws.register("c1", nil, sess.ID)
	if got := ws.GetSessionIDForConnection("c1"); got != sess.ID {
		t.Fatalf("GetSessionIDForConnection = %q, want %q", got, sess.ID)
	}
	if n := sess.GetConnectionCount(); n != 1 {
		t.Fatalf("session counts %d connections after register, want 1", n)
	}
	if !ws.HasConnectionsForSession(sess.ID) {
		t.Fatal("HasConnectionsForSession false after register")
	}

	ws.onDisconnect("c1")
	if got := ws.GetSessionIDForConnection("c1"); got != "" {
		t.Errorf("GetSessionIDForConnection after disconnect = %q, want empty", got)
	}
	if n := sess.GetConnectionCount(); n != 0 {
		t.Errorf("session counts %d connections after disconnect, want 0 (R244)", n)
	}
	if ws.HasConnectionsForSession(sess.ID) {
		t.Error("HasConnectionsForSession true after disconnect")
	}
}

// Test: a connection to a missing session holds nothing (R244, R246)
func TestConnectionToMissingSessionHoldsNothing(t *testing.T) {
	m := NewSessionManager(time.Hour)
	ws := NewWebSocketEndpoint(config.DefaultConfig(), m, nil)
	wc := ws.register("c1", nil, "no-such-session")
	if got := ws.GetSessionIDForConnection("c1"); got != "" {
		t.Errorf("GetSessionIDForConnection = %q, want empty", got)
	}
	if ws.HasConnectionsForSession("no-such-session") {
		t.Error("HasConnectionsForSession true for a session that does not exist")
	}
	if ws.postMessage("c1", wc, []byte(`{}`)) {
		t.Error("postMessage posted a message for a connection with no session (R246)")
	}
}

// Test: a destroyed session has no connections (R247)
func TestDestroyedSessionHasNoConnections(t *testing.T) {
	m := NewSessionManager(time.Hour)
	ws := NewWebSocketEndpoint(config.DefaultConfig(), m, nil)
	sess, _, err := m.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	ws.register("c1", nil, sess.ID) // still open: never disconnected
	m.DestroySession(sess.ID)
	if ws.HasConnectionsForSession(sess.ID) {
		t.Fatal("HasConnectionsForSession true for a destroyed session (R247)")
	}
	// A write to the nil socket would panic; Broadcast must find no connection to write to.
	if err := ws.Broadcast(sess.ID, &protocol.Message{Type: "update"}); err != nil {
		t.Fatalf("Broadcast to a destroyed session: %v", err)
	}
}
