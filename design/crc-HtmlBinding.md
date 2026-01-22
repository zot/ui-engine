# HtmlBinding

**Source Spec:** viewdefs.md

HTML bindings inject variable values as HTML content into elements. They support two modes: standard (innerHTML) and replace (element replacement).

## Responsibilities

### Knows
- widget: The Widget this binding belongs to (provides element ID and variable mappings)
- elementId: ID of the bound element (from Widget, for element lookup)
- childVarId: ID of the child variable created for this binding (NOT the parent context variable)
- path: Path property value sent to backend for resolution
- pathOptions: Parsed path options including `replace`, `access`, `scrollOnOutput`, etc.
- replaceMode: Boolean indicating if `replace` option was specified
- trackedElementIds: Array of element IDs currently in the DOM (for replace mode)
- originalElementId: The original element ID to preserve across replacements
- store: VariableStore reference for child variable management
- unbindValue: Callback to stop watching the child variable

### Does (implemented by BindingEngine)
- createChildVariable: Create child variable with path property for backend resolution
- watchChildVariable: Watch the child variable (not parent) for value updates
- applyHtml: Set element innerHTML or replace element(s) based on mode
- updateInnerHtml: For standard mode, set `element.innerHTML = value`
- replaceWithHtml: For replace mode, replace tracked elements with new HTML content
- parseHtmlFragment: Parse HTML string into DOM nodes using a scratch div
- assignElementIds: Assign IDs to new elements (first gets originalElementId, rest get vended IDs)
- removeTrackedElements: Remove all tracked elements from DOM
- insertElements: Insert new elements at the position of the first tracked element
- getElement: Look up DOM element by elementId (via document.getElementById)

## Unbind Handler

Each HTML binding registers an unbind handler with the Widget that:
1. Stops watching the child variable
2. Destroys the child variable
3. For replace mode: removes all tracked elements from the DOM

Called automatically when `widget.unbindAll()` is invoked.

## Child Variable Architecture

**Critical: HtmlBinding creates and manages a child variable for server-side path resolution.**

Like all bindings, HtmlBinding:
1. **Creates** a child variable: `store.create({parentId: contextVarId, properties: {path: "htmlContent"}})`
2. **Watches** the child variable for value updates (backend sends the HTML string)
3. **Destroys** the child variable when unbound

## Default Access Property

HtmlBinding always defaults to `access=r` (read-only) since HTML content flows from backend to frontend only.

| Binding Type | Default Access |
|--------------|----------------|
| `ui-html` | `r` |

## Standard Mode (innerHTML)

When `replace` is NOT specified, HtmlBinding simply sets the element's innerHTML:

```html
<div ui-html="description"></div>
```

**Behavior:**
1. Parse HTML string from child variable value
2. Set `element.innerHTML = value`
3. The element itself remains unchanged; only its children are replaced

**Null handling:** When value is null/undefined, sets innerHTML to empty string.

## Replace Mode (Element Replacement)

When `replace` IS specified in the path, the element is replaced with the HTML content:

```html
<div ui-html="renderedMarkdown?replace"></div>
```

### ID Preservation

The first element in the HTML content receives the original view element's ID:

1. When binding is created, store `originalElementId = widget.elementId`
2. When HTML is applied, the first element gets `id = originalElementId`
3. This ensures external references to the original element continue to work

### Fragment Handling

If the HTML produces multiple elements (a fragment):

1. Parse HTML into DOM nodes
2. First element gets `originalElementId`
3. Subsequent elements get IDs from ElementIdVendor
4. Store all element IDs in `trackedElementIds` array

**Example:**
```html
<!-- Original element: <div id="ui-5" ui-html="content?replace"></div> -->
<!-- HTML value: "<p>First</p><p>Second</p><p>Third</p>" -->

<!-- Result: -->
<p id="ui-5">First</p>
<p id="ui-17">Second</p>
<p id="ui-18">Third</p>
```

### Update Behavior

When the HTML content changes in replace mode:

1. Find the position of the first tracked element (for insertion point)
2. Get the parent node of the first tracked element
3. Remove ALL tracked elements from DOM
4. Parse new HTML into DOM nodes
5. Assign IDs (first gets `originalElementId`, rest get vended IDs)
6. Insert new elements at the stored position
7. Update `trackedElementIds` with the new element IDs

**Critical:** The widget's `elementId` is updated to remain valid:
- `widget.elementId` always points to the first tracked element
- This ensures subsequent operations can find the binding's "primary" element

### Cleanup

When the widget is unbound (replace mode):
1. Remove all elements in `trackedElementIds` from DOM
2. This is different from standard mode which leaves the element intact

## Parent Scroll Notifications

HTML content changes may trigger scrolling on an ancestor widget with `scrollOnOutput`:

- After applying HTML (in either mode), call `bindingEngine.addScrollNotification(parentVarId)`
- HTML content is typically content-resizable (like div, p, etc.)
- This follows the same pattern as ui-value on content-resizable elements

## Implementation Pattern

```typescript
// In BindingEngine
createHtmlBinding(widget, contextVarId, path, pathOptions) {
  const childVar = store.create({
    parentId: contextVarId,
    properties: { path, access: 'r' }
  });

  widget.registerBinding('ui-html', childVar.id);

  const replaceMode = pathOptions.replace === true;
  let trackedElementIds = [widget.elementId];
  const originalElementId = widget.elementId;

  const unbindValue = store.watch(childVar.id, (value) => {
    if (replaceMode) {
      trackedElementIds = applyReplaceHtml(
        trackedElementIds,
        originalElementId,
        value
      );
      widget.elementId = trackedElementIds[0];  // Keep widget reference valid
    } else {
      const element = document.getElementById(widget.elementId);
      element.innerHTML = value ?? '';
    }
    bindingEngine.addScrollNotification(childVar.parentId);
  });

  widget.addUnbindHandler('ui-html', () => {
    unbindValue();
    store.destroy(childVar.id);
    if (replaceMode) {
      trackedElementIds.forEach(id => {
        document.getElementById(id)?.remove();
      });
    }
  });
}
```

## Collaborators

- Widget: Binding context (provides element ID, variable mappings)
- BindingEngine: Creates and manages bindings
- Variable: Source of HTML content
- VariableStore: Child variable creation and watch
- ElementIdVendor: Vends IDs for fragment elements in replace mode

## Sequences

- seq-bind-element.md: Creating HTML binding
- seq-update-variable.md: Propagating HTML content changes
