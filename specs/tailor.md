# Tailor — sewing host work across the session executor

**Language:** Go
**Environment:** ui-engine backend (`internal/server`, re-exported through `cli`)

## Purpose

Host operations often need to leave the session executor to do slow work (run fossil,
read files, call the network) and come back to update Lua state. Blocking the executor
for that work stalls the whole session. Calling back synchronously from inside the
executor deadlocks.

Sewing solves both: control passes executor → worker goroutine → executor as posted
closures ("stitches"), and nothing ever waits on the executor. A **Tailor** is the
per-operation object that carries one sewing: it posts the stitches, keeps the
operation's state in one place, and makes a sewing that never finishes visible instead of
silent.

A Tailor exists for one operation and then disappears. It is not one per session.

## Concepts

- **Stitch:** a closure posted to the session executor. It runs with the session's Lua
  context set. A stitch is **passive** (no change detection afterward) or **active**
  (change detection afterward, pushing to the browser when one is connected).
- **Terminal stitch:** the stitch that ends the sewing. The code posting it declares it.
  Only the terminal stitch normally runs change detection; an active non-terminal stitch
  is for progress updates.
- **Closed:** a Tailor is closed when its terminal stitch has run, when a stitch fails,
  when its deadline passes, or when its session ends. A closed Tailor posts nothing more.
- **Pending list:** the session's list of Tailors not yet closed. It exists so a sewing
  that never completes can be seen. A sewing that never completes is a leak, not a hang:
  nothing waits on it.

## API

```go
// Creating a Tailor
func (v ExecRig) Tailor(name string, deadline time.Duration) *Tailor // deadline 0: long-lived

// Stitches: all four post and return at once
func (t *Tailor) Sew(fn func() error, updateOnErr bool) // passive
func (t *Tailor) SewUpdate(fn func() error)             // active
func (t *Tailor) End(fn func() error)                   // passive, terminal
func (t *Tailor) EndUpdate(fn func() error)             // active, terminal

// State
func (t *Tailor) IsValid() bool               // session valid and Tailor not closed
func (t *Tailor) Context() context.Context    // cancelled when the Tailor closes
func (t *Tailor) State() *lua.LState           // only inside a running stitch
func (t *Tailor) Ref() SessionRef

// Inspection
func (v ExecRig) PendingTailors() []TailorInfo
type TailorInfo struct {
    Name       string
    Created    time.Time
    LastStitch time.Time
    LastSite   string    // file:line of the last Sew/SewUpdate/End/EndUpdate call
    LongLived  bool
}
```

### Stitches

- Each stitch records the time and the calling site (file and line) in the Tailor's
  pending entry when it is posted.
- Before posting, a stitch checks `IsValid()`. If the session has ended, the Tailor closes
  as failed and nothing is posted.
- A stitch runs `fn` with panics recovered. A panic counts as an error.
- **An error ends the sewing.** When `fn` returns an error (or panics), the error is
  logged with the Tailor's name, and the Tailor closes as failed. The chain stops at
  the first error, so a stitch never has to remember to clean up after itself.
- `Sew` runs change detection after an error only when `updateOnErr` is set, so the
  browser can show the failure. `SewUpdate` and `EndUpdate` always run it.
- `End` and `EndUpdate` close the Tailor after `fn` runs, whether or not it errored.

### Posting to a closed Tailor

- **After `End` or `EndUpdate` has been posted,** any further stitch is a programming
  error and panics. Misuse is loud.
- **After a failure, a deadline, or the session ending,** a further stitch is logged once
  and dropped. A worker finishing late is not a programming error.

### Lua state

- `State()` returns the session's `*lua.LState` while a stitch of this Tailor is running,
  and panics otherwise. Workers on other goroutines must not touch Lua state.
- The Tailor never hands out the `LuaSession`, whose synchronous execution would let a
  stitch wait on its own executor. The `LState` is enough to read and write
  Lua values; nothing in a stitch needs the session object.

### Deadline and context

- A Tailor created with a non-zero deadline closes as failed when the deadline passes,
  and the failure is logged with its name and last calling site.
- A deadline of zero makes the Tailor **long-lived**: it never expires, and inspection
  reports it separately so it is not mistaken for a stuck sewing.
- `Context()` is cancelled when the Tailor closes for any reason. Workers pass it to
  anything they start (for example `exec.CommandContext`), so a deadline or a session
  ending also stops the child process.

### Pending list

- A Tailor joins its session's pending list when it is created and leaves when it
  closes.
- `PendingTailors()` lists the open Tailors of the session.
- When a session is destroyed, every open Tailor on it closes as failed, its context is
  cancelled, and each is logged with its name, age, and last calling site.

## Non-goals

- No debug page in this step. `PendingTailors()` is the data source; a page can be added
  later.
- No Lua API. Lua-facing host operations are a later step built on Tailor.
