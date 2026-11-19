# Test Design: WebSocketEndpoint
**Source:** crc-WebSocketEndpoint.md

The connection-to-session relationship used to be a map on the endpoint
(`sessionBindings`), duplicated in the other direction by each Session's own connection
set, and `HasConnectionsForSession` and `Broadcast` scanned every connection on the server
to answer from the endpoint's copy. These tests drive the binding through `register` and
`onDisconnect` with no real socket; a nil connection is enough, and any write to it would
fail the test.

## Test: connect and disconnect link and unlink both directions
**Purpose:** a connection to an existing session holds it and the session records it; disconnecting undoes both (R244, R245)
**Input:** one session; register a connection on it; then onDisconnect
**Expected:** after register: GetSessionIDForConnection is the session's ID, the session has one connection, HasConnectionsForSession is true. After onDisconnect: "", zero, false
**Refs:** crc-WebSocketEndpoint.md, crc-Session.md
**Code:** internal/server/connection_test.go
**Alarm:** 1
**Fire alarm:** drop the session's RemoveConnection call from onDisconnect; the session still counts one connection
**Inject:** internal/server/websocket.go:WebSocketEndpoint.onDisconnect
@alarm-hash: 967ba5c14cc944b3
**Pulled:** 2026-11-19 — rang: the session's RemoveConnection dropped from onDisconnect; "session counts 1 connections after disconnect, want 0 (R244)"

## Test: a connection to a missing session holds nothing
**Purpose:** a connection whose session does not exist is linked to nothing and its messages reach nothing (R244, R246)
**Input:** register a connection on a session ID that was never created; call postMessage with a message
**Expected:** GetSessionIDForConnection is ""; HasConnectionsForSession for that ID is false; postMessage reports false
**Refs:** crc-WebSocketEndpoint.md
**Code:** internal/server/connection_test.go

## Test: a destroyed session has no connections
**Purpose:** connection state comes from the session, so a destroyed one reports none and a broadcast to it sends nothing, even while the connection is still open (R247)
**Input:** one session; register a connection with a nil socket; destroy the session without disconnecting; ask HasConnectionsForSession; Broadcast a message to the session
**Expected:** HasConnectionsForSession is false; Broadcast returns without writing to the nil socket
**Refs:** crc-WebSocketEndpoint.md, crc-Session.md
**Code:** internal/server/connection_test.go
**Alarm:** 2
**Fire alarm:** make HasConnectionsForSession answer from the connections instead, true when any connection's session has that ID; it reports true after the destroy
**Inject:** internal/server/websocket.go:WebSocketEndpoint.HasConnectionsForSession
@alarm-hash: 5c782e99eb9ed106
**Pulled:** 2026-11-19 — rang: HasConnectionsForSession answered from the connections instead of the session; "HasConnectionsForSession true for a destroyed session (R247)"

## Not tested, deliberately
**No reconnect token (R248).** The requirement is an absence: nothing issues or checks a
token. Its implementation is the deletion of `GenerateReconnectToken`,
`ValidateReconnectToken` and their map, so there is nothing left to call.

**Message routing to the session's executor (R246, the positive case).** Posting runs
`processMessage`, which needs a protocol handler; with none, its own panic recovery fails.
The routing is one line in postMessage. No automated test covers it: no test in this
repository opens a WebSocket or drives readPump (checked 2026-11-19). It is exercised by
any live browser session.
