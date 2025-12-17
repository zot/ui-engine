# Plan to Refactor the Wrapper and Factory Systems

## Introduction

This plan outlines a comprehensive refactoring of the wrapper and object creation systems in `internal/lua/` and `internal/variable`. The key clarifications guiding this refactoring are:

1.  **Two Factory Registries:** There are two distinct factory systems:
    *   A **Create Factory** for `create` properties, which creates a new object from a value.
    *   A **Wrapper Factory** for `wrapper` properties, which creates a wrapper from a variable.
2.  **Factory Registry vs. Instance Storage:** The registries store **factory functions**, not wrapper instances. Instances are stored directly on the `Variable` object.
3.  **Wrapper as Stand-in Value:** Wrappers are "stand-in" values, not "transformers". There is no `ComputeValue` method.
4.  **Optional `Wrapper` Interface:** The `Wrapper` interface (which may contain `Destroy()`) is optional. Therefore, factories and variables will deal with `interface{}` and use type assertions to check for optional methods.

## Phase 1: Implement Two Factory Registries

1.  **Implement `CreateFactory` Registry:**
    *   Create a new global registry for `create` factories. The factory functions will take a value (`interface{}`) and return a new object (`interface{}`).
2.  **Update `WrapperFactory` Registry:**
    *   The existing "wrapper registry" will be clarified to be the `WrapperFactory` registry.
    *   Its factory functions will take a `WrapperVariable` and return `interface{}`.
    *   The `WrapperFactory` type definition will be updated to `type WrapperFactory func(runtime *Runtime, variable WrapperVariable) interface{}`.

## Phase 2: Update Core Wrapper and Variable Logic

1.  **Update `internal/lua/wrapper.go`:**
    *   The `Wrapper` interface will be removed, as wrappers are not required to implement any specific interface. Methods like `Destroy()` will be checked for via type assertion.
    *   `WrapperVariable.GetValue()` will return `interface{}`.
    *   Remove the `ComputeStoredValue` function.
    *   Remove `ComputeValue` from `LuaWrapper`.

2.  **Update `internal/variable/variable.go`:**
    *   The `wrapperInstance` field on the `Variable` struct will be of type `interface{}`.
    *   The `storedValue` of a variable will be its `wrapperInstance` if it exists; otherwise, it will be the raw `value`. The `computeStoredValue` method will be removed.

## Phase 3: Update `ViewList` Implementation

1.  **Update `internal/lua/viewlist.go`:**
    *   The `NewViewList` function will be the factory method registered in the `WrapperFactory` registry. It will take a `WrapperVariable` and return the new `ViewList` instance as an `interface{}`.
    *   The logic for processing the variable's value and syncing the `Items` will be moved into the `NewViewList` constructor.
    *   `ViewList` will no longer have a `ComputeValue` method.

## Phase 4: Update Documentation

1.  **Update `specs/protocol.md` and `specs/viewdefs.md`:**
    *   Describe the two-factory system (`create` and `wrapper`).
    *   Clarify that the registries are for factories, not instances.
    *   Remove all references to `ComputeValue`.
2.  **Update `design/crc-Wrapper.md` and `design/crc-Variable.md`:**
    *   Update the CRC cards to reflect the new two-factory system and the "stand-in" value pattern for wrappers.

## Phase 5: Testing

1.  **Update `internal/lua/viewlist_test.go`:**
    *   Update tests to reflect the new `ViewList` logic (syncing in the constructor, no `ComputeValue`).
2.  **Update `internal/lua/runtime_test.go`:**
    *   Update tests that rely on the old wrapper behavior.
3.  **Create `internal/variable/variable_test.go`:**
    *   If it doesn't exist, create a test file and add tests for the new `storedValue` logic.
4.  **Run all tests.**

## Phase 6: Final Verification

After all refactoring, documentation updates, and testing are complete, a final check will be performed to ensure the system is coherent and aligned with the updated design.