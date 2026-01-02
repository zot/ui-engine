# Test Design: Frontend Library

**Source Specs**: libraries.md, interfaces.md
**CRC Cards**: crc-FrontendApp.md, crc-SPANavigator.md, crc-ViewRenderer.md, crc-WidgetBinder.md
**Sequences**: seq-bootstrap.md, seq-spa-navigate.md, seq-render-view.md

## Overview

Tests for browser-side SPA navigation, view rendering, and widget bindings.

## Test Cases

### Test: Frontend app initialization

**Purpose**: Verify app bootstrap

**Input**:
- Page load with session URL

**References**:
- CRC: crc-FrontendApp.md - "Does: initialize"
- Sequence: seq-bootstrap.md

**Expected Results**:
- Session ID parsed from URL
- SharedWorker connected
- Variable 1 watched

---

### Test: Handle bootstrap viewdefs

**Purpose**: Verify initial viewdef loading

**Input**:
- update(1) received with viewdefs

**References**:
- CRC: crc-FrontendApp.md - "Does: handleBootstrap"
- Sequence: seq-bootstrap.md

**Expected Results**:
- Viewdefs stored locally
- Initial view rendered
- App ready for interaction

---

### Test: Handle variable update

**Purpose**: Verify update processing

**Input**:
- update(varId, value) received

**References**:
- CRC: crc-FrontendApp.md - "Does: handleVariableUpdate"

**Expected Results**:
- Bound elements updated
- View refreshed if needed
- No unnecessary re-renders

---

### Test: Send message via SharedWorker

**Purpose**: Verify outbound message path

**Input**:
- sendMessage(update)

**References**:
- CRC: crc-FrontendApp.md - "Does: sendMessage"

**Expected Results**:
- Message relayed through SharedWorker
- Reaches server
- No direct WebSocket access

---

### Test: Handle tab activation request

**Purpose**: Verify activation response

**Input**:
- Activation message from SharedWorker

**References**:
- CRC: crc-FrontendApp.md - "Does: handleTabActivation"

**Expected Results**:
- Window focused
- Navigation applied if path provided
- Confirmation sent

---

### Test: Show desktop notification

**Purpose**: Verify notification display

**Input**:
- showNotification("message")

**References**:
- CRC: crc-FrontendApp.md - "Does: showNotification"

**Expected Results**:
- Notification shown
- Click handler attached
- Permission handled

---

### Test: SPA bind to app presenter

**Purpose**: Verify history state binding

**Input**:
- bindToApp(appPresenter)

**References**:
- CRC: crc-SPANavigator.md - "Does: bindToApp"
- Sequence: seq-spa-navigate.md

**Expected Results**:
- historyIndex watched
- url watched
- State synchronized

---

### Test: SPA handle history index change

**Purpose**: Verify forward navigation

**Input**:
- historyIndex changes from 0 to 1

**References**:
- CRC: crc-SPANavigator.md - "Does: handleHistoryChange"

**Expected Results**:
- pushState called
- Browser URL updated
- View rendered

---

### Test: SPA pushState

**Purpose**: Verify browser history push

**Input**:
- pushState({}, "", "/page2")

**References**:
- CRC: crc-SPANavigator.md - "Does: pushState"

**Expected Results**:
- Browser history entry added
- URL changes
- No page reload

---

### Test: SPA replaceState

**Purpose**: Verify history replacement

**Input**:
- replaceState({}, "", "/page2")

**References**:
- CRC: crc-SPANavigator.md - "Does: replaceState"

**Expected Results**:
- Current entry replaced
- URL changes
- History length unchanged

---

### Test: SPA handle popstate

**Purpose**: Verify back/forward handling

**Input**:
- Browser back button clicked

**References**:
- CRC: crc-SPANavigator.md - "Does: handlePopState"

**Expected Results**:
- historyIndex updated
- App notified
- View rendered

---

### Test: View renderer render

**Purpose**: Verify view display

**Input**:
- render(presenter)

**References**:
- CRC: crc-ViewRenderer.md - "Does: render"
- Sequence: seq-render-view.md

**Expected Results**:
- Viewdef looked up by type
- HTML parsed
- DOM created and appended

---

### Test: View renderer clear

**Purpose**: Verify view cleanup

**Input**:
- clear()

**References**:
- CRC: crc-ViewRenderer.md - "Does: clear"

**Expected Results**:
- DOM content removed
- Bindings cleaned up
- Ready for new render

---

### Test: View renderer nested view

**Purpose**: Verify ui-view handling

**Input**:
- Element with ui-view="child" ui-namespace="Person"

**References**:
- CRC: crc-ViewRenderer.md - "Does: renderNestedView"

**Expected Results**:
- Child presenter resolved
- Person.DEFAULT viewdef used
- Nested content rendered

---

### Test: View renderer view list

**Purpose**: Verify ui-viewlist handling

**Input**:
- Element with ui-viewlist="items" ui-namespace="Item"

**References**:
- CRC: crc-ViewRenderer.md - "Does: renderViewList"

**Expected Results**:
- Array iterated
- Item.DEFAULT for each element
- All items rendered

---

### Test: View renderer dynamic content

**Purpose**: Verify ui-content handling

**Input**:
- Element with ui-content="htmlContent"
- Value: `<strong>Bold</strong>`

**References**:
- CRC: crc-ViewRenderer.md - "Does: updateDynamicContent"

**Expected Results**:
- innerHTML set
- HTML rendered
- Updates on change

---

### Test: View renderer collects scripts during cloning

**Purpose**: Verify script elements are collected from cloned content before DOM insertion

**Input**:
- Viewdef: `<template><div><script>window.testVar = 'collected';</script></div></template>`
- render(element, variable) called

**References**:
- CRC: crc-ViewRenderer.md - "Does: collectScripts"
- Sequence: seq-render-view.md

**Expected Results**:
- Script element collected during clone phase
- Script not yet executed (window.testVar undefined)
- Scripts stored for later activation

---

### Test: View renderer activates scripts after DOM insertion

**Purpose**: Verify scripts execute after binding (DOM-connected)

**Input**:
- Viewdef: `<template><div><script>window.scriptActivated = true;</script></div></template>`
- render(element, variable) called

**References**:
- CRC: crc-ViewRenderer.md - "Does: activateScripts"
- Sequence: seq-render-view.md

**Expected Results**:
- Script executes after appendToElement and binding
- window.scriptActivated is true
- Original script element replaced with new script element

---

### Test: View renderer script content executes

**Purpose**: Verify script content executes and can modify DOM/globals

**Input**:
- Viewdef: `<template><div id="target"><script>document.getElementById('target').textContent = 'modified';</script></div></template>`
- render(element, variable) called

**References**:
- CRC: crc-ViewRenderer.md - "Does: activateScripts"
- Sequence: seq-render-view.md

**Expected Results**:
- Script executes successfully
- DOM modified by script (target div contains 'modified')
- Script has access to document and window

---

### Test: View renderer multiple scripts execute in order

**Purpose**: Verify multiple scripts in viewdef execute sequentially in document order

**Input**:
- Viewdef: `<template><div><script>window.scriptOrder = [];</script><script>window.scriptOrder.push(1);</script><script>window.scriptOrder.push(2);</script></div></template>`
- render(element, variable) called

**References**:
- CRC: crc-ViewRenderer.md - "Does: collectScripts, activateScripts"
- Sequence: seq-render-view.md

**Expected Results**:
- All three scripts execute
- Scripts execute in document order
- window.scriptOrder equals [1, 2]

---

### Test: Widget binder Shoelace input

**Purpose**: Verify sl-input binding

**Input**:
- `<sl-input ui-value="name" ui-disabled="isLocked">`

**References**:
- CRC: crc-WidgetBinder.md - "Does: bindShoelaceInput"

**Expected Results**:
- value property bound
- disabled attribute bound
- Change events captured

---

### Test: Widget binder Shoelace button

**Purpose**: Verify sl-button binding

**Input**:
- `<sl-button ui-action="submit()">`

**References**:
- CRC: crc-WidgetBinder.md - "Does: bindShoelaceButton"

**Expected Results**:
- Click handler attached
- Action triggered on click
- Form values collected

---

### Test: Widget binder Shoelace select

**Purpose**: Verify sl-select binding

**Input**:
- `<sl-select ui-items="options" ui-index="selected">`

**References**:
- CRC: crc-WidgetBinder.md - "Does: bindShoelaceSelect"

**Expected Results**:
- Options populated from items
- Selection bound to index
- Change updates variable

---

### Test: Widget binder Tabulator

**Purpose**: Verify ui-tabulator binding

**Input**:
- `<div ui-tabulator="data" ui-columns="[...]">`

**References**:
- CRC: crc-WidgetBinder.md - "Does: bindTabulator"

**Expected Results**:
- Tabulator initialized
- Data bound
- Columns configured

---

### Test: ui-code binding executes JavaScript

**Purpose**: Verify ui-code binding executes code when variable updates

**Input**:
- `<div ui-code="codeVar"></div>`
- Variable `codeVar` updates to `"element.classList.add('highlighted')"`

**References**:
- CRC: crc-BindingEngine.md - "Does: createCodeBinding"
- CRC: crc-ValueBinding.md - "Does: executeCode"
- Sequence: seq-bind-element.md

**Expected Results**:
- Child variable created with path "codeVar"
- Code executed when variable value changes
- `element` parameter is the bound div
- `value` parameter is the current variable value
- Element has 'highlighted' class added

---

### Test: ui-code binding defaults to access=r

**Purpose**: Verify ui-code bindings are read-only by default

**Input**:
- `<div ui-code="codeVar"></div>`

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"
- CRC: crc-ValueBinding.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property
- Binding is read-only (no write to backend)

---

### Test: ui-code binding handles execution errors

**Purpose**: Verify ui-code binding catches and logs errors without throwing

**Input**:
- `<div ui-code="codeVar"></div>`
- Variable `codeVar` updates to `"throw new Error('test')"`

**References**:
- CRC: crc-ValueBinding.md - "Does: executeCode"

**Expected Results**:
- Error is caught and logged
- No exception thrown to caller
- Element remains bound

---

### Test: ui-value on non-interactive element defaults to access=r

**Purpose**: Verify ui-value on div/span defaults to read-only

**Input**:
- `<span ui-value="name"></span>`

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"
- CRC: crc-ValueBinding.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property
- Binding is read-only

---

### Test: ui-value on interactive element defaults to access=rw

**Purpose**: Verify ui-value on input/textarea defaults to read-write

**Input**:
- `<input ui-value="name">`

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"
- CRC: crc-ValueBinding.md - "Default Access Property"

**Expected Results**:
- Child variable created without explicit `access` property (defaults to rw)
- Binding supports two-way data flow

---

### Test: ui-attr binding defaults to access=r

**Purpose**: Verify ui-attr-* bindings default to read-only

**Input**:
- `<div ui-attr-disabled="isLocked"></div>`

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property
- Attribute updated from backend only

---

### Test: ui-class binding defaults to access=r

**Purpose**: Verify ui-class-* bindings default to read-only

**Input**:
- `<div ui-class-active="isActive"></div>`

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property
- Class updated from backend only

---

### Test: ui-style binding defaults to access=r

**Purpose**: Verify ui-style-* bindings default to read-only

**Input**:
- `<div ui-style-color="textColor"></div>`

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property
- Style updated from backend only

---

### Test: ui-view binding defaults to access=r

**Purpose**: Verify ui-view bindings default to read-only

**Input**:
- `<div ui-view="contact"></div>`

**References**:
- CRC: crc-View.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property
- View is read-only

---

### Test: ui-viewlist binding defaults to access=r

**Purpose**: Verify ui-viewlist bindings default to read-only

**Input**:
- `<div ui-viewlist="contacts"></div>`

**References**:
- CRC: crc-ViewList.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=r` property (and wrapper=lua.ViewList)
- ViewList is read-only

---

### Test: explicit access property overrides default

**Purpose**: Verify explicit access property takes precedence

**Input**:
- `<span ui-value="name?access=rw"></span>` (non-interactive with explicit rw)

**References**:
- CRC: crc-BindingEngine.md - "Default Access Property"

**Expected Results**:
- Child variable created with `access=rw` property
- Explicit property overrides default

---

## Coverage Summary

**Responsibilities Covered:**
- FrontendApp: initialize, handleBootstrap, handleVariableUpdate, sendMessage, navigateTo, handleTabActivation, showNotification
- SPANavigator: bindToApp, handleHistoryChange, pushState, replaceState, go, handlePopState, buildFullUrl
- ViewRenderer: render, clear, createElements, collectScripts, appendToElement, bindElements, activateScripts, handleViewChange, renderViewList, renderNestedView, updateDynamicContent
- WidgetBinder: bindWidget, bindShoelaceInput, bindShoelaceButton, bindShoelaceSelect, bindTabulator, bindDivContent, bindDivView, bindDivViewList, bindDynamicViewdef
- BindingEngine: createCodeBinding, determineDefaultAccess
- ValueBinding: executeCode (code binding execution)
- View: default access property
- ViewList: default access property

**Scenarios Covered:**
- seq-bootstrap.md: All paths
- seq-spa-navigate.md: All paths
- seq-render-view.md: All paths
- seq-bind-element.md: ui-code binding, default access property

**Gaps**: None identified
