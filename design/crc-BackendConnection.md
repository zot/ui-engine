# BackendConnection

**Source Spec:** libraries.md

## Responsibilities

### Knows
- uiServerUrl: UI server connection URL
- sessionId: Associated session ID
- rootVariable: Reference to variable 1
- connected: Connection state
- messageQueue: Pending outbound messages

### Does
- connect: Establish connection to UI server with root value
- disconnect: Close connection and invoke cleanup hook
- send: Send protocol message to UI server
- receive: Handle incoming protocol message
- setRootValue: Initialize variable 1 with root presenter
- onClose: Register connection close callback
- reconnect: Attempt reconnection after disconnect

## Collaborators

- ProtocolHandler: Message handling
- PathNavigator: Path resolution
- ChangeDetector: Change propagation
- AppPresenter: Root value binding

## Sequences

- seq-backend-connect.md: Connection establishment
- seq-backend-refresh.md: Change detection cycle
