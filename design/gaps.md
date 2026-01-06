# Gap Analysis

**Date:** 2026-01-05
**CRC Cards:** 45 | **Sequences:** 40 | **UI Specs:** 2 | **Test Designs:** 7
**Note:** MCP design elements moved to ui-mcp project
**External Package:** `change-tracker` (`github.com/zot/change-tracker`) provides variable tracking, change detection, object registry

## Summary

**Status:** Green
**Type A (Critical):** 0
**Type B (Quality):** 1 (Test coverage)
**Type C (Enhancements):** 3

---

## Type A Issues (Critical)

**None.** All critical issues from the previous analysis have been resolved or clarified:

### Resolved: A1 (MCP Sequence References)

**Previous Issue:** Three CRC cards referenced MCP-related sequence diagrams that did not exist.

**Resolution:** MCP sequence references removed from CRC cards. MCP functionality moved to separate ui-mcp project.

**Files cleaned:**
- `crc-ViewdefStore.md` - seq-mcp-create-presenter.md reference removed
- `crc-LuaPresenterLogic.md` - seq-mcp-create-presenter.md reference removed
- `crc-SessionManager.md` - seq-mcp-create-session.md reference removed

**Status:** Resolved

---

### Resolved: A2 (Widget Class Implementation)

**Previous Issue:** CRC card `crc-Widget.md` defines a Widget class, but implementation status was unclear.

**Resolution:** The Widget class IS implemented in `/home/deck/work/ui-engine/web/src/binding.ts` (lines 100-142). The implementation includes:
- `elementId` property (from global vendor via `ensureElementId`)
- `variables` Map for binding name to variable ID
- `unbindHandlers` Map for cleanup functions
- `registerBinding()`, `getVariableId()`, `hasBinding()`, `getElement()`, `unbindAll()` methods

Additionally, `syncValueBeforeEvent()` is implemented (lines 774-790) for event binding value synchronization.

**Status:** Resolved - Implementation exists and matches CRC design

---

## Type B Issues (Quality)

### B1: Test Code Coverage Needs Improvement

**Issue:** Test coverage has improved but frontend TypeScript tests are still missing.

**Current Test Files:**
- `internal/lua/runtime_test.go` - Lua runtime tests
- `internal/lua/viewlist_test.go` - ViewList tests
- `internal/session/session_test.go` - Session management tests
- `internal/protocol/protocol_test.go` - Protocol/batching tests
- `internal/server/http_test.go` - HTTP endpoint tests

**Test Designs Coverage:**
- [x] test-Session.md - Backend tests exist
- [x] test-Communication.md - Partially covered (HTTP, batching)
- [x] test-Lua.md - Partially covered
- [ ] test-VariableProtocol.md - Partially covered (protocol messages)
- [ ] test-Frontend.md - Not covered (TypeScript/browser tests needed)
- [ ] test-Viewdef.md - Not covered (TypeScript/browser tests needed)
- [ ] test-Backend.md - Path navigation tests needed

**Status:** In Progress

**Recommendation:**
1. Add TypeScript test infrastructure for frontend tests
2. Implement browser-based tests for viewdef binding system
3. Add path navigation tests for backend

---

## Type C Issues (Enhancements)

### C1: No Dedicated CRC for Select Views

**Enhancement:** specs/viewdefs.md documents Select Views as a pattern using ViewList with `<sl-select>`, but there is no dedicated CRC card.

**Current:** Select Views are handled generically by ViewList + WidgetBinder

**Better:** This is intentional - Select Views are a usage pattern, not a separate component. No action needed.

**Priority:** Low

---

### C2: No Dedicated CRC for Tabulator Integration

**Enhancement:** specs/components.md and specs/interfaces.md show Tabulator examples for advanced grids, but there is no dedicated CRC card.

**Current:** Handled generically through content binding (ui-tabulator attribute)

**Better:** If Tabulator integration requires special handling, consider a crc-TabulatorBinding.md. Otherwise, document the pattern in a "Widget Patterns" section.

**Priority:** Low

---

### C3: Test Design Missing Keypress Modifier Tests

**Enhancement:** test-Viewdef.md has keypress binding tests but does not include tests for modifier combinations (Ctrl+Enter, Ctrl+Shift+S, etc.).

**Current:** Basic keypress tests exist (enter, escape, arrow keys, letters)

**Better:** Add test cases for:
- `ui-event-keypress-ctrl-enter` - Ctrl+Enter combination
- `ui-event-keypress-ctrl-shift-s` - Multi-modifier combination
- Modifier exact matching (Ctrl+S should not fire when Ctrl+Shift+S is pressed)

**Priority:** Medium

---

## Coverage Summary

**CRC Responsibilities Coverage:**

| System | CRC Cards | Fully Traced | Notes |
|--------|-----------|--------------|-------|
| Variable Protocol | 5 | 5 | Complete |
| Presenter | 3 | 3 | Complete |
| Viewdef | 11 | 11 | Complete |
| Session | 3 | 3 | Complete |
| Backend | 2 | 2 | Complete |
| Communication | 6 | 6 | Complete |
| Backend Socket | 3 | 3 | Complete |
| Lua Runtime | 4 | 4 | Complete |
| Backend Library | 2 | 2 | Complete |
| Frontend Library | 4 | 4 | Complete |
| Cross-Cutting | 5 | 5 | Complete |

**External Package:** `change-tracker` (`github.com/zot/change-tracker`)
- Provides: Variable management, change detection, object registry, value serialization
- Not counted in CRC totals (external)

**Sequences Coverage:** 40/40 (100%)

**UI Specs Coverage:** 2/2 (100%)

**Test Design Coverage:**
| Test Design | Backend Tests | Frontend Tests | Coverage |
|-------------|---------------|----------------|----------|
| test-Session.md | Yes | N/A | Good |
| test-Communication.md | Partial | No | Partial |
| test-Lua.md | Partial | N/A | Partial |
| test-VariableProtocol.md | Partial | No | Partial |
| test-Frontend.md | N/A | No | Missing |
| test-Viewdef.md | N/A | No | Missing |
| test-Backend.md | Partial | N/A | Partial |

**Traceability:**
- All 10 spec files have corresponding design elements
- All CRC cards reference source specs

---

## Source Files with Traceability Comments

| File | CRC Reference | Spec Reference |
|------|---------------|----------------|
| `internal/backend/backend.go` | crc-Backend.md | main.md |
| `internal/backend/lua.go` | crc-LuaBackend.md | main.md, protocol.md |
| `internal/session/session.go` | crc-Session.md, crc-SessionManager.md | main.md, interfaces.md |
| `internal/lua/runtime.go` | crc-LuaSession.md, crc-LuaVariable.md | interfaces.md, deployment.md, libraries.md |
| `web/src/binding.ts` | crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md, crc-Widget.md | viewdefs.md |
| `web/src/variable.ts` | crc-Variable.md | protocol.md |
| `web/src/connection.ts` | crc-WebSocketEndpoint.md, crc-SharedWorker.md | interfaces.md |
| `web/src/element_id_vendor.ts` | crc-ElementIdVendor.md | viewdefs.md |

---

## Artifact Verification

### Sequence References Valid
- **Status:** PASS

### Complex Behaviors Have Sequences
- **Status:** PASS
- All major workflows have sequence diagrams (40 sequences exist)

### Collaborator Format Valid
- **Status:** PASS - All collaborators are valid CRC card names

### Architecture Updated
- **Status:** PASS
- All CRC cards appear in architecture.md systems

### Traceability Updated
- **Status:** PASS
- All CRC cards have entries in traceability.md
- External package (change-tracker) documented

### Test Designs Exist
- **Status:** PASS
- 7 test design files cover major systems

---

## Quality Checklist

**Completeness:**
- [x] All CRC cards analyzed (45)
- [x] All sequences analyzed (40 exist)
- [x] All source files examined

**Artifact Verification:**
- [x] Sequence references valid
- [x] Complex behaviors have sequences
- [x] Collaborators are CRC card names
- [x] All CRCs in architecture.md
- [x] All CRCs in traceability.md
- [x] Test designs exist for testable components

**Clarity:**
- [x] Issues have file/line references
- [x] Recommendations actionable
- [x] Impact explained

---

## Recommended Priority Order

1. **C3:** Add keypress modifier test cases to test-Viewdef.md
2. **B1:** Implement frontend test infrastructure (ongoing)
