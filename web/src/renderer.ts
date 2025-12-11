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
    for (const el of viewElements) {
      if (el instanceof HTMLElement) {
        this.setupView(el, contextVarId);
      }
    }

    // Find and process ui-viewlist elements
    const viewListElements = fragment.querySelectorAll('[ui-viewlist]');
    for (const el of viewListElements) {
      if (el instanceof HTMLElement) {
        this.setupViewList(el, contextVarId);
      }
    }
  }

  // Setup a ui-view element
  private setupView(element: HTMLElement, contextVarId: number): void {
    const view = createView(
      element,
      this.viewdefStore,
      this.variableStore,
      (el, varId) => this.bindElement(el, varId)
    );

    // Get path and resolve variable
    const path = element.getAttribute('ui-view');
    if (path) {
      const varId = this.resolvePathToVariable(path, contextVarId);
      if (varId !== null) {
        view.setVariable(varId);
      }
    }

    this.views.set(view.htmlId, view);
  }

  // Setup a ui-viewlist element
  private setupViewList(element: HTMLElement, contextVarId: number): void {
    const viewList = createViewList(
      element,
      this.viewdefStore,
      this.variableStore,
      (el, varId) => this.bindElement(el, varId)
    );

    // Get path and resolve variable
    const path = element.getAttribute('ui-viewlist');
    if (path) {
      const varId = this.resolvePathToVariable(path, contextVarId);
      if (varId !== null) {
        viewList.setVariable(varId);
      }
    }

    // Generate unique ID for tracking
    const id = element.id || `viewlist-${this.viewLists.size}`;
    if (!element.id) {
      element.id = id;
    }
    this.viewLists.set(id, viewList);
  }

  // Resolve a path to a variable ID
  // For now, just handles direct object references in the value
  private resolvePathToVariable(path: string, contextVarId: number): number | null {
    const data = this.variableStore.get(contextVarId);
    if (!data) return null;

    const value = data.value;
    if (typeof value !== 'object' || value === null) return null;

    // Navigate path
    const segments = path.split('.');
    let current: unknown = value;

    for (const segment of segments) {
      if (current === null || current === undefined) return null;

      if (typeof current === 'object') {
        if (Array.isArray(current) && /^\d+$/.test(segment)) {
          current = current[parseInt(segment, 10)];
        } else {
          current = (current as Record<string, unknown>)[segment];
        }
      } else {
        return null;
      }
    }

    // Check if result is an object reference
    if (typeof current === 'object' && current !== null && 'obj' in current) {
      return (current as { obj: number }).obj;
    }

    return null;
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
