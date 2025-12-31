# UI Manifest

**Global UI concerns for the UI Platform frontend**

**Source**: interfaces.md, libraries.md, components.md

---

## Routes

| Path Pattern | Handler | Description |
|--------------|---------|-------------|
| `/` | HTTPEndpoint | Redirect to `/NEW-SESSION-ID` |
| `/{sessionId}` | FrontendApp | Main app shell, renders currentPage() |
| `/{sessionId}/{path}` | Router | Registered presenter paths |

---

## View Hierarchy

```
FrontendApp (root)
  +-- SharedWorker (coordination)
  +-- AppShell (ui-app-shell.md)
       +-- CurrentPage (currentPage() presenter)
            +-- Nested Views (ui-view)
            +-- View Lists (ui-viewlist)
            +-- Dynamic Content (ui-content)
```

---

## Global Components

### SharedWorker
- Coordinates multiple browser tabs
- Maintains single WebSocket connection
- Designates main tab for backend communication
- Handles tab activation and notifications

### ViewdefStore (Frontend)
- Caches viewdefs by TYPE.NAMESPACE key
- Validates viewdefs (single template root element)
- Receives updates via variable 1's viewdefs property
- Provides viewdefs to ViewRenderer on demand
- Manages pending views list (views waiting for viewdefs)

### BindingEngine
- Processes all ui-* attributes
- Creates/destroys bindings as elements enter/leave DOM
- Manages variable watches for bound elements

---

## UI Patterns

### Value Bindings
- `ui-value="path"` - Bind to element value property
- `ui-attr-NAME="path"` - Bind to HTML attribute
- `ui-class-NAME="path"` - Bind value as CSS class
- `ui-style-PROP-SUBPROP="path"` - Bind to CSS style property

### Event Bindings
- `ui-event-EVENT="path"` - Update variable on DOM event
- `ui-action="method()"` - Trigger method call on action

### Container Bindings
- `ui-content="path"` - Render HTML content
- `ui-view="path"` - Create View for object reference (renders with TYPE.NAMESPACE viewdef)
- `ui-viewlist="path"` - Create ViewList for array of object references
- `ui-viewdef="path"` - Render computed viewdef string
- `ui-namespace="NAMESPACE"` - Viewdef namespace (default: DEFAULT)

### Path Parameters
- `path?create=Type&prop=value` - URL-style parameters in paths
- Parameters set variable properties on creation

---

## Theme

### Component Library
- **Shoelace** - Primary web components (sl-*)
- **Tabulator** - Data grids and tables

### CSS Classes
- Defined per viewdef
- No global theme enforced (backend controls styling)

### Error State Classes
- `ui-error` - Applied to elements when binding has error condition (e.g., path-failure)
- `ui-error-code` - Attribute containing error code (e.g., "path-failure")
- `ui-error-description` - Attribute containing human-readable error description
- Error state clears automatically on successful update to the same variable

---

## Browser History

### State Management
- `historyIndex` - Current position in app history
- `url` - Current URL path (after session ID)
- `history` - Array of page presenter references

### Navigation Triggers
- Backend update to historyIndex/url
- Browser back/forward (popstate)
- Link clicks (if registered path)

### URL Structure
```
http://SITE/SESSION-ID/PATH
         ^          ^    ^
         |          |    +-- Optional registered path
         |          +------- Unique session identifier
         +------------------ Server host
```

---

## View Lifecycle

1. **Watch variable 1** - Frontend watches root variable
2. **Receive viewdefs** - Store viewdefs from update
3. **Get currentPage()** - Determine active page presenter
4. **Lookup viewdef** - Find TYPE.VIEW for presenter type
5. **Create DOM** - Parse viewdef HTML
6. **Apply bindings** - Process ui-* attributes
7. **Watch variables** - Subscribe to bound variable changes
8. **Update on change** - Re-apply bindings when values change
9. **Cleanup on navigate** - Unbind and clear old view

---

*This file documents cross-cutting UI concerns - reference from ui-*.md specs*
