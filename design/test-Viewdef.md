# Test Design: Viewdef System

**Source Specs**: viewdefs.md, libraries.md
**CRC Cards**: crc-Viewdef.md, crc-ViewdefStore.md, crc-View.md, crc-ViewList.md, crc-Widget.md, crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md
**Sequences**: seq-load-viewdefs.md, seq-viewdef-delivery.md, seq-render-view.md, seq-viewlist-update.md, seq-bind-element.md, seq-handle-event.md (value sync, local value setting), seq-handle-keypress-event.md

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

### Test: Parse ui-event-keypress-enter binding

**Purpose**: Verify keypress event binding for Enter key

**Input**:
- HTML: `<input ui-event-keypress-enter="onEnter">`

**References**:
- CRC: crc-EventBinding.md - "Does: attach, handleKeypressEvent"
- CRC: crc-BindingEngine.md - "Does: createKeypressEventBinding, parseKeypressAttribute"
- Sequence: seq-handle-keypress-event.md

**Expected Results**:
- Keydown event listener attached
- Target key: "Enter" (normalized from "enter")
- Path: "onEnter"
- Handler filters for Enter key only

---

### Test: Parse ui-event-keypress-escape binding

**Purpose**: Verify keypress event binding for Escape key

**Input**:
- HTML: `<div ui-event-keypress-escape="onCancel" tabindex="0">`

**References**:
- CRC: crc-EventBinding.md - "Does: attach, matchesTargetKey"
- Sequence: seq-handle-keypress-event.md

**Expected Results**:
- Keydown event listener attached
- Target key: "Escape" (normalized from "escape")
- Path: "onCancel"

---

### Test: Parse ui-event-keypress-arrow bindings

**Purpose**: Verify keypress event bindings for arrow keys

**Input**:
- HTML: `<div ui-event-keypress-left="onLeft" ui-event-keypress-right="onRight">`

**References**:
- CRC: crc-EventBinding.md - "Keypress Binding"
- CRC: crc-BindingEngine.md - "Does: normalizeKeyName"

**Expected Results**:
- Two keydown listeners attached
- Target keys: "ArrowLeft", "ArrowRight"
- Each binding independent

---

### Test: Parse ui-event-keypress-letter binding

**Purpose**: Verify keypress event binding for letter keys

**Input**:
- HTML: `<input ui-event-keypress-a="onLetterA">`

**References**:
- CRC: crc-EventBinding.md - "Keypress Binding"
- Sequence: seq-handle-keypress-event.md

**Expected Results**:
- Keydown event listener attached
- Target key: "a" (case-insensitive match)
- Matches both "a" and "A"

---

### Test: Keypress binding fires only on matching key

**Purpose**: Verify keypress event filtering

**Input**:
- Input with `ui-event-keypress-enter="onEnter"`
- User presses "a" key
- User presses "Enter" key

**References**:
- CRC: crc-EventBinding.md - "Does: matchesTargetKey, handleKeypressEvent"
- Sequence: seq-handle-keypress-event.md

**Expected Results**:
- "a" key press: no variable update
- "Enter" key press: update(varId, "enter") sent
- Variable value is "enter" (lowercase key name)

---

### Test: Keypress binding sets variable to key name

**Purpose**: Verify keypress binding value

**Input**:
- Input with `ui-event-keypress-escape="onEscape"`
- User presses Escape key

**References**:
- CRC: crc-EventBinding.md - "handleKeypressEvent"
- Sequence: seq-handle-keypress-event.md

**Expected Results**:
- update(varId, "escape") sent
- Variable value is "escape" (the attribute key name, not "Escape")

---

### Test: Multiple keypress bindings on same element

**Purpose**: Verify multiple keypress bindings work independently

**Input**:
- Input with `ui-event-keypress-enter="onEnter" ui-event-keypress-tab="onTab"`
- User presses Enter, then Tab

**References**:
- CRC: crc-EventBinding.md - "Keypress Binding"
- Sequence: seq-handle-keypress-event.md

**Expected Results**:
- Enter press: updates onEnter variable to "enter"
- Tab press: updates onTab variable to "tab"
- Each binding has its own variable

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

### Test: Local value setting on variable update

**Purpose**: Verify local value is set before sending update to backend

**Input**:
- Input with ui-value="name" bound to variable 5
- User changes input value to "Alice"

**References**:
- CRC: crc-BindingEngine.md - "Does: sendVariableUpdate", "Local Value Setting"
- CRC: crc-EventBinding.md - "Local Value Setting"
- Sequence: seq-handle-event.md

**Expected Results**:
- Variable 5's local value set to "Alice" immediately
- update(5, "Alice") message sent to backend
- UI reflects "Alice" before backend response (optimistic update)

---

### Test: Event binding syncs ui-value before sending event

**Purpose**: Verify event binding checks for ui-value binding on same widget and syncs first

**Input**:
- Input with `ui-value="name"` (variable 5) and `ui-event-keypress-enter="onSubmit"` (variable 7)
- User types "John" in input (not yet synced, variable 5 still has old value "Jo")
- User presses Enter

**References**:
- CRC: crc-EventBinding.md - "Does: syncValueBinding", "Event Update Behavior"
- CRC: crc-Widget.md - "Does: hasBinding, getVariableId"
- Sequence: seq-handle-event.md

**Expected Results**:
- EventBinding checks widget for "ui-value" binding
- Found: variable 5 with path "name"
- Current element value "John" differs from variable 5's value "Jo"
- update(5, "John") sent first (with local value set)
- update(7, "enter") sent second

---

### Test: Event binding skips sync when values match (duplicate suppression)

**Purpose**: Verify event binding skips value sync when values already match (duplicate suppression)

**Input**:
- Input with `ui-value="name"` (variable 5) and `ui-event-click="onClick"` (variable 7)
- Variable 5 value is "John", input value is "John"
- User clicks input

**References**:
- CRC: crc-EventBinding.md - "Does: syncValueBinding", "Event Update Behavior"
- CRC: crc-BindingEngine.md - "Duplicate Update Suppression"
- Sequence: seq-handle-event.md

**Expected Results**:
- EventBinding checks widget for "ui-value" binding
- Found: variable 5, values match
- Duplicate suppression: no update sent for variable 5
- Only update(7, clickValue) sent

---

### Test: Event binding skips sync when no ui-value binding

**Purpose**: Verify event binding works normally without ui-value binding

**Input**:
- Button with `ui-event-click="onClick"` (no ui-value binding)
- User clicks button

**References**:
- CRC: crc-EventBinding.md - "Does: syncValueBinding"
- CRC: crc-Widget.md - "Does: hasBinding"
- Sequence: seq-handle-event.md

**Expected Results**:
- EventBinding checks widget for "ui-value" binding
- Not found
- Event update sent normally (no value sync step)

---

### Test: Duplicate update suppression - value unchanged on blur

**Purpose**: Verify no update sent when input value hasn't changed (blur event)

**Input**:
- Input with `ui-value="name"` bound to variable 5
- Variable 5 value is "John"
- User focuses input, doesn't change value, then blurs

**References**:
- CRC: crc-ValueBinding.md - "Duplicate Update Suppression"
- CRC: crc-BindingEngine.md - "shouldSuppressUpdate"
- Sequence: seq-input-value-binding.md

**Expected Results**:
- Blur event fires
- extractValue gets "John"
- shouldSuppressUpdate(variable5, "John") returns true (values equal)
- No update message sent
- No local value set (already correct)

---

### Test: Duplicate update suppression - value changed sends update

**Purpose**: Verify update sent when input value has changed

**Input**:
- Input with `ui-value="name"` bound to variable 5
- Variable 5 value is "John"
- User changes input to "Jane", then blurs

**References**:
- CRC: crc-ValueBinding.md - "Duplicate Update Suppression"
- CRC: crc-BindingEngine.md - "shouldSuppressUpdate"
- Sequence: seq-input-value-binding.md

**Expected Results**:
- Blur event fires
- extractValue gets "Jane"
- shouldSuppressUpdate(variable5, "Jane") returns false (values differ)
- variable5.value set to "Jane" locally first
- update(5, "Jane") message sent

---

### Test: Duplicate update suppression - action binding always sends

**Purpose**: Verify action bindings (access=action) always send, even with same value

**Input**:
- Button with `ui-action="save()"` bound to variable 7 (access=action)
- Variable 7 has null value
- User clicks button twice

**References**:
- CRC: crc-BindingEngine.md - "Duplicate Update Suppression"
- CRC: crc-EventBinding.md - "Does: isAction"
- Sequence: seq-handle-event.md

**Expected Results**:
- First click: update(7, null) sent
- Second click: update(7, null) sent (not suppressed)
- Both clicks processed because access=action bypasses suppression

---

### Test: Duplicate update suppression - write-only binding always sends

**Purpose**: Verify write-only bindings (access=w) always send, even with same value

**Input**:
- Element with binding having access=w property
- Same value sent twice

**References**:
- CRC: crc-BindingEngine.md - "Duplicate Update Suppression"
- CRC: crc-ValueBinding.md - "Duplicate Update Suppression"

**Expected Results**:
- First update sent
- Second update with same value also sent (not suppressed)
- access=w bypasses duplicate suppression

---

### Test: Duplicate update suppression - rw binding suppresses duplicates

**Purpose**: Verify read-write bindings (access=rw) suppress duplicate updates

**Input**:
- Input with `ui-value="name"` (default access=rw on interactive element)
- Variable value is "Test"
- User triggers blur without changing value

**References**:
- CRC: crc-BindingEngine.md - "Duplicate Update Suppression", "Default Access Property"
- CRC: crc-ValueBinding.md - "shouldSuppressUpdate"

**Expected Results**:
- access=rw is not action or w
- Values are equal
- shouldSuppressUpdate returns true
- No update sent

---

### Test: Widget hasBinding returns correct result

**Purpose**: Verify Widget.hasBinding method for sibling binding lookup

**Input**:
- Element with ui-value="name" and ui-event-click="onClick"
- Check hasBinding("ui-value"), hasBinding("ui-event-click"), hasBinding("ui-attr-disabled")

**References**:
- CRC: crc-Widget.md - "Does: hasBinding"

**Expected Results**:
- hasBinding("ui-value") returns true
- hasBinding("ui-event-click") returns true
- hasBinding("ui-attr-disabled") returns false

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

### Test: Widget created for element with bindings

**Purpose**: Verify Widget creation during binding

**Input**:
- Element with ui-value="name" attribute
- Element has no id attribute

**References**:
- CRC: crc-Widget.md - "Does: create, vendElementId"
- CRC: crc-BindingEngine.md - "Does: getOrCreateWidget"
- Sequence: seq-bind-element.md

**Expected Results**:
- Widget created for element
- Element assigned auto-vended ID (format: ui-widget-{counter})
- Widget tracks binding name to variable ID mapping

---

### Test: Widget uses existing element ID

**Purpose**: Verify Widget respects existing element IDs

**Input**:
- Element with id="my-input" and ui-value="name"

**References**:
- CRC: crc-Widget.md - "Does: create"
- CRC: crc-BindingEngine.md - "Does: getOrCreateWidget"

**Expected Results**:
- Widget uses "my-input" as elementId
- No auto-vended ID assigned
- Element id attribute unchanged

---

### Test: Widget tracks multiple bindings

**Purpose**: Verify Widget variable mapping with multiple bindings

**Input**:
- Element with ui-value="name" and ui-attr-disabled="isLocked"

**References**:
- CRC: crc-Widget.md - "Knows: variables"
- CRC: crc-BindingEngine.md - "Does: bind"

**Expected Results**:
- Single Widget created for element
- Widget.variables has entries for both bindings
- Each binding mapped to its child variable ID

---

### Test: Widget cleanup on unbind

**Purpose**: Verify Widget destroyed when all bindings removed

**Input**:
- Element with single binding
- Binding removed via unbind

**References**:
- CRC: crc-Widget.md - "Does: destroy"
- CRC: crc-BindingEngine.md - "Does: unbind"
- Sequence: seq-bind-element.md

**Expected Results**:
- Widget destroyed
- Auto-vended element ID removed (if applicable)
- No memory leaks

---

### Test: Variable stores elementId not DOM reference

**Purpose**: Verify variable-Widget relationship via elementId

**Input**:
- Element with ui-code binding
- Variable created for binding

**References**:
- CRC: crc-Widget.md - "Variable-Widget relationship"
- CRC: crc-ValueBinding.md - "Knows: elementId"

**Expected Results**:
- Variable has elementId property (string)
- Variable does NOT have direct element reference
- Element accessible via document.getElementById(elementId)

---

### Test: ui-code binding receives extended scope

**Purpose**: Verify ui-code execution scope includes variable and store

**Input**:
- `<div ui-code="myCode"></div>`
- Variable value: `element.dataset.test = variable.id + ':' + typeof store`

**References**:
- CRC: crc-ValueBinding.md - "Does: executeCode"
- CRC: crc-BindingEngine.md - "ui-code Binding"

**Expected Results**:
- Code executes with `element` (DOM element)
- Code executes with `value` (the code string)
- Code executes with `variable` (the child variable)
- Code executes with `store` (the VariableStore)

---

### Test: ui-code element lookup by ID

**Purpose**: Verify ui-code looks up element by ID (not stored reference)

**Input**:
- Element with ui-code binding
- Element replaced in DOM (same ID)
- Variable value changes

**References**:
- CRC: crc-ValueBinding.md - "Does: executeCode, getElement"
- CRC: crc-Widget.md - "Why Element ID"

**Expected Results**:
- Code execution finds current element (not stale reference)
- New element modified by code
- No errors from stale DOM reference

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

### Test: View destroy sends variable destroy message

**Purpose**: Verify View.destroy() destroys its associated variable

**Input**:
- View created with variableId 5
- View.destroy() called

**References**:
- CRC: crc-View.md - "Does: destroy", "Variable Destruction"
- Spec: viewdefs.md - "Variable destruction on re-render"
- Sequence: seq-viewdef-hotload.md

**Expected Results**:
- destroy(5) message sent to backend
- View's variableId set to null
- Backend recursively destroys child variables

---

### Test: ViewList destroy sends variable destroy message

**Purpose**: Verify ViewList.destroy() destroys its associated variable

**Input**:
- ViewList created with variableId 7
- ViewList.destroy() called

**References**:
- CRC: crc-ViewList.md - "Does: destroy", "Variable Destruction"
- Spec: viewdefs.md - "Variable destruction on re-render"
- Sequence: seq-viewdef-hotload.md

**Expected Results**:
- destroy(7) message sent to backend
- ViewList's variableId set to null
- Backend recursively destroys child variables (including item views)

---

### Test: Hot-reload re-render destroys old child variables

**Purpose**: Verify viewdef hot-reload destroys child variables before re-rendering

**Input**:
- View with viewdefKey "Contact.DEFAULT" containing child ui-view (variableId 10)
- Viewdef "Contact.DEFAULT" updated via hot-reload
- View.forceRender() called

**References**:
- CRC: crc-View.md - "Does: rerender", "Variable Destruction"
- Spec: viewdefs.md - "Variable destruction on re-render"
- Sequence: seq-viewdef-hotload.md

**Expected Results**:
- clearChildren() called which calls childView.destroy()
- destroy(10) message sent for old child variable
- New child variable created with new variableId
- No variable leak - old variables cleaned up on backend

---

## Coverage Summary

**Responsibilities Covered:**
- Viewdef: getKey, getTemplate, parseBindings, hasBinding, clone
- ViewdefStore: store, get, getForType, has, validate, batchUpdate, flushUpdates, addPendingView, processPendingViews, removePendingView
- View: create, render, setVariable, clear, destroy (variable destruction), getHtmlId, markPending, removePending
- ViewList: create, setExemplar, update, addItem, removeItem, reorder, clear, destroy (variable destruction), setDelegate, notifyAdd, notifyRemove
- Widget: create, vendElementId, registerBinding, unregisterBinding, getVariableId, hasBinding, getElement, destroy
- BindingEngine: bind, unbind, getOrCreateWidget, createValueBinding, createEventBinding, createKeypressEventBinding, parseKeypressAttribute, normalizeKeyName, parsePath, updateBinding, sendVariableUpdate, shouldSuppressUpdate
- ValueBinding: apply, update, getElement, getTargetProperty, transformValue, executeCode (extended scope), shouldSuppressUpdate, destroy
- EventBinding: attach, detach, handleEvent, handleKeypressEvent, extractEventValue, matchesTargetKey, isAction, isKeypressBinding, syncValueBinding, destroy

**Scenarios Covered:**
- seq-load-viewdefs.md: All paths (including validation)
- seq-viewdef-delivery.md: All paths
- seq-render-view.md: All paths (including pending views, namespace fallback)
- seq-viewlist-update.md: All paths (add, remove, reorder)
- seq-bind-element.md: All paths (including Widget creation, element ID vending)
- seq-handle-event.md: All paths (including value sync, local value setting, duplicate suppression)
- seq-handle-keypress-event.md: All paths (key matching, filtering, value setting)
- seq-input-value-binding.md: All paths (including duplicate suppression)

**Gaps**: None identified
