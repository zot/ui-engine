# Sequence: Navigate Page

**Source Spec:** main.md, libraries.md
**Use Case:** Navigating to a different page in the app

## Participants

- Trigger: User action or backend call
- AppPresenter: App state holder
- VariableStore: Variable storage
- SPANavigator: Frontend navigation
- ViewRenderer: View display

## Sequence

```
     Trigger           AppPresenter          VariableStore          SPANavigator          ViewRenderer
        |                      |                      |                      |                      |
        |---navigate(page)---->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---pushHistory------->|                      |                      |
        |                      |   (page)             |                      |                      |
        |                      |                      |                      |                      |
        |                      |---incrementIndex---->|                      |                      |
        |                      |                      |                      |                      |
        |                      |---setUrl(path)------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |---update(var,------->|                      |                      |
        |                      |    historyIndex,url) |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---notifyWatchers---->|                      |
        |                      |                      |                      |                      |
        |                      |                      |------update(historyIndex,url)------------->|
        |                      |                      |                      |                      |
        |                      |                      |                      |---pushState()------->|
        |                      |                      |                      |   (browser history)  |
        |                      |                      |                      |                      |
        |                      |                      |                      |---currentPage()----->|
        |                      |                      |                      |                      |
        |                      |                      |                      |<--page presenter-----|
        |                      |                      |                      |                      |
        |                      |                      |                      |---render()---------->|
        |                      |                      |                      |                      |
        |                      |                      |                      |                      |<--display
        |                      |                      |                      |                      |
```

## Notes

- Navigation updates history array and historyIndex
- URL property updated for browser address bar
- Frontend watches historyIndex and url variables
- Browser history updated via pushState/replaceState
- currentPage() returns history[historyIndex]
- ViewRenderer displays viewdef for page presenter type
