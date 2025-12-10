# ProtocolDetector

**Source Spec:** deployment.md

## Responsibilities

### Knows
- httpPrefixes: ASCII prefixes indicating HTTP protocol (GET, POST, PUT, DELETE, HEAD, PATCH, OPTIONS)
- packetProtocol: PacketProtocol handler reference
- httpHandler: HTTPEndpoint handler reference

### Does
- detect: Peek first 4 bytes of connection to determine protocol
- isHTTPPrefix: Check if bytes match HTTP method pattern
- routeToPacket: Hand connection to PacketProtocol handler
- routeToHTTP: Hand connection to HTTPEndpoint handler

## Collaborators

- BackendSocket: Provides raw connections
- PacketProtocol: Receives packet-protocol connections
- HTTPEndpoint: Receives HTTP connections

## Sequences

- seq-backend-socket-accept.md: Protocol detection flow

## Notes

- Detection pattern: `^(GET |POST|PUT |DELE|HEAD|PATC|OPTI)`
- Non-matching bytes interpreted as 4-byte big-endian length (packet protocol)
- Detection is non-destructive (peek, not read)
