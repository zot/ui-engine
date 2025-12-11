// Viewdef client-side representation
// CRC: crc-Viewdef.md
// Spec: viewdefs.md

export interface Viewdef {
  type: string;
  namespace: string;
  template: HTMLTemplateElement;
}

// Parse a TYPE.NAMESPACE key
export function parseKey(key: string): { type: string; namespace: string } | null {
  const dotIndex = key.indexOf('.');
  if (dotIndex === -1) {
    return null;
  }
  return {
    type: key.substring(0, dotIndex),
    namespace: key.substring(dotIndex + 1),
  };
}

// Build a TYPE.NAMESPACE key
export function buildKey(type: string, namespace: string): string {
  return `${type}.${namespace}`;
}

// Parse viewdef HTML content into a template element
// Returns null if content is invalid (not a single template element)
export function parseViewdef(content: string): HTMLTemplateElement | null {
  // Create scratch div to parse HTML
  const scratch = document.createElement('div');
  scratch.innerHTML = content.trim();

  // Validate: exactly one element that is a template
  if (scratch.children.length !== 1) {
    return null;
  }

  const element = scratch.children[0];
  if (!(element instanceof HTMLTemplateElement)) {
    return null;
  }

  return element;
}

// Create a Viewdef from key and content
export function createViewdef(
  key: string,
  content: string
): Viewdef | null {
  const parsed = parseKey(key);
  if (!parsed) {
    return null;
  }

  const template = parseViewdef(content);
  if (!template) {
    return null;
  }

  return {
    type: parsed.type,
    namespace: parsed.namespace,
    template,
  };
}

// Clone the template contents for rendering
export function cloneViewdefContent(viewdef: Viewdef): DocumentFragment {
  return viewdef.template.content.cloneNode(true) as DocumentFragment;
}

// Get the key for a viewdef
export function getViewdefKey(viewdef: Viewdef): string {
  return buildKey(viewdef.type, viewdef.namespace);
}
