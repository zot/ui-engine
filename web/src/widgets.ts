// Widget-specific bindings for Shoelace and Tabulator
// CRC: crc-WidgetBinder.md
// Spec: components.md, viewdefs.md

import { Variable } from './variable';

/** Widget binding handler function */
export type WidgetHandler = (
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
) => void;

/** Registered widget handlers by tag name */
const widgetHandlers: Map<string, WidgetHandler> = new Map();

/**
 * Register a widget handler for a tag name.
 */
export function registerWidget(tagName: string, handler: WidgetHandler): void {
  widgetHandlers.set(tagName.toLowerCase(), handler);
}

/**
 * Get the handler for a widget tag.
 */
export function getWidgetHandler(tagName: string): WidgetHandler | undefined {
  return widgetHandlers.get(tagName.toLowerCase());
}

/**
 * Check if a tag has a registered widget handler.
 */
export function hasWidgetHandler(tagName: string): boolean {
  return widgetHandlers.has(tagName.toLowerCase());
}

/**
 * Apply widget-specific bindings to an element.
 * @returns true if a handler was found and applied
 */
export function bindWidget(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): boolean {
  const handler = widgetHandlers.get(element.tagName.toLowerCase());
  if (handler) {
    handler(element, variable, bindings);
    return true;
  }
  return false;
}

// ============================================================================
// Shoelace Bindings
// ============================================================================

/** Bind sl-input element */
function bindShoelaceInput(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  const input = element as HTMLInputElement & { value: string };

  // Handle ui-value
  const valuePath = bindings.get('ui-value');
  if (valuePath && variable.value !== undefined) {
    input.value = String(variable.value);
  }

  // Handle ui-attr-disabled
  const disabledPath = bindings.get('ui-attr-disabled');
  if (disabledPath) {
    const disabled = resolveBindingValue(variable, disabledPath);
    if (disabled) {
      input.setAttribute('disabled', '');
    } else {
      input.removeAttribute('disabled');
    }
  }

  // Handle ui-attr-placeholder
  const placeholderPath = bindings.get('ui-attr-placeholder');
  if (placeholderPath) {
    const placeholder = resolveBindingValue(variable, placeholderPath);
    if (placeholder !== undefined) {
      input.setAttribute('placeholder', String(placeholder));
    }
  }

  // Listen for sl-change event (fires on blur/commit - when user tabs out)
  element.addEventListener('sl-change', ((e: CustomEvent) => {
    const target = e.target as HTMLInputElement;
    // Fire change event for binding engine to handle
    element.dispatchEvent(
      new CustomEvent('ui-value-change', {
        detail: { value: target.value, path: valuePath },
        bubbles: true,
      })
    );
  }) as EventListener);
}

/** Bind sl-textarea element */
function bindShoelaceTextarea(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  // Same as input
  bindShoelaceInput(element, variable, bindings);
}

/** Bind sl-button element */
function bindShoelaceButton(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  // Handle ui-action
  const actionPath = bindings.get('ui-action');
  if (actionPath) {
    element.addEventListener('click', () => {
      element.dispatchEvent(
        new CustomEvent('ui-action-trigger', {
          detail: { action: actionPath },
          bubbles: true,
        })
      );
    });
  }

  // Handle ui-attr-disabled
  const disabledPath = bindings.get('ui-attr-disabled');
  if (disabledPath) {
    const disabled = resolveBindingValue(variable, disabledPath);
    if (disabled) {
      element.setAttribute('disabled', '');
    } else {
      element.removeAttribute('disabled');
    }
  }

  // Handle ui-attr-loading
  const loadingPath = bindings.get('ui-attr-loading');
  if (loadingPath) {
    const loading = resolveBindingValue(variable, loadingPath);
    if (loading) {
      element.setAttribute('loading', '');
    } else {
      element.removeAttribute('loading');
    }
  }
}

/** Bind sl-select element */
function bindShoelaceSelect(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  const select = element as HTMLSelectElement & { value: string };

  // Handle ui-value for selected value
  const valuePath = bindings.get('ui-value');
  if (valuePath && variable.value !== undefined) {
    select.value = String(variable.value);
  }

  // Listen for sl-change event
  element.addEventListener('sl-change', ((e: CustomEvent) => {
    const target = e.target as HTMLSelectElement;
    element.dispatchEvent(
      new CustomEvent('ui-value-change', {
        detail: { value: target.value, path: valuePath },
        bubbles: true,
      })
    );
  }) as EventListener);
}

/** Bind sl-checkbox element */
function bindShoelaceCheckbox(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  const checkbox = element as HTMLInputElement & { checked: boolean };

  // Handle ui-value for checked state
  const valuePath = bindings.get('ui-value');
  if (valuePath && variable.value !== undefined) {
    checkbox.checked = Boolean(variable.value);
  }

  // Listen for sl-change event
  element.addEventListener('sl-change', ((e: CustomEvent) => {
    const target = e.target as HTMLInputElement;
    element.dispatchEvent(
      new CustomEvent('ui-value-change', {
        detail: { value: target.checked, path: valuePath },
        bubbles: true,
      })
    );
  }) as EventListener);
}

/** Bind sl-switch element */
function bindShoelaceSwitch(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  // Same as checkbox
  bindShoelaceCheckbox(element, variable, bindings);
}

/** Bind sl-radio-group element */
function bindShoelaceRadioGroup(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  const radioGroup = element as HTMLElement & { value: string };

  // Handle ui-value for selected value
  const valuePath = bindings.get('ui-value');
  if (valuePath && variable.value !== undefined) {
    radioGroup.value = String(variable.value);
  }

  // Listen for sl-change event
  element.addEventListener('sl-change', ((e: CustomEvent) => {
    const target = e.target as HTMLElement & { value: string };
    element.dispatchEvent(
      new CustomEvent('ui-value-change', {
        detail: { value: target.value, path: valuePath },
        bubbles: true,
      })
    );
  }) as EventListener);
}

/** Bind sl-option element (used in selects) */
function bindShoelaceOption(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  // Handle ui-attr-value for option value
  const valueAttrPath = bindings.get('ui-attr-value');
  if (valueAttrPath) {
    const value = resolveBindingValue(variable, valueAttrPath);
    if (value !== undefined) {
      element.setAttribute('value', String(value));
    }
  }
}

/** Bind sl-dialog element */
function bindShoelaceDialog(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  const dialog = element as HTMLElement & { open: boolean };

  // Handle ui-attr-open
  const openPath = bindings.get('ui-attr-open');
  if (openPath) {
    const open = resolveBindingValue(variable, openPath);
    dialog.open = Boolean(open);
  }
}

// ============================================================================
// Tabulator Bindings
// ============================================================================

/** Bind tabulator grid element */
function bindTabulator(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  // Tabulator requires the tabulator library to be loaded
  // This is a placeholder that creates the grid configuration

  const columnsPath = bindings.get('ui-columns');
  const dataPath = bindings.get('ui-value');

  // Store configuration for later initialization
  (element as HTMLElement).dataset.tabulatorConfig = JSON.stringify({
    columns: columnsPath,
    data: dataPath,
  });

  // Dispatch event for tabulator initialization
  element.dispatchEvent(
    new CustomEvent('ui-tabulator-init', {
      detail: {
        columns: columnsPath,
        data: variable.value,
      },
      bubbles: true,
    })
  );
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Resolve a binding path to a value from the variable.
 */
function resolveBindingValue(variable: Variable, path: string): unknown {
  if (!path) return undefined;

  // Simple case: direct property on value
  if (typeof variable.value === 'object' && variable.value !== null) {
    const parts = path.split('.');
    let current: unknown = variable.value;

    for (const part of parts) {
      if (current === null || current === undefined) {
        return undefined;
      }
      if (typeof current === 'object') {
        current = (current as Record<string, unknown>)[part];
      } else {
        return undefined;
      }
    }
    return current;
  }

  // If the path matches the whole value
  if (!path.includes('.')) {
    return variable.value;
  }

  return undefined;
}

/**
 * Update widget when variable changes.
 */
export function updateWidget(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): void {
  // Re-apply widget bindings with new variable value
  bindWidget(element, variable, bindings);
}

// ============================================================================
// Register Default Handlers
// ============================================================================

// Shoelace components
registerWidget('sl-input', bindShoelaceInput);
registerWidget('sl-textarea', bindShoelaceTextarea);
registerWidget('sl-button', bindShoelaceButton);
registerWidget('sl-select', bindShoelaceSelect);
registerWidget('sl-checkbox', bindShoelaceCheckbox);
registerWidget('sl-switch', bindShoelaceSwitch);
registerWidget('sl-radio-group', bindShoelaceRadioGroup);
registerWidget('sl-option', bindShoelaceOption);
registerWidget('sl-dialog', bindShoelaceDialog);

// Tabulator (requires data-tabulator attribute or similar)
registerWidget('div[data-tabulator]', bindTabulator);
