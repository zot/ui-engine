# Sequence: Load Viewdefs

**Source Spec:** viewdefs.md
**Use Case:** Loading viewdefs from variable 1 on frontend

## Participants

- FrontendApp: Frontend application
- VariableStore: Variable storage
- ViewdefStore: Viewdef storage
- BindingEngine: Binding processor

## Sequence

```
     FrontendApp         VariableStore          ViewdefStore          BindingEngine
        |                      |                      |                      |
        |---watch(1)---------->|                      |                      |
        |                      |                      |                      |
        |<--update(1,{viewdefs:|                      |                      |
        |    {T.V: html,...}}) |                      |                      |
        |                      |                      |                      |
        |---parseViewdefs()--->|                      |                      |
        |                      |                      |                      |
        |     [for each TYPE.VIEW in viewdefs]        |                      |
        |---store(T.V,html)--->|                      |                      |
        |                      |---store()----------->|                      |
        |                      |                      |                      |
        |                      |                      |---parseHtml()------->|
        |                      |                      |                      |
        |                      |                      |---extractBindings--->|
        |                      |                      |                      |
        |                      |                      |<--bindings-----------|
        |                      |                      |                      |
        |                      |<--viewdef-----------|                      |
        |                      |                      |                      |
        |     [subsequent updates replace viewdefs property]                 |
        |<--update(1,{viewdefs:|                      |                      |
        |    {NEW.V: html}})   |                      |                      |
        |                      |                      |                      |
        |---store(NEW.V,html)->|                      |                      |
        |                      |                      |                      |
```

## Notes

- Variable 1's viewdefs property contains TYPE.VIEW -> HTML mappings
- Frontend parses and stores viewdefs locally
- Previous viewdefs property values can be replaced (frontend stores separately)
- Viewdefs delivered when presenter type changes
- Backend batches viewdef updates and prioritizes delivery
- Bindings parsed and cached for efficient rendering
