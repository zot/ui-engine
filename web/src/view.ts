// View - manages a ui-view element
// CRC: crc-View.md
// Spec: viewdefs.md

import { isObjectReference } from './variable';
import { ViewdefStore } from './viewdef_store';
import { cloneViewdefContent } from './viewdef';
import { VariableStore } from './connection';
import { ViewList, createViewList } from './viewlist';

// Counter for unique HTML ids
let nextHtmlId = 1;

function vendHtmlId(): string {
  return `ui-view-${nextHtmlId++}`;
}

export class View {
  readonly element: HTMLElement;
  readonly htmlId: string;
  readonly namespace: string;

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
    this.element = element;
    this.htmlId = element.id || vendHtmlId();
    if (!element.id) {
      element.id = this.htmlId;
    }

    // Get namespace from ui-namespace attribute
    this.namespace = element.getAttribute('ui-namespace') || 'DEFAULT';

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

  // Set the bound variable (object reference)
  setVariable(variableId: number): void {
    // Cleanup old watcher
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }

    this.variableId = variableId;
    this.rendered = false;

    // Watch the variable
    this.unwatch = this.variableStore.watch(variableId, () => {
      this.render();
    });

    // Initial render
    this.render();
  }

  // Set variable from an object reference value
  setVariableFromRef(value: unknown): void {
    if (isObjectReference(value)) {
      this.setVariable(value.obj);
    } else {
      this.clear();
    }
  }

  // Render the view using TYPE.NAMESPACE viewdef
  // Returns true if rendered successfully
  render(): boolean {
    if (this.variableId === null) {
      return false;
    }

    const data = this.variableStore.get(this.variableId);
    if (!data) {
      // Variable not in cache yet, wait for update
      this.markPending();
      return false;
    }

    const type = data.properties['type'];
    if (!type) {
      // No type property yet, wait for it
      this.markPending();
      return false;
    }

    const viewdef = this.viewdefStore.get(type, this.namespace);
    if (!viewdef) {
      // Viewdef not loaded yet, wait for it
      this.markPending();
      return false;
    }

    // Clear and render
    this.clear();

    // Clone template content
    const fragment = cloneViewdefContent(viewdef);

    // Process ui-viewlist elements before binding
    // Spec: viewdefs.md - Path Resolution: Server-Side Only
    this.processViewLists(fragment, this.variableId!);

    // Process ui-view elements before binding
    this.processChildViews(fragment, this.variableId!);

    // Apply bindings to cloned content - only bind top-level children,
    // bindElement will handle recursion internally
    if (this.bindCallback) {
      for (const child of fragment.children) {
        if (child instanceof HTMLElement) {
          this.bindCallback(child, this.variableId!);
        }
      }
    }

    this.element.appendChild(fragment);
    this.rendered = true;

    // Remove from pending if we were pending
    this.removePending();

    return true;
  }

  // Process ui-viewlist elements in a fragment
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private processViewLists(fragment: DocumentFragment, contextVarId: number): void {
    const viewListElements = fragment.querySelectorAll('[ui-viewlist]');
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

      // Create child variable with path and ViewList properties (wrapper, item, etc.)
      const properties: Record<string, string> = {
        path: basePath,
        ...props,
      };

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

  // Process ui-view elements in a fragment
  private processChildViews(fragment: DocumentFragment, contextVarId: number): void {
    const viewElements = fragment.querySelectorAll('[ui-view]');
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
      const [basePath] = path.split('?');

      // Create child variable with path property
      this.variableStore.create({
        parentId: contextVarId,
        properties: { path: basePath },
      }).then((childVarId) => {
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
    while (this.element.firstChild) {
      this.element.removeChild(this.element.firstChild);
    }
    this.rendered = false;
  }

  // Mark as pending (waiting for type or viewdef)
  private markPending(): void {
    this.viewdefStore.addPendingView({
      id: this.htmlId,
      render: () => this.render(),
    });
  }

  // Remove from pending list
  private removePending(): void {
    this.viewdefStore.removePendingView(this.htmlId);
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
