# ui-engine

**Build reactive web UIs with just backend code and HTML templates.**

No frontend JavaScript. No API layers. No state management boilerplate. Define your domain and presentation objects in backend code, add HTML templates with declarative bindings, and ui-engine handles the rest.

## The Problem

Traditional web apps require massive amounts of non-application code:

- API endpoints and OpenAPI specs
- Data fetching, serialization, DTOs
- State management (Redux, Vuex, etc.)
- Form handling and validation
- Real-time sync infrastructure

Your actual domain logic becomes a fraction of the codebase.

## The Solution

ui-engine eliminates the client/server boundary for application code:

```lua
-- Backend: domain objects + presenters
Contact = {type = "Contact"}
function Contact:fullName()
    return self.firstName .. " " .. self.lastName
end

ContactPresenter = {type = "ContactPresenter"}
function ContactPresenter:delete()
    app:removeContact(self.contact)
end
```

```html
<!-- Frontend: just HTML templates with bindings -->
<input ui-value="firstName">
<span ui-value="fullName()"></span>
<button ui-action="save()">Save</button>
```

Modify objects directly. UI updates automatically. No plumbing required.

## Quick Start

```bash
./build/ui-engine-demo --port 8000 --dir demo
```

Open http://localhost:8000 to see the Contact Manager demo.

See [demo/README.md](demo/README.md) for details.

## Documentation

- **[USAGE.md](USAGE.md)** — Complete guide: bindings, events, path properties, ViewList, namespaces
- **[TRADEOFFS.md](TRADEOFFS.md)** — When to use ui-engine vs traditional web architecture
- **[demo/](demo/README.md)** — Working examples (Contact Manager, Simple Adder)

## Key Features

- **Declarative bindings** — `ui-value`, `ui-action`, `ui-view`, `ui-attr-*`, `ui-class-*`
- **Automatic change detection** — no observer pattern, no boilerplate
- **Hot-reloading** — edit templates, see changes instantly
- **ViewList** — automatic presenter wrapping for collections
- **Namespace system** — multiple views per type (list-item, detail, etc.)

## Current Focus

Embedded Lua backend for the [ui-mcp](https://github.com/zot/ui-mcp) project. The architecture supports other backends (Go, proxied external programs) but Lua is the priority.
