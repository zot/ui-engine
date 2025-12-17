# Dynamic UI Architecture

MCP provides Claude with live control over UI state.

```
┌─────────────────────────────────────────────────┐
│  Claude                                         │
│  - Reads ui://reference for patterns            │
│  - Generates & uploads Lua + viewdefs           │
│  - Inspects, alters, controls UI state live     │
│  - Augments UI on the fly (add components)      │
│  - Handles user events                          │
└─────────────────────┬───────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│  UI MCP (Go)                                    │
│                                                 │
│  Setup:                                         │
│    ui_upload_lua(code)                          │
│    ui_upload_viewdef(type, ns, html)            │
│                                                 │
│  Inspect:                                       │
│    ui_get_state() → current app state           │
│    ui_get_url() → url for user                  │
│                                                 │
│  Control:                                       │
│    ui_run(lua) → execute code, return result    │
│    ui_set(path, value) → update specific field  │
│    ui_call(method, args) → call app method      │
│                                                 │
│  Augment:                                       │
│    ui_add_viewdef(type, ns, html)               │
│    ui_run(lua) → add new objects/methods        │
│                                                 │
│  Events:                                        │
│    ui_wait(timeout) → user interaction          │
│    ui_poll() → non-blocking event check         │
│                                                 │
│  Resources:                                     │
│    ui://reference → patterns & examples         │
│    ui://state → live state (inspectable)        │
└─────────────────────────────────────────────────┘
```

## Live Interaction Example

```
1. Create initial UI
   ui_upload_lua(app_code)
   ui_upload_viewdef("TaskUI", "DEFAULT", html)

2. Start work, update progress
   ui_set("tasks[1].progress", 0.3)
   ui_set("tasks[1].status", "running")

3. Check what user sees
   state = ui_get_state()

4. User clicks pause
   event = ui_wait()  → {action: "pause"}

5. React to user
   ui_call("pause")

6. Add new capability on the fly
   ui_run("function app:skip() ... end")
   ui_add_viewdef("TaskUI", "DEFAULT", updated_html_with_skip_button)

7. Continue updating
   ui_set("tasks[2].progress", 0.5)
```

## What Claude Needs to Know

To generate Lua + viewdefs, Claude needs:

1. **App structure** - Single app object, stored as variable 1
2. **Viewdef syntax** - `data-ui-path`, `data-ui-action`, `data-ui-viewlist`, etc.
3. **Patterns** - Common UI components (progress, forms, lists, charts)
4. **Events** - How user actions map to methods

This knowledge lives in `ui://reference`.

## Plug-and-Play

User: `ui-mcp serve`

Claude:
1. Reads `ui://reference`
2. Generates code for what it wants to show
3. Uploads and controls via MCP tools
4. Handles events, updates state, augments as needed

---

## TODO: Remaining Design Work

### 1. ui://reference Content
What Claude reads to learn patterns. Must be concise but complete.
- App structure (single object, variable 1)
- Viewdef syntax (`data-ui-path`, `data-ui-action`, `data-ui-viewlist`, etc.)
- Common patterns (progress, forms, lists, tables, charts)
- Event handling (actions → methods)

### 2. Tool Schemas
Exact parameters and return types for each MCP tool.
- `ui_upload_lua(code: string) → {ok: bool, error?: string}`
- `ui_upload_viewdef(type: string, namespace: string, html: string) → ...`
- `ui_get_state() → {app: object, ...}`
- `ui_set(path: string, value: any) → ...`
- `ui_call(method: string, args?: array) → result`
- `ui_run(lua: string) → result`
- `ui_wait(timeout_ms?: int) → event`
- `ui_poll() → event | null`
- `ui_get_url() → string`

### 3. Event Format
How user interactions are represented.
- Action events: `{type: "action", method: "pause", args: [...]}`
- Input events: `{type: "input", path: "searchQuery", value: "..."}`
- Selection events: `{type: "select", path: "selectedIndex", value: 2}`

### 4. Implementation
- MCP server in Go (or wrap existing ui server)
- Resource serving for ui://reference
- WebSocket bridge for events
- Session management
