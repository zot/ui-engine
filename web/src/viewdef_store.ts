// ViewdefStore - manages viewdef storage and pending views
// CRC: crc-ViewdefStore.md
// Spec: viewdefs.md

import { Viewdef, createViewdef, buildKey } from './viewdef';
import { ViewLike } from './binding';

// Interface for views waiting to render
export interface PendingView {
  id: string;
  render: () => boolean; // Returns true if rendered successfully
}

// Function to look up a View by element ID (via widget.view)
export type ViewLookup = (elementId: string) => ViewLike | undefined;

export class ViewdefStore {
  private viewdefs: Map<string, Viewdef> = new Map();
  private pendingViews: Map<string, PendingView> = new Map();
  private errorHandler?: (key: string, error: string) => void;
  private viewLookup?: ViewLookup;  // Set by BindingEngine for hot-reload

  // Set error handler for validation failures
  setErrorHandler(handler: (key: string, error: string) => void): void {
    this.errorHandler = handler;
  }

  // Set view lookup function (for hot-reload)
  // Called by BindingEngine to enable widget → view lookup
  setViewLookup(lookup: ViewLookup): void {
    this.viewLookup = lookup;
  }

  // Store a viewdef by key, validating the content
  // Returns true if stored successfully, false if validation failed
  store(key: string, content: string): boolean {
    const viewdef = createViewdef(key, content);
    if (!viewdef) {
      if (this.errorHandler) {
        this.errorHandler(key, `Invalid viewdef: must be a single <template> element`);
      }
      return false;
    }

    this.viewdefs.set(key, viewdef);
    return true;
  }

  // Process viewdefs from variable 1's viewdefs property
  // viewdefs is { "TYPE.NAMESPACE": "HTML content", ... }
  // Spec: viewdefs.md - Hot-reloading
  // CRC: crc-ViewdefStore.md - processViewdefs
  processViewdefs(viewdefs: Record<string, string>): void {
    const updatedKeys: string[] = [];

    for (const [key, content] of Object.entries(viewdefs)) {
      // Track if this is an update (viewdef already existed)
      const isUpdate = this.viewdefs.has(key);

      if (this.store(key, content) && isUpdate) {
        updatedKeys.push(key);
      }
    }

    // Try to render pending views
    this.processPendingViews();

    // Re-render views for updated viewdefs (hot-reload)
    // Spec: viewdefs.md - Hot-reload re-rendering
    for (const key of updatedKeys) {
      this.rerenderViewsForKey(key);
    }
  }

  // Re-render all views using a specific viewdef key
  // Queries DOM for elements with ui-viewdef attribute matching the key
  // Uses widget.view.forceRender() to trigger re-render
  // Spec: viewdefs.md - Hot-reload re-rendering
  // CRC: crc-ViewdefStore.md - rerenderViewsForKey
  rerenderViewsForKey(key: string): void {
    if (!this.viewLookup) {
      console.warn('[ViewdefStore] Hot-reload: viewLookup not set, cannot re-render');
      return;
    }

    const selector = `[ui-viewdef="${key}"]`;
    const elements = document.querySelectorAll(selector);

    console.log(`[ViewdefStore] Hot-reload: re-rendering ${elements.length} views for ${key}`);

    for (const element of elements) {
      if (element instanceof HTMLElement && element.id) {
        const view = this.viewLookup(element.id);
        if (view) {
          try {
            view.forceRender();
          } catch (err) {
            console.error(`[ViewdefStore] Hot-reload: error re-rendering view ${element.id}:`, err);
          }
        } else {
          console.warn(`[ViewdefStore] Hot-reload: no view found for element ${element.id}`);
        }
      }
    }
  }

  // Get viewdef by TYPE.NAMESPACE with 3-tier resolution:
  // 1. Try TYPE.namespace (if provided)
  // 2. Try TYPE.fallbackNamespace (if provided)
  // 3. Try TYPE.DEFAULT
  get(type: string, namespace?: string, fallbackNamespace?: string): Viewdef | undefined {
    // 1. Try explicit namespace
    if (namespace) {
      const key = buildKey(type, namespace);
      const viewdef = this.viewdefs.get(key);
      if (viewdef) {
        return viewdef;
      }
    }

    // 2. Try fallbackNamespace
    if (fallbackNamespace) {
      const fallbackKey = buildKey(type, fallbackNamespace);
      const viewdef = this.viewdefs.get(fallbackKey);
      if (viewdef) {
        return viewdef;
      }
    }

    // 3. Fallback to DEFAULT namespace
    const defaultKey = buildKey(type, 'DEFAULT');
    return this.viewdefs.get(defaultKey);
  }

  // Get viewdef by key
  getByKey(key: string): Viewdef | undefined {
    return this.viewdefs.get(key);
  }

  // Check if viewdef exists
  has(type: string, namespace: string): boolean {
    return this.get(type, namespace) !== undefined;
  }

  // Add a pending view
  addPendingView(view: PendingView): void {
    this.pendingViews.set(view.id, view);
  }

  // Remove a pending view
  removePendingView(id: string): void {
    this.pendingViews.delete(id);
  }

  // Process pending views, removing ones that render successfully
  processPendingViews(): void {
    const toRemove: string[] = [];

    for (const [id, view] of this.pendingViews) {
      if (view.render()) {
        toRemove.push(id);
      }
    }

    for (const id of toRemove) {
      this.pendingViews.delete(id);
    }
  }

  // Get count of pending views
  getPendingViewCount(): number {
    return this.pendingViews.size;
  }

  // Get count of stored viewdefs
  getViewdefCount(): number {
    return this.viewdefs.size;
  }

  // Get all keys
  getKeys(): string[] {
    return Array.from(this.viewdefs.keys());
  }

  // Clear all viewdefs (e.g., on reconnect)
  clear(): void {
    this.viewdefs.clear();
  }
}
