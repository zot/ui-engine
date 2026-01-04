# Gap Analysis

**Date:** 2026-01-04
**CRC Cards:** 45 | **Sequences:** 40 | **UI Specs:** 2 | **Test Designs:** 7
**Note:** MCP design elements moved to ui-mcp project
**External Package:** `change-tracker` (`github.com/zot/change-tracker`) provides variable tracking, change detection, object registry

## Summary

**Status:** Yellow
**Type A (Critical):** 2 (MCP sequence references)
**Type B (Quality):** 3 (Test coverage, implementation gaps)
**Type C (Enhancements):** 3

---

## Type A Issues (Critical)

### A1: Missing MCP Sequence Diagrams Referenced by CRC Cards

**Issue:** Three CRC cards reference MCP-related sequence diagrams that do not exist in the design directory.

**Required by:** Referenced in CRC cards

**Expected in:**
- `seq-mcp-create-presenter.md` - Referenced by crc-ViewdefStore.md (line 38), crc-LuaPresenterLogic.md (line 32)
- `seq-mcp-create-session.md` - Referenced by crc-SessionManager.md (line 39)

**Impact:** Incomplete design documentation for MCP integration. These sequences were likely planned for the ui-mcp project but the CRC card references were not updated.

**Status:** Open

**Recommendation:** Either:
1. Remove these sequence references from CRC cards (if MCP sequences are in ui-mcp project)
2. Create stub sequence files indicating sequences are documented in ui-mcp project
3. Create full sequence diagrams if MCP integration is part of this project

---

### A2: Widget Class Not Implemented

**Issue:** CRC card `crc-Widget.md` defines a Widget class for binding contexts, but the implementation in `web/src/binding.ts` does not have an explicit Widget class. Instead, bindings are managed directly in maps keyed by elementId.

**Required by:** viewdefs.md - Element References, Widgets section

**Expected in:** `web/src/widget.ts` or integrated into `web/src/binding.ts`

**Current State:** The functionality is partially covered:
- `ensureElementId()` is implemented in `element_id_vendor.ts`
- Bindings are tracked by elementId in `BindingEngine.bindings` map
- But no explicit Widget class with `variables` map for sibling binding lookup

**Impact:**
- `syncValueBinding` feature (Event binding syncs ui-value before event) may not work correctly
- `hasBinding` lookup for sibling bindings not implemented
- Widget-to-variable relationship not properly tracked

**Status:** Open

**Recommendation:** Either:
1. Implement explicit Widget class per CRC design
2. Update CRC card to reflect actual implementation approach
3. Add the `syncValueBinding` behavior to EventBinding using current structure

---

## Type B Issues (Quality)

### B1: Test Code Coverage Needs Improvement

**Issue:** Test coverage continues to improve but still has gaps.

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

### B2: Traceability Checkboxes Have Unchecked Items

**Issue:** Several items in traceability.md are unchecked, indicating implementation gaps or outdated tracking.

**Unchecked Items:**
- `web/src/variable.ts` - namespace/fallbackNamespace property inheritance
- `web/src/view.ts` - 3-tier namespace resolution, elementId usage
- `web/src/viewlist.ts` - exemplar namespace inheritance, elementId usage
- `web/src/app_view.ts` - elementId instead of element
- `web/src/element_id_vendor.ts` - ElementIdVendor implementation (ACTUALLY IMPLEMENTED - checkbox is wrong)
- `web/src/widget.ts` - Widget class (see A2)
- `web/src/binding.ts` - Multiple unchecked items (Widget management, elementId keys, sendVariableUpdate, widget reference to event bindings)
- `web/src/renderer.ts` - 3-tier namespace resolution, script collection/activation, elementId usage

**Status:** In Progress

**Recommendation:**
1. Update traceability.md checkboxes that are actually complete (e.g., element_id_vendor.ts)
2. Prioritize implementing remaining unchecked features
3. Review whether all unchecked items are still required

---

### B3: Some Design Elements Reference Non-Existent Collaborators

**Issue:** Some CRC cards reference collaborators or elements that may be outdated.

**Found Issues:**
- `crc-BindingEngine.md` references `WatchManager` as collaborator (line 204) - WatchManager was merged into LuaBackend
- `test-VariableProtocol.md` references `crc-WatchManager.md` (line 4) - This CRC was deleted

**Impact:** Minor - documentation inconsistency only

**Status:** Open

**Recommendation:** Update references to reflect current architecture (WatchManager functionality is now in LuaBackend)

---

## Type C Issues (Enhancements)

### C1: No Dedicated CRC for Select Views

**Enhancement:** specs/viewdefs.md documents Select Views as a pattern using ViewList with `<sl-select>`, but there is no dedicated CRC card.

**Current:** Select Views are handled generically by ViewList + WidgetBinder

**Better:** This is likely intentional - Select Views are a usage pattern, not a separate component. No action needed, but could add a note in architecture.md if clarification is desired.

**Priority:** Low

---

### C2: No Dedicated CRC for Tabulator Integration

**Enhancement:** specs/components.md and specs/interfaces.md show Tabulator examples for advanced grids, but there is no dedicated CRC card.

**Current:** Handled generically through content binding (ui-tabulator attribute)

**Better:** If Tabulator integration requires special handling, consider a crc-TabulatorBinding.md. Otherwise, document the pattern in a "Widget Patterns" section.

**Priority:** Low

---

### C3: Demo Application Has No Test Design

**Enhancement:** specs/demo.md describes the Contact Manager demo application, but there is no test-Demo.md design file.

**Current:** Demo is documented in specs only

**Better:** Create test-Demo.md with integration test scenarios for the demo application to ensure end-to-end functionality works.

**Priority:** Medium

---

## Coverage Summary

**CRC Responsibilities Coverage:**

| System | CRC Cards | Fully Traced | Notes |
|--------|-----------|--------------|-------|
| Variable Protocol | 5 | 5 | Complete |
| Presenter | 3 | 3 | Complete |
| Viewdef | 11 | 9 | Widget not fully implemented, some namespace features pending |
| Session | 3 | 3 | Complete |
| Backend | 2 | 2 | Complete |
| Communication | 6 | 6 | Complete |
| Backend Socket | 3 | 3 | Complete |
| Lua Runtime | 4 | 4 | Complete |
| Backend Library | 2 | 2 | change-tracker provides core tracking |
| Frontend Library | 4 | 4 | Complete |
| Cross-Cutting | 5 | 4 | ElementIdVendor implemented |

**External Package:** `change-tracker` (`github.com/zot/change-tracker`)
- Provides: Variable management, change detection, object registry, value serialization
- Not counted in CRC totals (external)

**Sequences Coverage:** 40/42 (95%)
- 40 sequence files exist
- 2 MCP sequences referenced but missing (see A1)

**UI Specs Coverage:** 2/2 (100%)
- ui-app-shell.md
- manifest-ui.md

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
- 2 broken sequence references found (MCP sequences)
- 1 broken collaborator reference (WatchManager in BindingEngine)

---

## Source Files with Traceability Comments

**Verified files with proper CRC/Spec references:**

| File | CRC Reference | Spec Reference |
|------|---------------|----------------|
| `internal/backend/backend.go` | crc-Backend.md | main.md |
| `internal/backend/lua.go` | crc-LuaBackend.md | main.md, protocol.md |
| `internal/session/session.go` | crc-Session.md, crc-SessionManager.md | main.md, interfaces.md |
| `internal/lua/runtime.go` | crc-LuaSession.md, crc-LuaVariable.md | interfaces.md, deployment.md, libraries.md |
| `web/src/binding.ts` | crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md | viewdefs.md |
| `web/src/variable.ts` | crc-Variable.md | protocol.md |
| `web/src/connection.ts` | crc-WebSocketEndpoint.md, crc-SharedWorker.md | interfaces.md |
| `web/src/element_id_vendor.ts` | crc-ElementIdVendor.md | viewdefs.md |

---

## Artifact Verification

### Sequence References Valid
- **Status:** FAIL - 2 missing sequences
- `seq-mcp-create-presenter.md` - Referenced but missing
- `seq-mcp-create-session.md` - Referenced but missing

### Complex Behaviors Have Sequences
- **Status:** PASS
- All major workflows have sequence diagrams (40 sequences exist)

### Collaborator Format Valid
- **Status:** WARN - 1 outdated reference
- `crc-BindingEngine.md` references `WatchManager` (deleted)
- All other collaborators reference CRC card names or marked "(External)"

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
- [x] All sequences analyzed (40 exist, 2 missing)
- [x] All source files examined

**Artifact Verification:**
- [ ] Sequence references valid (2 missing MCP sequences)
- [x] Complex behaviors have sequences
- [ ] Collaborators are CRC card names (1 outdated WatchManager ref)
- [x] All CRCs in architecture.md
- [x] All CRCs in traceability.md
- [x] Test designs exist for testable components

**Clarity:**
- [x] Issues have file/line references
- [x] Recommendations actionable
- [x] Impact explained

---

## Recommended Priority Order

1. **A2:** Decide on Widget implementation approach (high impact on binding system)
2. **A1:** Fix MCP sequence references in CRC cards
3. **B2:** Update traceability checkboxes for completed items
4. **B3:** Fix WatchManager reference in BindingEngine
5. **B1:** Implement frontend test infrastructure
6. **C3:** Create test-Demo.md (optional)

## Changes Since Last Analysis (2025-12-30)

**New Issues Found:**
- A1: MCP sequence references (previously not flagged)
- A2: Widget class implementation gap (newly identified)
- B3: WatchManager collaborator reference (newly identified)

**Previously Resolved (Retained):**
- Sequence reference fixes for ProtocolHandler, PacketProtocol
- Backend interface checkbox update
- Wrapper Lua documentation clarification

## Historical: Previously Resolved

- A1 (2025-12-30): Fixed missing sequence references in CRC cards (seq-lua-action-dispatch -> seq-lua-handle-action, seq-packet-protocol-message -> seq-backend-socket-accept)
- B2 (2025-12-30): Updated traceability.md Backend interface checkbox
- B3 (2025-12-30): Clarified Lua wrapper documentation
- ObjectRegistry status - Now documented as external change-tracker package
- WatchManager CRC - Deleted (functionality in LuaBackend + change-tracker)
- crc-ChangeDetector.md - Deleted (provided by change-tracker)
- crc-ObjectRegistry.md - Deleted (provided by change-tracker)
- seq-object-registry.md - Deleted (internal to change-tracker)
