# No-Flash View Rendering

## Problem

When a view re-renders with subviews:
1. Old elements are removed immediately
2. New elements are created, subviews need server data (ping-pong)
3. User sees intermediate states (flash of incomplete content)

## Solution

Ancestor-aware timer buffering:
- View checks: "is an ancestor already buffering?" (via `closest('.ui-new-view')`)
- If NO ancestor buffering: I am the root, use hide/reveal with 100ms timer
- If YES ancestor buffering: I render normally (already hidden by ancestor)
- Result: Only one view at a time has a pending timer, children just render

## Behavior

### Initial Render
- New elements are created with `.ui-new-view` class (hidden)
- After 100ms timer fires, `.ui-new-view` is removed (revealed)

### Re-render
- Old elements get `.ui-obsolete-view` class (kept visible until timer)
- New elements get `.ui-new-view` class (hidden)
- After 100ms timer fires:
  - Elements with `.ui-obsolete-view` are removed
  - Elements with `.ui-new-view` have class removed (revealed)

### Nested Views
- Parent view starts the buffer (is buffer root)
- Child views detect parent's `.ui-new-view` class
- Children render normally (already hidden by parent)
- Parent's timer reveals entire subtree at once

## CSS Classes

| Class | Purpose |
|-------|---------|
| `.ui-view-{n}` | Identifies all elements of view n |
| `.ui-new-view` | Hidden (pending reveal) |
| `.ui-obsolete-view` | Marked for removal |
