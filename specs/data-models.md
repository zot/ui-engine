# Backend Data Models

The platform supports two primary data models that determine where the source of truth resides:

## Unbound Model (UI Server as Database)

The backend treats the UI server as the source of truth:
- Backend creates/updates variables directly in UI server
- Uses `get(varId)` to retrieve current state when needed
- State resides in UI server memory

**Use cases:**
- Self-contained applications built entirely on the platform
- Shell scripts and CLI tools that create UI, exit, and resume later
- Multi-step workflows where state must survive process restarts
- Distributed coordination - multiple scripts/processes sharing UI state
- Ephemeral UIs - dialogs, notifications, transient interactions (memory-only)

## Bound Model (External Data Integration)

The backend binds UI variables to its own data, with the UI server acting as a view layer:
- Backend maintains the source of truth in its own systems
- UI variables are bound to backend data via paths (`father.name`)
- Changes sync bidirectionally between UI and backend data

**Use cases:**
- **Data sovereignty** - Organizations control where data resides; UI is just a view
- **Compliance** - Data stays in approved systems (GDPR, HIPAA, etc.)
- **System integration** - UI for existing data without migration

**Integration examples:**

| System            | Binding Pattern                                                             |
|-------------------|-----------------------------------------------------------------------------|
| IDE               | Project structure, open files, diagnostics bound to editor state            |
| Filesystem        | Directory as object, files as variable values, file watches trigger updates |
| System monitoring | CPU/memory/disk as live-updating variables                                  |
| Databases         | Tables/rows exposed through an adapter layer                                |
| External APIs     | REST/GraphQL endpoints as bindable paths                                    |
| Git repositories  | Branches, commits, working tree status as navigable objects                 |

## Hybrid Usage

Both models can coexist in a single application:
- Core application state in UI server (unbound)
- Integration points bound to external systems
- Example: A monitoring dashboard with its own preferences (unbound) displaying live system metrics (bound)
