# Gap Analysis

**Date:** 2025-12-23
**CRC Cards:** 46 | **Sequences:** 34 | **UI Specs:** 1

## Type A Issues (Critical)

### A1: Missing Implementation of MCP Lifecycle & Tools
**Issue:** The new MCP lifecycle FSM and tools are specified and designed but not yet implemented.
**Required by:** specs/mcp.md (Sections 3 & 5)
**Expected in:** internal/mcp/server.go, internal/mcp/tools.go
**Impact:** AI agents cannot configure or start the server, nor use the browser integration.
**Status:** Open

## Type B Issues (Quality)

### B1: Architecture.md missing seq-mcp-lifecycle.md
**Issue:** The new sequence diagram `seq-mcp-lifecycle.md` is not listed in `design/architecture.md`.
**Current:** Lists other mcp sequences but not the lifecycle one.
**Location:** design/architecture.md (MCP Integration System)
**Recommendation:** Add `seq-mcp-lifecycle.md` to the list.
**Status:** Open

### B2: Traceability.md references outdated spec
**Issue:** MCP components in `traceability.md` reference `interfaces.md` instead of the new `specs/mcp.md`.
**Current:** "Source Spec: interfaces.md"
**Location:** design/traceability.md (Level 1 <-> Level 2)
**Recommendation:** Create a new section for `specs/mcp.md` or update the references.
**Status:** Open

## Type C Issues (Enhancements)

### C1: SharedWorker Logic for Conserve Mode
**Issue:** The frontend SharedWorker logic for "Conserve Mode" is specified but not detailed in a specific CRC card.
**Current:** Mentioned in `specs/mcp.md` and `crc-MCPTool.md`.
**Better:** Detailed CRC card for the SharedWorker's role in conserve mode, or update `crc-SharedWorker.md` to explicitly include this responsibility.
**Priority:** Medium

## Coverage Summary

**CRC Responsibilities:** High (Design updated)
**Sequences:** High (New sequence created)
**UI Specs:** N/A

**Traceability:**
- ⚠️ MCP components point to old spec in traceability.md

## Summary

**Status:** Yellow
**Type A (Critical):** 1 (Pending Implementation)
**Type B (Quality):** 2 (Documentation updates needed)
**Type C (Enhancements):** 1

----

# Gap Analysis

**Date:** 2025-12-10
**CRC Cards:** 45 | **Sequences:** 35 | **UI Specs:** 1

## Summary

**Status:** Yellow (Implementation Gaps)
**Type A (Critical - Spec Required):** 15
**Type B (Quality Issues):** 8
**Type C (Enhancements):** 6

---

## Type A Issues (Critical - Spec Required Missing)

### A1: Viewdef System Entirely Missing

**Issue:** The Viewdef System (viewdefs, viewdef store, views, viewlists) is not implemented.
**Required by:** viewdefs.md, architecture.md
**Expected in:** crc-Viewdef.md, crc-ViewdefStore.md, crc-View.md, crc-ViewList.md, internal/viewdef/, web/src/
**Impact:** Frontend cannot receive view templates from backend. Dynamic UI rendering is not possible.
**Status:** Open

**Missing components:**
- No `internal/viewdef/` package
- No Viewdef struct implementation
- No ViewdefStore implementation with validation and pending views
- No View class for ui-view elements
- No ViewList class for ui-viewlist elements
- Variable 1 does not hold viewdefs property
- No viewdef format validation (template element check)
- No namespace fallback logic (TYPE.NAMESPACE to TYPE.DEFAULT)

---

### A2: MCP Integration System Not Implemented

**Issue:** Model Context Protocol server for AI assistant integration is not implemented.
**Required by:** interfaces.md, architecture.md
**Expected in:** crc-MCPServer.md, crc-MCPResource.md, crc-MCPTool.md, internal/mcp/
**Impact:** AI assistants cannot create sessions, presenters, or interact with the UI.
**Status:** Open

**Missing components:**
- No `internal/mcp/` package
- No MCP server, resource handlers, or tool implementations
- No MCP-specific sequences implemented

---

### A3: Lua Runtime System Not Implemented

**Issue:** Embedded Lua backend for presentation logic is not implemented.
**Required by:** interfaces.md, deployment.md, architecture.md
**Expected in:** crc-LuaRuntime.md, crc-LuaPresenterLogic.md, internal/lua/
**Impact:** Cannot embed presenter logic in Lua. Server cannot run Lua-based backends.
**Status:** Open

**Missing components:**
- No `internal/lua/` package
- No LuaRuntime implementation
- Config has LuaConfig but it's unused
- seq-load-lua-code.md and seq-lua-handle-action.md not implemented

---

### A4: SQLite Storage Not Implemented

**Issue:** SQLite storage backend is specified but not implemented.
**Required by:** deployment.md, architecture.md
**Expected in:** crc-SQLiteStorage.md, internal/storage/sqlite.go
**Impact:** Cannot persist data to SQLite database.
**Status:** Open

**Missing components:**
- No `internal/storage/sqlite.go`
- Config accepts `--storage sqlite` but would fail at runtime

---

### A5: PostgreSQL Storage Not Implemented

**Issue:** PostgreSQL storage backend is specified but not implemented.
**Required by:** deployment.md, architecture.md
**Expected in:** crc-PostgresStorage.md, internal/storage/postgres.go
**Impact:** Cannot persist data to PostgreSQL database.
**Status:** Open

**Missing components:**
- No `internal/storage/postgres.go`
- Config accepts `--storage postgresql` but would fail at runtime

---

### A6: Router Component Not Implemented

**Issue:** URL routing component for session-scoped paths is not implemented.
**Required by:** interfaces.md, architecture.md
**Expected in:** crc-Router.md, internal/session/router.go or internal/router/
**Impact:** URL path patterns cannot be registered or resolved. seq-navigate-url.md not functional.
**Status:** Open

**Missing components:**
- No Router struct implementation
- SessionManager has partial URL path support but lacks pattern matching
- No `buildUrl`, `parseUrl`, or `isRegisteredPath` methods

---

### A7: MessageRelay Component Not Implemented

**Issue:** Message relay for frontend/backend communication is not fully implemented.
**Required by:** protocol.md, deployment.md, architecture.md
**Expected in:** crc-MessageRelay.md, crc-MessageBatcher.md, internal/server/relay.go or internal/relay/
**Impact:** Messages not properly relayed between frontend and backend. Priority batching not implemented.
**Status:** Open

**Missing components:**
- No dedicated MessageRelay struct
- No MessageBatcher for priority-based batching
- No shouldRelay filtering logic
- No filterForUnbound handling
- No batch message processing (JSON arrays)
- No priority separation (high/medium/low) for viewdef delivery

---

### A8: PendingResponseQueue Not Implemented

**Issue:** Pending response queue for long-polling clients is not implemented.
**Required by:** deployment.md, architecture.md
**Expected in:** crc-PendingResponseQueue.md, internal/server/pending.go
**Impact:** CLI and REST clients cannot receive push notifications. `handlePoll` returns empty.
**Status:** Open

**Missing components:**
- No PendingResponseQueue struct
- `handlePoll` in handler.go returns empty pending list (line 268-274)
- No enqueuePending, drain, or notifyWaiters methods

---

### A9: Backend Library System Not Implemented

**Issue:** Go library for backend integration is not implemented.
**Required by:** libraries.md, architecture.md
**Expected in:** crc-BackendConnection.md, crc-PathNavigator.md, crc-ChangeDetector.md
**Impact:** External Go backends cannot connect and integrate with UI server.
**Status:** Open

**Missing components:**
- No `pkg/` or `lib/` directory for public library
- No BackendConnection implementation for Go clients
- No PathNavigator for path resolution
- No ChangeDetector for change propagation

---

### A10: Frontend Uses Inline Script Instead of TypeScript Library

**Issue:** The embedded site (`site/index.html`) uses an inline JavaScript implementation instead of the TypeScript library in `web/src/`.
**Required by:** libraries.md, architecture.md
**Expected in:** site/index.html should use compiled web/src/* output
**Impact:** Two separate frontend implementations exist. TypeScript library features (SharedWorker, full binding engine) not used.
**Status:** Open

**Current state:**
- `web/src/*.ts` - Full TypeScript implementation with SharedWorker, binding engine
- `site/index.html` - Simple inline JS without SharedWorker or proper bindings
- No build step connecting them

---

### A11: SharedWorker Not Used by Embedded Site

**Issue:** SharedWorker for tab coordination exists in `web/src/worker.ts` but is not used.
**Required by:** interfaces.md, architecture.md
**Expected in:** crc-SharedWorker.md, site integration
**Impact:** Multiple tabs create multiple WebSocket connections instead of sharing one.
**Status:** Open

**Missing:**
- `site/index.html` creates direct WebSocket, not via SharedWorker
- Tab coordination, main tab election not functional

---

### A12: ViewRenderer Not Implemented in Frontend

**Issue:** ViewRenderer component for dynamic view rendering is not implemented.
**Required by:** libraries.md, viewdefs.md, architecture.md
**Expected in:** crc-ViewRenderer.md, crc-View.md, crc-ViewList.md, web/src/renderer.ts or similar
**Impact:** Cannot dynamically render viewdefs, handle ui-viewlist, ui-view, ui-content.
**Status:** Open

**Missing methods from CRC:**
- render(element, variable, namespace) - Returns boolean, handles pending views
- createView (for ui-view elements with unique HTML ids)
- createViewList (for ui-viewlist with exemplar cloning)
- lookupViewdef (TYPE.NAMESPACE with fallback to TYPE.DEFAULT)
- vendHtmlId (unique id generation)
- updateDynamicContent (ui-content HTML updates)
- handleViewChange (re-render on type change)

---

### A13: WidgetBinder Not Implemented

**Issue:** Widget-specific binding component is not implemented.
**Required by:** libraries.md, architecture.md
**Expected in:** crc-WidgetBinder.md, web/src/widgets.ts or similar
**Impact:** Widget-specific bindings (Shoelace components) not properly handled.
**Status:** Open

---

### A14: SPANavigator Not Fully Implemented

**Issue:** SPA navigation component has partial implementation but missing key features.
**Required by:** libraries.md, architecture.md
**Expected in:** crc-SPANavigator.md, web/src/app.ts
**Impact:** History management incomplete. Back/forward navigation not properly tracked.
**Status:** Partial

**Implemented:**
- `navigateTo(url)` in UIApp
- `handleNavigation()` extracts path

**Missing:**
- Integration with AppPresenter history
- Browser back/forward button handling
- popstate event listener

---

### A15: Storage Not Integrated with VariableStore

**Issue:** Storage backend interface exists but is not connected to VariableStore.
**Required by:** data-models.md, architecture.md
**Expected in:** internal/variable/store.go should use storage.Backend
**Impact:** Variables are only in memory. No persistence to storage backend.
**Status:** Open

**Current state:**
- `internal/storage/backend.go` defines Backend interface
- `internal/storage/memory.go` implements MemoryStorage
- `internal/variable/store.go` does not use storage.Backend
- Variables stored in in-memory map only

---

## Type B Issues (Design/Quality)

### B1: Protocol Handler Missing Relay Logic

**Issue:** ProtocolHandler should relay messages between frontend and backend but lacks this logic.
**Current:** All messages handled locally, no forwarding
**Location:** `/home/deck/work/ui/internal/protocol/handler.go`
**Recommendation:** Add relay logic per crc-ProtocolHandler.md responsibilities
**Status:** Open

**Missing from CRC "Does":**
- relayMessage
- parsePropertyPriority
- processPropertiesByPriority

---

### B2: CLI Protocol Commands Not Connected

**Issue:** CLI commands build messages but cannot send them to server.
**Current:** `sendToServer()` prints "not implemented" (cmd/ui/main.go:362-368)
**Location:** `/home/deck/work/ui/cmd/ui/main.go`
**Recommendation:** Connect to backend socket and send messages
**Status:** Open

---

### B3: HTTP Endpoint for Backend Socket Incomplete

**Issue:** HTTP-over-socket returns 501 Not Implemented.
**Current:** Returns "HTTP/1.1 501 Not Implemented" (backend_socket.go:144)
**Location:** `/home/deck/work/ui/internal/server/backend_socket.go`
**Recommendation:** Implement HTTP parsing or use net/http with custom listener
**Status:** Open

---

### B4: Variable Store Missing Storage Backend Reference

**Issue:** VariableStore CRC specifies `storageBackend` but implementation doesn't have it.
**Current:** Store only has in-memory map
**Location:** `/home/deck/work/ui/internal/variable/store.go`
**Recommendation:** Add storageBackend field and persistence calls
**Status:** Open

---

### B5: WatchManager Recursive Lock Issue

**Issue:** IsInactive has potential deadlock with recursive RLock.
**Current:** Releases lock, recurses, re-acquires (watch.go:160-165)
**Location:** `/home/deck/work/ui/internal/variable/watch.go` lines 160-165
**Recommendation:** Refactor to avoid lock release/reacquire pattern
**Status:** Open

---

### B6: Frontend Connection Missing Request/Response Correlation

**Issue:** sendRequest creates pending request but response handling incomplete.
**Current:** pendingRequests map populated but never resolved
**Location:** `/home/deck/work/ui/web/src/connection.ts` lines 91-97
**Recommendation:** Add request ID to messages and correlation logic
**Status:** Open

---

### B7: Binding Engine Missing Path Cache

**Issue:** BindingEngine CRC specifies pathCache for performance but not implemented.
**Current:** parsePath called on every binding evaluation
**Location:** `/home/deck/work/ui/web/src/binding.ts`
**Recommendation:** Add pathCache as Map<string, ParsedPath>
**Status:** Open

---

### B8: Server Missing Graceful Startup Sequence

**Issue:** seq-server-startup.md defines startup sequence but implementation is basic.
**Current:** Simple sequential startup
**Location:** `/home/deck/work/ui/internal/server/server.go`
**Recommendation:** Add proper initialization order, health checks
**Status:** Open

---

## Type C Issues (Enhancements)

### C1: No Test Files

**Enhancement:** No test files exist in the codebase.
**Current:** No `*_test.go` files in internal/
**Better:** Add unit tests for Variable, VariableStore, WatchManager, ProtocolHandler
**Priority:** High

---

### C2: No Logging Framework

**Enhancement:** Config has LoggingConfig but no structured logging implemented.
**Current:** Uses standard `log.Printf`
**Better:** Integrate structured logging (zerolog, zap) with levels
**Priority:** Medium

---

### C3: Windows Named Pipe Support Incomplete

**Enhancement:** BackendSocket falls back to TCP on Windows.
**Current:** Comment notes "For full Windows support, would need npipe package"
**Better:** Add proper Windows named pipe support
**Priority:** Low

---

### C4: No Metrics/Observability

**Enhancement:** No metrics collection or health endpoints.
**Current:** Basic logging only
**Better:** Add Prometheus metrics, health check endpoint
**Priority:** Medium

---

### C5: No Configuration Validation

**Enhancement:** Config loaded but not validated.
**Current:** Invalid storage type silently accepted
**Better:** Add validation with meaningful error messages
**Priority:** Medium

---

### C6: Frontend Has No Error Boundary

**Enhancement:** Frontend errors may crash entire app.
**Current:** Try/catch in message parsing only
**Better:** Add global error handling, recovery UI
**Priority:** Low

---

## Coverage Summary

### CRC Card Coverage

| System | CRC Cards | Implemented | Coverage |
|--------|-----------|-------------|----------|
| Variable Protocol | 4 | 4 | 100% |
| Presenter | 3 | 1 (partial) | 33% |
| Viewdef | 7 | 0 | 0% |
| Session | 3 | 2 | 67% |
| Communication | 5 | 2 (partial) | 40% |
| Backend Socket | 4 | 3 (partial) | 75% |
| Storage | 4 | 2 | 50% |
| MCP Integration | 3 | 0 | 0% |
| Lua Runtime | 2 | 0 | 0% |
| Backend Library | 3 | 0 | 0% |
| Frontend Library | 4 | 2 (partial) | 50% |
| Cross-Cutting | 3 | 1 | 33% |
| **Total** | **45** | **17** | **38%** |

### Sequence Coverage

| Sequence | Implemented |
|----------|-------------|
| seq-create-variable.md | Yes |
| seq-update-variable.md | Yes |
| seq-watch-variable.md | Yes |
| seq-destroy-variable.md | Yes |
| seq-create-session.md | Yes |
| seq-frontend-connect.md | Partial |
| seq-backend-connect.md | Partial |
| seq-frontend-reconnect.md | Partial |
| seq-server-startup.md | Partial |
| seq-bootstrap.md | Partial |
| All MCP sequences | No |
| All Lua sequences | No |
| seq-load-viewdefs.md | No |
| seq-viewdef-delivery.md | No |
| seq-render-view.md | No |
| seq-viewlist-update.md | No |
| seq-bind-element.md | No |
| seq-handle-event.md | No |
| seq-relay-message.md | No |
| seq-poll-pending.md | No |
| seq-store-variable.md | No |
| seq-retrieve-variable.md | No |
| **Total** | **9/35 (26%)** |

### Traceability

- All implemented files have CRC card references
- All implemented files have Spec references
- Architecture.md systems partially covered

---

## Recommended Implementation Order

1. **Storage Integration** (A15, B4) - Connect VariableStore to StorageBackend
2. **PendingResponseQueue** (A8) - Enable CLI/REST push notifications
3. **SQLite Storage** (A4) - Most common persistence option
4. **MessageRelay** (A7) - Enable frontend/backend communication
5. **Frontend Build Integration** (A10, A11) - Use TypeScript library
6. **Viewdef System** (A1, A12) - Enable dynamic UI
7. **Router** (A6) - URL-based navigation
8. **Lua Runtime** (A3) - Embedded backend logic
9. **MCP Integration** (A2) - AI assistant support
10. **PostgreSQL Storage** (A5) - Production database option

---

## Design Artifact Verification

### Sequence References Valid
- **Status:** PASS
- All seq-*.md files referenced in CRC cards exist

### Complex Behaviors Have Sequences
- **Status:** PASS
- All non-trivial "Does" items have sequence diagrams

### Collaborator Format Valid
- **Status:** PASS
- All collaborators reference CRC card names or note "(External)"

### Architecture Updated
- **Status:** PASS
- All CRC cards listed in architecture.md
- All sequences listed in architecture.md

---

*This analysis compares Level 2 design specs against Level 3 implementation to identify gaps.*
