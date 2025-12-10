# PathSyntax

**Source Spec:** protocol.md, viewdefs.md

## Responsibilities

### Knows
- segments: Parsed path segments
- hasStandardPrefix: True if starts with @name
- hasUrlParams: True if contains ?key=value parameters

### Does
- parse: Split path string into segments and parameters
- getPropertyAccess: Extract simple property name segment
- getArrayIndex: Extract numeric index (1-based)
- getMethodCall: Extract method name with parentheses
- getParentTraversal: Identify ".." segment
- getStandardVariable: Extract @name prefix
- getUrlParams: Parse ?create=Type&prop=value parameters
- toString: Reconstruct path string

## Collaborators

- PathNavigator: Uses for path resolution
- BindingEngine: Parses ui-* attribute values
- Variable: Path stored in path property

## Sequences

- seq-path-resolve.md: Path parsing and resolution
- seq-bind-element.md: Binding path parsing
