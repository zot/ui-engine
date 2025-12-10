# Test Design: Session System

**Source Specs**: interfaces.md, main.md
**CRC Cards**: crc-Session.md, crc-SessionManager.md, crc-Router.md
**Sequences**: seq-create-session.md, seq-frontend-connect.md, seq-activate-tab.md, seq-navigate-url.md

## Overview

Tests for session lifecycle, URL routing, and tab coordination.

## Test Cases

### Test: Create new session on root URL

**Purpose**: Verify GET / creates session and redirects

**Input**:
- HTTP GET /

**References**:
- CRC: crc-SessionManager.md - "Does: createSession"
- Sequence: seq-create-session.md

**Expected Results**:
- New session ID generated
- Variable 1 created as root
- HTTP 302 redirect to /SESSION-ID
- Session stored in manager

---

### Test: Session ID uniqueness

**Purpose**: Verify unique session IDs generated

**Input**:
- Create 100 sessions

**References**:
- CRC: crc-SessionManager.md - "Does: generateSessionId"

**Expected Results**:
- All session IDs unique
- No collisions
- IDs are URL-safe

---

### Test: Access existing session URL

**Purpose**: Verify session access with valid ID

**Input**:
- Session created with ID "abc123"
- HTTP GET /abc123

**References**:
- CRC: crc-SessionManager.md - "Does: getSession"

**Expected Results**:
- Session found and validated
- Frontend HTML served
- Session marked as accessed

---

### Test: Access invalid session URL

**Purpose**: Verify error for non-existent session

**Input**:
- HTTP GET /nonexistent123

**References**:
- CRC: crc-SessionManager.md - "Does: sessionExists"
- Sequence: seq-activate-tab.md

**Expected Results**:
- Error page displayed
- No desktop notification sent
- Tab remains open

---

### Test: Register URL path for presenter

**Purpose**: Verify path registration

**Input**:
- Session created
- registerUrlPath("/users", presenterVarId)
- Navigate to /SESSION-ID/users

**References**:
- CRC: crc-Router.md - "Does: register"
- Sequence: seq-navigate-url.md

**Expected Results**:
- Path associated with presenter
- Navigation resolves to presenter
- Unregistered paths return null

---

### Test: URL path resolution

**Purpose**: Verify registered path lookup

**Input**:
- Path "/users" registered
- resolve("/users")

**References**:
- CRC: crc-Router.md - "Does: resolve"

**Expected Results**:
- Returns registered presenter variable
- Unregistered path returns null
- Session context maintained

---

### Test: Session connection tracking

**Purpose**: Verify connection add/remove

**Input**:
- Session created
- Add frontend connection
- Add backend connection
- Remove frontend connection

**References**:
- CRC: crc-Session.md - "Does: addConnection, removeConnection"

**Expected Results**:
- Connection count accurate
- isActive returns true with connections
- isActive returns false when empty

---

### Test: Session cleanup on inactivity

**Purpose**: Verify inactive session cleanup

**Input**:
- Session created
- All connections removed
- Wait for cleanup interval

**References**:
- CRC: crc-SessionManager.md - "Does: cleanupInactiveSessions"

**Expected Results**:
- Inactive session destroyed
- Variables cleaned up
- Session ID no longer valid

---

### Test: Tab activation with existing main tab

**Purpose**: Verify duplicate tab handling

**Input**:
- Main tab connected to session
- New tab opens same session URL

**References**:
- CRC: crc-SharedWorker.md - "Does: handleDuplicateTab"
- Sequence: seq-activate-tab.md

**Expected Results**:
- Desktop notification shown
- Main tab receives focus request
- New tab closes or goes back

---

### Test: Tab activation with path

**Purpose**: Verify navigation before activation

**Input**:
- Main tab at /session/page1
- New tab opens /session/page2

**References**:
- CRC: crc-SharedWorker.md - "Does: activateMainTab"
- Sequence: seq-activate-tab.md

**Expected Results**:
- Main tab navigates to /page2
- Then receives focus
- New tab closes

---

### Test: Build full URL from presenter

**Purpose**: Verify URL construction

**Input**:
- Session ID "abc123"
- Path "/users/5"
- buildUrl()

**References**:
- CRC: crc-Router.md - "Does: buildUrl"

**Expected Results**:
- Returns "http://site/abc123/users/5"
- Handles missing path
- Escapes special characters

---

### Test: Parse URL to extract session and path

**Purpose**: Verify URL parsing

**Input**:
- URL "http://site/abc123/users/5"
- parseUrl()

**References**:
- CRC: crc-Router.md - "Does: parseUrl"

**Expected Results**:
- sessionId: "abc123"
- path: "/users/5"
- Handles edge cases (no path, trailing slash)

---

## Coverage Summary

**Responsibilities Covered:**
- Session: getId, getAppVariable, addConnection, removeConnection, isActive, getConnectionCount, touch
- SessionManager: createSession, getSession, destroySession, sessionExists, registerUrlPath, resolveUrlPath, generateSessionId, cleanupInactiveSessions
- Router: register, unregister, resolve, match, buildUrl, parseUrl, isRegisteredPath

**Scenarios Covered:**
- seq-create-session.md: All paths
- seq-frontend-connect.md: Session validation path
- seq-activate-tab.md: All paths
- seq-navigate-url.md: All paths

**Gaps**: None identified
