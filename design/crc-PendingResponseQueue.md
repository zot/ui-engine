# PendingResponseQueue

**Source Spec:** deployment.md

## Responsibilities

### Knows
- queue: List of pending response messages
- waiters: Channels waiting for pending responses (long-poll)
- maxSize: Maximum queue size before oldest dropped

### Does
- enqueue: Add message to pending queue (update, error, destroy)
- drain: Return all pending messages and clear queue
- poll: Return pending messages, optionally waiting for availability
- notifyWaiters: Wake up any long-polling waiters when messages arrive
- isEmpty: Check if queue has pending messages

## Collaborators

- PacketProtocol: Attaches pending to responses
- HTTPEndpoint: Attaches pending to REST responses
- MessageRelay: Enqueues update/error/destroy messages
- ProtocolHandler: Enqueues operation results

## Sequences

- seq-poll-pending.md: Polling for pending responses
- seq-update-variable.md: Enqueue update notifications

## Notes

- Pending message types: update, error, destroy
- Long-poll via optional `--wait` timeout
- Queue drained on every REST/CLI response
- One queue per client connection/session
