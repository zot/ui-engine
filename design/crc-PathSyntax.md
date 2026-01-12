# PathSyntax

**Source Spec:** protocol.md, viewdefs.md

## Responsibilities

### Knows
- segments: Parsed path segments
- hasStandardPrefix: True if starts with @name
- hasUrlParams: True if contains ?key=value parameters
- hasMethodArg: True if method call has `_` argument
- urlParams: Parsed URL parameters as key-value map

### Does
- parse: Split path string into segments and parameters
- getPropertyAccess: Extract simple property name segment
- getArrayIndex: Extract numeric index (1-based)
- getMethodCall: Extract method name with parentheses and optional argument
- isMethodCallWithArg: Check if method call has `_` argument (e.g., `method(_)`)
- isMethodCallNoArg: Check if method call has no argument (e.g., `method()`)
- getParentTraversal: Identify ".." segment
- getStandardVariable: Extract @name prefix
- getUrlParams: Parse ?key=value&key2=value2 parameters
- getPathWithoutParams: Return path without URL parameters
- toString: Reconstruct path string
- toVariableProperties: Convert URL params to variable properties map

## Notes

PathSyntax only parses path strings - it does not handle nullish value behavior.
Nullish coalescing during path traversal is handled by PathNavigator (see crc-PathNavigator.md).

**Method call syntax for ui-action:**
- `method()` - call with no arguments
- `method(_)` - call with the update message's value as the argument

**Method path access constraints:**
- Paths ending in `()` must have access `r` or `action`
- Paths ending in `(_)` must have access `w` or `action`

**Path property syntax:**

Paths can include properties that are set on the created variable:

```
contacts?wrapper=lua.ViewList&itemWrapper=ContactPresenter
```

- Properties after `?` are set on the created variable
- Uses URL query string syntax: `key=value&key2=value2`
- **Properties without values default to `true`:** `x?a&b` parses as `x?a=true&b=true`
- Common properties:
  - `wrapper` - Wrapper type for value transformation
  - `itemWrapper` - Item presenter type (for ViewList)
  - `create` - Type to instantiate as variable value
  - `keypress` - Boolean, controls input update timing (see crc-BindingEngine.md)
  - `scrollOnOutput` - Boolean, auto-scrolls element to bottom on update (see crc-ValueBinding.md, crc-View.md, crc-ViewList.md); for Views, child render notifications bubble up until an ancestor with `scrollOnOutput` is found

**Examples:**
- `name` - Simple property access
- `father.name` - Nested property access
- `contacts.1` - Array index (1-based)
- `@customers.2.name` - Standard variable prefix
- `getName()` - Method call
- `contacts?wrapper=lua.ViewList` - Path with wrapper property
- `contacts?itemWrapper=ContactPresenter&editable=true` - Multiple properties
- `name?keypress` - Property defaults to true (equivalent to `name?keypress=true`)
- `log?scrollOnOutput` - Auto-scroll element to bottom on value updates (ui-value)
- `chatLog?scrollOnOutput` - Auto-scroll view on render/re-render (ui-view); child renders bubble notification up
- `messages?wrapper=lua.ViewList&scrollOnOutput` - Auto-scroll list when items added (ui-view/ui-viewlist)

## Collaborators

- PathNavigator: Uses for path resolution
- BindingEngine: Parses ui-* attribute values (including ui-action paths)
- Variable: Path stored in path property, URL params become variable properties
- ViewList: Receives itemWrapper property from path params

## Sequences

- seq-path-resolve.md: Path parsing and resolution
- seq-bind-element.md: Binding path parsing
- seq-lua-handle-action.md: Action path parsing for method dispatch
- seq-create-variable.md: Path params converted to variable properties
