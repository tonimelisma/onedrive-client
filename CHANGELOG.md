# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Resumable large file downloads with progress bar and interrupt handling.
- New `files mkdir <path>` command to create a directory.
- New `files upload <local-file> [remote-path]` command to upload files.
- New `files download <remote-path> [local-path]` command to download files.
- Added an SDK interface abstraction to allow for mock-based testing of commands.
- New `files` command to serve as the main entrypoint for file operations.
- New `files list [path]` command to list contents of a directory. Defaults to root if no path is provided.
- New `files stat <path>` command to view detailed metadata for a specific file or folder.
- Resumable large file uploads with progress bar and interrupt handling.
- New `drives list` command to show all available user drives.
- `drives list` command to list all available OneDrive drives.
- `drives quota` command to display storage quota for the default drive.
- New `auth` command group (`login`, `confirm`, `status`, `logout`) to manage authentication.
- Comprehensive authentication testing strategy with file locking, error handling, and edge case coverage.
- Enhanced test infrastructure with mock server support for Graph API endpoints.
- Thread-safe session management with proper Manager pattern instead of global variables.
- Improved test infrastructure with better output capture that doesn't mutate global state.

### Changed
- **BREAKING**: Standardized error handling across all commands to use `RunE` pattern instead of `log.Fatalf()`.
- **BREAKING**: Fixed path handling to use proper remote path utilities instead of `filepath.Join` for URL paths.
- Improved thread safety by ensuring token refresh callback uses configuration mutex properly.
- The underlying SDK now uses path-based addressing to look up items in OneDrive, allowing access to any file/folder, not just those in the root.
- Refactored core application to use an SDK interface for better testability.
- Switched to using `log` for user-facing success/error messages.
- Large file uploads now use a session-based approach instead of a single request.
- Renamed `LiveSDK` to `OneDriveSDK` for better clarity.
- Refactored test suite to use a shared `test_helpers_test.go` to avoid duplicated code and fix the build.
- **Overhauled Authentication Flow**: Replaced the previous blocking Device Code Flow with a non-blocking, stateful, and more user-friendly process.
  - `auth login` now starts the flow and exits immediately, allowing the user to continue working.
  - Any subsequent command (`files list`, `auth status`, etc.) will automatically attempt to complete the pending login.
  - The CLI now clearly reports its state: `Logged In`, `Logged Out`, or `Login Pending`.
  - Added file-locking to the authentication session file (`auth_session.json`) to prevent race conditions.
  - The `auth whoami` command was removed and its functionality merged into `auth status` for a cleaner interface.
- The local configuration file path can now be overridden with the `ONEDRIVE_CONFIG_PATH` environment variable for easier testing.
- Session management now uses proper dependency injection instead of global variable overrides for testing.

### Fixed
- **Critical**: Standardized error handling patterns across all commands for consistency and testability.
- **Critical**: Fixed race conditions in session management by removing global variable patterns.
- **Critical**: Fixed path handling bugs where `filepath.Join` was incorrectly used for remote URL paths.
- **Critical**: Fixed thread safety issues in token refresh callbacks.
- Resolved persistent build and test failures caused by Go module inconsistencies. This required manual adjustments to `go.mod` and repeated `go mod tidy` commands to correctly vendor a new dependency (`progressbar`).
- The build process was fixed by removing duplicated test helper code.
- Authentication command error handling to use RunE instead of log.Fatalf for better testability.
- Session directory creation in file locking operations to prevent "no such file or directory" errors.
- Test helper function signatures and session file path construction in authentication tests.
- Improved test output capture to properly handle log output without global state mutation.

### Removed
- Removed the old `drives` command, which was a temporary implementation for listing root items. Its functionality is now part of `files list`.
- Removed the old, interactive authentication flow that required pasting a URL back into the terminal.
- **BREAKING**: Removed global variable patterns in session management in favor of proper dependency injection.

### Security
- Enhanced session file locking to prevent race conditions in concurrent CLI invocations.
- Improved error message handling to avoid exposing sensitive information in logs.

## [0.1.0] - 2024-05-20

### Added
- Initial release with basic `files list` and `files stat` commands.
- OAuth2 authentication flow.
- Configuration file management for storing tokens.

### Changed
- Refactored SDK into an interface for testability.

### Fixed
- Test suite now uses a mock SDK to avoid live network calls.
- Build process is now more reliable.

## [0.1.1] - 2024-05-21

### Added
- Initial implementation of the CLI.
- `files list`: List files in the root directory.
- `files stat`: View metadata for items.
- `files download`: Download files.
- `files upload`: Upload files (including large files via session).
- `files mkdir`: Create new folders.

### Changed
- Refactored SDK into an interface for testability.

### Fixed
- Test suite now uses a mock SDK to avoid live network calls.
- Build process is now more reliable. 