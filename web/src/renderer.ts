// ViewRenderer - orchestrates view rendering
// CRC: crc-ViewRenderer.md
// Spec: viewdefs.md, libraries.md

import { View, createView } from './view';
import { ViewList, createViewList } from './viewlist';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';
import { BindingEngine } from './binding';
import { cloneViewdefContent } from './viewdef';

export class ViewRenderer {
  private rootElement: HTMLElement;
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private bindingEngine: BindingEngine;
  private views: Map<string, View> = new Map();
  private viewLists: Map<string, ViewList> = new Map();
  private currentVariableId: number | null = null;

  constructor(
    rootElement: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore
  ) {
    this.rootElement = rootElement;
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.bindingEngine = new BindingEngine(variableStore);
  }

  // Render a variable with namespace (main entry point)
  // Returns true if rendered successfully
  render(variableId: number, namespace = 'DEFAULT'): boolean {
    const data = this.variableStore.get(variableId);
    if (!data) {
      // Variable not in cache, wait for update
      this.addPendingRender(variableId, namespace);
      return false;
    }

    const type = data.properties['type'];
    if (!type) {
      // No type property, wait for it
      this.addPendingRender(variableId, namespace);
      return false;
    }

    const viewdef = this.viewdefStore.get(type, namespace);
    if (!viewdef) {
      // Viewdef not loaded, wait for it
      this.addPendingRender(variableId, namespace);
      return false;
    }

    // Clear existing content
    this.clear();

    // Clone template
    const fragment = cloneViewdefContent(viewdef);

    // Process ui-view and ui-viewlist elements before appending
    this.processFragment(fragment, variableId);

    // Append to root
    this.rootElement.appendChild(fragment);

    // Bind all elements in root
    for (const child of Array.from(this.rootElement.children)) {
      if (child instanceof Element) {
        this.bindingEngine.bindElement(child, variableId);
      }
    }

    this.currentVariableId = variableId;
    return true;
  }

  // Process a document fragment for ui-view and ui-viewlist elements
  private processFragment(fragment: DocumentFragment, contextVarId: number): void {
    // Find and process ui-view elements
    const viewElements = fragment.querySelectorAll('[ui-view]');
    console.log('[DEBUG] processFragment: found', viewElements.length, 'ui-view elements');
    for (const el of viewElements) {
      if (el instanceof HTMLElement) {
        this.setupView(el, contextVarId);
      }
    }

    // Find and process ui-viewlist elements
    const viewListElements = fragment.querySelectorAll('[ui-viewlist]');
    console.log('[DEBUG] processFragment: found', viewListElements.length, 'ui-viewlist elements');
    for (const el of viewListElements) {
      console.log('[DEBUG] ui-viewlist element:', el.getAttribute('ui-viewlist'));
      if (el instanceof HTMLElement) {
        this.setupViewList(el, contextVarId);
      }
    }
  }

  // Setup a ui-view element
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private setupView(element: HTMLElement, contextVarId: number): void {
    const view = createView(
      element,
      this.viewdefStore,
      this.variableStore,
      (el, varId) => this.bindElement(el, varId)
    );

    // Get path and create child variable for backend path resolution
    const path = element.getAttribute('ui-view');
    if (path) {
      // Parse path to extract base path (without query params)
      const [basePath] = path.split('?');

      // Create child variable with path property
      this.variableStore.create({
        parentId: contextVarId,
        properties: { path: basePath },
      }).then((childVarId) => {
        view.setVariable(childVarId);
        // Store cleanup info
        (view as unknown as { childVarId: number }).childVarId = childVarId;
      }).catch((err) => {
        console.error('Failed to create view variable:', err);
      });
    }

    this.views.set(view.htmlId, view);
  }

  // Setup a ui-viewlist element
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private setupViewList(element: HTMLElement, contextVarId: number): void {
    console.log('[DEBUG] setupViewList called for contextVarId:', contextVarId);
    const viewList = createViewList(
      element,
      this.viewdefStore,
      this.variableStore,
      (el, varId) => this.bindElement(el, varId)
    );

    // Get path and create child variable for backend path resolution
    const path = element.getAttribute('ui-viewlist');
    console.log('[DEBUG] setupViewList path:', path);
    if (path) {
      // Get the base path and additional properties from ViewList
      const basePath = viewList.getBasePath();
      const props = viewList.getVariableProperties();
      console.log('[DEBUG] setupViewList basePath:', basePath, 'props:', props);

      // Create child variable with path and ViewList properties (wrapper, item, etc.)
      const properties: Record<string, string> = {
        path: basePath,
        ...props,
      };

      console.log('[DEBUG] Creating viewlist variable with properties:', properties);
      this.variableStore.create({
        parentId: contextVarId,
        properties,
      }).then((childVarId) => {
        console.log('[DEBUG] ViewList variable created with id:', childVarId);
        viewList.setVariable(childVarId);
        // Store cleanup info
        (viewList as unknown as { childVarId: number }).childVarId = childVarId;
      }).catch((err) => {
        console.error('Failed to create viewlist variable:', err);
      });
    }

    // Generate unique ID for tracking
    const id = element.id || `viewlist-${this.viewLists.size}`;
    if (!element.id) {
      element.id = id;
    }
    this.viewLists.set(id, viewList);
  }

  // Bind an element and its children
  private bindElement(element: HTMLElement, variableId: number): void {
    this.bindingEngine.bindElement(element, variableId);
  }

  // Clear current view content
  clear(): void {
    // Destroy all views
    for (const view of this.views.values()) {
      view.destroy();
    }
    this.views.clear();

    // Destroy all view lists
    for (const viewList of this.viewLists.values()) {
      viewList.destroy();
    }
    this.viewLists.clear();

    // Unbind all elements
    for (const child of Array.from(this.rootElement.children)) {
      if (child instanceof Element) {
        this.bindingEngine.unbindElement(child);
      }
    }

    // Clear DOM
    while (this.rootElement.firstChild) {
      this.rootElement.removeChild(this.rootElement.firstChild);
    }

    this.currentVariableId = null;
  }

  // Add a pending render request
  private addPendingRender(variableId: number, namespace: string): void {
    const id = `render-${variableId}-${namespace}`;
    this.viewdefStore.addPendingView({
      id,
      render: () => this.render(variableId, namespace),
    });
  }

  // Get the binding engine for external use
  getBindingEngine(): BindingEngine {
    return this.bindingEngine;
  }

  // Get the viewdef store
  getViewdefStore(): ViewdefStore {
    return this.viewdefStore;
  }

  // Get the current variable ID being rendered
  getCurrentVariableId(): number | null {
    return this.currentVariableId;
  }
}

// Create a ViewRenderer
export function createViewRenderer(
  rootElement: HTMLElement,
  viewdefStore: ViewdefStore,
  variableStore: VariableStore
): ViewRenderer {
  return new ViewRenderer(rootElement, viewdefStore, variableStore);
}
