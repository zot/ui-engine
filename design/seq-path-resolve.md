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
