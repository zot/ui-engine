# Variable Browser

**Language:** Go (backend), HTML/CSS/JavaScript (frontend)
**Environment:** Web browser, HTTP endpoints

## Overview

Replace the current server-rendered Shoelace tree debug page with an interactive variable browser. The browser displays session variables in a tree-table with sortable columns, a flat view option, diagnostics display, and live polling.

## Architecture

The backend serves a JSON API endpoint and a static HTML page. The HTML page fetches variable data as JSON and renders it client-side with vanilla JavaScript. No build step, no JS framework, no external dependencies.

## JSON API

The existing `/{session-id}/variables` endpoint is replaced by:
- `/{session-id}/variables` — serves the static HTML browser page
- `/{session-id}/variables.json` — returns the variable array as JSON

The JSON endpoint extends the existing `DebugVariable` struct with:
- `computeTime` — duration of most recent path navigation (formatted string, e.g. "0.12ms")
- `maxComputeTime` — peak duration across all recomputes
- `active` — whether the variable participates in change detection
- `access` — access mode: `rw`, `r`, `w`, or `action`
- `diags` — array of diagnostic messages (present only when diagnostics are enabled)
- `depth` — nesting depth from root (0 for roots), for tree indentation

A `?diag=N` query parameter on the JSON endpoint sets the tracker's diagnostic level before collecting variables, enabling diagnostic capture for that request.

## Browser UI

### Toolbar

```
[x] Flat  [ ] Tree      [Refresh]  [ ] Poll [2s v]      [Columns v]
```

- **Flat/Tree toggle**: switches between display modes; flat is the default
- **Refresh button**: manual data reload (always visible, default interaction)
- **Poll toggle + interval**: enables automatic polling at 1s, 2s, or 5s intervals
- **Column picker**: dropdown with checkboxes to show/hide columns

### Table

An HTML table with fixed header and scrollable body. Columns are content-sized and left-justified; the table extends to the full container width (row borders and header draw to the right edge) via a spacer column that absorbs remaining space. ID column is fixed-width so the Path column doesn't shift when data changes.

**Columns** (in display order):

| Column   | Default visible | Sortable (flat mode) | Description                              |
|----------|-----------------|----------------------|------------------------------------------|
| Diags    | always          | no                   | Toggle button when diagnostics present   |
| ID       | yes             | yes                  | Variable ID                              |
| Path     | yes             | yes                  | Path property; indented in tree mode     |
| Type     | yes             | yes                  | Lua type from properties                 |
| GoType   | no              | yes                  | Go type from resolver                    |
| Value    | yes             | no (text)            | Truncated; tooltip shows full JSON       |
| Changes  | no              | yes (numeric)        | Variable change count                    |
| Time     | yes             | yes (numeric)        | ComputeTime                              |
| Avg Time | no              | yes (numeric)        | ComputeTime / tracker refresh count      |
| Max Time | no              | yes (numeric)        | MaxComputeTime                           |
| Error    | yes             | yes                  | Error message; red highlight when present|
| Access   | no              | yes                  | rw / r / w / action                      |
| Active   | no              | yes                  | Boolean indicator                        |
| Props    | no              | no                   | Remaining properties as key=value        |

### Tree Mode

- Rows indented by depth using left padding
- Parent rows have expand/collapse triangles; clicking the path name also toggles
- All nodes expanded by default
- Sorting disabled (tree order is structural)

### Flat Mode

- All variables as flat rows, no indentation
- Click column headers to sort ascending/descending
- Numeric columns (Time, Max Time) default to descending on first click; others default to ascending
- Sort indicator arrow on active column

### Diagnostics

When a variable has diagnostic messages, a small toggle button appears in the Diags column. Clicking it expands a sub-row below the variable row showing the diagnostic messages as an indented list. Collapsed by default.

### Value Display

Values are truncated inline (100 chars). Hovering shows the full JSON as a tooltip.

### Error Display

Variables with errors show the error message in the Error column with red background styling.
