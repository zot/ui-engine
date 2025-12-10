# Sequence: SPA Navigate

**Source Spec:** libraries.md, interfaces.md
**Use Case:** Frontend handling SPA navigation

## Participants

- AppPresenter: Backend app state
- FrontendApp: Frontend application
- SPANavigator: Navigation manager
- ViewRenderer: View display
- Browser: Browser history API

## Sequence

```
     AppPresenter          FrontendApp          SPANavigator          ViewRenderer             Browser
        |                      |                      |                      |                      |
        |---update(historyIdx,-|                      |                      |                      |
        |    url)------------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---onUpdate---------->|                      |                      |
        |                      |   (historyIndex,url) |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---compareState------>|                      |
        |                      |                      |                      |                      |
        |                      |          [if url changed]                   |                      |
        |                      |                      |---updateUrl--------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if historyIndex changed forward]  |                      |
        |                      |                      |---pushState()------->|                      |
        |                      |                      |                      |---history.push------>|
        |                      |                      |                      |                      |
        |                      |          [if historyIndex same, url changed]|                      |
        |                      |                      |---replaceState()---->|                      |
        |                      |                      |                      |---history.replace--->|
        |                      |                      |                      |                      |
        |                      |                      |---currentPage()----->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--page presenter-----|                      |
        |                      |                      |                      |                      |
        |                      |                      |---render(page)------>|                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---display view------>|
        |                      |                      |                      |                      |
        |                      |                      |                      |                      |
        |     [user clicks back/forward]              |                      |                      |
        |                      |                      |                      |<--popstate-----------|
        |                      |                      |                      |                      |
        |                      |                      |<--handlePopState-----|                      |
        |                      |                      |                      |                      |
        |                      |                      |---go(newIndex)------>|                      |
        |                      |                      |                      |                      |
        |<--update(historyIdx)-|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Frontend watches historyIndex and url on AppPresenter
- Changes trigger pushState or replaceState
- currentPage() returns history[historyIndex]
- Browser back/forward triggers popstate
- Popstate updates historyIndex on AppPresenter
- Bidirectional sync between app state and browser history
