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

## Coverage Summary

**Responsibilities Covered:**
- FrontendApp: initialize, handleBootstrap, handleVariableUpdate, sendMessage, navigateTo, handleTabActivation, showNotification
- SPANavigator: bindToApp, handleHistoryChange, pushState, replaceState, go, handlePopState, buildFullUrl
- ViewRenderer: render, clear, createElements, bindElements, handleViewChange, renderViewList, renderNestedView, updateDynamicContent
- WidgetBinder: bindWidget, bindShoelaceInput, bindShoelaceButton, bindShoelaceSelect, bindTabulator, bindDivContent, bindDivView, bindDivViewList, bindDynamicViewdef

**Scenarios Covered:**
- seq-bootstrap.md: All paths
- seq-spa-navigate.md: All paths
- seq-render-view.md: All paths

**Gaps**: None identified
