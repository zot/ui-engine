# ElementIdVendor

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- nextId: Counter starting at 1, increments on each vend

### Does
- vendId: Return next unique ID in format `ui-{counter}`, increment counter

## Notes

### Global Singleton

ElementIdVendor is a global singleton that provides unique element IDs to any frontend code that needs one. The counter starts at 1 and increments monotonically.

### ID Format

Format: `ui-{counter}` (e.g., `ui-1`, `ui-2`, `ui-3`)

### Usage

Used by any code that creates or manages elements needing unique IDs:
- Widget: When binding elements without IDs
- View: When creating view containers
- ViewList: When creating list containers
- ViewRenderer: For internal element tracking

### Example

```typescript
// Global vendor
const elementIdVendor = {
    nextId: 1,
    vendId(): string {
        return `ui-${this.nextId++}`;
    }
};

// Usage
const id = elementIdVendor.vendId();  // "ui-1"
element.id = id;
```

### Cross-Cutting Requirement

Frontend code MUST NOT store direct references to DOM elements. All element references must be stored as element IDs. Elements are looked up on demand via `document.getElementById(elementId)`.

**Rationale:**
- Avoids circular references and memory leaks from DOM element references
- Enables serialization of binding/widget state
- Simplifies garbage collection
- Elements can be looked up on demand

## Collaborators

- Widget: Uses vendId for elements without IDs
- View: Uses vendId for view containers
- ViewList: Uses vendId for list containers
- ViewRenderer: Uses vendId for managed elements

## Sequences

- seq-bind-element.md: Widget uses vendor when element lacks ID
