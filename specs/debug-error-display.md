# Debug Error Display

**Language:** Go (backend), HTML/CSS (frontend)
**Environment:** Web browser, HTTP endpoint

## Overview

The variable tree debug endpoint (`/{session-id}/variables`) should display error information when a variable has an error. This helps developers quickly identify problematic variables during debugging.

## Requirements

### Error Display

- When a variable has an error, display it prominently in the tree view
- Error text should be visually distinct (red color, error styling)
- Error display should not interfere with other variable information (ID, type, path, value)
- Variables with errors should still show all their normal properties

### Visual Styling

- Error text should use a red color scheme consistent with the existing `.error` class
- Error icon or indicator to draw attention
- Error should appear inline with the variable node, not as a separate element
