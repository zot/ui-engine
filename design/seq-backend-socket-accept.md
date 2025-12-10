# Sequence: Backend Socket Accept

**Source Spec:** deployment.md
**Use Case:** Backend connects via socket, protocol auto-detected

## Participants

- Backend: External backend program or CLI
- BackendSocket: Socket listener
- ProtocolDetector: Protocol detection
- PacketProtocol: Packet-based handler
- HTTPEndpoint: HTTP handler
- ProtocolHandler: Message processing
- PendingResponseQueue: Pending message accumulation

## Sequence

```
     Backend            BackendSocket        ProtocolDetector        PacketProtocol          HTTPEndpoint        ProtocolHandler    PendingResponseQueue
        |                      |                      |                      |                      |                      |                      |
        |---connect(socket)--->|                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |---accept()---------->|                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |---peek(4 bytes)----->|                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |---isHTTPPrefix()?--->|                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |   [ALT: HTTP detected - bytes match GET/POST/PUT/etc.]             |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |---routeToHTTP()------------------------------>|                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |---handleSocketHTTP-->|                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |   handleProtocol---->|                      |
        |                      |                      |                      |                      |   Command()          |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |<--result-------------|                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |---drain()------------------------------------>|
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |<--pending[]-----------------------------------|
        |                      |                      |                      |                      |                      |                      |
        |<--response+pending---------------------------------------------------------|                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |   [ALT: Packet detected - bytes are binary length]                 |                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |---routeToPacket()--->|                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---readPacket()------>|                      |                      |
        |                      |                      |                      |   (len+json)         |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---parseMessage()---->|                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---handle*()-------------------------------->|                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |<--result-------------------------------------|                      |
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---drain()--------------------------------------------------->|
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |<--pending[]--------------------------------------------------|
        |                      |                      |                      |                      |                      |                      |
        |                      |                      |                      |---writePacket()----->|                      |                      |
        |                      |                      |                      |   (result+pending)   |                      |                      |
        |                      |                      |                      |                      |                      |                      |
        |<--packet(result+pending)---------------------|                      |                      |                      |
        |                      |                      |                      |                      |                      |                      |
```

## Notes

- Detection happens on connection accept, before reading full message
- HTTP detection: first 4 bytes match `^(GET |POST|PUT |DELE|HEAD|PATC|OPTI)`
- Packet detection: first 4 bytes interpreted as big-endian length
- Every response includes drained pending messages (update, error, destroy)
- Same socket serves both protocol types without configuration
