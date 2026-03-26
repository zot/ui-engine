# Destroy Response Batching

Language: Go (backend)

## Problem

When a view re-renders, the frontend destroys all child views
depth-first and sends the destroy messages as a single batch.

The backend receives this batch and processes each destroy message
sequentially. Each `handleDestroy` call sends a destroy notification
back to the frontend via `h.sender.Send()` — which writes directly to
the WebSocket, bypassing the server's OutgoingBatcher. The result is N
individual WebSocket frames sent back to the frontend, one per
destroyed variable.

Since the frontend destroys bottom-up, by the time a parent's destroy
reaches the backend, its children are already gone. The backend's
recursive `DestroyVariable` returns only the single variable. So each
incoming destroy produces exactly one outgoing destroy notification —
N in, N out, all unbatched.

## Fix: Route Destroy Notifications Through OutgoingBatcher

The handler's `handleDestroy` currently calls `h.sender.Send()` to
send destroy notifications directly to the WebSocket. Instead, these
notifications should be queued through the session's OutgoingBatcher.

The OutgoingBatcher throttles: the first `Queue` call starts a 10ms
timer; subsequent calls accumulate in the pending queue. When the
timer fires, all accumulated messages are sent as a single batch.
Since the backend processes the entire incoming batch synchronously
(all N destroy messages), all N outgoing notifications accumulate
in the batcher and flush together.
