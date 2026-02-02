// Frontend application entry point
// CRC: crc-FrontendApp.md, crc-SPANavigator.md
// Spec: libraries.md, interfaces.md, js-api.md

import { Connection, VariableStore } from './connection';
import { BindingEngine } from './binding';
import { Message } from './protocol';
import { ViewdefStore } from './viewdef_store';
import { AppView, findAppElement, createAppView } from './app_view';
import { getSessionIdFromLocation } from './router';

export class UIApp {
  private connection: Connection;
  private store: VariableStore;
  private viewdefStore: ViewdefStore;
  private binding: BindingEngine;
  private sessionId: string;
  private appView: AppView | null = null;

  constructor() {
    this.sessionId = this.extractSessionId();
    this.connection = new Connection(this.sessionId);
    this.store = new VariableStore(this.connection);
    this.viewdefStore = new ViewdefStore();
    this.binding = new BindingEngine(this.store);

    // Wire up viewdef hot-reload: ViewdefStore can look up Views via BindingEngine
    // Spec: viewdefs.md - Hot-reload re-rendering
    this.viewdefStore.setViewLookup((elementId) => this.binding.getView(elementId));
  }

  private extractSessionId(): string {
    // Extract session ID from cookie or URL path
    // Cookie takes precedence (set by server for root session binding)
    return getSessionIdFromLocation();
  }

  async initialize(): Promise<void> {
    // Set up connection handlers
    this.connection.onConnect(() => {
      console.log('Connected to session:', this.sessionId);
    });

    this.connection.onDisconnect(() => {
      console.log('Disconnected from session');
    });

    this.connection.onError((error) => {
      console.error('Connection error:', error);
    });

    this.connection.onMessage((msg: Message) => {
      this.handleMessage(msg);
    });

    // Connect to server first
    await this.connection.connect();

    // Find and setup ui-app element after connection is established
    const appElement = findAppElement();
    if (appElement) {
      this.appView = createAppView(
        this.viewdefStore,
        this.store,
        this.binding
      );
    }
  }

  private handleMessage(msg: Message): void {
    switch (msg.type) {
      case 'error':
        const error = msg.data as { description: string };
        console.error('Server error:', error.description);
        break;
      // Other message types are handled by VariableStore
    }
  }

  // Navigation methods
  navigateTo(url: string): void {
    window.history.pushState({}, '', url);
    this.handleNavigation();
  }

  private handleNavigation(): void {
    // Extract path after session ID
    const path = window.location.pathname;
    const parts = path.split('/').filter(Boolean);
    const pagePath = '/' + parts.slice(1).join('/');

    // Update app presenter with new URL
    this.store.update(1, undefined, { url: pagePath });
  }

  // Get store for external access
  getStore(): VariableStore {
    return this.store;
  }

  // Get viewdef store for external access
  getViewdefStore(): ViewdefStore {
    return this.viewdefStore;
  }

  // Get connection for external access
  getConnection(): Connection {
    return this.connection;
  }

  // Get the AppView instance
  getAppView(): AppView | null {
    return this.appView;
  }

  // Get BindingEngine for external access
  getBinding(): BindingEngine {
    return this.binding;
  }

  // Update element's ui-value binding variable
  // Spec: js-api.md - updateValue method
  updateValue(elementId: string, value?: unknown): void {
    const widget = this.binding.getWidget(elementId);
    if (!widget) return;

    const varId = widget.getVariableId('ui-value');
    if (varId === undefined) return;

    // If no value provided, read from element's current value
    if (value === undefined) {
      const element = document.getElementById(elementId) as HTMLInputElement | null;
      value = element?.value;
    }

    this.store.update(varId, value);
  }
}

// Initialize on DOM ready
export function initUIApp(): Promise<UIApp> {
  const app = new UIApp();
  return app.initialize().then(() => app);
}

// Auto-initialize if ui-app attribute present
document.addEventListener('DOMContentLoaded', () => {
  const appElement = findAppElement();
  if (appElement) {
    initUIApp().then((app) => {
      // Expose for console debugging
      (window as any).uiApp = app;
      (window as any).uiStore = app.getStore();
    }).catch(console.error);
  }
});
