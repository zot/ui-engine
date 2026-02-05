// ViewRenderer - orchestrates view rendering
// CRC: crc-ViewRenderer.md
// Spec: viewdefs.md, libraries.md

import { View, createView } from './view';
import { ViewList, createViewList } from './viewlist';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';
import { BindingEngine, parsePath } from './binding';
import { cloneViewdefContent, collectScripts, activateScripts } from './viewdef';
import { ensureElementId } from './element_id_vendor';
import { buildNamespaceProperties } from './namespace';

export class ViewRenderer {
  private rootElementId: string;
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
    this.rootElementId = ensureElementId(rootElement);
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.bindingEngine = new BindingEngine(variableStore);
  }

  // Get the root element by ID lookup (no stored reference)
  // Spec: viewdefs.md - Element References (Cross-Cutting Requirement)
  getRootElement(): HTMLElement | null {
    return document.getElementById(this.rootElementId) as HTMLElement | null;
  }

  // Render a variable (main entry point)
  // Uses 3-tier namespace resolution from variable properties
  // Returns true if rendered successfully
  render(variableId: number): boolean {
    const data = this.variableStore.get(variableId);
    if (!data) {
      // Variable not in cache, wait for update
      this.addPendingRender(variableId);
      return false;
    }

    const type = data.properties['type'];
    if (!type) {
      // No type property, wait for it
      this.addPendingRender(variableId);
      return false;
    }

    // 3-tier namespace resolution from variable properties
    const namespace = data.properties['namespace'];
    const fallbackNamespace = data.properties['fallbackNamespace'];
    const viewdef = this.viewdefStore.get(type, namespace, fallbackNamespace);
    if (!viewdef) {
      // Viewdef not loaded, wait for it
      this.addPendingRender(variableId);
      return false;
    }

    // Clear existing content
    this.clear();

    // Get root element by ID lookup
    const rootElement = this.getRootElement();
    if (!rootElement) {
      console.error('Root element not found:', this.rootElementId);
      return false;
    }

    // Clone template (returns DocumentFragment, not yet in DOM)
    const fragment = cloneViewdefContent(viewdef);

    // Collect scripts before appending (store for later activation)
    const scripts = collectScripts(fragment);

    // Process ui-view and ui-viewlist elements before appending
    this.processFragment(fragment, variableId);

    // Append to root (nodes are now in DOM)
    rootElement.appendChild(fragment);

    // Bind all elements in root
    for (const child of Array.from(rootElement.children)) {
      if (child instanceof Element) {
        this.bindingEngine.bindElement(child, variableId);
      }
    }

    // Activate scripts (scripts are now DOM-connected)
    activateScripts(scripts);

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
      this.bindingEngine
    );

    // Get path and create child variable for backend path resolution
    const pathAttr = element.getAttribute('ui-view');
    if (pathAttr) {
      const parsed = parsePath(pathAttr);

      // Check for scrollOnOutput option
      // Spec: viewdefs.md - scrollOnOutput for ui-view
      // CRC: crc-View.md - pathOptions
      if (parsed.options.props?.['scrollOnOutput'] === 'true') {
        view.setScrollOnOutput(true);
      }

      // Build properties with namespace inheritance
      const properties: Record<string, string> = { path: parsed.path };
      buildNamespaceProperties(element, contextVarId, properties, this.variableStore);

      // Create child variable with path property
      const childVarId = this.variableStore.create({
        parentId: contextVarId,
        properties,
      });
      view.setVariable(childVarId);
      // Store cleanup info
      (view as unknown as { childVarId: number }).childVarId = childVarId;
    }

    this.views.set(view.elementId, view);
  }

  // Setup a ui-viewlist element
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private setupViewList(element: HTMLElement, contextVarId: number): void {
    console.log('[DEBUG] setupViewList called for contextVarId:', contextVarId);
    const viewList = createViewList(
      element,
      this.viewdefStore,
      this.variableStore,
      this.bindingEngine
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

      buildNamespaceProperties(element, contextVarId, properties, this.variableStore);

      console.log('[DEBUG] Creating viewlist variable with properties:', properties);
      const childVarId = this.variableStore.create({
        parentId: contextVarId,
        properties,
      });
      console.log('[DEBUG] ViewList variable created with id:', childVarId);
      viewList.setVariable(childVarId);
      // Store cleanup info
      (viewList as unknown as { childVarId: number }).childVarId = childVarId;
    }

    // Use element ID for tracking (ensured by ViewList constructor)
    this.viewLists.set(viewList.elementId, viewList);
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

    // Get root element by ID lookup
    const rootElement = this.getRootElement();
    if (rootElement) {
      // Unbind all elements
      for (const child of Array.from(rootElement.children)) {
        if (child instanceof Element) {
          this.bindingEngine.unbindElement(child);
        }
      }

      // Clear DOM
      while (rootElement.firstChild) {
        rootElement.removeChild(rootElement.firstChild);
      }
    }

    this.currentVariableId = null;
  }

  // Add a pending render request
  private addPendingRender(variableId: number): void {
    const id = `render-${variableId}`;
    this.viewdefStore.addPendingView({
      id,
      render: () => this.render(variableId),
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
