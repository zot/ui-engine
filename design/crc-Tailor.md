# Tailor
**Requirements:** R174, R175, R176, R177, R178, R179, R180, R181, R182, R183, R184, R185, R186, R187, R188, R189, R190, R191, R194

One sewing operation: a monadic wrapper that exists for that operation and then
disappears. It posts stitches through the session's ExecView, carries the operation's
state, and keeps itself on its session's pending list until it closes.

## Knows
- ref: the SessionRef it was created from (R194)
- session: the `*Session` whose pending list holds it (R191)
- name, created, longLived (R174, R175)
- mu: guards everything below except inStitch and state
- ended: an `End` or `EndUpdate` has been posted (R184)
- closed, failed: the Tailor posts nothing more (R183, R185)
- warnedLate: a late post after a close has been logged (R185)
- lastStitch, lastSite: time and calling site of the last post (R178)
- ctx, cancel: cancelled on close (R190)
- timer: the deadline timer, nil when long-lived (R189)
- inStitch: atomic flag, set while one of its stitches runs (R187)
- state: the session's `*lua.LState` while a stitch runs (R187, R188)

## Does
- new (ExecView.Tailor): build the Tailor with a context from `context.Background()`,
  start the deadline timer when the deadline is non-zero, and add it to the session's
  pending list (R174, R175, R189, R191).
- Sew / SewUpdate / End / EndUpdate: call post with passive or active and terminal or
  not (R176).
- post: under mu, panic when ended (R184); when closed, log once and return (R185);
  record time and calling site (R178); mark ended for a terminal stitch. Then, outside
  mu, close as failed when the ref is invalid (R179), else post runStitch through
  `ExecView.Post` (passive) or `ExecView.PostUpdate` (active) (R177).
- runStitch (on the executor): skip when closed by a failure, deadline or session end.
  Set state and inStitch, run fn with panics recovered (R180), clear them. On error, log
  with the name and close as failed (R181); for a passive non-terminal stitch with
  updateOnErr, call `BrowserView.Update` (R182). A terminal stitch closes the Tailor
  (R183).
- IsValid: not closed and the ref is valid (R186).
- State: the LState when inStitch, else panic (R187). Never the LuaSession (R188).
- Context, Ref (R190, R194).
- close(reason, failed): once only; stop the timer, cancel the context, leave the
  pending list; log a failure with the name, age and last site (R189, R190, R191).
- info: a TailorInfo snapshot for PendingTailors.

## Collaborators
- ExecView: posting stitches; creating Tailors
- BrowserView: change detection after a passive error with updateOnErr
- LuaView: the session's LState for a running stitch
- Session: the pending list
- Config: logging

## Sequences
- seq-tailor.md
