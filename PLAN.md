# Plan to Update ViewList and ViewListItem

## Introduction

This plan outlines the steps to refactor the `ViewItem` and `ViewListWrapper` code in `internal/lua/` to align with the design. The primary goals are:
1.  Rename `ViewItem` to `ViewListItem`.
2.  Rename `ViewListWrapper` to `ViewList`.
3.  The `ViewList` will create and hold a collection of `ViewListItem`s.
4.  The `ViewListItem` will be a simple wrapper around a Go object (`interface{}`), and will no longer have a `BaseItem`.
5.  Ensure the implementation adheres to the CRC cards and sequence diagrams in the `design/` directory.

## Phase 0: Design Review

This phase has been completed and the findings are documented in `PHASE0.md`. The plan has been updated based on those findings and subsequent user feedback.

## Phase 1: Rename Core Components

1.  **Rename `ViewItem` to `ViewListItem`:**
    *   Rename the file `internal/lua/viewitem.go` to `internal/lua/viewlistitem.go`.
    *   In the newly renamed file, change the `ViewItem` struct to `ViewListItem`.
    *   Globally replace all instances of `ViewItem` with `ViewListItem` and `NewViewItem` with `NewViewListItem` in the `internal/lua/` directory.

2.  **Rename `ViewListWrapper` to `ViewList`:**
    *   Rename the file `internal/lua/viewlist_wrapper.go` to `internal/lua/viewlist.go`.
    *   In the newly renamed file, change the `ViewListWrapper` struct to `ViewList`.
    *   Globally replace all instances of `ViewListWrapper` with `ViewList` and `NewViewListWrapper` with `NewViewList`.

## Phase 2: Update `ViewList` and `ViewListItem` Structs

1.  **Update `ViewListItem` Struct:**
    *   The struct will contain:
        *   `Item` of type `interface{}` (the domain object).
        *   `List` of type `*ViewList`.
        *   `Index` of type `int`.
    *   The `BaseItem` field will be removed.
    *   The constructor will be `NewViewListItem(item interface{}, list *ViewList, index int)`.

2.  **Update `ViewList` Struct (to match `crc-ViewList.md` and frontend access requirements):**
    *   The struct will contain:
        *   `variable` of type `WrapperVariable`.
        *   `value` of type `[]interface{}`.
        *   `Items` of type `[]*ViewListItem` (this is called `items` in the CRC card).
        *   `SelectionIndex` of type `int` (capitalized for frontend access).
        *   `itemType` of type `string`.

## Phase 3: Update `ViewList` Logic

1.  **Remove `ComputeValue`:**
    *   The `ComputeValue` method is obsolete and will be removed from `ViewList`. The value of the `ViewList` variable for the frontend will be a JSON array of object references to its `Items`.

2.  **Modify `SyncViewItems` Logic:**
    *   This method will synchronize the `Items` slice with the `value` slice (`[]interface{}`).
    *   **Grow:** If `len(value) > len(Items)`, append new `ViewListItem`s to the `Items` slice until the lengths match.
    *   **Shrink:** If `len(value) < len(Items)`, remove `ViewListItem`s from the end of the `Items` slice until the lengths match, ensuring the removed items are properly destroyed.
    *   **Update:** After resizing, iterate from `i = 0` to `len(value) - 1`. For each `i`, set `Items[i].Item = value[i]` and `Items[i].Index = i`. This ensures all items are correctly mapped and indices are updated, even after reordering.

## Phase 4: Address Object ID and JSON Marshalling

1.  **`ViewList` Value for Frontend:** The value of the `ViewList` itself, when sent to the frontend, will be a JSON array of object references to the `ViewListItem`s in its `Items` slice. The change tracker will handle this conversion.
2.  **Remove `ViewListItem.ToJSON()`:** This method is redundant, as the change tracker will handle the conversion of `ViewListItem` objects to JSON references.

## Phase 5: Scrub Obsolete Code

1.  **Remove `GetBaseItem` and `GetBaseItemID`:** These methods will be removed entirely.
2.  **Remove JSON parsing:** All code that manually parses `json.RawMessage` will be removed.

## Phase 6: Testing

1.  This is a major refactoring. All existing tests must be updated to reflect the renaming and structural changes.
2.  New unit tests will be written to specifically test the new `ViewList` and `ViewListItem` and their interaction.
3.  Integration tests should be run to ensure the frontend still behaves as expected after these backend changes.

## Phase 7: Final Design Verification

After the code is refactored and tested, a final check will be performed to ensure that the implementation matches the design specified in the CRC and sequence diagrams.