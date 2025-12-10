# Test Design: Variable Protocol

**Source Specs**: protocol.md
**CRC Cards**: crc-Variable.md, crc-VariableStore.md, crc-ProtocolHandler.md, crc-WatchManager.md
**Sequences**: seq-create-variable.md, seq-update-variable.md, seq-watch-variable.md, seq-destroy-variable.md

## Overview

Tests for the core variable protocol system including variable creation, updates, watching, and destruction.

## Test Cases

### Test: Create variable with value

**Purpose**: Verify basic variable creation with a simple value

**Input**:
- create(parentId: 0, value: "hello", properties: {})

**References**:
- CRC: crc-VariableStore.md - "Does: create"
- Sequence: seq-create-variable.md

**Expected Results**:
- New variable ID allocated (>= 1)
- Variable stored with value "hello"
- Variable accessible via get(varId)

---

### Test: Create variable with object reference value

**Purpose**: Verify creation with {obj: ID} reference value

**Input**:
- create(parentId: 0, value: {obj: 5}, properties: {})

**References**:
- CRC: crc-Variable.md - "Does: isObjectReference"
- CRC: crc-ObjectReference.md

**Expected Results**:
- Variable value stored as {obj: 5}
- isObjectReference returns true
- get resolves to actual object data

---

### Test: Create variable with create property

**Purpose**: Verify object instantiation via create property

**Input**:
- create(parentId: 0, value: null, properties: {create: "MyPresenter"})

**References**:
- CRC: crc-ProtocolHandler.md - "Does: handleCreate"
- Sequence: seq-create-variable.md

**Expected Results**:
- Value ignored, object created from type
- type property auto-set to "MyPresenter"
- Object instance stored as value

---

### Test: Create variable with property priorities

**Purpose**: Verify :high/:med/:low property processing order

**Input**:
- create(parentId: 0, value: null, properties: {
    "viewdefs:high": {...},
    "data:med": {...},
    "optional:low": {...}
  })

**References**:
- CRC: crc-ProtocolHandler.md - "Does: parsePropertyPriority"

**Expected Results**:
- viewdefs processed first
- data processed second
- optional processed last

---

### Test: Create unbound variable

**Purpose**: Verify UI server stores unbound variables

**Input**:
- create(parentId: 0, value: "test", properties: {}, nowatch: false, unbound: true)

**References**:
- CRC: crc-Variable.md - "Does: isUnbound"
- Sequence: seq-relay-message.md

**Expected Results**:
- Variable stored in UI server storage
- Updates not relayed to backend
- get() returns from UI server

---

### Test: Update variable value

**Purpose**: Verify value update propagation

**Input**:
- Variable created with value "old"
- update(varId, value: "new")

**References**:
- CRC: crc-VariableStore.md - "Does: update"
- Sequence: seq-update-variable.md

**Expected Results**:
- Value changed to "new"
- Watchers notified of change
- Previous value no longer accessible

---

### Test: Update variable properties

**Purpose**: Verify property-only update

**Input**:
- Variable created with properties {a: "1"}
- update(varId, properties: {b: "2"})

**References**:
- CRC: crc-Variable.md - "Does: setProperty"

**Expected Results**:
- Property b added
- Property a unchanged
- Value unchanged

---

### Test: Watch variable returns current value

**Purpose**: Verify watch immediately sends current value

**Input**:
- Variable created with value "test"
- watch(varId)

**References**:
- CRC: crc-WatchManager.md - "Does: watch"
- Sequence: seq-watch-variable.md

**Expected Results**:
- Update message sent immediately
- Update contains current value "test"
- Watcher added to subscription list

---

### Test: Watch tally for bound variables

**Purpose**: Verify watch only forwarded on 0->1 transition

**Input**:
- Bound variable created
- watch(varId) from frontend A
- watch(varId) from frontend B

**References**:
- CRC: crc-WatchManager.md - "Does: shouldForwardWatch"

**Expected Results**:
- First watch forwarded to backend
- Second watch NOT forwarded (tally > 1)
- Both frontends receive updates

---

### Test: Unwatch tally for bound variables

**Purpose**: Verify unwatch only forwarded on 1->0 transition

**Input**:
- Bound variable with 2 watchers
- unwatch(varId) from frontend A
- unwatch(varId) from frontend B

**References**:
- CRC: crc-WatchManager.md - "Does: shouldForwardUnwatch"

**Expected Results**:
- First unwatch NOT forwarded (tally still > 0)
- Second unwatch forwarded to backend
- Backend can stop tracking

---

### Test: Inactive variable suppresses updates

**Purpose**: Verify inactive property stops notifications

**Input**:
- Variable with inactive property set
- update(varId, value: "changed")

**References**:
- CRC: crc-WatchManager.md - "Does: isInactive"

**Expected Results**:
- Value updated in storage
- NO update message sent to watchers
- Children also suppressed

---

### Test: Destroy variable

**Purpose**: Verify variable destruction

**Input**:
- Variable created
- destroy(varId)

**References**:
- CRC: crc-VariableStore.md - "Does: destroy"
- Sequence: seq-destroy-variable.md

**Expected Results**:
- Variable removed from storage
- Watchers cleaned up
- get(varId) returns null/error

---

### Test: Destroy variable with children

**Purpose**: Verify recursive child destruction

**Input**:
- Parent variable created
- Child variable created with parentId
- destroy(parentId)

**References**:
- CRC: crc-VariableStore.md - "Does: getChildren"
- Sequence: seq-destroy-variable.md

**Expected Results**:
- Child destroyed first
- Parent destroyed after
- Both removed from storage

---

### Test: Standard variable registration

**Purpose**: Verify @NAME registration and lookup

**Input**:
- Variable created
- registerStandardVariable("@app", varId)
- getByName("@app")

**References**:
- CRC: crc-VariableStore.md - "Does: registerStandardVariable"
- CRC: crc-Variable.md - "Does: isStandardVariable"

**Expected Results**:
- @app resolves to variable
- isStandardVariable returns true
- Path "@app.name" navigates correctly

---

### Test: Error message on creation failure

**Purpose**: Verify error response for invalid create

**Input**:
- create with invalid create property type

**References**:
- CRC: crc-ProtocolHandler.md - "Does: sendError"

**Expected Results**:
- error(varId, description) sent to client
- No variable created
- Description explains failure

---

## Coverage Summary

**Responsibilities Covered:**
- Variable: getValue, setValue, getProperty, setProperty, isObjectReference, isUnbound, isStandardVariable
- VariableStore: create, get, getByName, update, destroy, registerStandardVariable, getChildren
- ProtocolHandler: handleCreate, handleUpdate, handleWatch, handleUnwatch, handleDestroy, sendError, parsePropertyPriority
- WatchManager: watch, unwatch, shouldForwardWatch, shouldForwardUnwatch, notifyWatchers, isInactive

**Scenarios Covered:**
- seq-create-variable.md: All paths
- seq-update-variable.md: All paths
- seq-watch-variable.md: All paths
- seq-destroy-variable.md: All paths

**Gaps**: None identified
