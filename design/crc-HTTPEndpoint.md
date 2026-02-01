# HTTPEndpoint

**Source Spec:** interfaces.md, deployment.md
**Requirements:** R23, R24, R25

## Responsibilities

### Knows
- server: HTTP server instance
- routes: Map of HTTP routes to handlers
- staticDir: Directory for static file serving
- embeddedSite: Bundled frontend webapp
- pendingQueues: Map of session to PendingResponseQueue

### Does
- handleRequest: Route HTTP request to handler
- serveStatic: Serve static files from directory or embedded site
- handleSessionRedirect: Redirect / to /NEW-SESSION-ID
- handleRESTApi: Process REST API requests
- handleFastCGI: Process FastCGI requests
- extractBundledFile: Serve file from embedded archive
- setCustomDir: Switch to custom site directory
- handleSocketHTTP: Handle HTTP requests received via BackendSocket
- handleProtocolCommand: Process CLI protocol commands (create, destroy, update, watch, unwatch, get, poll)
- attachPendingResponses: Add pending messages to every response
- renderVariableError: Display variable errors with red styling in debug tree (R23, R24, R25)

## Collaborators

- SessionManager: Creates sessions on redirect
- ProtocolHandler: REST API and protocol commands
- Router: URL routing
- ProtocolDetector: Routes socket HTTP connections here
- PendingResponseQueue: Accumulates push messages for polling

## Sequences

- seq-frontend-connect.md: Initial HTTP request handling
- seq-create-session.md: Session creation via redirect
- seq-backend-socket-accept.md: HTTP via socket handling
- seq-poll-pending.md: Long-poll for pending responses
