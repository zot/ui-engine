// Frontend application entry point
// CRC: crc-FrontendApp.md, crc-SPANavigator.md
// Spec: libraries.md, interfaces.md

import { Connection, VariableStore } from './connection';
import { BindingEngine } from './binding';
import { Message } from './protocol';
import { ViewdefStore } from './viewdef_store';
import { AppView, findAppElement, createAppView } from './app_view';

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
  }

  private extractSessionId(): string {
    // Extract session ID from URL path: /SESSION-ID/...
    const path = window.location.pathname;
    const parts = path.split('/').filter(Boolean);
    if (parts.length > 0) {
      return parts[0];
    }
    throw new Error('No session ID in URL');
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
        (el, varId) => this.binding.bindElement(el, varId)
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
    initUIApp().catch(console.error);
  }
});
