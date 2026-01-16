# Sequence: Path Resolve

**Source Spec:** protocol.md, libraries.md
**Use Case:** Resolving path string to actual value

## Participants

- Caller: BindingEngine or PathNavigator
- PathSyntax: Path parser
- PathNavigator: Path resolver
- VariableStore: Standard variable lookup
- Object: Target data object

## Sequence

```
     Caller             PathSyntax           PathNavigator          VariableStore             Object
        |                      |                      |                      |                      |
        |---resolve(path)----->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parse(path)------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |<--segments-----------|                      |                      |
        |                      |                      |                      |                      |
        |     [if starts with @name]                  |                      |                      |
        |                      |---getStandard------->|                      |                      |
        |                      |   (@name)            |---getByName--------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--variable-----------|                      |
        |                      |                      |                      |                      |
        |                      |<--startValue---------|                      |                      |
        |                      |                      |                      |                      |
        |     [for each segment]                      |                      |                      |
        |                      |---navigateSegment--->|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [if currentValue is null/undefined]     |                      |
        |                      |<--null/undefined-----|   (nullish coalescing - stop traversal)    |
        |                      |                      |                      |                      |
        |                      |     [property: "name"]                      |                      |
        |                      |                      |---getProperty------->|                      |
        |                      |                      |                      |---obj.name---------->|
        |                      |                      |                      |                      |
        |                      |     [index: "2"]     |                      |                      |
        |                      |                      |---getIndex---------->|                      |
        |                      |                      |                      |---array[1]---------->|
        |                      |                      |                      |   (1-based)          |
        |                      |                      |                      |                      |
        |                      |     [method: "getName()"]                   |                      |
        |                      |                      |---callMethod-------->|                      |
        |                      |                      |                      |---invoke------------>|
        |                      |                      |                      |                      |
        |                      |     [parent: ".."]   |                      |                      |
        |                      |                      |---getParent--------->|                      |
        |                      |                      |                      |                      |
        |                      |<--finalValue---------|                      |                      |
        |                      |                      |                      |                      |
        |<--value--------------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Path segments: property, index (1-based), method call, parent (..)
- Standard variables accessed via @name prefix
- Method calls include parentheses: getName()
- Array indices are 1-based (Lua convention)
- Parent traversal navigates up the object tree
- Caching improves repeated resolution performance
- **Nullish coalescing:** If any segment resolves to null/undefined, traversal stops and returns null/undefined (no error)
- **Write direction:** When resolveForWrite encounters nullish intermediate, caller sends `error(varId, 'path-failure', description)` message. UI shows error indicator (e.g., `ui-error` class). Error clears on next successful update.
- **Read/write methods (Lua only):** Paths ending in `()` with `access=rw` call the method with no args on read, with value arg on write. Uses Lua's optional argument support.
