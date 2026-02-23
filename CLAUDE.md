# Project Instructions

## RECORDING A DEMO

**VERY IMPORTANT:** do now open a system browser, **only use playwright**.
If there is trouble opening a page, the user will solve it.

## Essential Reading: Usage Guide

**ALWAYS read `USAGE.md` before working on frontend/backend integration.**

This document explains the core data flow:
- Frontend creates most variables (backend only creates variable 1)
- Viewdef paths like `ui-value="value1"` cause frontend to create child variables with `path` property
- Backend resolves paths and returns values
- Wrappers enable value transformation

### Running the demo
From the project directory, this command runs the demo `./build/ui-engine-demo --port 8000 --dir demo -vvvv --hotload`
You can use the playwright MCP browser to connect to it: `http://localhost:8000`

### Debugging variables
Use the debug endpoint to inspect variable state. The URL uses the session ID from your browser URL:
```bash
# If your browser is at http://localhost:8000/abc123def456...
curl http://localhost:8000/abc123def456.../variables
```

The endpoint returns an HTML page with a tree view of all variables and their values. You can also view it directly in a browser by navigating to `/{session-id}/variables`.

## 🎯 Core Principles
- Use **SOLID principles** in all implementations
- Create comprehensive **unit tests** for all components
- code and specs are as MINIMAL as POSSIBLE
- Before using a callback, see if a collaborator reference would be simpler

## Techniques Reference

Reusable patterns. Consult when:
- In-place array filtering/compaction → fast-slow pointer technique
- GC-friendly object tracking → `.claude/techniques/weak-refs.md`

## When committing
1. Check git status and diff to analyze changes
2. Ask about any new files to ensure test/temp files aren't added accidentally
3. Add all changes (or only staged files if you specify "staged only")
4. Generate a clear commit message with terse bullet points
5. Create the commit and verify success

## Versioning and Releasing

Release versions use semantic versioning in `README.md` (the `**Version: X.Y.Z**` line near the top).

**To create a release:**
1. Update `**Version: X.Y.Z**` in both `README.md`
2. Commit: `git commit -am "Release vX.Y.Z"`
3. Tag: `git tag vX.Y.Z`
4. Build: `make release-bundled` (creates binaries in `release/` for Linux, macOS, Windows)
5. Push: `git push && git push --tags`
6. Create GitHub release: `gh release create vX.Y.Z release/* --title "vX.Y.Z" --notes "Release notes here"`
