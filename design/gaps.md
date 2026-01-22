# Gap Analysis

**Date:** 2026-01-19
**CRC Cards:** 46 | **Sequences:** 40 | **UI Specs:** 2 | **Test Designs:** 7
**Note:** MCP design elements moved to frictionless project
**External Package:** `change-tracker` (`github.com/zot/change-tracker`) provides variable tracking, change detection, object registry

## Summary

**Status:** Yellow
**Type A (Critical):** 2 (New hot-loading API missing from design and implementation)
**Type B (Quality):** 1 (Test coverage)
**Type C (Enhancements):** 3

---

## Type A Issues (Critical)

### A1: Prototype Management API Not Implemented

**Issue:** `specs/libraries.md` specifies a prototype management API for hot-loading support (`session:prototype()` and `session:create()`), but this API is NOT implemented in the Go backend or the Lua session module.

**Required by:** specs/libraries.md (Prototype Management section, lines 69-135)

**Expected in:**
- `internal/lua/runtime.go` - Go implementation of `session:prototype()` and `session:create()`
- `lib/lua/session.lua` - Lua-side support for prototype/instance tracking

**Current State:**
- `/home/deck/work/ui-engine/internal/lua/runtime.go` has NO implementation of:
  - `session:prototype(name, init)` - Prototype declaration/update
  - `session:create(prototype, instance)` - Instance creation with tracking
  - Post-load mutation processing
  - Instance tracking with weak references
  - EMPTY global for optional field declaration
- `/home/deck/work/ui-engine/lib/lua/session.lua` has NO reference to prototype or instance tracking

**What IS Implemented (partial, version-based approach):**
- `runtime.go` lines 613-656: `newVersion`, `getVersion`, `needsMutation` methods
- These are low-level primitives for manual mutation checking
- They do NOT implement the automatic prototype/instance tracking specified in the spec

**Spec Requirements NOT Met:**
1. `session:prototype(name, init)` behavior (lines 109-119):
   - Create prototype if global is nil
   - Add default `new` method
   - Store shallow copy of init for change detection
   - On reload: compare inits, queue for mutation
   - Support `EMPTY` marker for optional fields

2. `session:create(prototype, instance)` behavior (lines 122-126):
   - Set metatable to prototype
   - Register instance with weak reference for tracking
   - Return the instance

3. Post-load mutation processing (lines 128-135):
   - Iterate queued prototypes
   - Call `prototype:mutate(instance)` if method exists
   - Nil out removed fields
   - Clear mutation queue

**Impact:** Hot-loading schema migrations cannot work automatically. Developers must manually track instances and call mutation logic, which is error-prone and defeats the purpose of the frictionless API.

**Status:** Open

---

### A2: Hot-Loading Design Documents Outdated

**Issue:** Design documents reference OLD hot-loading conventions that differ from the new spec.

**Affected Files:**
- `/home/deck/work/ui-engine/design/seq-lua-hotload.md` (line 82): Still references old pattern `MyApp = MyApp or {type = "MyApp"}`
- `/home/deck/work/ui-engine/design/crc-LuaSession.md`: Does NOT include `session:prototype()` or `session:create()` methods

**Required by:** specs/libraries.md, HOT-LOADING.md

**Expected in:**
- `design/crc-LuaSession.md` - Add prototype/create methods to "Does" section
- `design/seq-lua-hotload.md` - Update to show prototype-based hot-loading flow
- Potentially new sequence: `seq-prototype-mutation.md` for mutation processing

**Current State:**
- `crc-LuaSession.md` "Does" section lists:
  - `newVersion`, `getVersion`, `needsMutation` (low-level primitives)
  - Missing: `prototype`, `create`, instance tracking

**Gap between HOT-LOADING.md and Design:**
- `HOT-LOADING.md` (Level 1 spec) fully documents the API
- `design/crc-LuaSession.md` (Level 2) does NOT match
- `internal/lua/runtime.go` (Level 3) does NOT implement

**Impact:** Three-tier documentation is inconsistent. Level 2 design does not bridge spec to implementation.

**Status:** Open

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

**Additional Gap:** No tests for prototype management API (when implemented)

**Status:** In Progress

**Recommendation:**
1. Add TypeScript test infrastructure for frontend tests
2. Implement browser-based tests for viewdef binding system
3. Add path navigation tests for backend
4. Add tests for prototype/instance tracking when implemented

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
| Viewdef | 12 | 12 | Complete (added HtmlBinding) |
| Session | 3 | 3 | Complete |
| Backend | 2 | 2 | Complete |
| Communication | 6 | 6 | Complete |
| Backend Socket | 3 | 3 | Complete |
| Lua Runtime | 4 | 4 | **Missing: prototype/create API** |
| Backend Library | 2 | 2 | Complete |
| Frontend Library | 4 | 4 | Complete |
| Cross-Cutting | 5 | 5 | Complete |

**External Package:** `change-tracker` (`github.com/zot/change-tracker`)
- Provides: Variable management, change detection, object registry, value serialization
- Not counted in CRC totals (external)

**Sequences Coverage:** 40/40 (100%)
- **Missing:** seq-prototype-mutation.md for post-load processing

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
- **Gap:** libraries.md prototype API not traced to design/implementation

---

## Spec-Design-Implementation Gap Details

### libraries.md Prototype Management

**Spec (Level 1):** `/home/deck/work/ui-engine/specs/libraries.md` lines 69-135
```lua
-- Declare a prototype
Person = session:prototype("Person", {
    name = "",
    email = "",
    avatar = EMPTY,
})

-- Create instance
instance = session:create(Person, instance)
```

**Design (Level 2):** `/home/deck/work/ui-engine/design/crc-LuaSession.md`
- **Status:** MISSING - Does not document prototype/create methods
- **Has:** newVersion, getVersion, needsMutation (low-level primitives only)

**Implementation (Level 3):** `/home/deck/work/ui-engine/internal/lua/runtime.go`
- **Status:** MISSING - Only has version-based primitives
- **Has:** lines 613-656 (newVersion, getVersion, needsMutation)
- **Missing:** prototype(), create(), instance tracking, EMPTY global, mutation queue

**Reference Document:** `/home/deck/work/ui-engine/HOT-LOADING.md`
- Full pseudocode for prototype/create behavior
- Describes instance tracking with weak references
- Describes post-load mutation processing

---

## Artifact Verification

### Sequence References Valid
- **Status:** PASS

### Complex Behaviors Have Sequences
- **Status:** PARTIAL
- Missing: seq-prototype-mutation.md for post-load mutation processing
- All other major workflows have sequence diagrams

### Collaborator Format Valid
- **Status:** PASS - All collaborators are valid CRC card names

### Architecture Updated
- **Status:** PASS
- All CRC cards appear in architecture.md systems

### Traceability Updated
- **Status:** PARTIAL
- All existing CRC cards have entries in traceability.md
- Missing: prototype management API not traced

### Test Designs Exist
- **Status:** PARTIAL
- 7 test design files cover major systems
- Missing: prototype management test scenarios

---

## Quality Checklist

**Completeness:**
- [x] All CRC cards analyzed (45)
- [x] All sequences analyzed (40 exist)
- [x] All source files examined
- [ ] New spec features traced (prototype API missing)

**Artifact Verification:**
- [x] Sequence references valid
- [ ] Complex behaviors have sequences (prototype mutation missing)
- [x] Collaborators are CRC card names
- [x] All CRCs in architecture.md
- [ ] All CRCs in traceability.md (prototype API missing)
- [x] Test designs exist for testable components

**Clarity:**
- [x] Issues have file/line references
- [x] Recommendations actionable
- [x] Impact explained

---

## Recommended Priority Order

1. **A1:** Implement `session:prototype()` and `session:create()` in runtime.go
2. **A2:** Update design docs (crc-LuaSession.md, seq-lua-hotload.md)
3. **C3:** Add keypress modifier test cases to test-Viewdef.md
4. **B1:** Implement frontend test infrastructure (ongoing)
