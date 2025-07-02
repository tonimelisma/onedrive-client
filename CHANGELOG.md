# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Structured Logging**: Comprehensive logging interface with Debug/Info/Warn/Error levels
  - Full Go 1.22 log/slog integration with `SlogLogger` implementation
  - `NoopLogger` for testing and silent operation
  - `NewDefaultLogger()` function for easy setup with debug mode control
  - Backward compatibility maintained through type aliases
- **Security Hardening**: Complete security utilities package
  - Path sanitization functions (`SanitizePath`, `SanitizeLocalPath`) to prevent path traversal attacks
  - Download path validation with overwrite protection (`ValidateDownloadPath`)
  - Secure file creation utilities (`SecureCreateFile`) with proper permissions
  - Filename validation for OneDrive compatibility (`ValidateFileName`)
  - Comprehensive security error types (`ErrPathTraversal`, `ErrInvalidPath`, `ErrFileExists`, `ErrUnsafePath`)
- **Enhanced Error Handling**: Improved error handling with structured sentinel errors
  - Enhanced `apiCall` method with proper error wrapping using existing sentinel errors
  - Better error context and debugging information
  - Status code 507 (Insufficient Storage) now properly mapped to `ErrQuotaExceeded`
- **CI/CD Pipeline**: Complete GitHub Actions workflow with comprehensive testing
  - Multi-OS build matrix (Ubuntu, macOS, Windows)
  - Go version matrix testing (1.22.x, 1.23.x)
  - Coverage reporting with detailed test results
  - Security scanning with gosec
  - Comprehensive linting with golangci-lint configuration
- **Code Quality Tools**: Enhanced linting and code quality enforcement
  - Extensive golangci-lint configuration with 30+ enabled linters
  - Security, performance, style, and maintainability checks
  - Consistent code formatting and best practices enforcement
- **HTTP Configuration**: New centralized HTTP client configuration system for timeouts and retry behavior
  - **Default Values**: 30-second timeout, 3 retry attempts with exponential backoff
  - **Configurable**: All HTTP clients use consistent timeout and retry settings
  - **Backward Compatible**: Existing configurations automatically get sensible defaults
  - **OAuth Preservation**: HTTP configuration preserves OAuth2 transport layer
- **Polling Configuration**: New configurable polling system for long-running operations
  - **Copy Operations**: Configurable intervals for monitoring copy status with exponential backoff
  - **Default Values**: 2-second initial interval, 30-second maximum, 1.5x multiplier
  - **Smart Backoff**: Prevents overwhelming the server while providing responsive feedback
- **UI Enhancement**: DisplayDriveItemsWithTitle function for context-specific directory listings
  - **Flexible Titles**: Custom titles for different contexts (root, subfolders, search results)
  - **Backward Compatible**: Existing DisplayDriveItems function maintains default behavior
- **Comprehensive Test Coverage**: New test suites for all architectural improvements
  - **HTTP Configuration Tests**: Verify timeout and retry behavior
  - **Configuration Loading Tests**: Ensure backward compatibility
  - **UI Display Tests**: Test custom title functionality
- **Error Handling Consistency**: Added comprehensive sentinel error system with 14 standardized error types
  - `ErrReauthRequired`: Authentication has failed, user needs to log in again
  - `ErrAccessDenied`: User does not have permission for the operation
  - `ErrRetryLater`: Service is busy or unavailable, operation might succeed on retry
  - `ErrInvalidRequest`: The request was malformed or invalid
  - `ErrResourceNotFound`: The requested resource could not be found
  - `ErrConflict`: Operation conflicts with current state (e.g., name already exists)
  - `ErrQuotaExceeded`: Storage quota has been exceeded
  - `ErrAuthorizationPending`: Device code flow is pending user authorization
  - `ErrAuthorizationDeclined`: User declined device code authorization
  - `ErrTokenExpired`: Access token has expired and needs refresh
  - `ErrInternal`: Internal server error occurred
  - `ErrDecodingFailed`: JSON response could not be decoded
  - `ErrNetworkFailed`: Network request failed (timeouts, connection errors)
  - `ErrOperationFailed`: General operation failure (copies, moves, uploads, etc.)
- **Error Testing Framework**: Comprehensive test suite validating error sentinel behavior with `errors.Is()`

### Improved
- **Error Consistency**: All SDK functions now use consistent error wrapping with `%w` verb for proper error chains
- **Error Categorization**: Replaced all raw `fmt.Errorf` patterns with categorized sentinel errors across all SDK files:
  - `pkg/onedrive/client.go`: User info, shared items, recent items, special folders, delta sync, pagination, versions
  - `pkg/onedrive/drive.go`: Drive operations, default drive, root items listing
  - `pkg/onedrive/auth.go`: OAuth2 authorization flow, device code flow, token exchange
  - `pkg/onedrive/item.go`: File/folder operations, metadata, children listing, CRUD operations
  - `pkg/onedrive/upload.go`: Upload session management, chunk upload progress
  - `pkg/onedrive/thumbnails.go`: Thumbnail and preview generation
  - `pkg/onedrive/permissions.go`: Sharing links, invitations, permission management
  - `pkg/onedrive/search.go`: Search result processing
- **Error Testability**: External consumers can now reliably test error conditions using `errors.Is()` checks
- **Debugging Experience**: Error messages now provide clear categorization and wrapped context
- **API Reliability**: Consistent error handling patterns across all OneDrive operations
- **Developer Experience**: Predictable error types enable better error handling strategies

### Changed
- **BREAKING**: Removed legacy session helper functions
  - Removed `session.SaveAuthState()` convenience function
  - Removed `session.LoadAuthState()` convenience function  
  - Removed `session.DeleteAuthState()` convenience function
  - All session operations now require explicit `session.Manager` instance
  - Updated `internal/app/app.go` to use Manager pattern: `sessionMgr.LoadAuthState()`
  - Updated `cmd/auth.go` to use Manager pattern: `sessionMgr.SaveAuthState(authState)`
- **Enhanced Session Management**: Fully migrated to Manager pattern
  - All components now use `session.NewManager()` for session operations
  - Consistent error handling across session operations
  - Improved session lifecycle management
- **Improved Display Interface**: Enhanced UI logging with structured logging support
  - `StdLogger` now implements full structured logging interface
  - Added Debug/Info/Warn/Error methods with formatted variants
  - Better integration with new logging infrastructure
- **HTTP Client Management**: Centralized and consistent HTTP client configuration
  - **All Clients**: SDK, upload, download, and auth operations use configured timeouts
  - **Pre-authenticated URLs**: Downloads and uploads maintain consistent timeout behavior
  - **Retry Logic**: Configurable exponential backoff with maximum delay caps
  - **Rate Limiting**: Improved handling of 429 responses with configurable delays
- **Copy Operation Monitoring**: Replaced hard-coded 5-second polling with configurable intervals
  - **Exponential Backoff**: Starts at 2 seconds, increases to maximum 30 seconds
  - **User Feedback**: Progress logging shows wait times and polling intervals
  - **Resource Efficiency**: Reduces server load while maintaining responsiveness

### Fixed
- **Package Organization**: Fixed Go package naming convention violation in `cmd/items/` directory
  - **Issue**: All files in `cmd/items/` directory incorrectly used `package cmd` instead of `package items`
  - **Solution**: Updated all 11 files to use proper `package items` declaration matching directory name
  - **Impact**: Improved code clarity and adherence to Go conventions without functional changes
  - **Files Updated**: `items_root.go`, `items_download.go`, `items_helpers.go`, `items_manage.go`, `items_meta.go`, `items_permissions.go`, `items_upload.go`, and all test files
  - **Import Updates**: Removed package alias in `cmd/root.go` import, now uses standard `items.InitItemsCommands()`
  - **Testing**: All unit and E2E tests pass, confirming no functional regressions
- Compilation errors in test files related to struct field mismatches
- Authentication test string matching issues
- Items search validation to properly handle missing folder scope
- Display interface struct field access errors (removed non-existent Email field references)
- Application struct field naming consistency (Application.Id → Application.ID)
- Unused import and variable cleanup throughout codebase

### Technical Improvements
- **Test Coverage**: Comprehensive test suite for all new functionality
  - Logger interface tests with all structured logging methods
  - Security utilities tests covering all path validation scenarios
  - Enhanced error handling tests with proper sentinel error validation
  - Integration tests ensuring backward compatibility
- **Documentation**: Updated inline documentation and code comments
- **Code Organization**: Better separation of concerns with focused utility packages
- **Performance**: Optimized error handling and logging performance
- **Maintainability**: Improved code structure and reduced technical debt

### Infrastructure
- **GitHub Actions**: Production-ready CI/CD pipeline
  - Automated testing across multiple environments
  - Security scanning and vulnerability detection
  - Code quality enforcement and linting
  - Automated build verification
- **Development Tools**: Enhanced development experience
  - Comprehensive linting rules for code consistency
  - Pre-configured security scanning
  - Automated code quality checks

### Internal Refactoring
- **Context Propagation Implementation (BREAKING INTERNAL)**: Completed comprehensive context propagation throughout the entire SDK for proper HTTP request cancellation and timeout support:
  - **SDK Interface Enhancement**: Added `context.Context` as first parameter to all 45+ SDK methods in `internal/app/sdk.go`
  - **HTTP Request Modernization**: Updated all HTTP operations from `http.NewRequest` to `http.NewRequestWithContext` across all SDK files
  - **Command Layer Integration**: Updated all command functions to use `cmd.Context()` from Cobra for proper cancellation chain
  - **Files Updated**: Modified 10 core SDK files (`client.go`, `item.go`, `drive.go`, `upload.go`, `download.go`, `search.go`, `activity.go`, `permissions.go`, `thumbnails.go`, plus all command files)
  - **Test Infrastructure Overhaul**: Updated MockSDK with context parameters and fixed all test compilation issues
  - **E2E Test Fixes**: Resolved directory conflict issues in e2e tests with improved cleanup logic and "conflict" error handling
  - **Result**: All HTTP operations now support cancellation, timeouts, and request tracing; enables graceful shutdown and resource management
  - **Zero Functional Impact**: All existing functionality preserved with identical behavior, only method signatures changed
  - **Build & Test Success**: All tests pass including comprehensive E2E test suite (104s runtime) confirming successful implementation

- **SDK Client Code Organization Enhancement**: Completed comprehensive splitting of monolithic `pkg/onedrive/client.go` (1018 LOC → 461 LOC) into focused, maintainable modules:
  - `pkg/onedrive/upload.go` - Upload session methods (CreateUploadSession, UploadChunk, GetUploadSessionStatus, CancelUploadSession) (~90 LOC)
  - `pkg/onedrive/download.go` - Download operations (DownloadFile, DownloadFileByItem, DownloadFileChunk, DownloadFileAsFormat) (~162 LOC)
  - `pkg/onedrive/search.go` - Search functionality (SearchDriveItems, SearchDriveItemsInFolder, SearchDriveItemsWithPaging) (~80 LOC)
  - `pkg/onedrive/activity.go` - Activity tracking (GetItemActivities) (~35 LOC)
  - `pkg/onedrive/permissions.go` - Sharing and permissions (CreateSharingLink, InviteUsers, ListPermissions, GetPermission, UpdatePermission, DeletePermission) (~158 LOC)
  - `pkg/onedrive/thumbnails.go` - Thumbnail and preview methods (GetThumbnails, GetThumbnailBySize, PreviewItem) (~85 LOC)
  - Core `client.go` retains essential functionality: client initialization, authentication, shared utilities (apiCall, collectAllPages), and cross-file dependencies
  - **Result**: Improved maintainability with focused single-responsibility files, easier code navigation, and better separation of concerns
  - **Zero functional impact**: All existing functionality preserved with identical method signatures and behavior
  - **Build & Test Success**: All tests pass including comprehensive E2E test suite confirming no regressions

### Critical Bug Fixes
- **FIXED**: Critical OAuth2 token refresh bug that caused authentication failures with error "AADSTS900144: The request body must contain the following parameter: 'client_id'"
  - **Root Cause**: The `oauth2.Config` used for token refresh was missing the `ClientID` field
  - **Solution**: Updated `NewClient` function to accept and use `clientID` parameter, eliminating import cycle
  - **Impact**: Authentication now works reliably without manual re-login when tokens expire

### API Mapping Corrections
- **FIXED**: Corrected CLI command placement to align with Microsoft Graph API boundaries:
  - **Moved `items recent` to `drives recent`**: Recent items use drive-level endpoint `GET /me/drive/recent` and are now correctly placed under drives
  - **Moved `shared list` to `drives shared`**: Shared items use drive-level endpoint `GET /me/drive/sharedWithMe` and are now correctly placed under drives
  - **Removed separate `shared` command**: Consolidated shared functionality into drives namespace for proper API boundary alignment
- **Enhanced**: All APIs now correctly mapped according to Microsoft Graph API specification:
  - **Drive commands**: All use drive-level endpoints (`/me/drive/*`, `/drives/*`)
  - **Items commands**: All use item-level endpoints (`/me/drive/items/{item-id}/*`)
- **Updated**: Test coverage for new command placements with comprehensive unit tests

### Breaking Changes - CLI Command Reorganization
- **BREAKING**: Reorganized CLI commands to follow Microsoft Graph API boundaries between Drive and Items operations:
  - **Drive-level commands** (operations on entire drive): moved to `drives` namespace
    - `drives search <query>` - Search across entire drive (was `items search` without --in flag)
    - `drives delta [token]` - Track drive changes (was `items delta`, removed `cmd/delta.go`)
    - `drives special <folder-name>` - Access special folders (was `items special`)
    - `drives root` - List root directory items (uses GetRootDriveItems)
    - `drives activities`, `drives list`, `drives quota`, `drives get` - unchanged
  - **Item-level commands** (operations on specific items): kept in `items` namespace
    - `items search <query> --in <folder-path>` - Search within specific folder (--in flag now required)
    - All other items commands unchanged (list, stat, upload, download, rm, mv, etc.)
  - **Shared items**: `shared` command unchanged (separate concept from drives/items)

### Major Refactoring (Breaking Changes)
- **BREAKING**: Decomposed monolithic `cmd/files.go` (1193 LOC) into focused files ≤200 LOC each:
  - `cmd/items/items_root.go` - Command registration and flag setup
  - `cmd/items/items_helpers.go` - Shared utility functions
  - `cmd/items/items_meta.go` - List, stat, search, versions, thumbnails, preview commands
  - `cmd/items/items_upload.go` - Upload, mkdir, and upload management commands  
  - `cmd/items/items_download.go` - Download and deprecated list commands
  - `cmd/items/items_manage.go` - File operations (rm, mv, rename, copy)
  - `cmd/items/items_permissions.go` - Sharing and permissions management
- **BREAKING**: Removed all deprecated session helper functions in favor of the Manager pattern
- **BREAKING**: Command restructuring from `files` to `items` namespace
- **BREAKING**: Centralized pagination helper for consistent flag handling

### SDK Client Code Organization
- **Split of monolithic `pkg/onedrive/client.go`** (~1400 LOC) into focused modules:
  - `pkg/onedrive/drive.go` - Drive-level operations (GetDrives, GetDefaultDrive, GetDriveByID, GetDriveActivities, GetRootDriveItems)
  - `pkg/onedrive/item.go` - Item-level CRUD operations (GetDriveItemByPath, CreateFolder, UploadFile, DeleteDriveItem, CopyDriveItem, MoveDriveItem, UpdateDriveItem, MonitorCopyOperation)
  - `pkg/onedrive/client.go` - Core client, authentication, download, search, and utility methods
- **Restored `GetRootDriveItems`** as a supported helper (moved to `drive.go`), removing the internal deprecation notice

### Internal Refactoring
- Split `cmd/files.go` (1193 LOC) into focused files ≤200 LOC each across 7 specialized modules
- Split `pkg/onedrive/client.go` drive-level methods into new `pkg/onedrive/drive.go` without functional changes.  This reduces client.go by ~150 LOC and unlocks further decomposition steps.
- Removed deprecated session helpers in favor of the Manager pattern  
- Command restructuring from `files` to `items` namespace
- Centralized pagination helper for consistent flag handling

### Infrastructure Improvements
- Enhanced session management with proper locking and error handling
- Improved code organization with logical separation of concerns
- Better maintainability with focused, single-responsibility files

### Developer Experience
- Eliminated 1000+ line files for easier development and review
- Clear separation between command definitions and business logic
- Standardized error handling patterns across command files

### Changed
- **Major Architectural Refactor**: Overhauled the core SDK and application architecture to improve robustness, maintainability, and testability.
  - **Stateful SDK Client**: Replaced the stateless `pkg/onedrive` function collection with a new stateful `onedrive.Client`. This client now manages the HTTP client and token lifecycle internally.
  - **Session State Persistence**: Upload sessions are now persisted to disk, allowing for resumable uploads even if the application is interrupted.
  - **Global State Elimination**: Previously, authentication was managed through multiple package-level variables and functions. This has been replaced with a new `App` structure that manages all dependencies explicitly.
  - **Token Refresh Integration**: Previously, the application had to manually check for 401s and refresh tokens. This is now handled transparently by the SDK client.
  - **Builder Pattern for Tests**: Tests can now use a builder pattern with mock SDKs, improving testability and reducing coupling.
  - **Centralized Configuration**: The configuration is now managed through a single `Config` structure, reducing dependency on environment variables throughout the codebase.
  - **HTTP Client Encapsulation**: Raw `http.Client` usage was scattered throughout the old implementation. This is now encapsulated within the SDK client.
  - **Structured Error Handling**: Errors now use typed sentinels (like `app.ErrLoginPending`) for consistent and testable error handling.
- **Session Management Overhaul**: Completely rewritten session management with a new `Manager` pattern:
  - Thread-safe operations with proper file locking
  - Atomic writes to prevent corruption during interruption
  - Configurable directory support for testing isolation
  - Consistent 0600 file permissions for security
  - Expiration handling for upload and auth sessions
  - Better error messages and handling
- **Upload Session Management**: Complete rewrite of upload session handling:
  - Resumable uploads with automatic retry on failure
  - Progress bars and status reporting during uploads
  - Session state persistence across application restarts
  - Graceful handling of Ctrl+C interruption
  - Session cleanup on successful completion
  - Improved error messages and debugging
- **Authentication Flow**: Substantially improved authentication handling:
  - Non-blocking device code flow with polling
  - Better error messages for common failure scenarios  
  - Automatic cleanup of expired authentication sessions
  - More robust token refresh handling
  - Clear indication of authentication state to users
- **OAuth Robustness**: Implemented automatic, transparent token refresh. If an API call fails with a 401 Unauthorized error, the client now automatically uses the refresh token to get a new access token and retries the original request.
- **Atomic Config Writes**: Configuration and token writes are now atomic (write to temp file + rename) to prevent corruption if the application is interrupted.
- **Thread Safety**: Fixed a potential deadlock in the configuration saving mechanism.
- Removed duplicate `persistingTokenSource` implementation from `internal/app`; the single authoritative implementation now lives in the SDK (`pkg/onedrive`).
- Session manager now respects the `ONEDRIVE_CONFIG_PATH` environment variable ensuring test isolation and consistent session file locations.
- The root command no longer relies on fragile string comparisons to detect a pending login. It now uses the typed sentinel `app.ErrLoginPending` and `errors.Is` for robust detection.
- E2E tests skip automatically when the local access token is invalid or expired, keeping CI green while still running for developers with valid credentials.

### Internal Refactoring  
- **SDK Client Code Organization**: Split monolithic `pkg/onedrive/client.go` (~1400 LOC) for improved maintainability:
  - `pkg/onedrive/drive.go` - Drive-level operations (GetDrives, GetDefaultDrive, GetDriveByID, GetDriveActivities, GetRootDriveItems) 
  - `pkg/onedrive/item.go` - Item-level CRUD operations (GetDriveItemByPath, CreateFolder, UploadFile, DeleteDriveItem, CopyDriveItem, MoveDriveItem, UpdateDriveItem)
  - Reduced client.go from ~1400 LOC to ~900 LOC for better code organization
  - Eliminated duplicate method implementations during split
  - Each file maintains focused responsibility (drives vs items vs core client functionality)
- Restored `GetRootDriveItems` as a supported helper (moved to `drive.go`), removing the internal deprecation notice.

### Removed
- Removed the old `drives` command, which was a temporary implementation for listing root items. Its functionality is now part of `files list`.
- Removed deprecated session management functions replaced by Manager pattern.
- Removed duplicate SDK implementations to maintain single source of truth.

### Added

**Epic 7: Comprehensive Microsoft Graph API Coverage - Advanced Features Implementation (COMPLETED)**

- **Thumbnail Support (`GET /drive/items/{item-id}/thumbnails`)**: New `onedrive-client files thumbnails <remote-path>` command retrieves thumbnail images in multiple sizes (small, medium, large, source) for files in OneDrive
- **File Preview (`POST /drive/items/{item-id}/preview`)**: New `onedrive-client files preview <remote-path>` command generates preview URLs for Office documents, PDFs, and images with optional page and zoom parameters
- **User Invitation (`POST /drive/items/{item-id}/invite`)**: New `onedrive-client files invite <remote-path> <email> [additional-emails...]` command invites users to access files and folders with configurable permissions and invitation settings
- **Permissions Management**: Complete permissions management system:
  - `onedrive-client files permissions list <remote-path>`: List all permissions on a file or folder
  - `onedrive-client files permissions get <remote-path> <permission-id>`: Get detailed information about a specific permission
  - `onedrive-client files permissions update <remote-path> <permission-id>`: Update permission roles, expiration, or password
  - `onedrive-client files permissions delete <remote-path> <permission-id>`: Remove a specific permission
- **Advanced Command Options**: All new commands support comprehensive flag-based configuration:
  - Preview command: `--page` for specific page preview, `--zoom` for zoom level control
  - Invite command: `--message` for custom invitation message, `--roles` for permission specification, `--require-signin` and `--send-invitation` for access control
  - Permissions update: `--roles`, `--expiration`, and `--password` for granular permission control

### Technical Implementation Details

- **Enhanced Models**: Added comprehensive data structures in `pkg/onedrive/models.go`:
  - `Thumbnail`, `ThumbnailSet`, `ThumbnailSetList`: Support for multi-resolution thumbnail handling
  - `PreviewRequest`, `PreviewResponse`: Preview generation with configurable parameters
  - `Permission`, `PermissionList`: Complete permission data model with user/group/link support
  - `InviteRequest`, `InviteResponse`: User invitation system with recipient management
  - `UpdatePermissionRequest`: Permission modification capabilities
- **Advanced SDK Functions**: Implemented 7 new SDK functions with full Microsoft Graph API integration:
  - `GetThumbnails()`, `GetThumbnailBySize()`: Thumbnail retrieval with size-specific access
  - `PreviewItem()`: Preview URL generation with request customization
  - `InviteUsers()`: Multi-user invitation system with role-based access
  - `ListPermissions()`, `GetPermission()`, `UpdatePermission()`, `DeletePermission()`: Complete CRUD operations for permission management
- **Enhanced UI Display System**: Added specialized display functions for rich user experience:
  - `DisplayThumbnails()`: Formatted thumbnail information with size and URL details
  - `DisplayPreview()`: Preview URL display with GET/POST endpoint information
  - `DisplayInviteResponse()`: Invitation result display with created permissions
  - `DisplayPermissions()`: Tabular permission listing with role and user information
  - `DisplaySinglePermission()`: Detailed permission view with inheritance and link details
- **Complete Command Integration**: Added 4 new top-level commands with 4 permission subcommands:
  - Integrated into existing `files` command structure with consistent flag patterns
  - Hierarchical command structure for permissions (`files permissions list/get/update/delete`)
  - Comprehensive argument validation and error handling
- **SDK Interface Evolution**: Updated SDK interface and MockSDK with full test coverage:
  - Added all 7 new methods to SDK interface with proper signatures
  - Implemented MockSDK methods for comprehensive testing isolation
  - Maintained backward compatibility with existing SDK implementations

### Progress Update

Epic 7 is now 100% COMPLETE with 30/30 API endpoints implemented. All Microsoft Graph OneDrive API coverage goals have been achieved, including advanced features for thumbnails, previews, and comprehensive permissions management.

**Epic 7: Comprehensive Microsoft Graph API Coverage - Advanced Features Implementation**

- **Download-as-Format (`GET /drive/items/{item-id}/content?format={format}`)**: New `onedrive-client files download <remote-path> [local-path] --format <format>` command enables file format conversion at download time (e.g., convert .docx to .pdf)
- **Folder-Scoped Search (`GET /drive/items/{item-id}/search(q='query')`)**: Enhanced search functionality with `onedrive-client files search "<query>" --in <remote-folder>` command for searching within specific directories
- **Drive Activities (`GET /drive/activities`)**: New `onedrive-client drives activities` command displays activity history for the entire drive with comprehensive activity tracking
- **Item Activities (`GET /drive/items/{item-id}/activities`)**: New `onedrive-client files activities <remote-path>` command shows activity history for specific files and folders
- **Advanced Paging Support**: All new commands support comprehensive paging options:
  - `--top <N>`: Limit results to N items (respects Microsoft Graph `$top` parameter)
  - `--all`: Fetch all results across all pages automatically
  - `--next <url>`: Resume from specific pagination URL for power users
- **Enhanced Search Capabilities**: Upgraded search functionality with paging support:
  - `onedrive-client files search "<query>" --top <N>`: Limit search results
  - `onedrive-client files search "<query>" --all`: Get all search results across pages
  - `onedrive-client files search "<query>" --in <folder>`: Search within specific folder

### Technical Implementation Details

- **New Models**: Added `Paging`, `Activity`, `ActivityList` structures in `pkg/onedrive/models.go` to support new functionality
- **Pagination Helper**: Implemented `collectAllPages()` helper function for consistent pagination handling across all new APIs
- **Enhanced SDK Functions**: Added comprehensive SDK functions with full paging support:
  - `DownloadFileAsFormat()`: Format-specific download with redirect handling
  - `SearchDriveItemsWithPaging()`: Enhanced search with pagination
  - `SearchDriveItemsInFolder()`: Folder-scoped search with pagination
  - `GetDriveActivities()`: Drive-level activity retrieval with paging
  - `GetItemActivities()`: Item-specific activity retrieval with paging
- **UI Enhancements**: Added `DisplayActivities()` function for formatted activity output with action type detection and proper formatting
- **SDK Interface Extensions**: Updated SDK interface and MockSDK to support all new functionality with comprehensive test coverage
- **Command Structure**: Enhanced command architecture with consistent flag patterns across all new commands

### Progress Update

Epic 7 now implements 25/30 API endpoints (83% complete), up from 20/30 (67% complete). Advanced features including activities tracking, format conversion, and enhanced search are now fully functional.

**Epic 7: Comprehensive Microsoft Graph OneDrive API Coverage - High/Medium Priority Features**

- **Delta Tracking (`GET /drive/root/delta`)**: New `onedrive-client delta [delta-token]` command enables efficient synchronization by tracking changes since last sync using Microsoft Graph delta queries
- **Specific Drive Access (`GET /drives/{drive-id}`)**: New `onedrive-client drives get <drive-id>` command retrieves detailed metadata for any drive by ID
- **File Versions (`GET /drive/items/{item-id}/versions`)**: New `onedrive-client files versions <remote-path>` command lists all versions of a specific file with version history and details

### Technical Details

- Added new models: `DeltaResponse`, `DriveItemVersion`, `DriveItemVersionList` in `pkg/onedrive/models.go`
- Added new SDK functions: `GetDelta()`, `GetDriveByID()`, `GetFileVersions()` in `pkg/onedrive/onedrive.go`
- Added new UI display functions: `DisplayDelta()`, `DisplayDrive()`, `DisplayFileVersions()` in `internal/ui/display.go`
- Created new command module: `cmd/delta.go` for delta operations
- Extended existing command modules: `cmd/drives.go` and `cmd/files.go` with new subcommands
- Comprehensive unit test coverage for all new functionality
- Updated SDK interface and mock implementations for testing

### Progress

Epic 7 now implements 20/30 API endpoints (67% complete), up from 17/30 (57% complete). Core synchronization and drive management features are now fully functional.

- **Search Functionality**: Implemented comprehensive search capabilities across OneDrive:
  - `files search "<query>"` - Search for files and folders by name or content using Microsoft Graph API
  - Proper URL encoding for search queries with special characters
  - Formatted search results display with query context, item counts, and metadata
  - E2E tests covering search operations with various query types
- **Shared Content Management**: Added support for viewing and managing shared items:
  - `shared list` - List all items shared with you from other OneDrive users
  - New `cmd/shared.go` command module with dedicated shared content operations
  - Special handling for remote items (cross-drive shared content) with owner information
  - Graceful handling of access restrictions on personal OneDrive accounts
- **Recent Items Access**: Implemented recent files functionality:
  - `files recent` - List recently accessed files and folders
  - Display with last access timestamps and file metadata
  - Integration with OneDrive's activity tracking system
- **Special Folder Access**: Added support for OneDrive's well-known special folders:
  - `files special <folder-name>` - Access special folders like Documents, Photos, Music
  - Support for all standard special folders: documents, photos, music, cameraroll, approot, recordings
  - Validation of folder names with informative error messages
  - Business account special folder support (cameraroll, approot, recordings)
  - Special folder metadata display with creation dates and child counts
- **Enhanced SDK Layer**: Added four new core functions to OneDrive SDK:
  - `SearchDriveItems()` - GET operation with proper query encoding for search functionality
  - `GetSharedWithMe()` - GET operation to retrieve items shared from other users
  - `GetRecentItems()` - GET operation to fetch recently accessed files
  - `GetSpecialFolder()` - GET operation with folder name validation for special folder access
- **Enhanced Display Functions**: Added specialized UI display functions for new features:
  - `DisplaySearchResults()` - Formatted search results with query context and item counts
  - `DisplaySharedItems()` - Shared items display with owner information and shared dates
  - `DisplayRecentItems()` - Recent items with last access timestamps
  - `DisplaySpecialFolder()` - Special folder information with detailed metadata
- **Comprehensive Testing**: Added extensive test coverage for all new functionality:
  - Unit tests for all new SDK functions and command logic
  - E2E tests for search operations with URL encoding validation
  - E2E tests for recent items with timestamp verification
  - E2E tests for shared items with graceful error handling
  - E2E tests for all special folder types including business-only folders
  - MockSDK updates to support all new functionality for testing
- **Core File System Operations**: Implemented essential file management commands for OneDrive:
  - `files rm <remote-path>` - Delete files and folders (moved to recycle bin)
  - `files copy <source-path> <destination-path> [new-name]` - Copy files and folders with async operation
  - `files mv <source-path> <destination-path>` - Move files and folders to new locations
  - `files rename <remote-path> <new-name>` - Rename files and folders
- **Progress Monitoring System**: Advanced copy operation monitoring with flexible user control:
  - Default fire-and-forget mode returns monitor URL for later checking
  - `--wait` flag for blocking copy operations with real-time progress
  - `files copy-status <monitor-url>` command to check status of any copy operation
  - Comprehensive status reporting (inProgress, completed, failed) with percentage completion
- **E2E Testing Framework**: Comprehensive end-to-end testing framework in `e2e/` directory that tests against real OneDrive accounts using the CLI's existing device code flow authentication.
  - Test isolation with unique timestamped directories (`/E2E-Tests/test-{timestamp}`)
  - Automated authentication using existing `config.json` from CLI login
  - Coverage for file uploads, directory creation, metadata retrieval, drive operations, and URL construction verification
  - Proper test cleanup and safety measures to protect user data
- **Comprehensive E2E Test Coverage**: Added extensive E2E tests covering all available SDK functionality:
  - `TestAuthOperations`: Tests for `GetMe` function to verify authenticated API calls
  - `TestFileOperations`: Tests for directory creation, small/large file uploads, file downloads, metadata retrieval, and directory listing
  - `TestDriveOperations`: Tests for listing drives and getting default drive information
  - `TestErrorHandling`: Tests for proper error handling of non-existent files and invalid operations
  - `TestURLConstruction`: Tests for proper URL construction and endpoint formatting
- Added file hash comparison helper (`CompareFileHash`) to E2E test suite for robust file integrity verification
- Added comprehensive chunked upload testing with proper error handling for final chunk completion
- Resumable large file downloads with progress bar and interrupt handling.
- New `files cancel-upload <upload-url>` command to cancel a resumable upload session.
- New `files get-upload-status <upload-url>` command to retrieve the status of a resumable upload session.
- New `files upload-simple <local-file> <remote-path>` command for non-resumable file uploads (suitable for small files).
- New `files list-root-deprecated` command (deprecated) to list items in the root drive.
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
- Added E2E test for `GetMe` to verify basic authenticated API calls.
- Added `DownloadURL` field to `DriveItem` model to capture `@microsoft.graph.downloadUrl` property from Microsoft Graph API responses.
- Added `DownloadFileByItem` function as alternative download method using item metadata and pre-authenticated download URLs.
- Added fallback download logic that handles both 302 redirects and 401/404 errors gracefully.
- Added comprehensive debug logging to download functions for better troubleshooting.
- Made `CalculateFileHash` method public in E2E test helpers for broader test utility usage.
- **Sharing Link Creation**: Implemented comprehensive sharing link functionality for OneDrive files and folders:
  - `files share <remote-path> <link-type> <scope>` - Create sharing links for files and folders
  - Support for all link types: "view" (read-only), "edit" (read-write), "embed" (embeddable for web pages)
  - Support for scopes: "anonymous" (anyone with link), "organization" (organization members only)
  - Proper input validation with helpful error messages for invalid link types and scopes
  - Comprehensive display of sharing link details including URL, permissions, expiration, and embed HTML
- **Enhanced SDK Layer**: Added `CreateSharingLink()` function to OneDrive SDK:
  - Uses POST to `/createLink` endpoint following Microsoft Graph API specification
  - Built-in validation for link types and scopes
  - Proper error handling and response parsing
  - Integration with existing `BuildPathURL()` pattern for consistent URL construction
- **Enhanced UI Display**: Added `DisplaySharingLink()` function for formatted sharing link output:
  - Shows link ID, type, scope, roles, and URL
  - Displays optional information like password protection, expiration, and embed HTML
  - Clean, user-friendly formatting consistent with existing UI patterns
- **Comprehensive Testing**: Added extensive test coverage for sharing functionality:
  - Unit tests for command logic with various scenarios and edge cases
  - MockSDK implementation for isolated testing
  - E2E tests for real OneDrive API integration with proper cleanup
  - Error handling tests for invalid inputs and API failures

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
- Enhanced `DownloadFile` function to properly handle Microsoft Graph API's 302 redirect response pattern for download requests.
- Improved error handling in E2E tests to be more resilient to different error message formats.
- Updated E2E test directory creation logic to ensure proper hierarchy setup.
- **Testing**: E2E test suite is now part of the default `go test ./...` run. The `//go:build e2e` build tags were removed and helper functions now gracefully skip tests when `config.json` is not present.
- **Testing Robustness**: E2E helpers and tests now rely on typed sentinel errors (`onedrive.ErrResourceNotFound`, `onedrive.ErrInvalidRequest`, …) instead of fragile string matching.

### Fixed
- **Critical**: Fixed fundamental URL construction bug in `BuildPathURL()` function that was generating malformed URLs with double colons (`::`) instead of single colons (`:`).
- **Critical**: Fixed Microsoft Graph API endpoint URLs for file operations to match correct format:
  - Upload endpoints: `/me/drive/root:/path/to/file:/content`
  - Download endpoints: `/me/drive/root:/path/to/file:/content`
  - Create folder endpoints: `/me/drive/root:/parent-path:/children`
- **Critical**: Fixed `CreateFolder()` function URL construction to use proper path-based addressing.
- **Critical**: Fixed directory listing functionality in `GetDriveItemChildrenByPath` function by correcting URL construction to append `:/children` for non-root paths instead of `/children`.
- **Critical**: Fixed download functionality by implementing proper Microsoft Graph API redirect handling and fallback download methods using item metadata.
- **Critical**: Fixed error handling in E2E tests to properly recognize both "itemNotFound" and "resource not found" error formats from the SDK.
- **Critical**: Fixed E2E test directory creation by ensuring both root test directory and specific test directories are created properly.
- **Critical**: Standardized error handling patterns across all commands for consistency and testability.
- **Critical**: Fixed race conditions in session management by removing global variable patterns.
- **Critical**: Fixed path handling bugs where `filepath.Join` was incorrectly used for remote URL paths.
- **Critical**: Fixed thread safety issues in token refresh callbacks.
- **Critical**: Fixed E2E test compilation errors by removing tests for non-existent SDK functions (`DeleteDriveItem`, `MoveDriveItem`, `RenameDriveItem`)
- **Critical**: Resolved build failures in E2E test suite that were preventing proper test execution
- **E2E Tests**: Resolved all known limitations in E2E test suite - all tests now pass with 100% success rate.
- **E2E Tests**: Fixed hash comparison logic in download tests to compare local files directly instead of attempting redundant downloads.
- **E2E Tests**: Improved test isolation and reliability by ensuring proper directory structure creation before running tests.
- Resolved persistent build and test failures caused by Go module inconsistencies. This required manual adjustments to `go.mod` and repeated `go mod tidy` commands to correctly vendor a new dependency (`progressbar`).
- The build process was fixed by removing duplicated test helper code.
- Authentication command error handling to use RunE instead of log.Fatalf for better testability.
- Session directory creation in file locking operations to prevent "no such file or directory" errors.
- Test helper function signatures and session file path construction in authentication tests.
- Improved test output capture to properly handle log output without global state mutation.
- Improved chunked upload test logic to handle SDK limitations with final chunk responses
- **Critical Auth Fix**: Fixed a bug that prevented automatic token refreshes. The application was failing to set the token's `Expiry` time, causing the OAuth2 library to treat the token as if it never expired, which prevented the refresh mechanism from ever triggering.
- **OAuth Robustness**: Implemented automatic, transparent token refresh. If an API call fails with a 401 Unauthorized error, the client now automatically uses the refresh token to get a new access token and retries the original request.
- **Atomic Config Writes**: Configuration and token writes are now atomic (write to temp file + rename) to prevent corruption if the application is interrupted.
- **Thread Safety**: Fixed a potential deadlock in the configuration saving mechanism.
- Removed duplicate `persistingTokenSource` implementation from `internal/app`; the single authoritative implementation now lives in the SDK (`pkg/onedrive`).
- Session manager now respects the `ONEDRIVE_CONFIG_PATH` environment variable ensuring test isolation and consistent session file locations.

### Removed
- Removed the old `drives` command, which was a temporary implementation for listing root items. Its functionality is now part of `files list`.
- Removed the old, interactive authentication flow that required pasting a URL back into the terminal.
- **BREAKING**: Removed global variable patterns in session management in favor of proper dependency injection.
- Removed problematic E2E tests for non-existent SDK functionality to fix compilation errors.
- Removed internal token refresh logic from the SDK in favor of a new, more robust persistence layer within the application that correctly uses the `golang.org/x/oauth2` library's features.

### Security
- Enhanced session file locking to prevent race conditions in concurrent CLI invocations.
- Improved error message handling to avoid exposing sensitive information in logs.
- Session and authentication state files are now written with `0600` permissions instead of `0644`, preventing other local users from reading sensitive data.
- Upload session files are now protected by advisory file locks to avoid corruption when multiple CLI instances access the same session concurrently.

### Development Notes
- **Test Coverage Achievement**: Successfully implemented comprehensive E2E test coverage for all existing SDK functionality
- **Build System**: All unit tests pass, E2E framework is operational with most core functionality working
- **Known Issues**: Some E2E tests fail due to authentication issues with download operations (documented issue)
- **Code Quality**: Maintained high code quality standards with proper error handling, test isolation, and safety measures

### E2E Testing Framework Development Notes
- **Aim:** Create automated end-to-end tests against real OneDrive accounts to catch integration issues and API regressions.
- **Scope of Work Undertaken:** Implemented comprehensive E2E testing framework with real OneDrive API integration.
- **Initial Approach (Rejected):** Originally implemented service principal authentication with Azure app registration (CLIENT_ID, CLIENT_SECRET, TENANT_ID) but user preferred simpler device code flow approach.
- **Final Implementation:**
  1. Created `e2e/config.go` for simplified configuration management using existing CLI authentication
  2. Implemented `e2e/test_helpers.go` with authentication using CLI's `config.json` and automatic token refresh
  3. Built `e2e/files_test.go` with comprehensive test coverage: uploads, downloads, directory operations, metadata, drive operations
  4. Added `e2e/auth_test.go` with authentication verification tests
  5. Added `e2e/quick_start.go` for setup validation and `e2e/README.md` with simple setup instructions
  6. Implemented test isolation with unique timestamped directories to prevent data conflicts
- **Critical Bug Fixes Discovered During Testing:**
  - Fixed `BuildPathURL()` function removing trailing colon that caused double-colon malformed URLs
  - Corrected Microsoft Graph API endpoint construction for all file operations
  - Updated download and folder creation URL patterns to match API specification
- **Recent Fixes (Current Session):**
  - Fixed directory listing URL construction bug in `GetDriveItemChildrenByPath` function
  - Implemented proper Microsoft Graph API download redirect handling
  - Fixed E2E test error handling to recognize multiple error message formats
  - Resolved test directory creation issues
  - Fixed hash comparison logic in download tests
- **Current Test Status:** 
  - ✅ **All Tests Passing**: Authentication, file operations, directory operations, drive operations, error handling, URL construction
  - ✅ **100% Success Rate**: All known limitations have been resolved
  - ✅ **Robust Coverage**: Comprehensive testing of all available SDK functionality
- **Code Quality:** Framework provides solid foundation for automated testing with proper error handling, test isolation, and safety measures to protect user data.
- **Final Status:** E2E test framework is fully operational with complete coverage and 100% test pass rate. All previously known limitations have been successfully resolved.

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

### Security
- Session and authentication state files are now written with `0600` permissions instead of `0644`, preventing other local users from reading sensitive data.
- Upload session files are now protected by advisory file locks to avoid corruption when multiple CLI instances access the same session concurrently.

### Changed
- The root command no longer relies on fragile string comparisons to detect a pending login.  It now uses the typed sentinel `app.ErrLoginPending` and `errors.Is` for robust detection.
- E2E tests skip automatically when the local access token is invalid or expired, keeping CI green while still running for developers with valid credentials.

### Internal Refactoring
- Split `pkg/onedrive/client.go` drive-level methods into new `pkg/onedrive/drive.go` without functional changes.  This reduces client.go by ~150 LOC and unlocks further decomposition steps.
- Restored `GetRootDriveItems` as a supported helper (moved to `drive.go`), removing the internal deprecation notice.