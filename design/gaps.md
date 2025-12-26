# Gap Analysis

**Date:** 2025-12-26
**CRC Cards:** 42 | **Sequences:** 35 | **UI Specs:** 2 | **Test Designs:** 7
**Note:** MCP design elements moved to ui-mcp project
**External Package:** `change-tracker` (`github.com/zot/change-tracker`) provides variable tracking, change detection, object registry

## Summary

**Status:** Yellow
**Type A (Critical):** 1
**Type B (Quality):** 4
**Type C (Enhancements):** 3

---

## Type A Issues (Critical)

### A1: Missing Sequence Diagrams Referenced by CRC Cards

**Issue:** Two CRC cards reference sequence diagrams that do not exist in the design directory.

**Required by:** Referenced in CRC cards

**Expected in:**
- `seq-lua-action-dispatch.md` - Referenced by crc-ProtocolHandler.md (line 51)
- `seq-packet-protocol-message.md` - Referenced by crc-PacketProtocol.md (line 28)

**Impact:** Incomplete design documentation. Developers implementing these behaviors lack the step-by-step flow documentation. This could lead to inconsistent implementations.

**Status:** Open

**Recommendation:** Either create the missing sequence diagrams or update the CRC cards to reference existing sequences (e.g., `seq-lua-action-dispatch.md` could reference `seq-lua-handle-action.md`).

---

## Type B Issues (Quality)

### B1: Test Code Coverage Incomplete

**Issue:** Only 2 test files exist in the codebase, despite 8 test design documents being present.

**Current:**
- `internal/lua/runtime_test.go`
- `internal/lua/viewlist_test.go`

**Test Designs Available:**
- test-VariableProtocol.md
- test-Session.md
- test-Lua.md (partially covered by existing tests)
- test-Frontend.md
- test-Viewdef.md
- test-Communication.md
- test-Backend.md

**Location:** Tests should be in corresponding `*_test.go` files

**Recommendation:** Implement tests for all test design scenarios, starting with high-priority systems (Variable Protocol, Backend, Session).

**Status:** Open

---

### B2: Backend Interface Not Implemented

**Issue:** traceability.md shows `crc-Backend.md` implementation as unchecked, but this is the Backend interface that LuaBackend implements.

**Current (traceability.md lines 217-221):**
```markdown
### crc-Backend.md
**Source Spec:** main.md (UI Server Architecture)
**Implementation:**
- [ ] `internal/backend/backend.go` - Backend interface
```

**Location:** `internal/backend/backend.go` exists but contains interface definition

**Recommendation:** Review if this should be checked (interface is defined) or if there's additional work needed. Interface definitions are typically "implemented" when defined.

**Status:** Open

---

### B3: Wrapper Lua Implementation Marked Optional but Incomplete

**Issue:** traceability.md shows `lib/wrapper.lua` as optional/incomplete for crc-Wrapper.md, but Lua wrapper types are documented in libraries.md.

**Current (traceability.md lines 453-458):**
```markdown
### crc-Wrapper.md
**Implementation:**
- [x] `internal/lua/wrapper.go` - Wrapper interface and registry
- [x] `internal/lua/viewlist.go` - ViewList wrapper implementation
- [ ] `lib/wrapper.lua` - Lua wrapper base (optional - Go implementation complete)
```

**Location:** libs.md documents Lua wrapper conventions

**Recommendation:** Clarify if Lua-side wrapper base class is truly optional or if the Go implementation handles all Lua wrapper creation. If optional, mark as N/A rather than unchecked.

**Status:** Open

---

### B4: Lua Change Detection File Marked for Removal

**Issue:** `lib/lua/change.lua` is marked as "to be removed" in traceability.md but still exists.

**Current (traceability.md lines 395-397):**
```markdown
- [x] `lib/lua/change.lua` - Lua change detection (to be removed - superseded by Go change-tracker)
```

**Also in Removed Design Elements (lines 468-469):**
```markdown
- [ ] `lib/lua/change.lua` - Lua change detection (remove - superseded by Go change-tracker)
```

**Recommendation:** If change-tracker has fully superseded this, remove the file. If still needed for compatibility, update the documentation.

**Status:** Open

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
| Viewdef | 9 | 9 | Complete |
| Session | 3 | 3 | Complete |
| Backend | 2 | 1 | Backend interface unchecked |
| Communication | 5 | 5 | Complete |
| Backend Socket | 3 | 3 | Complete |
| Lua Runtime | 4 | 4 | Complete |
| Backend Library | 2 | 2 | change-tracker provides core tracking |
| Frontend Library | 4 | 4 | Complete |
| Cross-Cutting | 5 | 5 | Complete |

**External Package:** `change-tracker` (`github.com/zot/change-tracker`)
- Provides: Variable management, change detection, object registry, value serialization
- Not counted in CRC totals (external)

**Sequences Coverage:** 35/37 (95%)
- Missing: seq-lua-action-dispatch.md, seq-packet-protocol-message.md
- Removed: seq-object-registry.md (internal to change-tracker)
- Moved to ui-mcp: 9 MCP sequences

**UI Specs Coverage:** 2/2 (100%)
- ui-app-shell.md
- manifest-ui.md

**Test Implementation Coverage:** ~10%
- 2 test files exist vs 7 test designs
- Focus areas needed: Variable Protocol, Backend, Session, Communication

**Traceability:**
- All 10 spec files have corresponding design elements
- All CRC cards reference source specs
- 2 broken sequence references found (see A1)

---

## Artifact Verification

### Sequence References Valid
- **Status:** FAIL
- **Issues:** 2 CRC cards reference non-existent sequences
  - crc-ProtocolHandler.md -> seq-lua-action-dispatch.md (MISSING)
  - crc-PacketProtocol.md -> seq-packet-protocol-message.md (MISSING)

### Complex Behaviors Have Sequences
- **Status:** PASS
- All major workflows have sequence diagrams

### Collaborator Format Valid
- **Status:** PASS
- All collaborators reference CRC card names or marked "(External)"

### Architecture Updated
- **Status:** PASS
- All CRC cards appear in architecture.md systems

### Traceability Updated
- **Status:** PASS
- All CRC cards have entries in traceability.md
- External package (change-tracker) documented

### Test Designs Exist
- **Status:** PASS
- 8 test design files cover major systems

---

## Quality Checklist

**Completeness:**
- [x] All CRC cards analyzed (49)
- [x] All sequences analyzed (45 existing, 2 missing)
- [x] All source files examined

**Artifact Verification:**
- [ ] Sequence references valid (2 missing)
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

1. **A1:** Create missing sequence diagrams or fix references
2. **B1:** Implement test code for test designs (incremental)
3. **B2:** Verify Backend interface implementation status
4. **B3/B4:** Clean up Lua wrapper and change detection documentation
5. **C3:** Create test-Demo.md (optional)

## Recently Resolved

- ~~A2: ObjectRegistry status~~ - Now documented as external change-tracker package
- ~~B2 (old): WatchManager CRC~~ - Deleted (functionality in LuaBackend + change-tracker)
- ~~crc-ChangeDetector.md~~ - Deleted (provided by change-tracker)
- ~~crc-ObjectRegistry.md~~ - Deleted (provided by change-tracker)
- ~~seq-object-registry.md~~ - Deleted (internal to change-tracker)
