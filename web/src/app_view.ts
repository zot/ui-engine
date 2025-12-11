// AppView - manages the ui-app element (root app view)
// CRC: crc-AppView.md
// Spec: viewdefs.md

import { View } from './view';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';

// Root app variable ID is always 1
const ROOT_VARIABLE_ID = 1;

export class AppView {
  readonly element: HTMLElement;
  readonly variableId: number = ROOT_VARIABLE_ID;
  readonly namespace: string;

  private view: View | null = null;
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  private bindCallback?: (element: HTMLElement, variableId: number) => void;

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    bindCallback?: (element: HTMLElement, variableId: number) => void
  ) {
    this.element = element;
    this.namespace = element.getAttribute('ui-namespace') || 'DEFAULT';
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.bindCallback = bindCallback;
  }

  // Initialize the AppView: create View and watch variable 1
  initialize(): void {
    // Create View for the ui-app element
    this.view = new View(
      this.element,
      this.viewdefStore,
      this.variableStore,
      this.bindCallback
    );

    // Set the variable to 1 (root app variable)
    this.view.setVariable(this.variableId);

    // Also watch variable 1 for viewdefs property updates
    this.unwatch = this.variableStore.watch(this.variableId, (value, props) => {
      this.handleRootUpdate(value, props);
    });
  }

  // Handle updates to variable 1
  private handleRootUpdate(value: unknown, _props: Record<string, string>): void {
    // Check for viewdefs property in the value
    if (typeof value === 'object' && value !== null && 'viewdefs' in value) {
      const viewdefs = (value as { viewdefs: Record<string, string> }).viewdefs;
      if (typeof viewdefs === 'object' && viewdefs !== null) {
        this.viewdefStore.processViewdefs(viewdefs);
      }
    }

    // The View will re-render automatically when type property changes
    // since it watches the variable
  }

  // Get the View instance
  getView(): View | null {
    return this.view;
  }

  // Check if rendered
  isRendered(): boolean {
    return this.view?.isRendered() ?? false;
  }

  // Cleanup
  destroy(): void {
    if (this.unwatch) {
      this.unwatch();
      this.unwatch = null;
    }
    if (this.view) {
      this.view.destroy();
      this.view = null;
    }
  }
}

// Find the ui-app element in the document
export function findAppElement(): HTMLElement | null {
  return document.querySelector('[ui-app]') as HTMLElement | null;
}

// Create an AppView from the ui-app element
export function createAppView(
  viewdefStore: ViewdefStore,
  variableStore: VariableStore,
  bindCallback?: (element: HTMLElement, variableId: number) => void
): AppView | null {
  const element = findAppElement();
  if (!element) {
    return null;
  }

  const appView = new AppView(element, viewdefStore, variableStore, bindCallback);
  appView.initialize();
  return appView;
}
