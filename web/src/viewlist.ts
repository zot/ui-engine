// ViewList - manages a ui-viewlist element (array of object refs)
// CRC: crc-ViewList.md
// Spec: viewdefs.md, protocol.md
// Sequence: seq-viewlist-update.md

import { View } from './view';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';

// Delegate interface for notifications
export interface ViewListDelegate {
  onItemAdd?(view: View, index: number): void;
  onItemRemove?(view: View, index: number): void;
}

// Parsed path with optional URL parameters
export interface ParsedViewListPath {
  path: string;
  wrapper?: string;
  item?: string;
  props: Record<string, string>;
}

// Parse a ViewList path like "contacts?item=ContactPresenter"
export function parseViewListPath(fullPath: string): ParsedViewListPath {
  const [path, queryPart] = fullPath.split('?');
  const result: ParsedViewListPath = { path, props: {} };

  if (queryPart) {
    const params = new URLSearchParams(queryPart);
    params.forEach((value, key) => {
      if (key === 'wrapper') {
        result.wrapper = value;
      } else if (key === 'item') {
        result.item = value;
      } else {
        result.props[key] = value;
      }
    });

    // If item is specified but wrapper is not, default wrapper to ViewList
    if (result.item && !result.wrapper) {
      result.wrapper = 'ViewList';
    }
  }

  return result;
}

export class ViewList {
  readonly element: HTMLElement;
  readonly namespace: string;

  private variableId: number | null = null;
  private views: View[] = [];
  private exemplar: HTMLElement;
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  private delegate: ViewListDelegate | null = null;
  private bindCallback?: (element: HTMLElement, variableId: number) => void;

  // Path properties for wrapper configuration
  private pathConfig: ParsedViewListPath | null = null;

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    bindCallback?: (element: HTMLElement, variableId: number) => void
  ) {
    this.element = element;
    // ViewList uses list-item namespace by default (not DEFAULT)
    // per specs/viewdefs.md
    this.namespace = element.getAttribute('ui-namespace') || 'list-item';
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.bindCallback = bindCallback;

    // Check for exemplar in element (first child element)
    const firstChild = element.firstElementChild;
    if (firstChild instanceof HTMLElement) {
      this.exemplar = firstChild.cloneNode(true) as HTMLElement;
      // Remove the exemplar from DOM
      element.removeChild(firstChild);
    } else {
      // Default exemplar is a div
      this.exemplar = document.createElement('div');
    }
  }

  // Set a custom exemplar element (e.g., sl-option)
  setExemplar(exemplar: HTMLElement): void {
    this.exemplar = exemplar.cloneNode(true) as HTMLElement;
  }

  // Set delegate for notifications
  setDelegate(delegate: ViewListDelegate): void {
    this.delegate = delegate;
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
      // ViewList always uses ViewList wrapper
      props['wrapper'] = this.pathConfig.wrapper || 'ViewList';

      if (this.pathConfig.item) {
        props['item'] = this.pathConfig.item;
      }

      // Include any additional properties from path
      Object.assign(props, this.pathConfig.props);
    } else {
      // Default wrapper for ViewList elements
      props['wrapper'] = 'ViewList';
    }

    return props;
  }

  // Get the base path (without query parameters)
  getBasePath(): string {
    return this.pathConfig?.path || '';
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
    if (this.variableId === null) {
      return;
    }

    const data = this.variableStore.get(this.variableId);
    if (!data) {
      return;
    }

    const value = data.value;
    if (!Array.isArray(value)) {
      this.clear();
      return;
    }

    const itemCount = value.length;

    // Build map of current views by index
    const existingViewsByIndex = new Map<number, View>();
    for (let i = 0; i < this.views.length; i++) {
      existingViewsByIndex.set(i, this.views[i]);
    }

    // Build new views array - one per array index
    const newViews: View[] = [];

    for (let index = 0; index < itemCount; index++) {
      let view = existingViewsByIndex.get(index);
      if (view) {
        // Reuse existing view at this index
        newViews.push(view);
        existingViewsByIndex.delete(index);
      } else {
        // Create new view with child variable for this index
        view = this.createItemView();
        newViews.push(view);

        // Create child variable with path = index (1-based per protocol)
        const indexPath = String(index + 1);
        this.variableStore.create({
          parentId: this.variableId!,
          properties: { path: indexPath },
        }).then((childVarId) => {
          view!.setVariable(childVarId);
        }).catch((err) => {
          console.error('Failed to create viewlist item variable:', err);
        });

        // Notify delegate
        if (this.delegate?.onItemAdd) {
          this.delegate.onItemAdd(view, newViews.length - 1);
        }
      }
    }

    // Remove views that are beyond the new array length
    for (const [index, view] of existingViewsByIndex) {
      // Notify delegate before removing
      if (this.delegate?.onItemRemove) {
        this.delegate.onItemRemove(view, index);
      }
      view.destroy();
      if (view.element.parentNode) {
        view.element.parentNode.removeChild(view.element);
      }
    }

    // Update views array
    this.views = newViews;

    // Reorder DOM to match array order
    this.reorderDOM();
  }

  // Create a view element for an item
  private createItemView(): View {
    const element = this.exemplar.cloneNode(true) as HTMLElement;
    // Don't set ui-view attribute - ViewList manages the variable directly
    const view = new View(
      element,
      this.viewdefStore,
      this.variableStore,
      this.bindCallback
    );
    return view;
  }

  // Reorder DOM elements to match views array
  private reorderDOM(): void {
    for (const view of this.views) {
      // Append moves element to end if already in DOM, or adds if not
      this.element.appendChild(view.element);
    }
  }

  // Add an item manually at the end of the list
  // Creates a child variable with the next index path
  addItem(): View {
    const view = this.createItemView();
    const index = this.views.length;
    this.views.push(view);
    this.element.appendChild(view.element);

    // Create child variable with path = index (1-based per protocol)
    if (this.variableId !== null) {
      const indexPath = String(index + 1);
      this.variableStore.create({
        parentId: this.variableId,
        properties: { path: indexPath },
      }).then((childVarId) => {
        view.setVariable(childVarId);
      }).catch((err) => {
        console.error('Failed to create viewlist item variable:', err);
      });
    }

    if (this.delegate?.onItemAdd) {
      this.delegate.onItemAdd(view, this.views.length - 1);
    }

    return view;
  }

  // Remove an item by index
  removeItem(index: number): void {
    if (index < 0 || index >= this.views.length) {
      return;
    }

    const view = this.views[index];

    if (this.delegate?.onItemRemove) {
      this.delegate.onItemRemove(view, index);
    }

    view.destroy();
    if (view.element.parentNode) {
      view.element.parentNode.removeChild(view.element);
    }

    this.views.splice(index, 1);
  }

  // Clear all items
  clear(): void {
    for (let i = this.views.length - 1; i >= 0; i--) {
      const view = this.views[i];
      if (this.delegate?.onItemRemove) {
        this.delegate.onItemRemove(view, i);
      }
      view.destroy();
    }
    this.views = [];
    while (this.element.firstChild) {
      this.element.removeChild(this.element.firstChild);
    }
  }

  // Get number of items
  getCount(): number {
    return this.views.length;
  }

  // Get view at index
  getView(index: number): View | undefined {
    return this.views[index];
  }

  // Get all views
  getViews(): View[] {
    return [...this.views];
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
