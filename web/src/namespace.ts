// Namespace resolution utilities
// Shared between ViewRenderer and View for consistent namespace handling
// CRC: crc-View.md
// Spec: viewdefs.md

import { VariableStore } from './connection';

/**
 * Resolves namespace by checking DOM hierarchy and parent variable.
 * Uses element.closest() to find nearest ui-namespace in DOM,
 * then compares with parent variable's namespace to determine which is "closer".
 */
export function resolveNamespace(
  element: HTMLElement,
  parentVarId: number,
  variableStore: VariableStore
): string | undefined {
  const closest = element.closest('[ui-namespace]') as HTMLElement | null;

  const parentData = variableStore.get(parentVarId);
  const parentNs = parentData?.properties['namespace'];
  const parentElementId = parentData?.properties['elementId'];
  const parentElement = parentElementId ? document.getElementById(parentElementId) : null;

  if (closest && parentElement && parentNs) {
    // Both exist: check containment
    // If closest is inside parent element, use closest's namespace
    // Otherwise, parent's namespace is "closer" in view hierarchy
    if (parentElement.contains(closest)) {
      return closest.getAttribute('ui-namespace') || undefined;
    } else {
      return parentNs;
    }
  } else if (closest) {
    // Only closest DOM element exists
    return closest.getAttribute('ui-namespace') || undefined;
  } else if (parentNs) {
    // Only parent variable namespace exists
    return parentNs;
  }

  return undefined;
}

/**
 * Builds namespace properties for a variable based on element and parent context.
 * Handles namespace, fallbackNamespace, and default access.
 */
export function buildNamespaceProperties(
  element: HTMLElement,
  contextVarId: number,
  properties: Record<string, string>,
  variableStore: VariableStore
): void {
  // 1. If element has ui-namespace, use it directly
  const elementNs = element.getAttribute('ui-namespace');
  if (elementNs) {
    properties['namespace'] = elementNs;
  } else {
    // 2. Otherwise resolve via DOM/parent hierarchy
    const namespace = resolveNamespace(element, contextVarId, variableStore);
    if (namespace) {
      properties['namespace'] = namespace;
    }
  }

  // 3. Inherit fallbackNamespace from parent if not already set
  const parentData = variableStore.get(contextVarId);
  if (!properties['fallbackNamespace'] && parentData?.properties['fallbackNamespace']) {
    properties['fallbackNamespace'] = parentData.properties['fallbackNamespace'];
  }

  // Default to read-only access for views/viewlists
  if (!properties['access']) {
    properties['access'] = 'r';
  }
}
