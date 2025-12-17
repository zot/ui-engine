# Phase 0 Findings: Design Review for ViewList and ViewListItem

## 1. Introduction

This document summarizes the findings from the design review phase (Phase 0) of the `ViewList` and `ViewListItem` refactoring plan. The review was conducted by analyzing `design/crc-ViewList.md`, `design/crc-ViewListItem.md`, and `design/seq-viewlist-presenter-sync.md`.

The primary goal of this review was to ensure the refactoring plan aligns with the specified design before any code modifications are made.

## 2. Key Findings

### 2.1. Naming Convention

*   **Finding:** The design documents consistently use the names `ViewList` and `ViewListItem`. The current code uses `ViewListWrapper` and `ViewItem`.
*   **Impact:** This confirms that **Phase 1** of the plan, which involves renaming these components, is correct and necessary.

### 2.2. `ViewListItem.item` as a Go Object

*   **Finding:** The `crc-ViewListItem.md` is explicit that the `item` field should be the "actual backend object from the array (taken from variable's Value, not ValueJSON)".
*   **Impact:** This is a major discrepancy with the current implementation, which uses `json.RawMessage`. The plan to change the `Item` field to `interface{}` to hold a direct Go object pointer is **correct and critical** for design alignment.

### 2.3. Absence of `BaseItem`

*   **Finding:** The design documents do not mention a `BaseItem` field in `ViewListItem`. The `ViewListItem` is designed to be a simple wrapper with `item`, `list`, and `index`.
*   **Impact:** The current implementation's `BaseItem` field is a deviation from the design. The plan to **remove the `BaseItem` field is correct**.

### 2.4. `ViewList.sync` Logic

*   **Finding:** The design documents provide a clear, three-step algorithm for the `sync` method: 1) update existing items, 2) trim excess items, and 3) add new items.
*   **Impact:** The Go implementation of `syncViewItems` must adhere to this logic. While the current code follows a similar pattern, the refactoring must ensure the new implementation correctly follows the design.

### 2.5. Object ID Marshalling Contradiction

*   **Finding:** There is a minor contradiction between the CRC and sequence diagrams. `crc-ViewListItem.md` states `item` is the "actual backend object", while a note in `seq-viewlist-presenter-sync.md` describes it as a JSON reference (`{obj: ID}`).
*   **Impact:** This is interpreted as a distinction between internal representation (Go object) and communication format (JSON reference). The plan's approach to have `ViewListItem` hold the Go object internally and use the change tracker's `ToValueJSON` for frontend communication is a valid strategy that resolves this contradiction.

### 2.6. Key for `viewItems` Map

*   **Finding:** The design does not specify the implementation detail of the key for the `viewItems` map within `ViewList`.
*   **Impact:** The plan's proposal to use the object's pointer address (`uintptr`) as the key is a reasonable implementation choice that does not conflict with the design.

## 3. Conclusion

The design review confirms that the refactoring outlined in `PLAN.md` is necessary and correctly interprets the design specifications. The existing code has diverged significantly from the design, and the planned changes will bring it back into alignment.

The plan to rename the components, change `ViewListItem.Item` to `interface{}`, remove `BaseItem`, and adjust the surrounding logic is fully supported by the design documents.
