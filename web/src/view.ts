// View - manages a ui-view element
// CRC: crc-View.md
// Spec: viewdefs.md

import { ViewdefStore } from './viewdef_store';
import { cloneViewdefContent, collectScripts, activateScripts } from './viewdef';
import { VariableStore } from './connection';
import { ViewList, createViewList } from './viewlist';
import { parsePath, BindingEngine } from './binding';
import { ensureElementId } from './element_id_vendor';

// Inject CSS for flash-free hot-reload re-rendering
// The marker and everything after it is hidden until the swap completes
const RENDER_MARKER_STYLE = `
.ui-render-marker,
.ui-render-marker ~ * {
  display: none !important;
}
`;

(function injectRenderMarkerStyle() {
  if (document.getElementById('ui-render-marker-style')) return;
  const style = document.createElement('style');
  style.id = 'ui-render-marker-style';
  style.textContent = RENDER_MARKER_STYLE;
  document.head.appendChild(style);
})();

export class View {
  readonly elementId: string;

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

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    binding?: BindingEngine
  ) {
    this.elementId = ensureElementId(element);

    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.binding = binding;
  }

  // Get the element by ID lookup (no stored reference)
  // Spec: viewdefs.md - Element References (Cross-Cutting Requirement)
  getElement(): HTMLElement | null {
    return document.getElementById(this.elementId) as HTMLElement | null;
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

  // Resolve namespace for a child element
  // Spec: viewdefs.md - Namespace variable properties
  // Uses closest('[ui-namespace]') and checks if parent variable's element contains it
  private resolveNamespace(element: HTMLElement, parentVarId: number): string | undefined {
    // Find closest element with ui-namespace attribute
    const closestNsElement = element.closest('[ui-namespace]');
    if (!closestNsElement) {
      // No ui-namespace found, inherit from parent variable
      const parentData = this.variableStore.get(parentVarId);
      return parentData?.properties['namespace'];
    }

    // Get the parent variable's element
    const parentData = this.variableStore.get(parentVarId);
    const parentElementId = parentData?.properties['elementId'];
    const parentElement = parentElementId ? document.getElementById(parentElementId) : null;

    // If no parent variable or parent element contains the found namespace element, use it
    if (!parentElement || parentElement.contains(closestNsElement)) {
      return closestNsElement.getAttribute('ui-namespace') || undefined;
    }

    // Otherwise, inherit namespace from parent variable
    return parentData?.properties['namespace'];
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

    console.log('SET VIEW VARIABLE', this, this.variableStore.get(this.variableId!))
    // Watch the variable
    this.unwatch = this.variableStore.watch(variableId, (_v, _value, _props) => {
      this.render();
    }, send);

    // Initial render
    this.render();
  }

  // Set variable from an object reference value
  setVariableFromRef(_value: unknown): void {
    console.error('THIS IS ERRONEOUS CODE')
    //if (isObjectReference(value)) {
    //  this.setVariable(value.obj);
    //} else {
    //  this.clear();
    //}
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
  // Returns true if rendered successfully
  // Spec: viewdefs.md - Render process
  // CRC: crc-View.md
  render(): boolean {
    console.log('1')
    if (this.variableId === null) {
      return false;
    }
    const data = this.variableStore.get(this.variableId);
    if (!data) {
      // Variable not in cache yet, wait for update
      this.markPending();
      return false;
    }
    console.log('2')
    const type = data.properties['type'];
    if (!type) {
      // No type property yet, wait for it
      this.markPending();
      return false;
    }
    // 3-tier namespace resolution: namespace -> fallbackNamespace -> DEFAULT
    const namespace = this.getNamespace();
    const fallbackNamespace = this.getFallbackNamespace();
    const viewdef = this.viewdefStore.get(type, namespace, fallbackNamespace);
    if (!viewdef) {
      // Viewdef not loaded yet, wait for it
      this.markPending();
      return false;
    }
    console.log('3')
    if (this.rendered && type === this.valueType) {
      // no need to re-render
      return false
    }

    console.log('RENDER VIEW', this)

    // Get element by ID lookup
    const element = this.getElement();
    if (!element) {
      console.error('View element not found:', this.elementId);
      return false;
    }

    // Flash-free re-render: use marker to hide new content until swap completes
    // Only needed when re-rendering (element already has children)
    const hasExistingContent = element.childNodes.length > 0;
    let marker: HTMLElement | null = null;

    if (hasExistingContent) {
      // Insert marker - CSS hides marker and everything after it
      marker = document.createElement('div');
      marker.className = 'ui-render-marker';
      element.appendChild(marker);
    } else {
      // Initial render - just clear any stale state
      this.clear();
    }

    // Clone template content (returns DocumentFragment, not yet in DOM)
    const fragment = cloneViewdefContent(viewdef);

    // Collect scripts before appending (store for later activation)
    const scripts = collectScripts(fragment);

    // Set data-ui-viewdef attribute for hot-reload targeting
    // Spec: viewdefs.md - Hot-reloading
    // CRC: crc-View.md - viewdefKey
    element.setAttribute('data-ui-viewdef', viewdef.key);

    // Append to element (after marker if present, so hidden by CSS)
    element.appendChild(fragment);

    // Process ui-viewlist elements before binding
    // Spec: viewdefs.md - Path Resolution: Server-Side Only
    // Note: fragment is now empty after appendChild, query element
    this.processViewLists(element, this.variableId!);

    // Process ui-view elements before binding
    this.processChildViews(element, this.variableId!);

    // Apply bindings to cloned content (only bind new content after marker)
    if (this.binding) {
      let pastMarker = !marker;
      for (const child of element.children) {
        if (child === marker) {
          pastMarker = true;
          continue;
        }
        if (pastMarker && child instanceof HTMLElement) {
          this.binding.bindElement(child, this.variableId!);
        }
      }
    }

    // Swap: remove old content (before marker) and marker itself
    if (marker) {
      // Destroy old child views/viewlists (but don't clear DOM - we handle that below)
      this.clearChildren();
      // Remove everything before the marker
      while (element.firstChild !== marker) {
        element.removeChild(element.firstChild!);
      }
      // Remove the marker - new content now visible
      element.removeChild(marker);
    }

    // Activate scripts (after content is visible)
    activateScripts(scripts);

    this.rendered = true;
    this.valueType = type;

    // Register view with widget for hot-reload
    // Spec: viewdefs.md - Hot-reload re-rendering
    if (this.binding) {
      this.binding.setViewForElement(this.elementId, this);
    }

    // Remove from pending if we were pending
    this.removePending();

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

  // Process ui-viewlist elements in an element
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private processViewLists(container: Element, contextVarId: number): void {
    const viewListElements = container.querySelectorAll('[ui-viewlist]');
    for (const el of viewListElements) {
      if (el instanceof HTMLElement) {
        this.setupViewList(el, contextVarId);
      }
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

    // Get path and create child variable for backend path resolution
    const path = element.getAttribute('ui-viewlist');
    if (path) {
      // Get the base path and additional properties from ViewList
      const basePath = viewList.getBasePath();
      const props = viewList.getVariableProperties();

      // Ensure element has an ID for tracking
      // Spec: viewdefs.md - Variable Element Tracking
      const elId = ensureElementId(element);

      // Create child variable with path and ViewList properties (wrapper, item, etc.)
      const properties: Record<string, string> = {
        path: basePath,
        elementId: elId,
        ...props,
      };

      // Resolve namespace using closest ui-namespace element
      // Spec: viewdefs.md - Namespace variable properties
      const namespace = this.resolveNamespace(element, contextVarId);
      if (namespace) {
        properties['namespace'] = namespace;
      }

      // Always inherit fallbackNamespace from parent variable
      const parentData = this.variableStore.get(contextVarId);
      if (parentData?.properties['fallbackNamespace']) {
        properties['fallbackNamespace'] = parentData.properties['fallbackNamespace'];
      }

      // Default to access=r for ui-viewlist (read-only binding)
      // Spec: viewdefs.md - ViewLists
      if (!properties['access']) {
        properties['access'] = 'r';
      }

      this.variableStore.create({
        parentId: contextVarId,
        properties,
      }).then((childVarId) => {
        viewList.setVariable(childVarId);
      }).catch((err) => {
        console.error('Failed to create viewlist variable:', err);
      });
    }

    this.viewLists.push(viewList);
  }

  // Process ui-view elements in an element
  private processChildViews(container: Element, contextVarId: number): void {
    const viewElements = container.querySelectorAll('[ui-view]');
    for (const el of viewElements) {
      if (el instanceof HTMLElement) {
        this.setupChildView(el, contextVarId);
      }
    }
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

    // Get path and create child variable for backend path resolution
    const path = element.getAttribute('ui-view');
    if (path) {
      // Parse path to extract base path (without query params)
      const [basePath, ] = path.split('?');
      const props = parsePath(path)

      console.log('CREATING VIEW FOR ', path, ' parent: ', contextVarId)
      let extra = props.options.props
      if (extra) {
        delete props.options.props
      } else {
        extra = {}
      }

      // Set scrollOnOutput on View if specified (will be applied to widget on render)
      // Spec: viewdefs.md - scrollOnOutput (universal property on widget)
      // CRC: crc-Widget.md - scrollOnOutput property
      if (extra['scrollOnOutput'] === 'true') {
        view.setScrollOnOutput(true);
      }
      delete extra['scrollOnOutput'];  // Don't send to backend (handled locally)

      // Ensure element has an ID for tracking
      // Spec: viewdefs.md - Variable Element Tracking
      const elId = ensureElementId(element);

      // Build properties with namespace inheritance
      const properties: Record<string, string> = {
        path: basePath,
        elementId: elId,
        ...(props.options as any),
        ...extra,
      };

      // Resolve namespace using closest ui-namespace element
      // Spec: viewdefs.md - Namespace variable properties
      const namespace = this.resolveNamespace(element, contextVarId);
      if (namespace) {
        properties['namespace'] = namespace;
      }

      // Always inherit fallbackNamespace from parent variable
      const parentData = this.variableStore.get(contextVarId);
      if (parentData?.properties['fallbackNamespace']) {
        properties['fallbackNamespace'] = parentData.properties['fallbackNamespace'];
      }

      // Default to access=r for ui-view (read-only binding)
      // Spec: viewdefs.md - Views
      if (!properties['access']) {
        properties['access'] = 'r';
      }

      // Create child variable with path property
      this.variableStore.create({
        parentId: contextVarId,
        properties,
      }).then((childVarId) => {
        console.log("GET VIEW VARIABLE ", childVarId, " props: ", JSON.stringify(this.variableStore.get(childVarId)?.properties))
        view.setVariable(childVarId);
      }).catch((err) => {
        console.error('Failed to create view variable:', err);
      });
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

  // Clear rendered content (destroys children AND clears DOM)
  clear(): void {
    this.clearChildren();

    // Clear DOM
    const element = this.getElement();
    if (element) {
      while (element.firstChild) {
        element.removeChild(element.firstChild);
      }
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
