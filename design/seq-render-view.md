# Sequence: Render View

**Source Spec:** libraries.md
**Use Case:** Rendering viewdef for current page presenter

## Participants

- SPANavigator: Navigation trigger
- ViewRenderer: View display manager
- ViewdefStore: Viewdef storage
- BindingEngine: Binding processor
- WidgetBinder: Widget-specific bindings

## Sequence

```
     SPANavigator          ViewRenderer          ViewdefStore          BindingEngine          WidgetBinder
        |                      |                      |                      |                      |
        |---render(presenter)->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---getType()--------->|                      |                      |
        |                      |   (from presenter)   |                      |                      |
        |                      |                      |                      |                      |
        |                      |---get(TYPE.VIEW)---->|                      |                      |
        |                      |                      |                      |                      |
        |                      |<--viewdef------------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---clear()----------->|                      |                      |
        |                      |   (remove old view)  |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseHtml()------->|                      |                      |
        |                      |   (viewdef.html)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |---createElements---->|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each element with ui-*]            |                      |
        |                      |---bind(element)----->|                      |                      |
        |                      |                      |---processBindings--->|                      |
        |                      |                      |                      |                      |
        |                      |                      |     [for special widgets]                   |
        |                      |                      |---bindWidget-------->|                      |
        |                      |                      |                      |---apply bindings---->|
        |                      |                      |                      |                      |
        |                      |     [handle nested views]                   |                      |
        |                      |---renderNested------>|                      |                      |
        |                      |   (ui-view elements) |                      |                      |
        |                      |                      |                      |                      |
        |                      |     [handle view lists]                     |                      |
        |                      |---renderViewList---->|                      |                      |
        |                      |   (ui-viewlist)      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---appendToDOM------->|                      |                      |
        |                      |                      |                      |                      |
        |<--complete-----------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- View name defaults to "DEFAULT" if not specified
- Old view content cleared before new render
- ui-view renders single nested object with its viewdef
- ui-viewlist renders array of objects
- ui-content renders raw HTML
- ui-viewdef renders computed viewdef string
- Widget bindings handle Shoelace and Tabulator components
