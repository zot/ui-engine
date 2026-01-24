# Bundle
**Source Spec:** deployment.md, executable-attrs.md

## Knows
- MagicMarker: identifies bundled binaries ("UISERVER")
- FooterSize: 24 bytes (offset + size + magic)
- IGNORE_FILES: regex for files to skip (backup/temp files)

## Does
- CreateBundle: creates bundled binary from source binary + site directory
- addDirToZip: recursively adds files to ZIP, preserving relative symlinks and file modes
- addRegularFileToZip: adds regular file with mode preservation
- GetBinarySize: returns executable size excluding any bundle
- IsBundled: checks if current binary has bundled content
- GetBundleReader: returns zip.Reader for bundled content
- ExtractBundle: extracts bundle to directory, recreating symlinks and file modes
- extractZipFile: extracts single file or symlink, preserving mode
- ListFiles: lists files in bundle (names only)
- ListFilesWithInfo: lists files with metadata (name, isSymlink, symlinkTarget, mode)
- ReadFile: reads file content from bundle
- ReadFileInfo: reads file info (mode) from bundle
- ListFilesInDir: lists files in a bundle subdirectory
- validateSymlinkTarget: ensures symlink stays within bundle root

## Collaborators
- ZipFileSystem: serves bundled files via fs.FS interface

## Sequences
- seq-bundle-create.md (if needed)
