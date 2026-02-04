// View - manages a ui-view element
// CRC: crc-View.md
// Spec: viewdefs.md

import { ViewdefStore } from './viewdef_store';
import { cloneViewdefContent, collectScripts, activateScripts } from './viewdef';
import { VariableStore } from './connection';
import { ViewList, createViewList } from './viewlist';
import { parsePath, BindingEngine } from './binding';
import { ensureElementId, vendElementId } from './element_id_vendor';

// Inject CSS for no-flash view rendering
// New elements are hidden until the reveal timer fires
const NO_FLASH_STYLE = `
.ui-new-view {
  display: none !important;
}
`;

(function injectNoFlashStyle() {
  if (document.getElementById('ui-no-flash-style')) return;
  const style = document.createElement('style');
  style.id = 'ui-no-flash-style';
  style.textContent = NO_FLASH_STYLE;
  document.head.appendChild(style);
})();

// View class counter for unique viewClass assignment
let nextViewId = 1;

export class View {
  readonly elementId: string;
  private readonly viewClass: string;  // e.g. "ui-view-42", identifies all elements of this view
  private bufferTimeoutId?: number;  // Pending reveal timer (only set on buffer root)

  private valueType: string = '';
  private variableId: number | null = null;
  private rendered = false;
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  private binding?: BindingEngine;
  private viewLists: ViewList[] = [];
  private childViews: View[] = [];
  private _scrollOnOutput = false;  // Pending scrollOnOutput to set on widget
  private viewUnbindHandlers: (() => void)[] = [];  // View-specific cleanup handlers

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    binding?: BindingEngine
  ) {
    this.elementId = ensureElementId(element);
    this.viewClass = `ui-view-${nextViewId++}`;

    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.binding = binding;
  }

  // Get the first element by ID lookup (no stored reference)
  // Spec: viewdefs.md - Element References (Cross-Cutting Requirement)
  getElement(): HTMLElement | null {
    return document.getElementById(this.elementId) as HTMLElement | null;
  }

  // Get all elements owned by this view (marked with viewClass)
  getElements(): HTMLElement[] {
    // Query all elements with this view's class marker
    const parent = document.getElementById(this.elementId)?.parentElement;
    if (!parent) return [];
    return [...parent.querySelectorAll(`:scope > .${this.viewClass}`)] as HTMLElement[];
  }

  // Set scrollOnOutput flag (will be applied to widget on render)
  // Spec: viewdefs.md - scrollOnOutput (universal property on widget)
  // CRC: crc-Widget.md - scrollOnOutput property
  setScrollOnOutput(value: boolean): void {
    this._scrollOnOutput = value;
  }

  // Notify parent that this view rendered (for scrollOnOutput bubbling)
  // Spec: viewdefs.md - Render notifications (for scrollOnOutput)
  // CRC: crc-View.md - notifyParentRendered
  private notifyParentRendered(): void {
    if (!this.binding || this.variableId === null) return;
    const data = this.variableStore.get(this.variableId);
    if (data?.parentId) {
      this.binding.addScrollNotification(data.parentId);
    }
  }

  // Get namespace from variable properties (set from ui-namespace or inherited)
  private getNamespace(): string | undefined {
    if (this.variableId === null) return undefined;
    const data = this.variableStore.get(this.variableId);
    return data?.properties['namespace'];
  }

  // Get fallbackNamespace from variable properties (inherited from parent)
  private getFallbackNamespace(): string | undefined {
    if (this.variableId === null) return undefined;
    const data = this.variableStore.get(this.variableId);
    return data?.properties['fallbackNamespace'];
  }

  // Resolve namespace for a child element (when element doesn't have ui-namespace directly)
  // Spec: viewdefs.md - Namespace variable properties
  // Uses closest('[ui-namespace]') and checks containment with parent variable's element
  private resolveNamespace(element: HTMLElement, parentVarId: number): string | undefined {
    const closest = element.closest('[ui-namespace]') as HTMLElement | null;

    const parentData = this.variableStore.get(parentVarId);
    const parentNs = parentData?.properties['namespace'];
    const parentElementId = parentData?.properties['elementId'];
    const parentElement = parentElementId ? document.getElementById(parentElementId) : null;

    if (closest && parentElement && parentNs) {
      // Both exist: check containment
      // If closest is inside parent element, use closest's namespace
      // Otherwise, parent's namespace is "closer" in view hierarchy
      if (parentElement.contains(closest)) {
        return closest.getAttribute('ui-namespace') || undefined;
      } else {
        return parentNs;
      }
    } else if (closest) {
      // Only closest DOM element exists
      return closest.getAttribute('ui-namespace') || undefined;
    } else if (parentNs) {
      // Only parent variable namespace exists
      return parentNs;
    }

    return undefined;
  }

  // Set the bound variable (object reference)
  setVariable(variableId: number, send?: boolean): void {
    // Cleanup old watcher
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }

    this.variableId = variableId;
    this.rendered = false;

    // Watch the variable
    this.unwatch = this.variableStore.watch(variableId, (_v, _value, _props) => {
      this.render();
    }, send);

    // Initial render
    this.render();
  }

  // Get the resolved viewdef key for this view
  getViewdefKey(): string | undefined {
    if (this.variableId === null) return undefined;
    const data = this.variableStore.get(this.variableId);
    if (!data) return undefined;
    const type = data.properties['type'];
    if (!type) return undefined;

    const namespace = this.getNamespace();
    const fallbackNamespace = this.getFallbackNamespace();
    const viewdef = this.viewdefStore.get(type, namespace, fallbackNamespace);
    return viewdef?.key;
  }

  // Render the view using TYPE.NAMESPACE viewdef
  // Views replace their element(s) in the DOM rather than adding children
  // Returns true if rendered successfully
  // Spec: viewdefs.md - Render process
  // CRC: crc-View.md
  render(): boolean {
    if (this.variableId === null) return false;

    const data = this.variableStore.get(this.variableId);
    if (!data) {
      this.markPending();  // Variable not in cache yet
      return false;
    }

    const type = data.properties['type'];
    if (!type) {
      this.markPending();  // No type property yet
      return false;
    }

    // 3-tier namespace resolution: namespace -> fallbackNamespace -> DEFAULT
    const namespace = this.getNamespace();
    const fallbackNamespace = this.getFallbackNamespace();
    const viewdef = this.viewdefStore.get(type, namespace, fallbackNamespace);
    if (!viewdef) {
      this.markPending();  // Viewdef not loaded yet
      return false;
    }

    if (this.rendered && type === this.valueType) {
      return false;  // No need to re-render
    }

    // Get first element by ID lookup
    const firstElement = this.getElement();
    if (!firstElement) {
      console.error('View element not found:', this.elementId);
      return false;
    }

    // Remember insertion point before destroying old content
    const parent = firstElement.parentNode as HTMLElement | null;
    if (!parent) {
      console.error('View element has no parent:', this.elementId);
      return false;
    }

    // Check if an ancestor is already buffering (has ui-new-view class)
    // If so, we render normally since we're already hidden by ancestor
    const isInsideAncestorBuffer = parent.closest('.ui-new-view') !== null;

    // Get old elements to replace
    // On first render, getElements() returns [] because no elements have viewClass yet
    // So we fall back to firstElement (the placeholder)
    let oldElements = this.getElements();
    if (oldElements.length === 0) {
      oldElements = [firstElement];  // First render: use placeholder element
    }
    const insertBefore = oldElements[oldElements.length - 1].nextSibling;

    // CRITICAL: Capture OLD descendant views BEFORE any new children are added
    // Views sharing this widget are stored ancestor-to-descendant order
    // CRC: crc-View.md - descendant view cleanup on re-render
    type ViewLikeWithUnbind = { onWidgetUnbind?(): void };
    let oldDescendantViews: ViewLikeWithUnbind[] = [];
    if (this.binding) {
      const widget = this.binding.getWidget(this.elementId);
      if (widget) {
        const myIndex = widget.views.indexOf(this);
        if (myIndex >= 0) {
          // All views after this one are OLD descendants
          oldDescendantViews = widget.views.slice(myIndex + 1);
          // Truncate now so new children get added after this view
          widget.views.length = myIndex + 1;
        }
      }
    }

    // CRITICAL: Destroy old child views/viewlists FIRST
    // Widgets are keyed by elementId, so we must clear them before reusing IDs
    this.clearChildren();

    // Handle old elements based on buffering mode
    if (isInsideAncestorBuffer) {
      // Inside ancestor's buffer - remove immediately (already hidden)
      for (const el of oldElements) {
        el.remove();
      }
    } else {
      // We are the buffer root - mark old elements as obsolete (keep hidden until timer)
      // Also add viewClass so the timer's selector can find them
      for (const el of oldElements) {
        el.classList.add(this.viewClass, 'ui-obsolete-view');
      }
    }

    // Clone template content (returns DocumentFragment)
    const fragment = cloneViewdefContent(viewdef);

    // Collect scripts before inserting (store for later activation)
    const scripts = collectScripts(fragment);

    // Get root elements from fragment and assign IDs and classes
    const rootElements = Array.from(fragment.children) as HTMLElement[];
    if (rootElements.length === 0) {
      console.error('Viewdef has no root elements:', viewdef.key);
      return false;
    }

    // First element gets the stable elementId, rest get vended IDs
    // All elements get the viewClass for tracking
    rootElements[0].id = this.elementId;
    for (let i = 1; i < rootElements.length; i++) {
      rootElements[i].id = vendElementId();
    }

    // Add viewClass to all new elements for tracking
    // Only add ui-new-view if we're the buffer root (not inside ancestor's buffer)
    for (const el of rootElements) {
      el.classList.add(this.viewClass);
      if (!isInsideAncestorBuffer) {
        el.classList.add('ui-new-view');
      }
    }

    // Set ui-viewdef attribute on first element for hot-reload targeting
    // Spec: viewdefs.md - Hot-reloading
    // CRC: crc-View.md - viewdefKey
    rootElements[0].setAttribute('ui-viewdef', viewdef.key);

    // Insert new elements at the remembered position
    // Note: insertBefore with null appends to the end
    parent.insertBefore(fragment, insertBefore);

    // Process ui-viewlist and ui-view elements in each root element
    // Spec: viewdefs.md - Path Resolution: Server-Side Only
    for (const el of rootElements) {
      this.processViewLists(el, this.variableId!);
      this.processChildViews(el, this.variableId!);
    }

    // Apply bindings to each root element
    if (this.binding) {
      for (const el of rootElements) {
        this.binding.bindElement(el, this.variableId!);
      }
    }

    // AFTER new content is added, unbind the OLD descendants
    // This ensures new children are NOT in oldDescendantViews
    for (const descendant of oldDescendantViews) {
      descendant.onWidgetUnbind?.();
    }

    this.rendered = true;
    this.valueType = type;

    // Register view with widget for hot-reload
    // Spec: viewdefs.md - Hot-reload re-rendering
    if (this.binding) {
      this.binding.setViewForElement(this.elementId, this);
    }

    // Remove from pending if we were pending
    this.removePending();

    // Start reveal timer if we're the buffer root and no timer pending
    if (!isInsideAncestorBuffer && !this.bufferTimeoutId) {
      this.bufferTimeoutId = window.setTimeout(() => {
        // Remove obsolete elements for this view
        document.querySelectorAll(`.${this.viewClass}.ui-obsolete-view`).forEach(el => el.remove());

        // Reveal new elements for this view
        document.querySelectorAll(`.${this.viewClass}.ui-new-view`).forEach(el => {
          el.classList.remove('ui-new-view');
        });

        // Activate scripts (after content is in DOM)
        activateScripts(scripts);

        this.bufferTimeoutId = undefined;
      }, 100);
    }

    // Set scrollOnOutput on widget if View has the flag, then scroll
    // Spec: viewdefs.md - scrollOnOutput (universal property on widget)
    // CRC: crc-Widget.md - scrollOnOutput property
    if (this.binding) {
      const widget = this.binding.getWidget(this.elementId);
      if (widget) {
        if (this._scrollOnOutput) {
          widget.scrollOnOutput = true;
        }
        if (widget.scrollOnOutput) {
          widget.scrollToBottom();
        }
      }
    }

    // Notify parent that we rendered (for scrollOnOutput bubbling)
    // Spec: viewdefs.md - Render notifications (for scrollOnOutput)
    this.notifyParentRendered();

    return true;
  }

  // Process ui-viewlist elements in an element (including the element itself)
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private processViewLists(container: Element, contextVarId: number): void {
    this.processAttributeElements(container, 'ui-viewlist', (el) => {
      this.setupViewList(el, contextVarId);
    });
  }

  // Generic helper to process elements with a given attribute
  // Checks container first, then descendants (but not both)
  private processAttributeElements(
    container: Element,
    attribute: string,
    handler: (el: HTMLElement) => void
  ): void {
    // Check if the container itself has the attribute
    if (container instanceof HTMLElement && container.hasAttribute(attribute)) {
      handler(container);
      return; // Don't process descendants if container itself matches
    }

    // Otherwise process matching descendants
    for (const el of container.querySelectorAll(`[${attribute}]`)) {
      if (el instanceof HTMLElement) {
        handler(el);
      }
    }
  }

  // Build common namespace properties for child variables
  // Namespace is recorded BEFORE element replacement to preserve container's ui-namespace
  // Spec: viewdefs.md - Namespace variable properties
  private buildNamespaceProperties(
    element: HTMLElement,
    contextVarId: number,
    properties: Record<string, string>
  ): void {
    // 1. If element has ui-namespace, use it directly
    const elementNs = element.getAttribute('ui-namespace');
    if (elementNs) {
      properties['namespace'] = elementNs;
    } else {
      // 2. Otherwise resolve via DOM/parent hierarchy
      const namespace = this.resolveNamespace(element, contextVarId);
      if (namespace) {
        properties['namespace'] = namespace;
      }
    }

    // 3. Inherit fallbackNamespace from parent if not already set
    const parentData = this.variableStore.get(contextVarId);
    if (!properties['fallbackNamespace'] && parentData?.properties['fallbackNamespace']) {
      properties['fallbackNamespace'] = parentData.properties['fallbackNamespace'];
    }

    // Default to read-only access for views/viewlists
    if (!properties['access']) {
      properties['access'] = 'r';
    }
  }

  // Setup a ui-viewlist element
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private setupViewList(element: HTMLElement, contextVarId: number): void {
    const viewList = createViewList(
      element,
      this.viewdefStore,
      this.variableStore,
      this.binding
    );

    const path = element.getAttribute('ui-viewlist');
    if (path) {
      const basePath = viewList.getBasePath();
      const elId = ensureElementId(element);

      const properties: Record<string, string> = {
        path: basePath,
        elementId: elId,
        ...viewList.getVariableProperties(),
      };

      this.buildNamespaceProperties(element, contextVarId, properties);

      const childVarId = this.variableStore.create({
        parentId: contextVarId,
        properties,
      });
      viewList.setVariable(childVarId);
    }

    this.viewLists.push(viewList);
  }

  // Process ui-view elements in an element (including the element itself)
  private processChildViews(container: Element, contextVarId: number): void {
    this.processAttributeElements(container, 'ui-view', (el) => {
      this.setupChildView(el, contextVarId);
    });
  }

  // Setup a ui-view element
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private setupChildView(element: HTMLElement, contextVarId: number): void {
    const view = new View(
      element,
      this.viewdefStore,
      this.variableStore,
      this.binding
    );

    const path = element.getAttribute('ui-view');
    if (path) {
      const [basePath] = path.split('?');
      const parsed = parsePath(path);

      // Extract extra props (handled locally, not sent to backend)
      const extra = parsed.options.props ?? {};
      delete parsed.options.props;

      // Set scrollOnOutput on View if specified
      // Spec: viewdefs.md - scrollOnOutput (universal property on widget)
      if (extra['scrollOnOutput'] === 'true') {
        view.setScrollOnOutput(true);
      }
      delete extra['scrollOnOutput'];

      const elId = ensureElementId(element);

      const properties: Record<string, string> = {
        path: basePath,
        elementId: elId,
        ...(parsed.options as Record<string, string>),
        ...extra,
      };

      this.buildNamespaceProperties(element, contextVarId, properties);

      const childVarId = this.variableStore.create({
        parentId: contextVarId,
        properties,
      });
      view.setVariable(childVarId);
    }

    this.childViews.push(view);
  }

  // Destroy child views and viewlists without clearing DOM
  // Used during hot-reload swap where DOM cleanup is handled separately
  private clearChildren(): void {
    for (const viewList of this.viewLists) {
      viewList.destroy();
    }
    this.viewLists = [];

    for (const view of this.childViews) {
      view.destroy();
    }
    this.childViews = [];
  }

  // Clear rendered content (destroys children AND removes view elements from DOM)
  clear(): void {
    this.clearChildren();

    // Remove all elements from elementId to lastElementId
    const elements = this.getElements();
    for (const el of elements) {
      el.remove();
    }
    this.rendered = false;
  }

  // Mark as pending (waiting for type or viewdef)
  private markPending(): void {
    this.viewdefStore.addPendingView({
      id: this.elementId,
      render: () => this.render(),
    });
  }

  // Remove from pending list
  private removePending(): void {
    this.viewdefStore.removePendingView(this.elementId);
  }

  // Check if rendered
  isRendered(): boolean {
    return this.rendered;
  }

  // Force a re-render (for hot-reload)
  // Clears rendered state and re-renders with current variable
  // Spec: viewdefs.md - Hot-reload re-rendering
  // CRC: crc-View.md - rerender
  forceRender(): void {
    this.rendered = false;
    this.valueType = '';
    this.render();
  }

  // Callback from Widget.unbindAll() - runs view-specific cleanup
  // CRC: crc-View.md - onWidgetUnbind
  onWidgetUnbind(): void {
    for (const handler of this.viewUnbindHandlers) {
      handler();
    }
    this.viewUnbindHandlers = [];
  }

  // Get the variable ID
  getVariableId(): number | null {
    return this.variableId;
  }

  // Cleanup - destroys view and its associated variable
  // Spec: viewdefs.md - Variable destruction on re-render
  // CRC: crc-View.md - destroy
  destroy(): void {
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }

    // Clear pending buffer timer
    if (this.bufferTimeoutId) {
      clearTimeout(this.bufferTimeoutId);
      this.bufferTimeoutId = undefined;
    }

    this.removePending();
    this.clear();

    // Destroy the associated variable (notifies backend)
    // Backend destruction is recursive - destroys all child variables
    if (this.variableId !== null) {
      this.variableStore.destroy(this.variableId);
      this.variableId = null;
    }
  }
}

// Create a View from an element with ui-view attribute
export function createView(
  element: HTMLElement,
  viewdefStore: ViewdefStore,
  variableStore: VariableStore,
  binding?: BindingEngine
): View {
  return new View(element, viewdefStore, variableStore, binding);
}
