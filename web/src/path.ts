// Path parsing and navigation for variable bindings
// CRC: crc-PathSyntax.md, crc-PathNavigator.md
// Spec: protocol.md

/** Segment type in a path */
export enum SegmentType {
  Property = 'property',   // Simple property: name
  Index = 'index',         // Array index: 1, 2 (1-based)
  Parent = 'parent',       // Parent traversal: ..
  Method = 'method',       // Method call: getName()
  Standard = 'standard',   // Standard variable: @name
}

/** A single segment in a parsed path */
export interface Segment {
  type: SegmentType;
  value: string;  // property name, method name, or @name (without @)
  index?: number; // for Index type (1-based)
}

/** Parsed path with segments and optional URL params */
export interface ParsedPath {
  segments: Segment[];
  urlParams: URLSearchParams;
  hasStandard: boolean;
  standardName: string | null;
  raw: string;
}

const METHOD_PATTERN = /^([a-zA-Z_][a-zA-Z0-9_]*)\(\)$/;
const STANDARD_PATTERN = /^@([a-zA-Z_][a-zA-Z0-9_]*)$/;
const INDEX_PATTERN = /^[1-9][0-9]*$/;

/**
 * Parse a path string into segments.
 * Examples:
 *   - "name" -> property access
 *   - "father.name" -> property.property
 *   - "@customers.2.name" -> standard.index.property
 *   - "getName()" -> method call
 *   - ".." -> parent traversal
 *   - "path?create=Type" -> path with URL params
 */
export function parsePath(pathStr: string): ParsedPath {
  const result: ParsedPath = {
    segments: [],
    urlParams: new URLSearchParams(),
    hasStandard: false,
    standardName: null,
    raw: pathStr,
  };

  if (!pathStr) {
    return result;
  }

  // Split off URL parameters
  let pathPart = pathStr;
  const queryIdx = pathStr.indexOf('?');
  if (queryIdx !== -1) {
    pathPart = pathStr.substring(0, queryIdx);
    result.urlParams = new URLSearchParams(pathStr.substring(queryIdx + 1));
  }

  if (!pathPart) {
    return result;
  }

  // Split on dots
  const parts = pathPart.split('.');

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    if (!part) continue;

    // Check for standard variable (@name) - only valid at start
    if (i === 0 && part.startsWith('@')) {
      const match = part.match(STANDARD_PATTERN);
      if (match) {
        result.segments.push({
          type: SegmentType.Standard,
          value: match[1], // without @
        });
        result.hasStandard = true;
        result.standardName = match[1];
        continue;
      }
    }

    // Check for parent traversal
    if (part === '..') {
      result.segments.push({
        type: SegmentType.Parent,
        value: '..',
      });
      continue;
    }

    // Check for method call
    const methodMatch = part.match(METHOD_PATTERN);
    if (methodMatch) {
      result.segments.push({
        type: SegmentType.Method,
        value: methodMatch[1],
      });
      continue;
    }

    // Check for array index (1-based)
    if (INDEX_PATTERN.test(part)) {
      result.segments.push({
        type: SegmentType.Index,
        value: part,
        index: parseInt(part, 10),
      });
      continue;
    }

    // Default: property access
    result.segments.push({
      type: SegmentType.Property,
      value: part,
    });
  }

  return result;
}

/**
 * Convert parsed path back to string.
 */
export function pathToString(path: ParsedPath): string {
  if (path.segments.length === 0 && path.urlParams.toString() === '') {
    return '';
  }

  const parts = path.segments.map((seg) => {
    switch (seg.type) {
      case SegmentType.Standard:
        return '@' + seg.value;
      case SegmentType.Parent:
        return '..';
      case SegmentType.Method:
        return seg.value + '()';
      case SegmentType.Index:
        return String(seg.index);
      case SegmentType.Property:
        return seg.value;
    }
  });

  let result = parts.join('.');
  const params = path.urlParams.toString();
  if (params) {
    result += '?' + params;
  }
  return result;
}

/** Result of path resolution */
export interface ResolveResult {
  value: unknown;
  found: boolean;
}

/** Result of resolveForWrite */
export interface WriteTarget {
  parent: unknown;
  key: string | number;
  found: boolean;
}

/**
 * Navigate a path to get a value from an object.
 * @param root Starting object
 * @param path Parsed path or path string
 * @param standardResolver Optional function to resolve @name to a value
 */
export function resolve(
  root: unknown,
  path: ParsedPath | string,
  standardResolver?: (name: string) => unknown
): ResolveResult {
  const parsed = typeof path === 'string' ? parsePath(path) : path;

  if (parsed.segments.length === 0) {
    return { value: root, found: true };
  }

  let current: unknown = root;

  for (const segment of parsed.segments) {
    if (current === null || current === undefined) {
      return { value: undefined, found: false };
    }

    switch (segment.type) {
      case SegmentType.Standard:
        if (standardResolver) {
          current = standardResolver(segment.value);
        } else {
          return { value: undefined, found: false };
        }
        break;

      case SegmentType.Property:
        if (typeof current === 'object' && current !== null) {
          current = (current as Record<string, unknown>)[segment.value];
        } else {
          return { value: undefined, found: false };
        }
        break;

      case SegmentType.Index:
        if (Array.isArray(current) && segment.index !== undefined) {
          // 1-based indexing
          current = current[segment.index - 1];
        } else {
          return { value: undefined, found: false };
        }
        break;

      case SegmentType.Method:
        if (typeof current === 'object' && current !== null) {
          const method = (current as Record<string, unknown>)[segment.value];
          if (typeof method === 'function') {
            current = method.call(current);
          } else {
            return { value: undefined, found: false };
          }
        } else {
          return { value: undefined, found: false };
        }
        break;

      case SegmentType.Parent:
        // Parent traversal requires context not available here
        // Should be handled by the caller with variable tree knowledge
        return { value: undefined, found: false };
    }
  }

  return { value: current, found: true };
}

/**
 * Navigate a path and return parent + key for setting a value.
 * Does not navigate the final segment.
 */
export function resolveForWrite(
  root: unknown,
  path: ParsedPath | string,
  standardResolver?: (name: string) => unknown
): WriteTarget {
  const parsed = typeof path === 'string' ? parsePath(path) : path;

  if (parsed.segments.length === 0) {
    return { parent: null, key: '', found: false };
  }

  if (parsed.segments.length === 1) {
    const seg = parsed.segments[0];
    if (seg.type === SegmentType.Property) {
      return { parent: root, key: seg.value, found: true };
    }
    if (seg.type === SegmentType.Index && seg.index !== undefined) {
      return { parent: root, key: seg.index - 1, found: true };
    }
    return { parent: null, key: '', found: false };
  }

  // Navigate all but the last segment
  const parentPath: ParsedPath = {
    ...parsed,
    segments: parsed.segments.slice(0, -1),
  };

  const parentResult = resolve(root, parentPath, standardResolver);
  if (!parentResult.found) {
    return { parent: null, key: '', found: false };
  }

  const lastSeg = parsed.segments[parsed.segments.length - 1];
  if (lastSeg.type === SegmentType.Property) {
    return { parent: parentResult.value, key: lastSeg.value, found: true };
  }
  if (lastSeg.type === SegmentType.Index && lastSeg.index !== undefined) {
    return { parent: parentResult.value, key: lastSeg.index - 1, found: true };
  }

  return { parent: null, key: '', found: false };
}

/**
 * Check if a path starts with a standard variable reference.
 */
export function hasStandardVariable(path: ParsedPath | string): boolean {
  const parsed = typeof path === 'string' ? parsePath(path) : path;
  return parsed.hasStandard;
}

/**
 * Get the standard variable name from a path (without @).
 */
export function getStandardVariable(path: ParsedPath | string): string | null {
  const parsed = typeof path === 'string' ? parsePath(path) : path;
  return parsed.standardName;
}
