# ui-engine vs Traditional Web Architecture

**Write only your application logic.** No API layers, no state management, no serialization, no client/server synchronization. Just domain objects, presenters, and HTML templates.

**Rapid prototyping.** No frontend/backend setup, no API design upfront, no boilerplate. Start with your domain model, add HTML templates, iterate with hot-reload.

## Non-Application Code Comparison

Traditional web apps (React, Vue, Angular, Svelte, etc.) require significant "plumbing" code that isn't domain or presentation logic. This is inherent to client/server architecture, not specific to any framework:

| Category         | Traditional                                                    | ui-engine            |
|------------------|----------------------------------------------------------------|----------------------|
| API layer        | REST/GraphQL endpoints, route handlers                         | None                 |
| API contracts    | OpenAPI specs, code generation, version management, spec drift | None                 |
| Data fetching    | useQuery, useFetch, loading states, error handling             | None                 |
| Serialization    | JSON parsing, type converters, DTOs                            | None                 |
| State management | Redux/Vuex/Pinia setup, actions, reducers, selectors           | None                 |
| Form handling    | Form libraries, validation schemas, submit handlers            | Automatic (ui-value) |
| Real-time sync   | WebSocket setup, reconnection logic, state reconciliation      | Built-in             |
| Auth plumbing    | Token storage, refresh logic, protected routes (both sides)    | Backend only         |

**ui-engine**: Write domain objects + presenters + HTML templates. That's it.

**Traditional**: Domain logic is a fraction of the codebase. Most code moves data around.

## ui-engine advantages

- **Single codebase** - no separate frontend JS to write/maintain
- **No API layer** - no REST/GraphQL endpoints to design, no serialization
- **Direct object manipulation** - just mutate objects, UI updates automatically
- **Simpler mental model** - everything lives in one place (backend)

## ui-engine tradeoffs

- **Every interaction hits the backend** - WebSocket round-trip for each action (vs local JS state)
- **Backend holds session state** - scales with concurrent users, not just requests
- **No offline/client-only interactions** - requires connection to backend
- **Latency-sensitive UIs** - games, drag-and-drop may feel sluggish

## Best fit

- Internal tools, admin panels, dashboards
- Apps where backend logic dominates anyway
- Rapid prototyping
- Teams without frontend expertise

## Less ideal for

- Highly interactive UIs needing sub-10ms response
- Offline-first applications
- Apps with millions of concurrent users
