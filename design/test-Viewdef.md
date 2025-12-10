# Test Design: Viewdef System

**Source Specs**: viewdefs.md, libraries.md
**CRC Cards**: crc-Viewdef.md, crc-ViewdefStore.md, crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md
**Sequences**: seq-load-viewdefs.md, seq-bind-element.md, seq-handle-event.md, seq-render-view.md

## Overview

Tests for viewdef loading, binding creation, and event handling.

## Test Cases

### Test: Load viewdefs from variable 1

**Purpose**: Verify initial viewdef loading on bootstrap

**Input**:
- watch(1) returns {viewdefs: {"Person.DEFAULT": "<div ui-value='name'></div>"}}

**References**:
- CRC: crc-ViewdefStore.md - "Does: store"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- Viewdef stored with key "Person.DEFAULT"
- HTML template accessible
- Bindings parsed and cached

---

### Test: Viewdef update replaces previous

**Purpose**: Verify viewdef replacement on update

**Input**:
- Viewdef "Person.DEFAULT" stored
- Update with new viewdefs property

**References**:
- CRC: crc-ViewdefStore.md - "Does: batchUpdate"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- New viewdef replaces old
- Previous versions not retained
- Active views re-rendered

---

### Test: Parse ui-value binding

**Purpose**: Verify value binding extraction

**Input**:
- HTML: `<input ui-value="name">`

**References**:
- CRC: crc-Viewdef.md - "Does: parseBindings"
- CRC: crc-ValueBinding.md

**Expected Results**:
- Binding type: value
- Path: "name"
- Target: input element value property

---

### Test: Parse ui-attr-* binding

**Purpose**: Verify attribute binding extraction

**Input**:
- HTML: `<input ui-attr-disabled="isLocked">`

**References**:
- CRC: crc-BindingEngine.md - "Does: createValueBinding"

**Expected Results**:
- Binding type: attr
- Attribute: disabled
- Path: "isLocked"

---

### Test: Parse ui-class-* binding

**Purpose**: Verify class binding extraction

**Input**:
- HTML: `<div ui-class-status="statusClass">`

**References**:
- CRC: crc-ValueBinding.md - "Knows: bindingType"

**Expected Results**:
- Binding type: class
- Value becomes CSS class
- Multiple classes supported

---

### Test: Parse ui-style-*-* binding

**Purpose**: Verify style binding extraction

**Input**:
- HTML: `<div ui-style-background-color="bgColor">`

**References**:
- CRC: crc-ValueBinding.md

**Expected Results**:
- Binding type: style
- Property: backgroundColor (camelCase)
- Value applied to element.style

---

### Test: Parse ui-event-* binding

**Purpose**: Verify event binding extraction

**Input**:
- HTML: `<input ui-event-change="onNameChange">`

**References**:
- CRC: crc-EventBinding.md - "Does: attach"
- Sequence: seq-bind-element.md

**Expected Results**:
- Event listener attached for "change"
- Path: "onNameChange"
- Handler updates variable on event

---

### Test: Parse ui-action binding

**Purpose**: Verify action binding (button)

**Input**:
- HTML: `<sl-button ui-action="submit()">Send</sl-button>`

**References**:
- CRC: crc-EventBinding.md - "Does: isAction"
- Sequence: seq-handle-event.md

**Expected Results**:
- Click listener attached
- Action path: "submit()"
- Triggers method call on presenter

---

### Test: Path with URL parameters

**Purpose**: Verify ?key=value parsing in paths

**Input**:
- HTML: `<div ui-view="child?create=Person&name=test">`

**References**:
- CRC: crc-BindingEngine.md - "Does: parsePath"
- CRC: crc-PathSyntax.md

**Expected Results**:
- Base path: "child"
- Parameters: {create: "Person", name: "test"}
- Variable created with properties

---

### Test: Apply value binding to element

**Purpose**: Verify binding updates element

**Input**:
- Binding created for ui-value="name"
- Variable value: "John"

**References**:
- CRC: crc-ValueBinding.md - "Does: apply"

**Expected Results**:
- Element value set to "John"
- DOM updated immediately
- No extra events fired

---

### Test: Update binding on variable change

**Purpose**: Verify reactive updates

**Input**:
- Binding active with value "old"
- Variable updated to "new"

**References**:
- CRC: crc-ValueBinding.md - "Does: update"
- Sequence: seq-update-variable.md

**Expected Results**:
- Element value changes to "new"
- Update triggered by watch notification
- Other bindings unaffected

---

### Test: Handle DOM event updates variable

**Purpose**: Verify event-to-variable flow

**Input**:
- Input with ui-event-input="name"
- User types "Alice"

**References**:
- CRC: crc-EventBinding.md - "Does: handleEvent"
- Sequence: seq-handle-event.md

**Expected Results**:
- update(varId, "Alice") sent
- Variable value changed
- Other watchers notified

---

### Test: Handle action event

**Purpose**: Verify action triggers method

**Input**:
- Button with ui-action="save()"
- User clicks button

**References**:
- CRC: crc-EventBinding.md - "Does: handleEvent"
- Sequence: seq-handle-event.md

**Expected Results**:
- update(varId, {action: "save"}) sent
- Includes form values if applicable
- Presenter method invoked

---

### Test: Cleanup bindings on element removal

**Purpose**: Verify binding cleanup

**Input**:
- Element with bindings in DOM
- Element removed from DOM

**References**:
- CRC: crc-BindingEngine.md - "Does: unbind"
- CRC: crc-ValueBinding.md - "Does: destroy"

**Expected Results**:
- Event listeners removed
- Variable unwatch sent
- No memory leaks

---

### Test: Render ui-content HTML

**Purpose**: Verify HTML content rendering

**Input**:
- `<div ui-content="htmlContent"></div>`
- Variable value: `<strong>Bold</strong>`

**References**:
- CRC: crc-WidgetBinder.md - "Does: bindDivContent"
- Sequence: seq-render-view.md

**Expected Results**:
- innerHTML set to variable value
- HTML rendered (not escaped)
- Updates on value change

---

### Test: Render ui-view nested object

**Purpose**: Verify nested view rendering

**Input**:
- `<div ui-view="child" ui-namespace="Person"></div>`
- child is object reference to Person presenter

**References**:
- CRC: crc-ViewRenderer.md - "Does: renderNestedView"
- Sequence: seq-render-view.md

**Expected Results**:
- Person.DEFAULT viewdef loaded
- Child presenter rendered inside div
- Bindings applied to nested content

---

### Test: Render ui-viewlist array

**Purpose**: Verify array rendering

**Input**:
- `<div ui-viewlist="items" ui-namespace="Item"></div>`
- items is array of 3 object references

**References**:
- CRC: crc-ViewRenderer.md - "Does: renderViewList"
- Sequence: seq-render-view.md

**Expected Results**:
- 3 Item.DEFAULT views rendered
- Each bound to respective object
- Array updates re-render list

---

## Coverage Summary

**Responsibilities Covered:**
- Viewdef: getKey, getHtml, parseBindings, hasBinding, clone
- ViewdefStore: store, get, getForType, has, batchUpdate, flushUpdates
- BindingEngine: bind, unbind, createValueBinding, createEventBinding, parsePath, updateBinding
- ValueBinding: apply, update, getTargetProperty, transformValue, destroy
- EventBinding: attach, detach, handleEvent, extractEventValue, isAction, destroy

**Scenarios Covered:**
- seq-load-viewdefs.md: All paths
- seq-bind-element.md: All paths
- seq-handle-event.md: All paths
- seq-render-view.md: All paths

**Gaps**: None identified
