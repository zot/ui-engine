# BackendSocket

**Source Spec:** deployment.md

## Responsibilities

### Knows
- socketPath: Path to socket (POSIX: Unix domain, Windows: named pipe)
- listener: Active socket listener
- connections: Active backend connections
- protocolDetector: Protocol detection handler

### Does
- listen: Start listening on platform-appropriate socket
- accept: Accept incoming connection and hand to ProtocolDetector
- getDefaultPath: Return platform-specific default path (/tmp/ui.sock or \\.\pipe\ui)
- close: Close listener and all active connections
- broadcast: Send message to all connected backends

## Collaborators

- Config: Provides socket path configuration
- ProtocolDetector: Determines protocol for each connection
- PacketProtocol: Handles packet-based connections
- HTTPEndpoint: Handles HTTP-based connections

## Sequences

- seq-server-startup.md: Socket initialization
- seq-backend-socket-accept.md: Connection acceptance and protocol detection
