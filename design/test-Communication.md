# Test Design: Communication System

**Source Specs**: interfaces.md, deployment.md, libraries.md
**CRC Cards**: crc-WebSocketEndpoint.md, crc-HTTPEndpoint.md, crc-SharedWorker.md, crc-MessageRelay.md
**Sequences**: seq-frontend-connect.md, seq-backend-connect.md, seq-relay-message.md, seq-bootstrap.md

## Overview

Tests for WebSocket/HTTP transport, SharedWorker coordination, and message relay.

## Test Cases

### Test: WebSocket connection establishment

**Purpose**: Verify WebSocket handshake and session binding

**Input**:
- Frontend opens WebSocket to /ws/SESSION-ID

**References**:
- CRC: crc-WebSocketEndpoint.md - "Does: accept"
- Sequence: seq-frontend-connect.md

**Expected Results**:
- Connection accepted
- Connection bound to session
- Acknowledgment sent to client

---

### Test: WebSocket message send

**Purpose**: Verify message delivery to specific connection

**Input**:
- Connection established
- send(connectionId, message)

**References**:
- CRC: crc-WebSocketEndpoint.md - "Does: send"

**Expected Results**:
- Message delivered to connection
- Message properly serialized
- Delivery confirmed

---

### Test: WebSocket broadcast to session

**Purpose**: Verify message to all session connections

**Input**:
- Multiple connections in session
- broadcast(sessionId, message)

**References**:
- CRC: crc-WebSocketEndpoint.md - "Does: broadcast"

**Expected Results**:
- All session connections receive message
- Message identical for all
- Non-session connections unaffected

---

### Test: WebSocket connection close cleanup

**Purpose**: Verify cleanup on disconnect

**Input**:
- Connection established
- WebSocket closed by client

**References**:
- CRC: crc-WebSocketEndpoint.md - "Does: close"
- CRC: crc-Session.md - "Does: removeConnection"

**Expected Results**:
- Connection removed from session
- Resources cleaned up
- Session notified

---

### Test: HTTP redirect to session

**Purpose**: Verify GET / creates session and redirects

**Input**:
- HTTP GET /

**References**:
- CRC: crc-HTTPEndpoint.md - "Does: handleSessionRedirect"
- Sequence: seq-create-session.md

**Expected Results**:
- Session created
- HTTP 302 redirect to /SESSION-ID
- Session URL valid

---

### Test: HTTP serve static files from embedded site

**Purpose**: Verify embedded site serving

**Input**:
- HTTP GET /index.html (embedded mode)

**References**:
- CRC: crc-HTTPEndpoint.md - "Does: serveStatic"

**Expected Results**:
- File served from embedded archive
- Correct content type
- Compressed delivery if supported

---

### Test: HTTP serve static files from custom directory

**Purpose**: Verify --dir flag serving

**Input**:
- Server started with --dir /custom
- HTTP GET /app.js

**References**:
- CRC: crc-HTTPEndpoint.md - "Does: setCustomDir"

**Expected Results**:
- File served from /custom/app.js
- Falls back to embedded if not found
- Directory traversal prevented

---

### Test: SharedWorker first tab becomes main

**Purpose**: Verify main tab designation

**Input**:
- First tab connects to SharedWorker

**References**:
- CRC: crc-SharedWorker.md - "Does: setMainTab"
- Sequence: seq-frontend-connect.md

**Expected Results**:
- Tab designated as main
- WebSocket opened by worker
- isMainTab returns true

---

### Test: SharedWorker second tab relays through first

**Purpose**: Verify non-main tab coordination

**Input**:
- Main tab connected
- Second tab connects

**References**:
- CRC: crc-SharedWorker.md - "Does: connect"

**Expected Results**:
- Second tab NOT main
- Messages relay through SharedWorker
- No duplicate WebSocket

---

### Test: SharedWorker relay to backend

**Purpose**: Verify tab-to-backend message flow

**Input**:
- Tab sends message via SharedWorker

**References**:
- CRC: crc-SharedWorker.md - "Does: relayToBackend"
- Sequence: seq-relay-message.md

**Expected Results**:
- Message sent via main tab's WebSocket
- Message arrives at server
- Response relayed back

---

### Test: SharedWorker relay to all tabs

**Purpose**: Verify backend-to-tabs message flow

**Input**:
- Server sends message via WebSocket
- Multiple tabs connected

**References**:
- CRC: crc-SharedWorker.md - "Does: relayToTabs"

**Expected Results**:
- All tabs receive message
- Message content identical
- Delivered via postMessage

---

### Test: SharedWorker desktop notification

**Purpose**: Verify notification for tab activation

**Input**:
- Duplicate tab opens session URL

**References**:
- CRC: crc-SharedWorker.md - "Does: sendNotification"
- Sequence: seq-activate-tab.md

**Expected Results**:
- Desktop notification shown
- "Click to focus" message
- Notification clickable

---

### Test: MessageRelay forward to frontend

**Purpose**: Verify backend-to-frontend relay

**Input**:
- Backend sends update message

**References**:
- CRC: crc-MessageRelay.md - "Does: relayToFrontend"
- Sequence: seq-relay-message.md

**Expected Results**:
- Message forwarded to frontend
- All watchers receive update
- Message unmodified

---

### Test: MessageRelay forward to backend

**Purpose**: Verify frontend-to-backend relay

**Input**:
- Frontend sends update message

**References**:
- CRC: crc-MessageRelay.md - "Does: relayToBackend"

**Expected Results**:
- Message forwarded to backend
- Backend receives message
- Response path established

---

### Test: MessageRelay handles unbound locally

**Purpose**: Verify unbound variable handling

**Input**:
- Update for unbound variable

**References**:
- CRC: crc-MessageRelay.md - "Does: filterForUnbound"
- Sequence: seq-relay-message.md

**Expected Results**:
- Message NOT forwarded to backend
- UI server stores change
- Frontend watchers notified

---

### Test: MessageRelay batches messages

**Purpose**: Verify message batching

**Input**:
- Multiple updates in quick succession

**References**:
- CRC: crc-MessageRelay.md - "Does: batchMessages"

**Expected Results**:
- Updates combined into single batch
- Reduced network overhead
- All changes delivered

---

## Coverage Summary

**Responsibilities Covered:**
- WebSocketEndpoint: accept, close, send, broadcast, receive, bindToSession, isConnected, getSessionId
- HTTPEndpoint: handleRequest, serveStatic, handleSessionRedirect, handleRESTApi, extractBundledFile, setCustomDir
- SharedWorker: connect, disconnect, setMainTab, isMainTab, relayToBackend, relayToTabs, activateMainTab, sendNotification, handleDuplicateTab
- MessageRelay: relayToFrontend, relayToBackend, shouldRelay, filterForUnbound, batchMessages, flushBatch

**Scenarios Covered:**
- seq-frontend-connect.md: All paths
- seq-backend-connect.md: All paths
- seq-relay-message.md: All paths
- seq-bootstrap.md: Connection paths

**Gaps**: None identified
