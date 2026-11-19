# Test Design: Bundle
**Source:** crc-Bundle.md

The older bundle tests (symlinks, executable bits, listing) have no test design yet; this
document starts with the file-skipping rule, whose failure is silent: a bundle that ships
a fossil checkout database builds, installs and runs, and only breaks checkpoints on every
install, far from the bundling that caused it.

## Test: bundling leaves out editor leftovers and fossil state
**Purpose:** a site directory holding a fossil checkout, fossil repositories and editor leftovers bundles only its content (R217)
**Input:** `apps/prefs` holding `app.lua`, `fossil.txt`, `.fslckout`, `_FOSSIL_`, `checkpoint.fossil`, `original.fossil`, `#app.lua#`, `.#app.lua` and `app.lua~`
**Expected:** the ZIP holds exactly `apps/prefs/app.lua` and `apps/prefs/fossil.txt`
**Refs:** crc-Bundle.md
**Code:** internal/bundle/bundle_test.go
**Alarm:** 1
**Fire alarm:** in addDirToZip, match against the old editor-only pattern; the four fossil files are bundled
**Inject:** internal/bundle/bundle.go:addDirToZip
@alarm-hash: 06bb8b6e7c48496b
**Pulled:** 2026-11-19 — rang: addDirToZip matched the old editor-only pattern; `bundled [apps/prefs/.fslckout apps/prefs/_FOSSIL_ ... checkpoint.fossil ... original.fossil], want only [apps/prefs/app.lua apps/prefs/fossil.txt]`
