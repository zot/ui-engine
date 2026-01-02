// View - manages a ui-view element
// CRC: crc-View.md
// Spec: viewdefs.md

import { ViewdefStore } from './viewdef_store';
import { cloneViewdefContent, collectScripts, activateScripts } from './viewdef';
import { VariableStore } from './connection';
import { ViewList, createViewList } from './viewlist';
import { parsePath } from './binding';
import { ensureElementId } from './element_id_vendor';

export class View {
  readonly elementId: string;

  private valueType: string = '';
  private variableId: number | null = null;
  private rendered = false;
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  private bindCallback?: (element: HTMLElement, variableId: number) => void;
  private viewLists: ViewList[] = [];
  private childViews: View[] = [];

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    bindCallback?: (element: HTMLElement, variableId: number) => void
  ) {
    this.elementId = ensureElementId(element);

    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.bindCallback = bindCallback;

    // Get path from ui-view attribute
    const path = element.getAttribute('ui-view');
    if (path) {
      // Path creates a variable reference - not implemented yet
      // For now, we expect variable to be set directly
    }
  }

  // Get the element by ID lookup (no stored reference)
  // Spec: viewdefs.md - Element References (Cross-Cutting Requirement)
  getElement(): HTMLElement | null {
    return document.getElementById(this.elementId) as HTMLElement | null;
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

  // Render the view using TYPE.NAMESPACE viewdef
  // Returns true if rendered successfully
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

    // Clear and render
    this.clear();

    console.log('RENDER VIEW', this)

    // Clone template content (returns DocumentFragment, not yet in DOM)
    const fragment = cloneViewdefContent(viewdef);

    // Collect scripts before appending (store for later activation)
    const scripts = collectScripts(fragment);

    // Get element by ID lookup
    const element = this.getElement();
    if (!element) {
      console.error('View element not found:', this.elementId);
      return false;
    }

    // Append to element (nodes are now in DOM)
    element.appendChild(fragment);

    // Process ui-viewlist elements before binding
    // Spec: viewdefs.md - Path Resolution: Server-Side Only
    // Note: fragment is now empty after appendChild, query element
    this.processViewLists(element, this.variableId!);

    // Process ui-view elements before binding
    this.processChildViews(element, this.variableId!);

    // Apply bindings to cloned content
    if (this.bindCallback) {
      for (const child of element.children) {
        if (child instanceof HTMLElement) {
          this.bindCallback(child, this.variableId!);
        }
      }
    }

    // Activate scripts (scripts are now DOM-connected)
    activateScripts(scripts);

    this.rendered = true;
    this.valueType = type;

    // Remove from pending if we were pending
    this.removePending();

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
      this.bindCallback
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
      this.bindCallback
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

  // Clear rendered content
  clear(): void {
    // Destroy viewLists
    for (const viewList of this.viewLists) {
      viewList.destroy();
    }
    this.viewLists = [];

    // Destroy child views
    for (const view of this.childViews) {
      view.destroy();
    }
    this.childViews = [];

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

  // Get the variable ID
  getVariableId(): number | null {
    return this.variableId;
  }

  // Cleanup
  destroy(): void {
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }
    this.removePending();
    this.clear();
  }
}

// Create a View from an element with ui-view attribute
export function createView(
  element: HTMLElement,
  viewdefStore: ViewdefStore,
  variableStore: VariableStore,
  bindCallback?: (element: HTMLElement, variableId: number) => void
): View {
  return new View(element, viewdefStore, variableStore, bindCallback);
}
