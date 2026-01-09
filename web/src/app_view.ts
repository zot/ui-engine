// AppView - manages the ui-app element (root app view)
// CRC: crc-AppView.md
// Spec: viewdefs.md

import { View } from './view';
import { ViewdefStore } from './viewdef_store';
import { VariableStore } from './connection';
import { BindingEngine } from './binding';
import { ensureElementId } from './element_id_vendor';

// Root app variable ID is always 1
const ROOT_VARIABLE_ID = 1;

export class AppView {
  readonly elementId: string;
  readonly variableId: number = ROOT_VARIABLE_ID;
  readonly namespace: string;

  private view: View | null = null;
  private viewdefStore: ViewdefStore;
  private variableStore: VariableStore;
  private unwatch: (() => void) | null = null;
  private binding?: BindingEngine;

  constructor(
    element: HTMLElement,
    viewdefStore: ViewdefStore,
    variableStore: VariableStore,
    binding?: BindingEngine
  ) {
    this.elementId = ensureElementId(element);
    this.namespace = element.getAttribute('ui-namespace') || 'DEFAULT';
    this.viewdefStore = viewdefStore;
    this.variableStore = variableStore;
    this.binding = binding;
  }

  // Get the element by ID lookup (no stored reference)
  // Spec: viewdefs.md - Element References (Cross-Cutting Requirement)
  getElement(): HTMLElement | null {
    return document.getElementById(this.elementId) as HTMLElement | null;
  }

  // Initialize the AppView: create View and watch variable 1
  initialize(): void {
    const element = this.getElement();
    if (!element) {
      console.error('AppView element not found:', this.elementId);
      return;
    }

    // Create View for the ui-app element
    this.view = new View(
      element,
      this.viewdefStore,
      this.variableStore,
      this.binding
    );

    // Set the variable to 1 (root app variable)
    this.view.setVariable(this.variableId, true);

    //// Also watch variable 1 for viewdefs property updates
    this.unwatch = this.variableStore.watch(this.variableId, (_v, value, props) => {
      this.handleRootUpdate(value, props ?? {});
    }, false);
  }

  // Handle updates to variable 1
  private handleRootUpdate(_value: unknown, props: Record<string, string>): void {
    console.log('handleRootUpdate called, props:', Object.keys(props));

    // Check for viewdefs property (JSON string containing TYPE.NAMESPACE -> HTML mappings)
    // Per spec: "Variable 1 has a viewdefs property containing TYPE.NAMESPACE → HTML mappings"
    const viewdefsJson = props['viewdefs'];
    if (viewdefsJson) {
      console.log('Found viewdefs property, length:', viewdefsJson.length);
      try {
        const viewdefs = JSON.parse(viewdefsJson) as Record<string, string>;
        if (typeof viewdefs === 'object' && viewdefs !== null) {
          console.log('Parsed viewdefs, keys:', Object.keys(viewdefs));
          this.viewdefStore.processViewdefs(viewdefs);
        }
      } catch (e) {
        console.error('Failed to parse viewdefs property:', e);
      }
    } else {
      console.log('No viewdefs property found');
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
  binding?: BindingEngine
): AppView | null {
  const element = findAppElement();
  if (!element) {
    return null;
  }

  const appView = new AppView(element, viewdefStore, variableStore, binding);
  appView.initialize();
  return appView;
}
