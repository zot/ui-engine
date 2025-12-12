# Test Design: Backend Library

**Source Specs**: libraries.md
**CRC Cards**: crc-BackendConnection.md, crc-PathNavigator.md, crc-ChangeDetector.md, crc-ObjectRegistry.md
**Sequences**: seq-backend-connect.md, seq-backend-refresh.md, seq-path-resolve.md, seq-object-registry.md

## Overview

Tests for Go/Lua backend library supporting connection, path navigation, and change detection.

## Test Cases

### Test: Backend connection establishment

**Purpose**: Verify connection to UI server

**Input**:
- connect(url, rootValue)

**References**:
- CRC: crc-BackendConnection.md - "Does: connect"
- Sequence: seq-backend-connect.md

**Expected Results**:
- WebSocket opened
- Session bound
- Root value sent to variable 1

---

### Test: Backend disconnect with cleanup hook

**Purpose**: Verify disconnect callback

**Input**:
- onClose(callback) registered
- Connection closed

**References**:
- CRC: crc-BackendConnection.md - "Does: disconnect, onClose"

**Expected Results**:
- Callback invoked
- Resources cleaned up
- Reconnection possible

---

### Test: Backend send message

**Purpose**: Verify message sending

**Input**:
- send(update message)

**References**:
- CRC: crc-BackendConnection.md - "Does: send"

**Expected Results**:
- Message sent via WebSocket
- Proper serialization
- Delivery confirmed

---

### Test: Backend receive message

**Purpose**: Verify message receipt handling

**Input**:
- Server sends watch message

**References**:
- CRC: crc-BackendConnection.md - "Does: receive"

**Expected Results**:
- Message parsed
- Handler invoked
- Response generated

---

### Test: Path resolve simple property

**Purpose**: Verify property navigation

**Input**:
- Object: {name: "John"}
- resolve("name")

**References**:
- CRC: crc-PathNavigator.md - "Does: resolve"
- Sequence: seq-path-resolve.md

**Expected Results**:
- Returns "John"
- Correct type
- No side effects

---

### Test: Path resolve nested property

**Purpose**: Verify dot notation navigation

**Input**:
- Object: {father: {name: "Bob"}}
- resolve("father.name")

**References**:
- CRC: crc-PathNavigator.md - "Does: navigateSegment"

**Expected Results**:
- Returns "Bob"
- Intermediate object traversed
- Null-safe handling

---

### Test: Path resolve array index

**Purpose**: Verify 1-based array indexing

**Input**:
- Object: {items: ["a", "b", "c"]}
- resolve("items.2")

**References**:
- CRC: crc-PathNavigator.md - "Does: handleArrayIndex"

**Expected Results**:
- Returns "b" (index 2 = second element)
- 1-based indexing
- Bounds checking

---

### Test: Path resolve method call

**Purpose**: Verify method invocation in path

**Input**:
- Object with getName() method returning "Alice"
- resolve("getName()")

**References**:
- CRC: crc-PathNavigator.md - "Does: handleMethodCall"

**Expected Results**:
- Method called
- Returns "Alice"
- No arguments passed

---

### Test: Path resolve parent traversal

**Purpose**: Verify ".." segment handling

**Input**:
- Child object in parent
- resolve("..") from child

**References**:
- CRC: crc-PathNavigator.md - "Does: handleParentTraversal"

**Expected Results**:
- Returns parent object
- Maintains tree structure
- Works multiple levels

---

### Test: Path resolve standard variable

**Purpose**: Verify @name prefix handling

**Input**:
- @app registered with app object
- resolve("@app.url")

**References**:
- CRC: crc-PathNavigator.md - "Does: resolveStandardVariable"

**Expected Results**:
- @app resolved to registered variable
- .url navigated from there
- Returns current URL

---

### Test: Path resolve for write

**Purpose**: Verify parent + key extraction

**Input**:
- resolveForWrite("father.name")

**References**:
- CRC: crc-PathNavigator.md - "Does: resolveForWrite"

**Expected Results**:
- Returns {parent: father object, key: "name"}
- Enables setting value
- Works for all path types

---

### Test: Change detector add watch

**Purpose**: Verify watch registration

**Input**:
- addWatch(varId)

**References**:
- CRC: crc-ChangeDetector.md - "Does: addWatch"

**Expected Results**:
- Variable tracked
- Initial value stored
- Included in refresh

---

### Test: Change detector remove watch

**Purpose**: Verify watch unregistration

**Input**:
- removeWatch(varId)

**References**:
- CRC: crc-ChangeDetector.md - "Does: removeWatch"

**Expected Results**:
- Variable no longer tracked
- Previous value cleared
- Excluded from refresh

---

### Test: Change detector refresh

**Purpose**: Verify refresh cycle

**Input**:
- Multiple watched variables
- Some values changed
- refresh()

**References**:
- CRC: crc-ChangeDetector.md - "Does: refresh"
- Sequence: seq-backend-refresh.md

**Expected Results**:
- All watched variables computed
- Changed values detected
- Updates sent for changes only

---

### Test: Change detector auto-refresh after message

**Purpose**: Verify automatic refresh trigger

**Input**:
- Client message received

**References**:
- CRC: crc-ChangeDetector.md - "Does: afterMessage"

**Expected Results**:
- Refresh triggered automatically
- Changes detected
- Updates sent

---

### Test: Change detector throttling

**Purpose**: Verify background refresh throttling

**Input**:
- Multiple scheduleRefresh() calls in quick succession

**References**:
- CRC: crc-ChangeDetector.md - "Does: scheduleRefresh"

**Expected Results**:
- Only one refresh executed
- Throttle interval respected
- No flooding

---

### Test: Change detection with reflection

**Purpose**: Verify any object can be watched

**Input**:
- Plain object (no observer pattern)
- Watch and modify

**References**:
- CRC: crc-ChangeDetector.md

**Expected Results**:
- Changes detected via value comparison
- No special interface required
- Works with any backend object

---

### Test: Object registry register and lookup

**Purpose**: Verify object registration with weak reference

**Input**:
- register(object, varId)
- lookup(object)

**References**:
- CRC: crc-ObjectRegistry.md - "Does: register, lookup"
- Sequence: seq-object-registry.md

**Expected Results**:
- Object mapped to varId
- Weak reference created
- lookup returns varId

---

### Test: Object registry unregister

**Purpose**: Verify object removal from registry

**Input**:
- register(object, varId)
- unregister(varId)
- lookup(object)

**References**:
- CRC: crc-ObjectRegistry.md - "Does: unregister"

**Expected Results**:
- Entry removed
- lookup returns nil
- No memory leak

---

### Test: Object registry serialization with refs

**Purpose**: Verify object reference serialization

**Input**:
- Contact registered as varId 5
- App with contacts array containing Contact
- serializeWithRefs(app)

**References**:
- CRC: crc-ObjectRegistry.md - "Does: serializeWithRefs"
- Sequence: seq-object-registry.md

**Expected Results**:
- Contact serializes as {"obj": 5}
- Non-registered objects inline
- Nested objects handled

---

### Test: Object identity across multiple references

**Purpose**: Verify same object serializes identically

**Input**:
- Contact in contacts[0] and selected
- Both point to same object
- serializeWithRefs(app)

**References**:
- CRC: crc-ObjectRegistry.md - "Does: lookup, serializeWithRefs"
- Sequence: seq-object-registry.md

**Expected Results**:
- Both locations emit {"obj": id}
- Same id value
- Identity preserved

---

### Test: Object registry weak reference cleanup

**Purpose**: Verify GC'd objects removed from registry

**Input**:
- register(object, varId)
- Drop all references to object
- Force GC
- cleanup()

**References**:
- CRC: crc-ObjectRegistry.md - "Does: cleanup"
- Sequence: seq-object-registry.md

**Expected Results**:
- Weak pointer returns nil
- Entry removed from registry
- No memory leak

---

### Test: Object registry thread safety

**Purpose**: Verify concurrent access safety

**Input**:
- Concurrent register/lookup/unregister calls

**References**:
- CRC: crc-ObjectRegistry.md - "Knows: mu"

**Expected Results**:
- No race conditions
- Consistent state
- Operations atomic

---

### Test: Change detector with object registry integration

**Purpose**: Verify addWatch registers objects

**Input**:
- addWatch(varId, path)
- Object at path

**References**:
- CRC: crc-ChangeDetector.md - "Does: addWatch"
- CRC: crc-ObjectRegistry.md - "Does: register"
- Sequence: seq-object-registry.md

**Expected Results**:
- Object registered in ObjectRegistry
- Serialization uses object refs
- Identity-based comparison

---

## Coverage Summary

**Responsibilities Covered:**
- BackendConnection: connect, disconnect, send, receive, setRootValue, onClose, reconnect
- PathNavigator: resolve, resolveForWrite, parsePath, navigateSegment, handleMethodCall, handleArrayIndex, handleParentTraversal, resolveStandardVariable
- ChangeDetector: addWatch, removeWatch, refresh, detectChange, scheduleRefresh, sendUpdates, afterMessage
- ObjectRegistry: register, unregister, lookup, serializeWithRefs, cleanup, startCleanup, stopCleanup

**Scenarios Covered:**
- seq-backend-connect.md: All paths
- seq-backend-refresh.md: All paths
- seq-path-resolve.md: All paths
- seq-object-registry.md: All paths

**Gaps**: None identified
