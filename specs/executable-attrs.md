# Executable Attribute Preservation

## Overview

Bundles must preserve executable file permissions (+x bit) so that scripts and binaries bundled in a site work correctly after extraction.

## Affected Operations

**bundle command:** When creating a ZIP archive from a directory, executable permissions must be stored in the ZIP file headers.

**extract command:** When extracting files, the original file permissions must be restored from the ZIP headers.

**cp command:** When copying individual files from a bundle, their original permissions must be preserved.

## Requirements

1. Store Unix file mode (permissions) for all regular files in ZIP headers
2. Restore file mode when extracting files
3. Restore file mode when copying files via `cp` command
4. Work correctly on POSIX systems (Linux, macOS)
5. Graceful fallback on Windows (permissions may not apply)

## Example Use Case

```
site/
  html/
    index.html      # 0644 - regular file
  scripts/
    deploy.sh       # 0755 - executable script
```

After `bundle` + `extract`, `deploy.sh` should remain executable without manual `chmod`.

## Technology

- **Language:** Go
- **Environment:** Linux/macOS (primary), Windows (best-effort)
