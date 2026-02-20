# VariableBrowser

**Source Spec:** variable-browser.md
**Requirements:** R63, R64, R65, R66, R67, R68, R69, R70, R71, R72, R73, R74, R75, R76, R77, R78, R79, R82, R83

## Responsibilities

### Knows
- sessionId: extracted from URL path
- variables: array of DebugVariable objects from JSON API
- trackerRefreshCount: global change count from X-Change-Count header
- viewMode: "flat" (default) or "tree"
- polling: enabled/disabled + interval
- visibleColumns: set of column keys currently shown
- sortColumn: current sort column (flat mode only)
- sortDirection: "asc" or "desc"
- expandedDiags: set of variable IDs with expanded diagnostics

### Does
- fetchVariables: GET `/{session-id}/variables.json`, parse response and extract X-Change-Count header as trackerRefreshCount (R57, R83)
- renderTable: build HTML table rows from variable data including Changes and Avg Time columns (R63, R82, R83)
- renderTreeMode: indent rows by depth, add expand/collapse triangles; path name also clickable to toggle (R64)
- renderFlatMode: flat rows, enable column header sort; numeric columns default to descending (R65, R72)
- toggleViewMode: switch between tree and flat (R66)
- refresh: manual data reload (R67)
- togglePolling: start/stop interval timer (R68)
- toggleColumn: show/hide column via picker (R69, R70, R71)
- sortByColumn: sort flat rows by column value (R72)
- toggleDiagRow: expand/collapse diagnostic sub-row (R73, R74)
- renderValueTooltip: truncated cell with title attribute for full JSON (R75)
- renderErrorCell: red-highlighted error display (R76)

## Collaborators

- HTTPEndpoint: serves the HTML page and JSON API

## Sequences

- (none — single-page client, no multi-component sequences)
