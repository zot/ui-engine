// ViewList - manages a ui-viewlist element (array of object refs)
// CRC: crc-ViewList.md
// Spec: viewdefs.md, protocol.md
// Sequence: seq-viewlist-update.md

import { View } from './view';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';
import { BindingEngine, parsePath } from './binding';
import { ensureElementId } from './element_id_vendor';

// Parsed path with optional URL parameters
export interface ParsedViewListPath {
  path: string;
  wrapper?: string;
  item?: string;
  props: Record<string, string>;
}

// Parse a ViewList path like "contacts?wrapper=CustomListPresenter&item=ContactPresenter"
export function parseViewListPath(fullPath: string): ParsedViewListPath {
  const parsed = parsePath(fullPath);
  return {
    path: parsed.path,
    wrapper: parsed.options.wrapper,
    item: parsed.options.item,
    props: parsed.options.props ?? {},
  };
}

export class ViewList {
  readonly elementId: string;

  private itemWrapper?: string;
  private variableId: number | null = null;
  private itemViews: View[] = [];  // View instances for child items
  private exemplarHtml: string;    // store as HTML string, not element reference
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  private binding?: BindingEngine;
  private _scrollOnOutput = false;  // Pending scrollOnOutput to set on widget

  // Path properties for wrapper configuration
  private pathConfig: ParsedViewListPath | null = null;

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    binding?: BindingEngine
  ) {
    this.elementId = ensureElementId(element);
    this.itemWrapper = element.getAttribute('ui-item-wrapper') || undefined;
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.binding = binding;

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

  // Set scrollOnOutput flag (will be applied to widget)
  // Spec: viewdefs.md - scrollOnOutput (universal property on widget)
  // CRC: crc-Widget.md - scrollOnOutput property
  setScrollOnOutput(value: boolean): void {
    this._scrollOnOutput = value;
  }

  // Register with widget and set scrollOnOutput if flagged
  // CRC: crc-ViewList.md - Widget Registration
  private registerWidget(): void {
    if (!this.binding) return;
    // Ensure widget exists for this element
    this.binding.setViewForElement(this.elementId, { forceRender: () => this.update() });
    const widget = this.binding.getWidget(this.elementId);
    if (widget && this._scrollOnOutput) {
      widget.scrollOnOutput = true;
    }
  }

  // Notify parent that items were added (for scrollOnOutput bubbling)
  // Spec: viewdefs.md - Render notifications (for scrollOnOutput)
  // CRC: crc-ViewList.md - notifyParentRendered
  private notifyParentRendered(): void {
    if (!this.binding || this.variableId === null) return;
    const data = this.variableStore.get(this.variableId);
    if (data?.parentId) {
      this.binding.addScrollNotification(data.parentId);
    }
  }

  // Scroll element to bottom if widget has scrollOnOutput
  // CRC: crc-Widget.md - scrollToBottom
  private scrollToBottom(): void {
    if (!this.binding) return;
    const widget = this.binding.getWidget(this.elementId);
    if (widget?.scrollOnOutput) {
      widget.scrollToBottom();
    }
  }

  // Set a custom exemplar element (e.g., sl-option)
  setExemplar(exemplar: HTMLElement): void {
    this.exemplarHtml = exemplar.outerHTML;
  }

  // Parse and store path configuration from ui-viewlist attribute
  setPathConfig(fullPath: string): void {
    this.pathConfig = parseViewListPath(fullPath);

    // Extract scrollOnOutput option (will be applied to widget)
    // Spec: viewdefs.md - scrollOnOutput (universal property on widget)
    // CRC: crc-Widget.md - scrollOnOutput property
    if (this.pathConfig.props['scrollOnOutput'] === 'true' ||
        this.pathConfig.props['scrollOnOutput'] === '') {
      this._scrollOnOutput = true;
      delete this.pathConfig.props['scrollOnOutput'];  // Don't send to backend
    }
  }

  // Get properties to set on the variable when created
  getVariableProperties(): Record<string, string> {
    if (!this.pathConfig) return {};

    const props: Record<string, string> = { ...this.pathConfig.props };
    if (this.pathConfig.item) {
      props.item = this.pathConfig.item;
    }
    return props;
  }

  // Get the base path (without query parameters)
  getBasePath(): string {
    return this.pathConfig?.path || '';
  }

  // Resolve namespace for an item element
  // Spec: viewdefs.md - Namespace variable properties
  private resolveNamespace(
    itemElement: HTMLElement,
    listVariable: { properties: Record<string, string> }
  ): string | undefined {
    const closestNsElement = itemElement.closest('[ui-namespace]');
    const listElement = this.getElement();

    // Use ui-namespace if found within our list element
    if (closestNsElement && listElement?.contains(closestNsElement)) {
      return closestNsElement.getAttribute('ui-namespace') || undefined;
    }

    // Otherwise inherit from list variable
    return listVariable.properties.namespace;
  }

  // Set the bound variable (should contain array of object refs)
  setVariable(variableId: number): void {
    // Cleanup old watcher
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }

    this.variableId = variableId;

    // Register widget for scrollOnOutput support
    // CRC: crc-ViewList.md - Widget Registration
    this.registerWidget();

    // Watch the variable
    this.unwatch = this.variableStore.watch(variableId, () => {
      this.update();
    });

    // Initial update
    this.update();
  }

  // Update views to match the bound array
  // Creates child variables with index paths (0, 1, 2...) for each item
  update(): void {
    if (this.variableId === null) return;

    const listVariable = this.variableStore.get(this.variableId);
    if (!listVariable) return;

    const value = listVariable.value;
    if (!Array.isArray(value)) {
      this.clear();
      return;
    }

    // Add views for new items
    while (this.itemViews.length < value.length) {
      const index = this.itemViews.length;
      const view = this.createItemView();
      this.itemViews.push(view);
      this.createItemVariable(view, index, listVariable);
    }

    // Remove views beyond the new array length
    while (this.itemViews.length > value.length) {
      this.itemViews.pop()!.destroy();
    }

    this.scrollToBottom();
    this.notifyParentRendered();
  }

  // Create a child variable for an item view at the given index
  // Spec: viewdefs.md - Namespace variable properties
  private createItemVariable(
    view: View,
    index: number,
    listVariable: { parentId?: number; properties: Record<string, string> }
  ): void {
    const parentVariable = listVariable.parentId
      ? this.variableStore.get(listVariable.parentId)
      : undefined;

    const indexPath = this.itemWrapper
      ? `${index}?wrapper=${this.itemWrapper}`
      : String(index);

    const itemProps: Record<string, string> = {
      path: indexPath,
      elementId: view.elementId,
    };

    // Inherit wrapper from parent if specified
    if (parentVariable?.properties.itemWrapper) {
      itemProps.wrapper = parentVariable.properties.itemWrapper;
    }

    // Resolve namespace from closest ui-namespace element
    const viewElement = view.getElement();
    if (viewElement) {
      const namespace = this.resolveNamespace(viewElement, listVariable);
      if (namespace) {
        itemProps.namespace = namespace;
      }
    }

    // Inherit fallbackNamespace from list variable
    if (listVariable.properties.fallbackNamespace) {
      itemProps.fallbackNamespace = listVariable.properties.fallbackNamespace;
    }

    const childVarId = this.variableStore.create({
      parentId: this.variableId!,
      properties: itemProps,
    });
    view.setVariable(childVarId);
  }

  // Create item view and append to DOM (must be in DOM for closest() to work)
  private createItemView(): View {
    const template = document.createElement('template');
    template.innerHTML = this.exemplarHtml;
    const element = template.content.firstElementChild as HTMLElement;

    this.getElement()?.appendChild(element);

    return new View(element, this.viewdefStore, this.variableStore, this.binding);
  }

  // Clear all items
  clear(): void {
    for (const view of this.itemViews) {
      view.destroy();
    }
    this.itemViews = [];

    // Clear any remaining children from list element
    const listElement = this.getElement();
    if (listElement) {
      listElement.replaceChildren();
    }
  }

  // Get number of items
  getCount(): number {
    return this.itemViews.length;
  }

  // Get view element at index (returns first element of view)
  getViewElement(index: number): HTMLElement | null {
    return this.itemViews[index]?.getElement() ?? null;
  }

  // Get all view element IDs (returns first element ID of each view)
  getViewIds(): string[] {
    return this.itemViews.map(v => v.elementId);
  }

  // Cleanup - destroys viewlist and its associated variable
  // Spec: viewdefs.md - Variable destruction on re-render
  // CRC: crc-ViewList.md - destroy
  destroy(): void {
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }
    this.clear();

    // Destroy the associated variable (notifies backend)
    // Backend destruction is recursive - destroys all child variables
    if (this.variableId !== null) {
      this.variableStore.destroy(this.variableId);
      this.variableId = null;
    }
  }
}

// Create a ViewList from an element with ui-viewlist attribute
export function createViewList(
  element: HTMLElement,
  viewdefStore: ViewdefStore,
  variableStore: VariableStore,
  binding?: BindingEngine
): ViewList {
  const viewList = new ViewList(element, viewdefStore, variableStore, binding);

  // Parse path config from ui-viewlist attribute
  const path = element.getAttribute('ui-viewlist');
  if (path) {
    viewList.setPathConfig(path);
  }

  return viewList;
}
