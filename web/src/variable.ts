// Variable client-side representation
// CRC: crc-Variable.md
// Spec: protocol.md

export interface Variable {
  id: number;
  parentId?: number;
  value: unknown;
  properties: Record<string, string>;
  unbound?: boolean;
}

export interface ObjectReference {
  obj: number;
}

export function isObjectReference(value: unknown): value is ObjectReference {
  return (
    typeof value === 'object' &&
    value !== null &&
    'obj' in value &&
    typeof (value as ObjectReference).obj === 'number'
  );
}

export function getObjectReferenceId(value: unknown): number | null {
  if (isObjectReference(value)) {
    return value.obj;
  }
  return null;
}
