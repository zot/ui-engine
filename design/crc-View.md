# View

**Source Spec:** viewdefs.md
**Requirements:** R32, R33, R34, R35, R36, R37, R38, R50, R51, R52, R53, R54, R55, R56, R87, R89, R90, R91

## Responsibilities

### Knows
- elementId: ID of first element for this view (NOT direct DOM reference)
- viewClass: CSS class identifying all elements of this view (e.g., "ui-view-42")
- bufferTimeoutId: Pending reveal timer (only set on buffer root views)
- variable: Variable bound to this view (object reference)
- rendered: Whether view has been successfully rendered
- viewdefKey: The resolved viewdef key (e.g., "Contact.COMPACT") stored as `ui-viewdef` attribute on first element and as `viewdef` property on the variable
- viewUnbindHandlers: Array of cleanup handlers called when widget unbinds
- originalClass: Class attribute from original ui-view element (applied to first rendered element)
- originalStyle: Style attribute from original ui-view element (applied to first rendered element)

### Does
- create: Initialize view from element with ui-view attribute, vend element ID if needed, set variable namespace properties, register widget
- render: Render variable by replacing view's element(s) with template content, supports multi-element templates, sets `viewdef` property on variable, returns boolean
- setVariable: Update bound variable (triggers re-render)
- clear: Remove all view elements from DOM (from elementId to lastElementId), destroy child views
- destroy: Cleanup view - unwatch, remove from pending, clear DOM, destroy associated variable
- getElement: Look up first DOM element by elementId (via document.getElementById)
- getElements: Look up all DOM elements owned by this view (via querySelectorAll with viewClass)
- clearIfRendered: When type becomes empty on a rendered view, clear DOM content, insert placeholder, mark pending
- markPending: Add to pending views list (missing type or viewdef)
- removePending: Remove from pending views list after successful render
- resolveNamespace: Apply 3-tier resolution (namespace -> fallbackNamespace -> DEFAULT)
- rerender: Hot-reload re-render using updated viewdef (destroys old widgets/children, re-renders)
- notifyParentRendered: After rendering, add parent variable ID to BindingEngine's pendingScrollNotifications set
- onWidgetUnbind: Callback for view-level cleanup when widget unbinds (runs viewUnbindHandlers)

## Collaborators

- ElementIdVendor: Vends unique element ID if element lacks one
- ViewdefStore: Retrieves viewdefs by TYPE.NAMESPACE
- ViewRenderer: Creates and manages Views
- BindingEngine: Applies bindings to rendered content
- Variable: Source of object reference and type property

## Notes

### Default Access Property

The `ui-view` binding automatically adds `access=r` (read-only) if no `access` property is specified. Views are typically read-only bindings that display object references.

### Namespace Property Setting

When creating a view's variable, namespace resolution uses shared utilities (`namespace.ts`):

1. **resolveNamespace(element, parentVarId, variableStore):** Finds the namespace by:
   - Using `element.closest('[ui-namespace]')` to find nearest ancestor with namespace
   - Comparing DOM containment with parent variable's element to determine which is "closer"
   - Returns the namespace from whichever source is more specific

2. **buildNamespaceProperties(element, contextVarId, properties, variableStore):** Sets namespace properties:
   - If element has `ui-namespace`, uses it directly
   - Otherwise calls `resolveNamespace()` to find inherited namespace
   - Inherits `fallbackNamespace` from parent variable if not already set
   - Sets default `access=r` for views/viewlists

These utilities are shared between `View`, `ViewList`, and `ViewRenderer` to ensure consistent namespace handling.

### 3-Tier Namespace Resolution

When rendering, viewdef lookup uses this resolution order:
1. If variable has `namespace` property and `TYPE.{namespace}` viewdef exists, use it
2. Otherwise, if variable has `fallbackNamespace` property and `TYPE.{fallbackNamespace}` viewdef exists, use it
3. Otherwise, use `TYPE.DEFAULT`

This allows custom namespaces to fall back gracefully when specific viewdefs don't exist.

## Notes

### Element Replacement Model

Views replace their element(s) in the DOM rather than adding children to a container:
1. Template content replaces the view's current element(s)
2. First new element gets the stable `elementId`, additional elements get vended IDs
3. `viewClass` CSS class identifies all elements belonging to this view
4. On re-render: destroy old children first, mark/remove old elements, insert new elements at same position

### Class and Style Preservation

The original `ui-view` element's `class` and `style` attributes are preserved and applied to the first rendered element that is not a `<script>` or `<style>`:

1. **Constructor captures:** When the View is created, it stores the original element's `class` and `style` attribute values, filtering out internal classes (`ui-view-*`, `ui-new-view`, `ui-obsolete-view`) to prevent class leakage between views
2. **Render applies** to the first non-script/style root element:
   - Classes are split and added individually to preserve any existing classes from the viewdef template
   - Style is merged with existing style (viewdef style first, original style appended with `;`)
3. **Use case:** Allows styling the view container without modifying viewdef templates

```html
<!-- Original element -->
<div ui-view="person" class="highlight" style="margin: 10px"></div>

<!-- After render, first non-script/style element has both viewdef classes AND original classes/style -->
<div id="ui-42" class="person-card highlight" style="padding: 5px; margin: 10px">...</div>
```

### No-Flash Buffering

Views use ancestor-aware timer buffering to prevent visual flashing during re-renders:

1. **Buffer Root Detection**: View checks if parent.closest('.ui-new-view') exists
   - If YES: ancestor is buffering, render normally (already hidden by ancestor)
   - If NO: this view is the buffer root, use hide/reveal mechanism

2. **Buffer Root Behavior**:
   - Old elements get `.ui-obsolete-view` class (stay visible until timer fires)
   - Old elements have their `id` attribute removed (prevents `getElementById` from finding stale elements)
   - New elements get `.ui-new-view` class (hidden via CSS)
   - After 100ms timer: remove obsolete elements (found via class selector), reveal new elements

3. **CSS Classes**:
   - `.ui-view-{n}`: Identifies all elements of view n (replaces ID-based tracking)
   - `.ui-new-view`: Hidden (pending reveal)
   - `.ui-obsolete-view`: Marked for removal

4. **Edge Cases**:
   - Rapid re-renders: Timer already pending → add obsolete class, don't start new timer
   - View destroyed during buffer: Clear timeout, remove elements
   - Nested views: Render normally (ancestor handles buffering)

### Null View Clearing

When a rendered view's type property becomes empty (backend value became nil):
1. Destroy child views/viewlists via `clearChildren()` (removes their elements from DOM)
2. Unbind remaining elements via `BindingEngine.unbindElement()` to destroy binding-created variables
3. Remove all rendered elements from the DOM
4. Insert a plain `<div>` placeholder with the view's `elementId` and `viewClass`
5. Set `rendered = false` and `valueType = ''`
6. Clear the `viewdef` property on the variable (R91)
7. Call `markPending()` so the view re-renders if type becomes non-empty later

This is handled inline in `render()` when `!type && this.rendered`.

### Hot-Reload Support

Views support hot-reload re-rendering:
1. `ui-viewdef` attribute on first element stores the resolved viewdef key
2. When updated viewdefs arrive, ViewdefStore queries `[ui-viewdef="KEY"]` to find views
3. Each matching view's `forceRender()` method is called
4. Re-rendering reuses the same variable; old elements are replaced with new ones
5. Old child views/viewlists are destroyed before new elements are inserted (widgets keyed by elementId)

## Widget Registration

Views register themselves with the BindingEngine's widgets map. This enables:
- Consistent cleanup and lifecycle management
- `scrollOnOutput` support (set on the widget, not the view)
- Hot-reload targeting

When `scrollOnOutput` is specified in the path (e.g., `ui-view="chatLog?scrollOnOutput"`), it is set on the element's widget, not on the View itself. See crc-Widget.md for details.

## Render Notifications

After a view renders, it notifies its parent so ancestor widgets with `scrollOnOutput` can scroll:

1. After rendering, call `notifyParentRendered()` which adds the parent variable ID to BindingEngine's `pendingScrollNotifications` set
2. The BindingEngine processes these notifications after the batch completes (see crc-BindingEngine.md)
3. If an ancestor widget has `scrollOnOutput`, it scrolls to bottom

This batched approach ensures multiple child renders cause only one scroll.

## Variable Destruction

When a View is destroyed, it must destroy its associated variable:
1. The variable was created when the View was set up (via `setupChildView`)
2. `destroy()` calls `VariableStore.destroy(varId)` to notify the backend
3. Backend destruction is recursive - destroys all child variables
4. This prevents variable leaks during hot-reload re-render cycles

## Sequences

- seq-render-view.md: View creation and rendering with namespace resolution
- seq-viewdef-delivery.md: Processing pending views when viewdefs arrive
- seq-viewdef-hotload.md: Hot-reload re-rendering when viewdefs change
- seq-no-flash.md: Double-buffered re-render flow for no-flash rendering
