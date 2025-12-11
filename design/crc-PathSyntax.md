# PathSyntax

**Source Spec:** protocol.md, viewdefs.md

## Responsibilities

### Knows
- segments: Parsed path segments
- hasStandardPrefix: True if starts with @name
- hasUrlParams: True if contains ?key=value parameters
- hasMethodArg: True if method call has `_` argument

### Does
- parse: Split path string into segments and parameters
- getPropertyAccess: Extract simple property name segment
- getArrayIndex: Extract numeric index (1-based)
- getMethodCall: Extract method name with parentheses and optional argument
- isMethodCallWithArg: Check if method call has `_` argument (e.g., `method(_)`)
- isMethodCallNoArg: Check if method call has no argument (e.g., `method()`)
- getParentTraversal: Identify ".." segment
- getStandardVariable: Extract @name prefix
- getUrlParams: Parse ?create=Type&prop=value parameters
- toString: Reconstruct path string

## Notes

PathSyntax only parses path strings - it does not handle nullish value behavior.
Nullish coalescing during path traversal is handled by PathNavigator (see crc-PathNavigator.md).

**Method call syntax for ui-action:**
- `method()` - call with no arguments
- `method(_)` - call with the update message's value as the argument

## Collaborators

- PathNavigator: Uses for path resolution
- BindingEngine: Parses ui-* attribute values (including ui-action paths)
- Variable: Path stored in path property

## Sequences

- seq-path-resolve.md: Path parsing and resolution
- seq-bind-element.md: Binding path parsing
- seq-lua-handle-action.md: Action path parsing for method dispatch
