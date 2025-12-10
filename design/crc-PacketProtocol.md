# PacketProtocol

**Source Spec:** deployment.md

## Responsibilities

### Knows
- connections: Map of connection ID to connection state
- pendingQueues: Map of connection ID to PendingResponseQueue

### Does
- handleConnection: Process packet-protocol connection
- readPacket: Read 4-byte length prefix + JSON payload
- writePacket: Write length-prefixed JSON packet
- parseMessage: Parse JSON payload into protocol message
- serializeMessage: Serialize protocol message to JSON
- attachPendingResponses: Add pending messages to response

## Collaborators

- ProtocolDetector: Routes connections here
- ProtocolHandler: Processes protocol messages
- PendingResponseQueue: Accumulates push messages for polling

## Sequences

- seq-backend-socket-accept.md: Connection handling
- seq-packet-protocol-message.md: Message read/write cycle

## Notes

- Packet format: `[4-byte big-endian length][JSON payload]`
- Efficient for persistent connections with many messages
- Every response includes attached pending messages
