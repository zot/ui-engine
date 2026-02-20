# Change Count Tracking

**Language:** Go (backend)
**Environment:** HTTP endpoints, variable browser

## Overview

Expose change-tracker's count fields in the debug variable API. The change-tracker already tracks:
- `Tracker.ChangeCount` — incremented each time `DetectChanges()` finds changes (tree refresh count)
- `Variable.ChangeCount` — incremented each time a variable's value actually changes

## JSON API

The `DebugVariable` struct gains a new field:
- `changeCount` — number of times this variable's value changed (`Variable.ChangeCount`)

The `/{session-id}/variables.json` endpoint gains a response header:
- `X-Change-Count` — the tracker's global `ChangeCount` value (stringified int64)

This header lets the variable browser detect whether any changes occurred since the last poll without parsing the full JSON body.
