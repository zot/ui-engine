# No-Flash View Rendering

**Source Spec:** viewdefs.md
**CRC:** crc-View.md

## Trigger

View.render() is called when a view needs to render or re-render.

## Participants

- View: Manages render with no-flash buffering
- DOM: Browser Document Object Model

## Sequence

```
View                              DOM
  |                                |
  |-- Check ancestor buffering --->|
  |   parent.closest('.ui-new-view')
  |<--------- result --------------|
  |                                |
  |-- [If buffer root] ----------->|
  |   Mark old elements:           |
  |   .ui-obsolete-view            |
  |                                |
  |-- Create new elements -------->|
  |   Add classes:                 |
  |   .ui-view-{n} .ui-new-view    |
  |                                |
  |-- Insert new elements -------->|
  |   (hidden by CSS)              |
  |                                |
  |-- Process child views -------->|
  |   Children see .ui-new-view    |
  |   on ancestor, render normally |
  |                                |
  |-- [If buffer root] ----------->|
  |   Start 100ms timer            |
  |                                |
  |   ... 100ms passes ...         |
  |                                |
  |-- Timer fires ---------------->|
  |   Remove .ui-obsolete-view     |
  |   Remove .ui-new-view (reveal) |
  |                                |
```

## CSS Rules

```css
.ui-new-view {
  display: none !important;
}
```

## Notes

### Ancestor-Aware Buffering

Only one view at a time acts as the buffer root:
- Root view: Manages the timer and class transitions
- Descendant views: Render normally (already hidden by ancestor's `.ui-new-view`)

### Rapid Re-renders

If a view re-renders while a timer is already pending:
- Old elements: Already have `.ui-obsolete-view`, will be removed by existing timer
- New elements: Get `.ui-new-view`, will be revealed by existing timer
- No new timer is started

### View Destruction

If a view is destroyed while a buffer timer is pending:
- Timer is cleared
- Elements with `.ui-view-{n}` class are removed by normal cleanup
