// ViewList - manages a ui-viewlist element (array of object refs)
// CRC: crc-ViewList.md
// Spec: viewdefs.md, protocol.md
// Sequence: seq-viewlist-update.md

import { View } from './view';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';
import { ensureElementId } from './element_id_vendor';

// Parsed path with optional URL parameters
export interface ParsedViewListPath {
  path: string;
  wrapper?: string;
  item?: string;
  props: Record<string, string>;
}

// Parse a ViewList path like "contacts?wrapper=CustomListPresenter&itemWrapper=ContactPresenter"
export function parseViewListPath(fullPath: string): ParsedViewListPath {
  const [path, queryPart] = fullPath.split('?');
  const result: ParsedViewListPath = { path, props: {} };

  if (queryPart) {
    const params = new URLSearchParams(queryPart);
    params.forEach((value, key) => {
      if (key === 'item') {
        result.item = value;
      } else {
        result.props[key] = value;
      }
    });
  }

  return result;
}

export class ViewList {
  readonly elementId: string;

  private itemWrapper?: string;
  private variableId: number | null = null;
  private viewIds: string[] = [];  // element IDs of child views (no direct references)
  private exemplarHtml: string;    // store as HTML string, not element reference
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  //private delegate: ViewListDelegate | null = null;
  private bindCallback?: (element: HTMLElement, variableId: number) => void;

  // Path properties for wrapper configuration
  private pathConfig: ParsedViewListPath | null = null;

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    bindCallback?: (element: HTMLElement, variableId: number) => void
  ) {
    this.elementId = ensureElementId(element);
    this.itemWrapper = element.getAttribute('ui-item-wrapper') || undefined;
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.bindCallback = bindCallback;

    // Check for exemplar in element (first child element)
    const firstChild = element.firstElementChild;
    if (firstChild instanceof HTMLElement) {
      this.exemplarHtml = firstChild.outerHTML;
      // Remove the exemplar from DOM
      element.removeChild(firstChild);
    } else {
      // Default exemplar is a div
      this.exemplarHtml = '<div></div>';
    }
  }

  // Get the element by ID lookup (no stored reference)
  // Spec: viewdefs.md - Element References (Cross-Cutting Requirement)
  getElement(): HTMLElement | null {
    return document.getElementById(this.elementId) as HTMLElement | null;
  }

  getItemWrapper(): string | undefined {
    // RETURN PARENT's itemWrapper property
    return
  }

  // Set a custom exemplar element (e.g., sl-option)
  setExemplar(exemplar: HTMLElement): void {
    this.exemplarHtml = exemplar.outerHTML;
  }

  // Parse and store path configuration from ui-viewlist attribute
  setPathConfig(fullPath: string): void {
    this.pathConfig = parseViewListPath(fullPath);
  }

  // Get properties to set on the variable when created
  // These include wrapper, item, and any additional props from path
  getVariableProperties(): Record<string, string> {
    const props: Record<string, string> = {};

    if (this.pathConfig) {
      if (this.pathConfig.item) {
        props['item'] = this.pathConfig.item;
      }

      // Include any additional properties from path
      Object.assign(props, this.pathConfig.props);
    }

    return props;
  }

  // Get the base path (without query parameters)
  getBasePath(): string {
    return this.pathConfig?.path || '';
  }

  // Resolve namespace for an item element
  // Spec: viewdefs.md - Namespace variable properties
  // Uses closest('[ui-namespace]') and checks if ViewList's element contains it
  private resolveNamespace(itemElement: HTMLElement, parentVar: { properties: Record<string, string> } | undefined): string | undefined {
    // Find closest element with ui-namespace attribute
    const closestNsElement = itemElement.closest('[ui-namespace]');
    if (!closestNsElement) {
      // No ui-namespace found, inherit from ViewList's variable
      return parentVar?.properties['namespace'];
    }

    // Get the ViewList's element (which is also the parent variable's element)
    const listElement = this.getElement();
    // If the ViewList element contains the found namespace element, use it
    if (listElement && listElement.contains(closestNsElement)) {
      return closestNsElement.getAttribute('ui-namespace') || undefined;
    }

    // Otherwise, inherit namespace from ViewList's variable
    return parentVar?.properties['namespace'];
  }

  // Set the bound variable (should contain array of object refs)
  setVariable(variableId: number): void {
    // Cleanup old watcher
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }

    this.variableId = variableId;

    // Watch the variable
    this.unwatch = this.variableStore.watch(variableId, () => {
      this.update();
    });

    // Initial update
    this.update();
  }

  // Update views to match the bound array
  // Creates child variables with index paths (1, 2, 3...) for each item
  update(): void {
    //console.log('update 1')
    if (this.variableId === null) {
      return;
    }

    const data = this.variableStore.get(this.variableId);
    if (!data) {
      return;
    }

    //console.log('update 2')
    const value = data.value;
    if (!Array.isArray(value)) {
      this.clear();
      return;
    }
    const itemCount = value.length;
    // Build map of current view IDs by index
    const newViewIds: string[] = this.viewIds.slice()

    while (newViewIds.length < itemCount) {
      // Create new view with child variable for this index
      // createItemView appends element to DOM first so closest() works
      const view = this.createItemView();
      const index = newViewIds.length
      newViewIds.push(view.elementId);
      // Create child variable with path = index (0-based)
      const indexPath = this.itemWrapper ? `${index}?wrapper=${this.itemWrapper}`
        : String(index);
      const variable = this.variableStore.get(this.variableId!)
      const parent = variable?.parentId ? this.variableStore.get(variable.parentId) : undefined
      console.log('CREATE LIST ITEM VIEW WITH PARENT', JSON.stringify(parent))

      // View already has an element ID from ensureElementId
      const itemProps: Record<string, string> = { path: indexPath, elementId: view.elementId }
      const itemWrapper = parent?.properties.itemWrapper
      if (itemWrapper) {
        itemProps.wrapper = itemWrapper
      }

      // Resolve namespace using closest ui-namespace element
      // Spec: viewdefs.md - Namespace variable properties
      // The view element is already appended to the ViewList element
      const viewElement = view.getElement();
      if (viewElement) {
        const namespace = this.resolveNamespace(viewElement, variable);
        if (namespace) {
          itemProps['namespace'] = namespace;
        }
      }

      // Always inherit fallbackNamespace from ViewList's variable
      if (variable?.properties['fallbackNamespace']) {
        itemProps['fallbackNamespace'] = variable.properties['fallbackNamespace'];
      }

      this.variableStore.create({
        parentId: this.variableId!,
        properties: itemProps,
      }).then((childVarId) => {
        console.log('setting view ', index, ' variable ', childVarId)
        view!.setVariable(childVarId);
      }).catch((err) => {
        console.error('Failed to create viewlist item variable:', err);
      });
    }

    // Remove views that are beyond the new array length
    while (newViewIds.length > itemCount) {
      const viewId = newViewIds.pop()!
      const viewElement = document.getElementById(viewId);
      if (viewElement && viewElement.parentNode) {
        viewElement.parentNode.removeChild(viewElement);
      }
    }

    // Update viewIds array
    this.viewIds = newViewIds;

  }

  // Create a view element for an item
  // Appends element to DOM first so closest() works for namespace resolution
  private createItemView(): View {
    // Create element from stored HTML
    const template = document.createElement('template');
    template.innerHTML = this.exemplarHtml;
    const element = template.content.firstElementChild as HTMLElement;

    // Append to DOM before creating View so closest() works
    const listElement = this.getElement();
    if (listElement) {
      listElement.appendChild(element);
    }

    // Don't set ui-view attribute - ViewList manages the variable directly
    const view = new View(
      element,
      this.viewdefStore,
      this.variableStore,
      this.bindCallback
    );
    return view;
  }

  // Clear all items
  clear(): void {
    // Remove child elements by ID
    for (const viewId of this.viewIds) {
      const viewElement = document.getElementById(viewId);
      if (viewElement && viewElement.parentNode) {
        viewElement.parentNode.removeChild(viewElement);
      }
    }
    this.viewIds = [];

    // Clear any remaining children from list element
    const listElement = this.getElement();
    if (listElement) {
      while (listElement.firstChild) {
        listElement.removeChild(listElement.firstChild);
      }
    }
  }

  // Get number of items
  getCount(): number {
    return this.viewIds.length;
  }

  // Get view element at index
  getViewElement(index: number): HTMLElement | null {
    const viewId = this.viewIds[index];
    return viewId ? document.getElementById(viewId) as HTMLElement | null : null;
  }

  // Get all view element IDs
  getViewIds(): string[] {
    return [...this.viewIds];
  }

  // Cleanup
  destroy(): void {
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }
    this.clear();
  }
}

// Create a ViewList from an element with ui-viewlist attribute
export function createViewList(
  element: HTMLElement,
  viewdefStore: ViewdefStore,
  variableStore: VariableStore,
  bindCallback?: (element: HTMLElement, variableId: number) => void
): ViewList {
  const viewList = new ViewList(element, viewdefStore, variableStore, bindCallback);

  // Parse path config from ui-viewlist attribute
  const path = element.getAttribute('ui-viewlist');
  if (path) {
    viewList.setPathConfig(path);
  }

  return viewList;
}
