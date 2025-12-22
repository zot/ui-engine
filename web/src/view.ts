// View - manages a ui-view element
// CRC: crc-View.md
// Spec: viewdefs.md

import { ViewdefStore } from './viewdef_store';
import { cloneViewdefContent } from './viewdef';
import { VariableStore } from './connection';
import { ViewList, createViewList } from './viewlist';
import { parsePath } from './binding';

// Counter for unique HTML ids
let nextHtmlId = 1;

function vendHtmlId(): string {
  return `ui-view-${nextHtmlId++}`;
}

export class View {
  readonly element: HTMLElement;
  readonly htmlId: string;
  readonly namespace: string;

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
    const viewdef = this.viewdefStore.get(type, this.namespace);
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
    this.valueType = type;

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
      const [basePath, ] = path.split('?');
      const props = parsePath(path)

      console.log('CREATING VIEW FOR ', path, ' parent: ', contextVarId)
      let extra = props.options.props
      if (extra) {
        delete props.options.props
      } else {
        extra = {}
      }
      // Create child variable with path property
      this.variableStore.create({
        parentId: contextVarId,
        properties: { path: basePath, ...(props.options as any), ...extra },
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
