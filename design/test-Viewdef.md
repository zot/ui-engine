# Test Design: Viewdef System

**Source Specs**: viewdefs.md, libraries.md
**CRC Cards**: crc-Viewdef.md, crc-ViewdefStore.md, crc-View.md, crc-ViewList.md, crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md
**Sequences**: seq-load-viewdefs.md, seq-viewdef-delivery.md, seq-render-view.md, seq-viewlist-update.md, seq-bind-element.md, seq-handle-event.md

## Overview

Tests for viewdef loading and validation, View/ViewList rendering, binding creation, and event handling.

## Test Cases

### Test: Load viewdefs from variable 1

**Purpose**: Verify initial viewdef loading on bootstrap

**Input**:
- watch(1) returns {viewdefs: {"Person.DEFAULT": "<template><div ui-value='name'></div></template>"}}

**References**:
- CRC: crc-ViewdefStore.md - "Does: store, validate"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- Viewdef stored with key "Person.DEFAULT"
- Template element accessible
- Bindings parsed and cached

---

### Test: Validate viewdef format - valid template

**Purpose**: Verify viewdef validation accepts proper template

**Input**:
- HTML: `<template><div class="person"><span ui-value="name"></span></div></template>`

**References**:
- CRC: crc-ViewdefStore.md - "Does: validate"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- Validation passes
- Template stored
- No error sent

---

### Test: Validate viewdef format - reject non-template root

**Purpose**: Verify viewdef validation rejects non-template root

**Input**:
- HTML: `<div ui-value="name"></div>`

**References**:
- CRC: crc-ViewdefStore.md - "Does: validate"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- Validation fails
- error(1, "invalid viewdef") sent to backend
- Viewdef not stored

---

### Test: Validate viewdef format - reject multiple roots

**Purpose**: Verify viewdef validation rejects multiple root elements

**Input**:
- HTML: `<template><div>A</div></template><template><div>B</div></template>`

**References**:
- CRC: crc-ViewdefStore.md - "Does: validate"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- Validation fails
- error(1, "multiple root elements") sent
- Viewdef not stored

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

### Test: Parse ui-style-* binding

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

### Test: View render returns true when ready

**Purpose**: Verify successful view rendering

**Input**:
- Variable with value (object ref), type property "Contact"
- Viewdef "Contact.DEFAULT" exists

**References**:
- CRC: crc-View.md - "Does: render"
- Sequence: seq-render-view.md

**Expected Results**:
- render() returns true
- Template cloned into element
- Bindings applied
- View has unique HTML id

---

### Test: View render returns false when missing type

**Purpose**: Verify pending view when type not available

**Input**:
- Variable with value but no type property
- Viewdef exists

**References**:
- CRC: crc-View.md - "Does: render, markPending"
- CRC: crc-ViewdefStore.md - "Does: addPendingView"
- Sequence: seq-render-view.md

**Expected Results**:
- render() returns false
- View added to pending list
- Element remains empty

---

### Test: View render returns false when missing viewdef

**Purpose**: Verify pending view when viewdef not available

**Input**:
- Variable with value and type "Contact"
- No viewdef for "Contact.DEFAULT"

**References**:
- CRC: crc-View.md - "Does: render, markPending"
- CRC: crc-ViewdefStore.md - "Does: addPendingView"
- Sequence: seq-render-view.md

**Expected Results**:
- render() returns false
- View added to pending list
- Element remains empty

---

### Test: View falls back to TYPE.DEFAULT namespace

**Purpose**: Verify namespace fallback

**Input**:
- Variable with type "Contact"
- ui-namespace="COMPACT"
- No viewdef for "Contact.COMPACT"
- Viewdef for "Contact.DEFAULT" exists

**References**:
- CRC: crc-ViewdefStore.md - "Does: get"
- CRC: crc-ViewRenderer.md - "Does: lookupViewdef"
- Sequence: seq-render-view.md

**Expected Results**:
- Falls back to "Contact.DEFAULT"
- render() returns true
- View rendered with DEFAULT viewdef

---

### Test: Pending views processed when viewdefs arrive

**Purpose**: Verify pending view mechanism

**Input**:
- View added to pending (missing viewdef)
- Update to variable 1 delivers "Contact.DEFAULT" viewdef

**References**:
- CRC: crc-ViewdefStore.md - "Does: processPendingViews, removePendingView"
- Sequence: seq-load-viewdefs.md

**Expected Results**:
- processPendingViews called after storing
- Pending view renders successfully
- View removed from pending list

---

### Test: View gets unique HTML id

**Purpose**: Verify frontend-vended HTML ids

**Input**:
- Create multiple Views

**References**:
- CRC: crc-View.md - "Knows: htmlId"
- CRC: crc-ViewRenderer.md - "Does: vendHtmlId"

**Expected Results**:
- Each View has unique htmlId
- htmlId set as element id attribute
- Ids never duplicate

---

### Test: ViewList creates views for array items

**Purpose**: Verify ViewList initialization

**Input**:
- `<div ui-viewlist="contacts"></div>`
- Variable value: [{obj:5}, {obj:7}, {obj:9}]

**References**:
- CRC: crc-ViewList.md - "Does: create, addItem"
- Sequence: seq-viewlist-update.md

**Expected Results**:
- 3 child elements created
- Each is clone of exemplar (div)
- Each has View bound to respective object

---

### Test: ViewList adds items on array grow

**Purpose**: Verify ViewList handles additions

**Input**:
- ViewList with 2 items
- Array updated to 3 items

**References**:
- CRC: crc-ViewList.md - "Does: update, addItem, notifyAdd"
- Sequence: seq-viewlist-update.md

**Expected Results**:
- New element cloned from exemplar
- Variable created for new item
- View rendered and appended
- Delegate notified of addition

---

### Test: ViewList removes items on array shrink

**Purpose**: Verify ViewList handles removals

**Input**:
- ViewList with 3 items
- Array updated to 2 items

**References**:
- CRC: crc-ViewList.md - "Does: update, removeItem, notifyRemove"
- Sequence: seq-viewlist-update.md

**Expected Results**:
- Removed element destroyed
- Variable destroyed
- Element removed from DOM
- Delegate notified of removal

---

### Test: ViewList reorders items on array reorder

**Purpose**: Verify ViewList handles reordering

**Input**:
- ViewList with [{obj:5}, {obj:7}, {obj:9}]
- Array updated to [{obj:9}, {obj:5}, {obj:7}]

**References**:
- CRC: crc-ViewList.md - "Does: update, reorder"
- Sequence: seq-viewlist-update.md

**Expected Results**:
- DOM elements reordered (not recreated)
- Views maintain bindings
- No variables destroyed

---

### Test: ViewList uses custom exemplar

**Purpose**: Verify custom exemplar element

**Input**:
- ViewList with sl-option exemplar
- Array with 2 items

**References**:
- CRC: crc-ViewList.md - "Does: setExemplar"
- Sequence: seq-viewlist-update.md

**Expected Results**:
- Each item cloned from sl-option
- sl-option elements created (not div)
- Works for Select Views

---

### Test: ViewList notifies delegate

**Purpose**: Verify delegate notifications

**Input**:
- ViewList with delegate set
- Add and remove items

**References**:
- CRC: crc-ViewList.md - "Does: setDelegate, notifyAdd, notifyRemove"

**Expected Results**:
- notifyAdd called with item info
- notifyRemove called with item info
- Delegate can respond to changes

---

## Coverage Summary

**Responsibilities Covered:**
- Viewdef: getKey, getTemplate, parseBindings, hasBinding, clone
- ViewdefStore: store, get, getForType, has, validate, batchUpdate, flushUpdates, addPendingView, processPendingViews, removePendingView
- View: create, render, setVariable, clear, getHtmlId, markPending, removePending
- ViewList: create, setExemplar, update, addItem, removeItem, reorder, clear, setDelegate, notifyAdd, notifyRemove
- BindingEngine: bind, unbind, createValueBinding, createEventBinding, parsePath, updateBinding
- ValueBinding: apply, update, getTargetProperty, transformValue, destroy
- EventBinding: attach, detach, handleEvent, extractEventValue, isAction, destroy

**Scenarios Covered:**
- seq-load-viewdefs.md: All paths (including validation)
- seq-viewdef-delivery.md: All paths
- seq-render-view.md: All paths (including pending views, namespace fallback)
- seq-viewlist-update.md: All paths (add, remove, reorder)
- seq-bind-element.md: All paths
- seq-handle-event.md: All paths

**Gaps**: None identified
