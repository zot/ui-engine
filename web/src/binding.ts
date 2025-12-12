// Binding engine for ui-* attributes
// CRC: crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md
// Spec: viewdefs.md

import { VariableStore, VariableError } from './connection';

export interface Binding {
  element: Element;
  unbind: () => void;
}

export interface PathOptions {
  create?: string;
  wrapper?: string;
  item?: string;
  props?: Record<string, string>;
}

export interface ParsedPath {
  segments: string[];
  options: PathOptions;
}

// Parse a path like "father.name?create=Person&wrapper=ViewList&item=ContactPresenter"
// Spec: protocol.md - Path property syntax
export function parsePath(path: string): ParsedPath {
  const [pathPart, queryPart] = path.split('?');
  const segments = pathPart.split('.');
  const options: PathOptions = {};

  if (queryPart) {
    const params = new URLSearchParams(queryPart);

    // Extract well-known properties
    if (params.has('create')) {
      options.create = params.get('create')!;
    }
    if (params.has('wrapper')) {
      options.wrapper = params.get('wrapper')!;
    }
    if (params.has('item')) {
      options.item = params.get('item')!;
    }

    // Collect remaining properties
    const props: Record<string, string> = {};
    params.forEach((value, key) => {
      if (key !== 'create' && key !== 'wrapper' && key !== 'item') {
        props[key] = value;
      }
    });
    if (Object.keys(props).length > 0) {
      options.props = props;
    }
  }

  return { segments, options };
}

// Convert path options to variable properties map
// Used when creating variables from paths with properties
export function pathOptionsToProperties(options: PathOptions): Record<string, string> {
  const props: Record<string, string> = {};

  if (options.create) {
    props['create'] = options.create;
  }
  if (options.wrapper) {
    props['wrapper'] = options.wrapper;
  }
  if (options.item) {
    props['item'] = options.item;
  }
  if (options.props) {
    Object.assign(props, options.props);
  }

  return props;
}

// Resolve a path against a variable value
export function resolvePath(value: unknown, segments: string[]): unknown {
  let current = value;
  for (const segment of segments) {
    if (current === null || current === undefined) {
      return undefined;
    }
    if (typeof current === 'object') {
      // Handle array index
      if (Array.isArray(current) && /^\d+$/.test(segment)) {
        current = current[parseInt(segment, 10)];
      } else {
        current = (current as Record<string, unknown>)[segment];
      }
    } else {
      return undefined;
    }
  }
  return current;
}

export class BindingEngine {
  private store: VariableStore;
  private bindings: Map<Element, Binding[]> = new Map();

  constructor(store: VariableStore) {
    this.store = store;
  }

  // Bind all ui-* attributes on an element
  bindElement(element: Element, contextVarId: number): void {
    const elementBindings: Binding[] = [];

    // ui-value binding
    const uiValue = element.getAttribute('ui-value');
    if (uiValue) {
      const binding = this.createValueBinding(element, contextVarId, uiValue);
      if (binding) elementBindings.push(binding);
    }

    // ui-attr-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-attr-')) {
        const targetAttr = attr.name.substring(8); // Remove "ui-attr-"
        const binding = this.createAttrBinding(element, contextVarId, attr.value, targetAttr);
        if (binding) elementBindings.push(binding);
      }
    }

    // ui-class-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-class-')) {
        const className = attr.name.substring(9); // Remove "ui-class-"
        const binding = this.createClassBinding(element, contextVarId, attr.value, className);
        if (binding) elementBindings.push(binding);
      }
    }

    // ui-style-*-* bindings (e.g., ui-style-background-color)
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-style-')) {
        const styleProp = attr.name.substring(9); // Remove "ui-style-"
        const binding = this.createStyleBinding(element, contextVarId, attr.value, styleProp);
        if (binding) elementBindings.push(binding);
      }
    }

    // ui-event-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-event-')) {
        const eventName = attr.name.substring(9); // Remove "ui-event-"
        const binding = this.createEventBinding(element, contextVarId, attr.value, eventName);
        if (binding) elementBindings.push(binding);
      }
    }

    // ui-action binding (shorthand for click action)
    const uiAction = element.getAttribute('ui-action');
    if (uiAction) {
      const binding = this.createActionBinding(element, contextVarId, uiAction);
      if (binding) elementBindings.push(binding);
    }

    if (elementBindings.length > 0) {
      this.bindings.set(element, elementBindings);
    }

    // Recursively bind children
    for (const child of Array.from(element.children)) {
      this.bindElement(child, contextVarId);
    }
  }

  // Unbind all bindings from an element and its children
  unbindElement(element: Element): void {
    const elementBindings = this.bindings.get(element);
    if (elementBindings) {
      elementBindings.forEach((b) => b.unbind());
      this.bindings.delete(element);
    }

    for (const child of Array.from(element.children)) {
      this.unbindElement(child);
    }
  }

  // Create a value binding (sets textContent or value, and handles changes)
  // Spec: viewdefs.md - Nullish path handling with error indicators
  private createValueBinding(element: Element, varId: number, path: string): Binding | null {
    const parsed = parsePath(path);

    const update = (value: unknown) => {
      const resolved = resolvePath(value, parsed.segments);
      if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
        element.value = resolved?.toString() ?? '';
      } else if (element instanceof HTMLSelectElement) {
        element.value = resolved?.toString() ?? '';
      } else {
        element.textContent = resolved?.toString() ?? '';
      }
    };

    // Handle error state changes - add/remove ui-error class and data-ui-error-* attributes
    const updateError = (error: VariableError | null) => {
      if (error) {
        element.classList.add('ui-error');
        element.setAttribute('data-ui-error-code', error.code);
        element.setAttribute('data-ui-error-description', error.description);
      } else {
        element.classList.remove('ui-error');
        element.removeAttribute('data-ui-error-code');
        element.removeAttribute('data-ui-error-description');
      }
    };

    const unbindValue = this.store.watch(varId, (value) => update(value));
    const unbindError = this.store.watchErrors(varId, updateError);

    // Initial update
    const current = this.store.get(varId);
    if (current) update(current.value);

    // Two-way binding: listen for ui-value-change events from widgets
    const changeHandler = (e: Event) => {
      const customEvent = e as CustomEvent;
      const { value, path: changePath } = customEvent.detail;
      if (changePath === path) {
        // Send update to server with the new value at the path
        this.updateValueAtPath(varId, parsed.segments, value);
      }
    };
    element.addEventListener('ui-value-change', changeHandler);

    return {
      element,
      unbind: () => {
        unbindValue();
        unbindError();
        element.removeEventListener('ui-value-change', changeHandler);
        // Clean up error state on unbind
        element.classList.remove('ui-error');
        element.removeAttribute('data-ui-error-code');
        element.removeAttribute('data-ui-error-description');
      },
    };
  }

  // Update a value at a path within a variable
  // Spec: viewdefs.md - Nullish path handling
  private updateValueAtPath(varId: number, segments: string[], newValue: unknown): void {
    const current = this.store.get(varId);
    if (!current) return;

    if (segments.length === 0) {
      // Direct value update
      this.store.update(varId, newValue);
      return;
    }

    // Clone the value and update at path
    const value = JSON.parse(JSON.stringify(current.value ?? {}));
    let target = value;

    for (let i = 0; i < segments.length - 1; i++) {
      const segment = segments[i];
      // Nullish path handling: if intermediate segment is null/undefined,
      // send error message with path-failure code (allows UI to show error indicator)
      if (target[segment] === null || target[segment] === undefined) {
        const pathStr = segments.slice(0, i + 1).join('.');
        this.store.sendError(varId, 'path-failure', `Cannot write to path: '${pathStr}' is ${target[segment]}`);
        return;
      }
      target = target[segment];
    }

    target[segments[segments.length - 1]] = newValue;
    this.store.update(varId, value);
  }

  // Create an attribute binding
  private createAttrBinding(
    element: Element,
    varId: number,
    path: string,
    targetAttr: string
  ): Binding | null {
    const parsed = parsePath(path);

    const update = (value: unknown) => {
      const resolved = resolvePath(value, parsed.segments);
      if (resolved !== null && resolved !== undefined && resolved !== false) {
        element.setAttribute(targetAttr, resolved.toString());
      } else {
        element.removeAttribute(targetAttr);
      }
    };

    const unbind = this.store.watch(varId, (value) => update(value));

    const current = this.store.get(varId);
    if (current) update(current.value);

    return { element, unbind };
  }

  // Create a class binding
  private createClassBinding(
    element: Element,
    varId: number,
    path: string,
    className: string
  ): Binding | null {
    const parsed = parsePath(path);

    const update = (value: unknown) => {
      const resolved = resolvePath(value, parsed.segments);
      if (resolved) {
        element.classList.add(className);
      } else {
        element.classList.remove(className);
      }
    };

    const unbind = this.store.watch(varId, (value) => update(value));

    const current = this.store.get(varId);
    if (current) update(current.value);

    return { element, unbind };
  }

  // Create a style binding
  private createStyleBinding(
    element: Element,
    varId: number,
    path: string,
    styleProp: string
  ): Binding | null {
    const parsed = parsePath(path);
    const htmlElement = element as HTMLElement;

    const update = (value: unknown) => {
      const resolved = resolvePath(value, parsed.segments);
      if (resolved !== null && resolved !== undefined) {
        htmlElement.style.setProperty(styleProp, resolved.toString());
      } else {
        htmlElement.style.removeProperty(styleProp);
      }
    };

    const unbind = this.store.watch(varId, (value) => update(value));

    const current = this.store.get(varId);
    if (current) update(current.value);

    return { element, unbind };
  }

  // Create an event binding
  private createEventBinding(
    element: Element,
    varId: number,
    actionExpr: string,
    eventName: string
  ): Binding | null {
    const handler = (event: Event) => {
      this.executeAction(varId, actionExpr, event);
    };

    element.addEventListener(eventName, handler);

    return {
      element,
      unbind: () => element.removeEventListener(eventName, handler),
    };
  }

  // Create an action binding (click)
  private createActionBinding(element: Element, varId: number, actionExpr: string): Binding | null {
    const handler = (event: Event) => {
      event.preventDefault();
      this.executeAction(varId, actionExpr, event);
    };

    element.addEventListener('click', handler);

    return {
      element,
      unbind: () => element.removeEventListener('click', handler),
    };
  }

  // Execute an action expression like "submit()" or "deleteItem(id)"
  private executeAction(varId: number, actionExpr: string, _event: Event): void {
    // Parse action expression: methodName(args)
    const match = actionExpr.match(/^(\w+)\((.*)\)$/);
    if (!match) {
      console.error('Invalid action expression:', actionExpr);
      return;
    }

    const [, methodName, argsStr] = match;
    const args = argsStr ? argsStr.split(',').map((s) => s.trim()) : [];

    // For now, send an update with the action in properties
    this.store.update(varId, undefined, {
      action: methodName,
      'action-args': JSON.stringify(args),
    });
  }
}
