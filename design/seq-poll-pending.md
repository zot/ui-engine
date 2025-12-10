# Sequence: Poll Pending Responses

**Source Spec:** deployment.md
**Use Case:** CLI/REST client polls for push-based messages

## Participants

- Client: CLI or REST client
- HTTPEndpoint: HTTP handler (or PacketProtocol)
- PendingResponseQueue: Message queue
- MessageRelay: Message routing (background)

## Sequence

```
     Client               HTTPEndpoint       PendingResponseQueue          MessageRelay
        |                      |                      |                      |
        |   [Background: push messages accumulate]   |                      |
        |                      |                      |                      |
        |                      |                      |<--enqueuePending-----|
        |                      |                      |   (update msg)       |
        |                      |                      |                      |
        |                      |                      |<--enqueuePending-----|
        |                      |                      |   (error msg)        |
        |                      |                      |                      |
        |---poll-------------->|                      |                      |
        |                      |                      |                      |
        |                      |---isEmpty()--------->|                      |
        |                      |                      |                      |
        |                      |<--false--------------|                      |
        |                      |                      |                      |
        |                      |---drain()----------->|                      |
        |                      |                      |                      |
        |                      |<--pending[]----------|                      |
        |                      |   [{update...},      |                      |
        |                      |    {error...}]       |                      |
        |                      |                      |                      |
        |<--{pending:[...]}----|                      |                      |
        |                      |                      |                      |
        |   [ALT: Long-poll with --wait]             |                      |
        |                      |                      |                      |
        |---poll(wait=30s)---->|                      |                      |
        |                      |                      |                      |
        |                      |---isEmpty()--------->|                      |
        |                      |                      |                      |
        |                      |<--true---------------|                      |
        |                      |                      |                      |
        |                      |---poll(timeout)----->|                      |
        |                      |                      |                      |
        |                      |      [waits...]      |                      |
        |                      |                      |                      |
        |                      |                      |<--enqueuePending-----|
        |                      |                      |   (destroy msg)      |
        |                      |                      |                      |
        |                      |                      |---notifyWaiters()--->|
        |                      |                      |                      |
        |                      |<--pending[]----------|                      |
        |                      |   [{destroy...}]     |                      |
        |                      |                      |                      |
        |<--{pending:[...]}----|                      |                      |
        |                      |                      |                      |
```

## Notes

- Pending message types: update, error, destroy
- Regular poll returns immediately with any accumulated messages
- Long-poll (`--wait`) blocks until messages arrive or timeout
- Every protocol response (not just poll) includes pending messages
- Queue is drained on each response (one-time delivery)
- CLI: `ui poll` or `ui poll --wait 30s`
- REST: `GET /poll` or `GET /poll?wait=30s`
