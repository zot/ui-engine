# Test Design: MCP Integration

**Source Specs**: interfaces.md
**CRC Cards**: crc-MCPServer.md, crc-MCPResource.md, crc-MCPTool.md
**Sequences**: seq-mcp-create-session.md, seq-mcp-create-presenter.md, seq-mcp-receive-event.md

## Overview

Tests for Model Context Protocol server integration enabling AI assistants to create and control UIs.

## Test Cases

### Test: MCP server initialization

**Purpose**: Verify MCP server setup

**Input**:
- initialize() called by MCP client

**References**:
- CRC: crc-MCPServer.md - "Does: initialize"

**Expected Results**:
- Server ready for requests
- Resources and tools listed
- Connection to UI server established

---

### Test: List available resources

**Purpose**: Verify resource enumeration

**Input**:
- resources/list request

**References**:
- CRC: crc-MCPServer.md - "Does: listResources"

**Expected Results**:
- Presenter types resource
- Viewdefs resource
- Session state resource
- Pending messages resource

---

### Test: List available tools

**Purpose**: Verify tool enumeration

**Input**:
- tools/list request

**References**:
- CRC: crc-MCPServer.md - "Does: listTools"

**Expected Results**:
- create_session tool
- create_presenter tool
- update_presenter tool
- create_viewdef tool
- load_presenter_logic tool
- register_url_path tool
- activate_tab tool

---

### Test: Resource - list presenter types

**Purpose**: Verify presenter type query

**Input**:
- resources/read for presenter types

**References**:
- CRC: crc-MCPResource.md - "Does: listPresenterTypes"

**Expected Results**:
- App presenter type
- List presenter type
- Custom registered types
- Properties for each type

---

### Test: Resource - list viewdefs

**Purpose**: Verify viewdef listing

**Input**:
- resources/read for viewdefs

**References**:
- CRC: crc-MCPResource.md - "Does: listViewdefs"

**Expected Results**:
- TYPE.VIEW keys listed
- HTML content available
- Binding info extracted

---

### Test: Resource - get session state

**Purpose**: Verify session state query

**Input**:
- resources/read for session state

**References**:
- CRC: crc-MCPResource.md - "Does: getSessionState"

**Expected Results**:
- Current session ID
- Variable tree structure
- Active presenter info

---

### Test: Resource - get pending messages

**Purpose**: Verify user message queue

**Input**:
- resources/read for pending messages

**References**:
- CRC: crc-MCPResource.md - "Does: getPendingMessages"

**Expected Results**:
- Queued user messages
- Action events
- Form submissions

---

### Test: Tool - create session

**Purpose**: Verify session creation via MCP

**Input**:
- tools/call create_session

**References**:
- CRC: crc-MCPTool.md - "Does: createSession"
- Sequence: seq-mcp-create-session.md

**Expected Results**:
- Session ID returned
- Full URL returned
- Session ready for use

---

### Test: Tool - create presenter

**Purpose**: Verify presenter creation via MCP

**Input**:
- tools/call create_presenter with type and properties

**References**:
- CRC: crc-MCPTool.md - "Does: createPresenter"
- Sequence: seq-mcp-create-presenter.md

**Expected Results**:
- Variable ID returned
- Presenter created with properties
- Type property set

---

### Test: Tool - update presenter

**Purpose**: Verify presenter update via MCP

**Input**:
- tools/call update_presenter with properties

**References**:
- CRC: crc-MCPTool.md - "Does: updatePresenter"

**Expected Results**:
- Properties updated
- Watchers notified
- UI reflects changes

---

### Test: Tool - update presenter with method call

**Purpose**: Verify presenter method invocation

**Input**:
- tools/call update_presenter with call: "update", args: ["ACME", 142.50]

**References**:
- CRC: crc-MCPTool.md - "Does: updatePresenter"

**Expected Results**:
- Method invoked on presenter
- Arguments passed correctly
- State updated by method

---

### Test: Tool - create viewdef

**Purpose**: Verify viewdef creation via MCP

**Input**:
- tools/call create_viewdef with TYPE.VIEW and HTML

**References**:
- CRC: crc-MCPTool.md - "Does: createViewdef"
- Sequence: seq-mcp-create-presenter.md

**Expected Results**:
- Viewdef stored
- Bindings parsed
- Available for rendering

---

### Test: Tool - load presenter logic

**Purpose**: Verify Lua code loading via MCP

**Input**:
- tools/call load_presenter_logic with Lua code

**References**:
- CRC: crc-MCPTool.md - "Does: loadPresenterLogic"

**Expected Results**:
- Lua code executed
- Presenter type registered
- Methods available

---

### Test: Tool - register URL path

**Purpose**: Verify path registration via MCP

**Input**:
- tools/call register_url_path with path and presenter

**References**:
- CRC: crc-MCPTool.md - "Does: registerUrlPath"

**Expected Results**:
- Path associated with presenter
- Navigation to path works
- URL reflects path

---

### Test: Tool - activate tab

**Purpose**: Verify tab activation via MCP

**Input**:
- tools/call activate_tab

**References**:
- CRC: crc-MCPTool.md - "Does: activateTab"

**Expected Results**:
- Desktop notification shown
- Main tab focused
- User attention directed

---

### Test: MCP receive user event

**Purpose**: Verify event notification to AI

**Input**:
- User clicks button with ui-action

**References**:
- CRC: crc-MCPServer.md - "Does: sendNotification"
- Sequence: seq-mcp-receive-event.md

**Expected Results**:
- AI notified of event
- Action name included
- Form values included

---

### Test: MCP conversation loop

**Purpose**: Verify two-way interaction flow

**Input**:
- AI creates chat UI
- User sends message
- AI receives and responds

**References**:
- Sequence: seq-mcp-receive-event.md

**Expected Results**:
- User message queued
- AI receives notification
- AI updates UI with response
- User sees response

---

### Test: MCP frictionless UI creation

**Purpose**: Verify on-the-fly UI creation

**Input**:
- AI creates new presenter type (not pre-registered)
- AI creates viewdef
- AI creates presenter instance

**References**:
- CRC: crc-MCPTool.md

**Expected Results**:
- No prior registration needed
- Type created dynamically
- UI displays immediately

---

## Coverage Summary

**Responsibilities Covered:**
- MCPServer: initialize, listResources, listTools, handleResourceRequest, handleToolCall, sendNotification, shutdown
- MCPResource: listPresenterTypes, listViewdefs, getSessionState, getPendingMessages, getPresenterState
- MCPTool: createSession, createPresenter, updatePresenter, destroyPresenter, createViewdef, updateViewdef, loadPresenterLogic, registerUrlPath, activateTab

**Scenarios Covered:**
- seq-mcp-create-session.md: All paths
- seq-mcp-create-presenter.md: All paths
- seq-mcp-receive-event.md: All paths

**Gaps**: None identified
