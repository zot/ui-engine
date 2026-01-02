// Global Element ID Vendor
// CRC: crc-ElementIdVendor.md
// Spec: viewdefs.md - Element References (Cross-Cutting Requirement)

let nextId = 1

// Vend a unique element ID
// Format: ui-{counter} (e.g., ui-1, ui-2, ui-3)
export function vendElementId(): string {
  return `ui-${nextId++}`
}

// Ensure an element has an ID, vending one if needed
// Returns the element's ID (existing or newly assigned)
export function ensureElementId(element: Element): string {
  if (!element.id) {
    element.id = vendElementId()
  }
  return element.id
}
