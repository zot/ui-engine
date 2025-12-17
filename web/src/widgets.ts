// Widget registry for custom element handlers
// CRC: crc-WidgetBinder.md
// Spec: components.md, viewdefs.md
//
// NOTE: The binding engine (binding.ts) handles all standard bindings
// (ui-value, ui-attr-*, ui-class-*, ui-style-*-*, ui-action, ui-event-*)
// by creating child variables with server-side path resolution.
// See: specs/viewdefs.md - Path Resolution: Server-Side Only
//
// This module provides a registry for widget-specific handlers that need
// custom behavior beyond what the binding engine provides. Widget handlers
// receive the already-resolved variable value (not a path to resolve).

import { Variable } from './variable';

/** Widget binding handler function */
export type WidgetHandler = (
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
) => (() => void) | void; // Returns optional cleanup function

/** Registered widget handlers by tag name */
const widgetHandlers: Map<string, WidgetHandler> = new Map();

/**
 * Register a widget handler for a tag name.
 * Widget handlers are for truly custom behavior that can't be handled
 * by the standard binding engine.
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
 * @returns cleanup function if handler was found and applied, undefined otherwise
 */
export function bindWidget(
  element: Element,
  variable: Variable,
  bindings: Map<string, string>
): (() => void) | undefined {
  const handler = widgetHandlers.get(element.tagName.toLowerCase());
  if (handler) {
    const cleanup = handler(element, variable, bindings);
    return cleanup || undefined;
  }
  return undefined;
}

// ============================================================================
// NOTE: No default handlers are registered.
//
// The binding engine (binding.ts) handles Shoelace elements directly:
// - sl-input/sl-textarea: ui-value with sl-input/sl-change events
// - sl-checkbox/sl-switch: ui-value with checked property
// - sl-select: ui-value with sl-change events
// - sl-button: ui-action with click events
// - All elements: ui-attr-*, ui-class-*, ui-style-*-* via child variables
//
// Register custom handlers here only for widgets that need behavior
// the binding engine can't provide.
// ============================================================================
