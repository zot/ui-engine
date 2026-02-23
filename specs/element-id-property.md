# elementId Variable Property

**Language:** TypeScript (frontend), Go (backend)
**Environment:** Web browser, WebSocket protocol

## Overview

Every variable created by a binding or view must carry an `elementId` property containing the DOM element ID of the widget or view element that created it. This allows the variable browser to report which DOM element a variable is associated with, enabling users and tools to navigate from variables to their corresponding UI elements.

## Current State

View and ViewList variable creation already sets `elementId` in properties. Widget binding variable creation (ui-value, ui-attr-*, ui-class-*, ui-style-*, ui-code, ui-html, ui-event-*, ui-action) does not — these pass the widget reference but omit `elementId` from the properties sent to the backend.

## Requirements

When `VariableStore.create()` receives a `widget` option, it must set `elementId` in the variable's properties to `widget.elementId`. This ensures:

1. All widget-bound variables carry `elementId` in their properties
2. The property is sent to the backend as part of the create message
3. The variable browser displays `elementId` for all variables (it already renders all properties)
4. The `processScrollNotifications` lookup (`properties['elementId']`) works correctly for widget-bound variables

## Centralized Assignment

Setting `elementId` in `VariableStore.create()` rather than in each binding method:
- Single point of assignment — no risk of forgetting it in new binding types
- View/ViewList already set `elementId` in properties before calling create — these take precedence since properties are set before the widget check
