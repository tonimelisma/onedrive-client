# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New `files` command to serve as the main entrypoint for file operations.
- New `files list [path]` command to list contents of a directory. Defaults to root if no path is provided.
- New `files stat <path>` command to view detailed metadata for a specific file or folder.

### Changed
- The underlying SDK now uses path-based addressing to look up items in OneDrive, allowing access to any file/folder, not just those in the root.

### Removed
- Removed the old `drives` command, which was a temporary implementation for listing root items. Its functionality is now part of `files list`. 