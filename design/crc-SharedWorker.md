# SharedWorker

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- mainTab: Reference to primary connected tab
- connectedTabs: List of all connected tab ports
- sessionId: Current session ID
- webSocket: Primary WebSocket connection

### Does
- connect: Register new tab connection
- disconnect: Remove tab from list
- setMainTab: Designate primary tab
- isMainTab: Check if tab is primary
- relayToBackend: Forward message from tab to WebSocket
- relayToTabs: Forward message from WebSocket to tabs
- activateMainTab: Bring main tab to focus
- sendNotification: Show desktop notification for tab activation
- handleDuplicateTab: Process duplicate session tab opening

## Collaborators

- WebSocketEndpoint: Backend communication
- FrontendApp: Tab instances
- Session: Session coordination

## Sequences

- seq-frontend-connect.md: Tab registration
- seq-activate-tab.md: Tab activation flow
- seq-relay-message.md: Message routing through worker
