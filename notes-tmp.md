# MCP

New, opinionated base code in the site directory for the default bundle, used by the MCP server:
- viewlist viewdefs
- simple main.lua
- html
- resources
If the config directory does not exist, extract the bundle into it.
Lua defines an `app` global containing the App object.
To present an object, assign it to `app.value`

Apps are based on the idea of editing objects. An Object's class name determines how it appears in a web page, using one of its defined `viewdefs`. A viewdef is an HTML template with elements that have `ui-*` attributes that bind to variables in the object using paths like `name` or `mother.name` (to traverse through data).

In this example Person.DEFAULT viewdef, one input field shows the person's name and another shows the person's number. When the person Lua object's name or number changes in the server, it will automatically update in the UI. When the user changes the name or number, it will automatically change in the server Lua object.

```html
    <div><sl-input ui-value="name"></sl-input></div>
    <div><sl-input ui-value="number"></sl-input></div>
```

A view is a div with a `ui-view` attribute. It will display the object at the path indicated by the `ui-vew` attribute's value as a child view. It will normally use the object's `CLASS.DEFAULT` viewdef to display it.

Each class can define a set of HTML viewdefs for different namespaces, each named `CLASS.NAMESPACE`. A view can specify which namespace to use with the `ui-namespace` attribute. This is just a string, it does not bind to a value on the server. Example: `<div ui-view="calendar" ui-namespace="special"></div>` will display the value for the current object's `calendar` object. If this is an instance of Calendar, it will use the `Calendar.special` viewdef if it exists, otherwise it will use the `Calendar.DEFAULT` viewdef.

To display an array `freinds` as a list in the UI, you can use a view and the `wrapper` property like this, `<div ui-view="friends?wrapper=ViewList"></div>`. This will wrap the `friends` array in a ViewList object so that it will display in place of the raw `friends` array. ViewLists  use the `list-item` namespace for their items, so if a friend is a `Person` and there is a `Person.list-item` viewdef, it will use that to display each friend. If there is no `Person.list-item` viewdef, it will use `Person.DEFAULT` instead.

In Claude, conserve context by using an agent to define apps. In Gemini, put the request in a file and run a separate Gemini instance. Consider having the UI MCP run gemini / claude itself to define the app. Once the app is defined, the host agent can interact with it.
