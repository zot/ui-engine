// ViewdefStore - manages viewdef storage and pending views
// CRC: crc-ViewdefStore.md
// Spec: viewdefs.md

import { Viewdef, createViewdef, buildKey } from './viewdef';

// Interface for views waiting to render
export interface PendingView {
  id: string;
  render: () => boolean; // Returns true if rendered successfully
}

export class ViewdefStore {
  private viewdefs: Map<string, Viewdef> = new Map();
  private pendingViews: Map<string, PendingView> = new Map();
  private errorHandler?: (key: string, error: string) => void;

  // Set error handler for validation failures
  setErrorHandler(handler: (key: string, error: string) => void): void {
    this.errorHandler = handler;
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
  processViewdefs(viewdefs: Record<string, string>): void {
    for (const [key, content] of Object.entries(viewdefs)) {
      this.store(key, content);
    }

    // Try to render pending views
    this.processPendingViews();
  }

  // Get viewdef by TYPE.NAMESPACE, with fallback to TYPE.DEFAULT
  get(type: string, namespace: string): Viewdef | undefined {
    const key = buildKey(type, namespace);
    const viewdef = this.viewdefs.get(key);
    if (viewdef) {
      return viewdef;
    }

    // Fallback to DEFAULT namespace
    if (namespace !== 'DEFAULT') {
      const defaultKey = buildKey(type, 'DEFAULT');
      return this.viewdefs.get(defaultKey);
    }

    return undefined;
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
