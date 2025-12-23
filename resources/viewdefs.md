# Viewdef Syntax Reference

View definitions (viewdefs) are HTML fragments wrapped in a `<template>` tag. They use special `ui-*` attributes to bind the UI to the underlying Lua objects.

## Core Binding Attributes

| Attribute    | Description                                           | Example                                    |
|:-------------|:------------------------------------------------------|:-------------------------------------------|
| `ui-value`   | Binds element value or content to a Lua path.         | `<span ui-value="name"></span>`            |
| `ui-action`  | Binds user events (click, etc.) to a Lua method call. | `<button ui-action="save()">Save</button>` |
| `ui-view`    | Renders a child object using its own viewdef.         | `<div ui-view="selectedItem"></div>`       |
| `ui-attr-*`  | Binds an HTML attribute to a Lua path.                | `<sl-alert ui-attr-open="hasError">`       |
| `ui-class-*` | Toggles a CSS class based on a Lua boolean path.      | `<div ui-class-active="isSelected">`       |
| `ui-style-*` | Binds a CSS style property to a Lua path.             | `<div ui-style-color="themeColor">`        |

## Path Syntax

Paths are resolved relative to the current object.

- **Properties:** `firstName`, `address.city`
- **Methods:** `fullName()`, `calculateTotal(0.1)`
- **Arrays:** `items[1]`, `tasks[currentTaskIndex]`
- **Properties:** `path?key=value&key2=value2`

### Special Path Properties

- **`?keypress`**: When used with `ui-value` on an input, triggers updates on every keystroke instead of just on blur.
  - Example: `<sl-input ui-value="searchQuery?keypress">`
- **`?wrapper=NAME`**: Instructs the backend to wrap the resolved value in a specific Transformer/Presenter before sending it to the frontend.
  - Example: `ui-view="contact?wrapper=ContactPresenter"`

## Lists and Collections

The `lua.ViewList` wrapper is used to efficiently render arrays of objects.

```html
<div class="list" ui-view="items?wrapper=lua.ViewList&itemWrapper=ItemPresenter"></div>
```

- **`wrapper=lua.ViewList`**: Identifies the variable as a list.
- **`itemWrapper=ItemPresenter`**: (Optional) Wraps each item in the array with the specified Lua class.

Each item is rendered using the `lua.ViewListItem` viewdef, which typically delegates to `ui-view="item"`.

## Component Libraries

The platform integrates **Shoelace** web components. Use `sl-input`, `sl-button`, `sl-icon`, etc., for a consistent, modern look.

```html
<sl-input label="Email" type="email" ui-value="email">
  <sl-icon name="envelope" slot="prefix"></sl-icon>
</sl-input>
```
