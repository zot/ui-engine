# viewdef Variable Property

**Language:** TypeScript (frontend)
**Environment:** Web browser, WebSocket protocol

## Overview

When a View renders, it resolves a viewdef key (e.g., `Contact.COMPACT`) via 3-tier namespace resolution. This key is currently stored as a DOM attribute (`ui-viewdef`) on the first rendered element for hot-reload targeting, but is not recorded on the view's variable. Recording it as a variable property makes it visible in the variable browser alongside `elementId`, enabling debugging and inspection of which viewdef each view is using.

## Requirements

1. After a View successfully renders, set a `viewdef` property on the view's variable containing the resolved viewdef key
2. The property must be sent to the backend so the variable browser can display it
3. The property must update when the viewdef changes (e.g., type change triggers re-render with different viewdef)
4. ViewList item views must also record their viewdef (they are View instances and render through the same path)
5. When a view is cleared (type becomes empty), the viewdef property should be cleared
