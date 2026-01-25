// Viewdef client-side representation
// CRC: crc-Viewdef.md
// Spec: viewdefs.md

export interface Viewdef {
  key: string;  // TYPE.NAMESPACE key for hot-reload targeting
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
    key,
    type: parsed.type,
    namespace: parsed.namespace,
    template,
  };
}

// Clone the template contents for rendering
export function cloneViewdefContent(viewdef: Viewdef): DocumentFragment {
  return viewdef.template.content.cloneNode(true) as DocumentFragment;
}

// Collect script elements from a fragment or element
// Spec: viewdefs.md - Render process step 4
export function collectScripts(container: DocumentFragment | Element): HTMLScriptElement[] {
  return Array.from(container.querySelectorAll('script')) as HTMLScriptElement[];
}

// Activate script elements by replacing with new ones
// Cloned scripts don't execute; creating new script elements triggers execution
// Spec: viewdefs.md - Render process step 7
export function activateScripts(scripts: HTMLScriptElement[]): void {
  for (const original of scripts) {
    const newScript = document.createElement('script');
    newScript.type = 'text/javascript';
    newScript.textContent = original.textContent;
    if (original.id) {
      newScript.id = original.id;
    }
    original.replaceWith(newScript);
  }
}

// Get the key for a viewdef
export function getViewdefKey(viewdef: Viewdef): string {
  return buildKey(viewdef.type, viewdef.namespace);
}
