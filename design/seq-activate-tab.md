# Sequence: Activate Tab

**Source Spec:** interfaces.md
**Use Case:** Opening session URL brings existing tab to focus

## Participants

- NewTab: Newly opened browser tab
- SharedWorker: Tab coordination worker
- MainTab: Primary connected tab
- Browser: Browser notification API

## Sequence

```
     NewTab            SharedWorker             MainTab               Browser
        |                      |                      |                      |
        |---connect(sessionId)->|                      |                      |
        |                      |                      |                      |
        |                      |---hasMainTab?------->|                      |
        |                      |                      |                      |
        |                      |<--yes----------------|                      |
        |                      |                      |                      |
        |                      |     [check URL path] |                      |
        |                      |---hasPath?---------->|                      |
        |                      |                      |                      |
        |                      |          [if has path]                      |
        |                      |---navigateTo(path)-->|                      |
        |                      |                      |                      |
        |                      |---requestFocus------>|                      |
        |                      |                      |                      |
        |                      |                      |---Notification.show->|
        |                      |                      |   "Click to focus"   |
        |                      |                      |                      |
        |                      |                      |     [user clicks]    |
        |                      |                      |<--notification click-|
        |                      |                      |                      |
        |                      |                      |---window.focus()---->|
        |                      |                      |                      |
        |                      |<--focused------------|                      |
        |                      |                      |                      |
        |<--shouldClose--------|                      |                      |
        |                      |                      |                      |
        |     [if history.length == 1]                |                      |
        |---window.close()---->|                      |                      |
        |                      |                      |                      |
        |     [else]           |                      |                      |
        |---history.back()---->|                      |                      |
        |                      |                      |                      |
```

## Notes

- New tab detects existing main tab via SharedWorker
- Path in URL can navigate main tab before focus
- Desktop notification prompts user to switch tabs
- New tab closes itself or goes back in history
- Enables backends/AIs to direct user attention
- If session doesn't exist, error page shown (no notification)
