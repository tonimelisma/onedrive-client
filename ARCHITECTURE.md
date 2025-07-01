# Architecture Document: onedrive-client (v2)

## 1. Purpose and Guiding Principles

This document outlines the technical architecture for the `onedrive-client` application. Its purpose is to provide a clear and consistent guide for developers, ensuring that the project remains maintainable, testable, and extensible as it evolves.

All development should adhere to these core principles:

1.  **Separation of Concerns:** Each component has a single, well-defined responsibility. The CLI is separate from the SDK, and the SDK is separate from the core application logic.
2.  **Testability:** Application logic must be testable without live network calls, achieved through interfaces and dependency injection.
3.  **Extensibility:** Adding new commands should be simple and not require major refactoring.

This document is divided into two parts:
*   **Current Architecture:** Describes the application as it is currently built.
*   **Future Architecture:** Outlines the vision for improving the design, extracting the SDK, and adding a background sync engine.

---

## 2. Current Architecture: A Command-Driven Utility

The application functions as a stateless, command-driven utility. The user executes a command, the action is performed, and the application exits. State is managed across invocations for specific features like authentication and resumable uploads via session files.

### 2.1. High-Level Data Flow

The flow for any given command is as follows:

```
[User] -> [main.go] -> [Cobra CLI (cmd/)] -> [App Core (internal/app)] -> [SDK Interface] -> [OneDrive SDK (pkg/onedrive)] -> [MS Graph API]
  ^
  |                                                                                                                        |
  +------------------------------------ [UI (internal/ui)] <----------------------- [Cobra CLI (cmd/)] <--------------------+
```

The `internal/app` component now creates a special `http.Client` that automatically refreshes tokens and persists them on change, abstracting this complexity away from both the commands and the SDK.

### 2.2. Component Breakdown

The project is organized into packages, each with a distinct role.

```
/onedrive-client/
├── main.go               // Application entry point. Executes the root command.
├── go.mod                // Manages project dependencies.
├── cmd/                  // All CLI command definitions (using Cobra).
│   ├── root.go
│   ├── files.go
│   ├── drives.go
│   └── auth.go
├── e2e/                  // End-to-end testing framework for real OneDrive API integration.
│   ├── config.go         // E2E test configuration management
│   ├── test_helpers.go   // Authentication and utility functions for E2E tests
│   ├── files_test.go     // Comprehensive file operation tests
│   ├── quick_start.go    // Setup validation and test environment verification
│   ├── README.md         // E2E testing setup and usage instructions
│   └── .gitignore        // Excludes test config files from version control
└── internal/
    ├── app/              // Core application logic (initialization, SDK abstraction).
    │   ├── app.go
    │   └── sdk.go
    ├── config/           // Configuration loading and saving (e.g., tokens).
    │   └── config.go
    ├── session/          // Manages temporary state for multi-step operations.
    │   ├── auth.go       // Handles the pending auth session.
    │   └── session.go    // Handles resumable upload sessions.
    └── ui/               // User interface formatting and output.
        └── display.go
└── pkg/
    └── onedrive/         // The Go SDK for interacting with the OneDrive API.
        ├── onedrive.go
        └── models.go
```

#### `cmd/` (The Command Layer)
*   **Technology:** [Cobra](https://cobra.dev/).
*   **Responsibility:** Defines the command structure (`files`, `drives`, etc.), arguments, and flags. The `RunE` function for each command is responsible for:
    1.  Initializing the App Core (`app.NewApp()`).
    2.  Calling the appropriate SDK function via the App Core's `SDK` interface.
    3.  Passing the results (or errors) to the UI Layer (`internal/ui`) for display.
*   **Constraint:** This layer contains **no business logic**. It only orchestrates calls between the user, the app core, and the UI.

**CLI command hierarchy update (2025-06-30):** The former `files`, `delta`, and `shared` root commands have been consolidated into a single `items` root command.  All previous subcommands are unchanged but are now invoked as `items <subcommand>`.

#### `internal/app/` (The App Core & SDK Abstraction)
*   **Responsibility:** Acts as the central hub for the application.
    *   `app.go`: The `NewApp()` function initializes the configuration and the OneDrive HTTP client. Crucially, it detects and completes any pending authentication flows, ensuring that any command that runs can assume it has a valid, authenticated client.
    *   `sdk.go`: Defines the `SDK` interface, which decouples the command layer from the concrete SDK implementation. This is key for testability. It also provides the `OneDriveSDK` struct which wraps the real `pkg/onedrive` functions.

#### `internal/config/` (Configuration Management)
*   **Responsibility:** Handles all logic for loading, parsing, and saving the `config.json` file. This file stores the final OAuth tokens and is located in the user's configuration directory (e.g., `~/.config/onedrive-client/`).

#### `internal/session/` (Session Management)
*   **Responsibility:** Manages temporary state files required for operations that span multiple CLI invocations. It uses file-locking to prevent race conditions from concurrent commands.
    *   `auth.go`: Manages the `auth_session.json` file, which stores the details of a pending Device Code Flow login.
    *   `session.go`: Manages session files for resumable uploads. It creates a unique session file for each upload, named with a SHA256 hash of the local and remote file paths. This allows the `upload` command to resume if interrupted.

#### `internal/ui/` (The Presentation Layer)
*   **Responsibility:** Handles all user-facing output. This includes printing tables of files, progress bars, success messages, and formatted errors.

#### `pkg/onedrive/` (The SDK Layer)
*   **Responsibility:** This package is the **only** component that knows how to communicate with the Microsoft Graph API. It handles creating API requests, parsing responses, and defining the data models (`DriveItem`, etc.).
*   **Authentication**: It implements the raw mechanics of the OAuth 2.0 Device Code Flow but has no awareness of the higher-level application's stateful, non-blocking flow. It simply provides the functions to initiate the flow and verify a device code.
*   **Independence:** This package has no dependencies on any other package in the project (`internal/`, `cmd/`), making it a candidate for future extraction into a standalone library.

#### Token Refresh & Persistence (Refined)
Prior refactors introduced two independent `persistingTokenSource` wrappers (one in `internal/app` and one inside the SDK).  The duplication led to divergent error-handling behaviour and extra maintenance overhead.  As of vNEXT the application relies exclusively on the implementation inside the SDK (`pkg/onedrive`).  The redundant version and its unit tests have been removed from `internal/app`.  All token persistence and refresh callbacks are therefore centralised in a single location and consumed transparently via `onedrive.NewClient()`.

### 2.3. Key Architectural Patterns

#### Non-Blocking Authentication Flow
The application uses a stateful, non-blocking implementation of the OAuth 2.0 Device Code Flow, providing a seamless CLI experience.
1.  **`auth login`**: The user runs this command. The CLI gets a `user_code` and `device_code` from Microsoft, saves them to `auth_session.json`, displays the code to the user, and immediately exits.
2.  **User Action**: The user authorizes the code in a browser.
3.  **Any Subsequent Command**: When the user runs another command (e.g., `files list`), `app.NewApp()` detects the `auth_session.json` file, automatically exchanges the `device_code` for the final tokens, saves them to `config.json`, and deletes the session file before proceeding.

#### Resumable Uploads
Large file uploads are handled via resumable sessions to be resilient to network interruptions.
1.  **`files upload`**: The command first checks if a session file exists for the given local and remote file paths.
2.  **New Upload**: If no session exists, it calls the SDK to create an upload session with the Graph API and saves the unique `uploadUrl` to a new session file.
3.  **Resumed Upload**: If a session file exists, it reads the `uploadUrl` and queries the API for the last successfully uploaded byte range.
4.  **Chunking**: The file is then uploaded in chunks. After each successful chunk upload, the progress is implicitly saved on the server side. If the command is interrupted, running it again will resume from the last completed chunk.
5.  **Completion**: Upon successful upload of all chunks, the session file is deleted.

#### Upload Session Management
The application provides advanced upload session management capabilities for fine-grained control over resumable uploads.
1.  **Session Status**: The `files get-upload-status` command allows users to query the current status of any active upload session using its upload URL.
2.  **Session Cancellation**: The `files cancel-upload` command enables users to cancel unwanted upload sessions, freeing server resources.
3.  **Simple Uploads**: The `files upload-simple` command provides a non-resumable upload option optimized for small files that don't require session management.
4.  **Legacy Support**: The `files list-root-deprecated` command maintains compatibility with the older root listing method.

#### Resumable Downloads
Large file downloads are handled via chunked downloads with session state management for resumption.
1.  **`files download`**: The command checks for existing session files to resume interrupted downloads.
2.  **Chunked Downloads**: Files are downloaded in chunks using HTTP Range requests to minimize memory usage.
3.  **Session Tracking**: Download progress is saved to session files, allowing seamless resumption after interruption.
4.  **Completion**: Upon successful download, the session file is automatically cleaned up.

#### Progress Monitoring for Async Operations
The application implements a sophisticated progress monitoring system for async operations like file copying, providing users with flexible control over operation tracking.

**Hybrid Monitoring Approach:**
1. **Fire-and-Forget (Default)**: `files copy src dest` returns immediately with a monitor URL for later checking
2. **Blocking Mode**: `files copy --wait src dest` polls until completion with real-time progress updates  
3. **Manual Monitoring**: `files copy-status <monitor-url>` allows checking any operation's status

**Implementation Details:**
- Copy operations return HTTP 202 with Location header containing monitor URL
- Monitor URL supports polling with different HTTP status codes indicating progress:
  - 202: Operation in progress (may include percentage complete)
  - 303: Operation completed (Location header points to new resource)
  - 4xx/5xx: Operation failed with error details
- Progress updates include status, percentage completion, and descriptive messages
- Blocking mode polls every 2 seconds until completion or failure

#### Core File System Operations
The application provides full CRUD operations for OneDrive files and folders through four core SDK functions:

1. **`DeleteDriveItem()`**: Uses DELETE HTTP method to move items to recycle bin (not permanent deletion)
2. **`CopyDriveItem()`**: Uses POST to `/copy` endpoint, returns monitor URL for async operation tracking
3. **`MoveDriveItem()`**: Uses PATCH to update `parentReference` property for item relocation
4. **`UpdateDriveItem()`**: Uses PATCH to update item properties like `name` for renaming

**Path-Based Addressing:** All operations use the existing `BuildPathURL()` pattern for consistent URL construction (`/me/drive/root:/path:`).

**Error Handling:** Leverages existing `apiCall()` function for standardized OneDrive API error categorization and retry logic.

#### Sharing Link Management
The application provides comprehensive sharing link functionality for OneDrive files and folders through the Microsoft Graph API's createLink endpoint:

1. **`CreateSharingLink()`**: Uses POST to `:/createLink` endpoint to generate sharing links
   - **Supported Link Types**: "view" (read-only), "edit" (read-write), "embed" (embeddable for web pages)
   - **Supported Scopes**: "anonymous" (anyone with link), "organization" (organization members only)
   - **Input Validation**: Built-in validation for link types and scopes with descriptive error messages
   - **Response Handling**: Parses complete sharing link response including URL, permissions, expiration, and embed HTML

**CLI Interface:** `files share <remote-path> <link-type> <scope>` command provides user-friendly access to sharing functionality.

**Error Handling:** Uses standard `apiCall()` function for consistent error handling and categorization.

**Display Integration:** `DisplaySharingLink()` function provides formatted output of sharing link details consistent with application UI patterns.

#### Advanced Features and Pagination (Latest Implementation)
The application now provides comprehensive support for advanced Microsoft Graph OneDrive features with sophisticated pagination capabilities:

**Download Format Conversion:**
1. **`DownloadFileAsFormat()`**: Uses GET to `/drive/items/{item-id}/content?format={format}` endpoint for file format conversion
   - **Format Support**: Enables conversion of Office documents (e.g., .docx to .pdf, .xlsx to .csv)
   - **Redirect Handling**: Properly handles Microsoft Graph's 302 redirect pattern for download URLs
   - **CLI Interface**: `onedrive-client files download <remote-path> [local-path] --format <format>` command
   - **Error Handling**: Robust error handling for unsupported formats and conversion failures

**Activities Tracking:**
1. **`GetDriveActivities()`**: Uses GET to `/drive/activities` endpoint for drive-level activity history
   - **Comprehensive Actions**: Tracks create, edit, delete, move, rename, share, comment, version, restore, and mention actions
   - **Actor Information**: Captures user details and timestamps for all activities
   - **CLI Interface**: `onedrive-client drives activities` command with full pagination support
   
2. **`GetItemActivities()`**: Uses GET to `/drive/items/{item-id}/activities` endpoint for item-specific activity history
   - **Item-Specific Tracking**: Focused activity history for individual files and folders
   - **Detailed Context**: Includes action-specific details like old names for rename operations
   - **CLI Interface**: `onedrive-client files activities <remote-path>` command

**Enhanced Search Capabilities:**
1. **`SearchDriveItemsWithPaging()`**: Enhanced version of existing search with full pagination support
   - **Backward Compatibility**: Maintains compatibility with existing search functionality
   - **Pagination Control**: Supports `--top`, `--all`, and `--next` flags for flexible result management
   
2. **`SearchDriveItemsInFolder()`**: Uses GET to `/drive/items/{item-id}/search(q='query')` for folder-scoped search
   - **Scoped Search**: Enables searching within specific directory trees
   - **Path Resolution**: Automatically converts folder paths to item IDs for API calls
   - **CLI Interface**: `onedrive-client files search "<query>" --in <remote-folder>` command

**Advanced Pagination System:**
The application implements a sophisticated pagination system that provides users with maximum flexibility:

1. **Paging Structure**: 
   ```go
   type Paging struct {
       Top      int    // 0 = Graph default (usually 200 items)
       FetchAll bool   // true → follow all @odata.nextLink URLs
       NextLink string // resume from specific pagination URL
   }
   ```

2. **Pagination Helper**: `collectAllPages()` function centralizes pagination logic:
   - **URL Parameter Handling**: Properly constructs `$top` query parameters
   - **Link Following**: Automatically follows `@odata.nextLink` URLs when `FetchAll` is true
   - **Resume Capability**: Supports resuming from any pagination URL for power users
   - **Memory Efficient**: Uses `json.RawMessage` for intermediate processing to minimize memory usage

3. **CLI Pagination Flags** (consistent across all commands):
   - `--top <N>`: Limit results to N items (respects Microsoft Graph limits)
   - `--all`: Automatically fetch all results across all pages
   - `--next <url>`: Resume from specific pagination URL (exposed for power users)
   - **Mutual Exclusivity**: `--all` overrides `--top` when both are specified

4. **User Experience**: 
   - **Default Behavior**: Returns first page (typically 200 items) for quick results
   - **Progress Indication**: Shows next page availability when applicable
   - **Power User Support**: Full pagination URLs exposed for scripting and automation

**Implementation Details:**
- **New Models**: `Activity`, `ActivityList`, `Paging` structures support comprehensive activity tracking and pagination
- **Error Handling**: Uses standard `apiCall()` function for consistent error categorization across all new endpoints
- **Display Functions**: `DisplayActivities()` provides formatted activity output with intelligent action type detection
- **Test Coverage**: Comprehensive unit tests for all new functionality with MockSDK updates
- **URL Construction**: Follows existing `customRootURL` + endpoint pattern with proper parameter encoding

**Integration with Existing Architecture:**
- **SDK Interface**: All new functions follow existing interface patterns for consistency and testability
- **Command Structure**: New commands integrate seamlessly with existing Cobra command hierarchy
- **UI Consistency**: Activity display follows established formatting patterns used throughout the application
- **Session Management**: Pagination state can be managed externally using the exposed next link URLs

This implementation significantly enhances the application's capabilities while maintaining architectural consistency and providing a foundation for future sync engine development through comprehensive activity tracking and efficient pagination.

#### Delta Tracking and Version Control (Epic 7 Implementation)
The application now provides advanced OneDrive synchronization capabilities through delta tracking and file version management:

**Delta Tracking:**
1. **`GetDelta()`**: Uses GET to `/drive/root/delta` endpoint for efficient change detection
   - **Token Support**: Accepts optional delta token to track changes since last sync
   - **Response Format**: Returns `DeltaResponse` with `@odata.deltaLink` and `@odata.nextLink` for pagination
   - **Use Cases**: Enables efficient synchronization by returning only changed items
   - **CLI Interface**: `onedrive-client delta [delta-token]` command for manual change tracking

**Drive-Specific Operations:**
1. **`GetDriveByID()`**: Uses GET to `/drives/{drive-id}` endpoint for specific drive access
   - **Metadata Retrieval**: Returns complete drive information including quota, owner, and type
   - **Multi-Drive Support**: Enables access to shared and organization drives beyond default personal drive
   - **CLI Interface**: `onedrive-client drives get <drive-id>` command for drive inspection

**File Version Management:**
1. **`GetFileVersions()`**: Uses GET to `/drive/items/{item-id}/versions` endpoint for version history
   - **Version Listing**: Returns complete version history with timestamps, sizes, and authors
   - **Path Resolution**: Automatically converts file paths to item IDs using existing path resolution
   - **CLI Interface**: `onedrive-client files versions <remote-path>` command for version inspection

**Implementation Details:**
- **New Models**: `DeltaResponse`, `DriveItemVersion`, `DriveItemVersionList` added to support API responses
- **URL Construction**: Follows existing `customRootURL` + endpoint pattern with proper parameter encoding
- **Error Handling**: Uses standard `apiCall()` function for consistent error categorization
- **Display Functions**: `DisplayDelta()`, `DisplayDrive()`, `DisplayFileVersions()` provide formatted output
- **Test Coverage**: Comprehensive unit tests for all new functionality with mock implementations

**Synchronization Foundation:** Delta tracking provides the technical foundation for future sync engine implementation, enabling efficient detection of remote changes without full directory traversals.

#### E2E Testing Infrastructure
The application includes a comprehensive end-to-end testing framework that validates functionality against real OneDrive accounts, ensuring API integration reliability and catching regressions that unit tests might miss.

##### Authentication Strategy
The E2E tests use the same device code flow authentication as the CLI application, ensuring consistency and avoiding the complexity of service principal setup.
1.  **Setup**: Tests expect a `config.json` file in the project root (copied from `~/.config/onedrive-client/config.json` after CLI login).
2.  **Token Management**: Tests automatically handle token refresh during execution, ensuring long-running test suites don't fail due to expired tokens.
3.  **Consistency**: By using the same authentication flow as the CLI, tests validate the actual user experience.

##### Test Isolation and Safety
The framework includes multiple layers of protection to prevent data corruption or conflicts during testing.
1.  **Unique Test Directories**: All tests run within timestamped directories (`/E2E-Tests/test-{timestamp}`) to prevent conflicts between test runs.
2.  **Cleanup Procedures**: Tests include proper cleanup of both local temporary files and remote test data.
3.  **Read-Only Operations**: Where possible, tests favor read-only operations to minimize risk to user data.
4.  **Build Tags**: E2E tests use the `e2e` build tag to prevent accidental execution during regular testing.

##### Test Coverage and Bug Detection
The E2E framework provides comprehensive coverage of the OneDrive SDK functionality:
1.  **File Operations**: Upload (small files, large file sessions), download verification, metadata retrieval
2.  **Directory Operations**: Creation, listing, navigation
3.  **Drive Operations**: List drives, quota checking
4.  **URL Construction**: Verification of proper Microsoft Graph API endpoint formatting
5.  **Error Handling**: Network failures, authentication issues, API errors

##### Critical Bug Discovery
The E2E testing implementation discovered and helped fix several critical bugs:
1.  **URL Construction**: Fixed `BuildPathURL()` function that was generating malformed URLs with double colons (`::`)
2.  **API Endpoint Formatting**: Corrected Microsoft Graph API endpoints to match specification
3.  **Authentication Flows**: Identified download authentication issues requiring further investigation

**Note on Current E2E Coverage:** While the framework is robust, comprehensive E2E tests for large, chunked file uploads and downloads have not been fully implemented due to tooling issues. The existing tests cover session creation and basic transfers, but full validation of resilient transfers remains a work in progress.

**Current E2E Test Coverage:** The E2E testing framework is now fully operational with comprehensive coverage and 100% test pass rate:
- **Authentication**: `GetMe` function validation and user information retrieval
- **File Operations**: Directory creation, small file uploads, large file chunked uploads with session management, metadata retrieval, file downloads with proper redirect handling
- **Drive Operations**: Drive listing, default drive information, quota checking  
- **Error Handling**: Non-existent file handling, invalid operation responses, proper error format recognition
- **URL Construction**: Microsoft Graph API endpoint validation and formatting
- **All Known Issues Resolved**: Successfully fixed directory listing URL construction, download redirect handling, error message format recognition, and test isolation issues

The framework successfully validates core OneDrive SDK integration and provides a solid foundation for regression testing with 100% reliability.

##### Future Enhancement: True CLI End-to-End Testing
While the current E2E framework provides excellent SDK-level integration testing, there is an identified need for true CLI end-to-end testing that validates the complete user experience:

**Current E2E Tests (SDK Level):**
```go
// Tests SDK functionality directly
item, err := helper.App.SDK.UploadFile(localFile, remotePath)
```

**Future CLI E2E Tests (True End-to-End):**
```bash
# Tests actual CLI commands
./onedrive-client files upload test.txt /remote/path/test.txt
./onedrive-client files list /remote/path/
./onedrive-client files download /remote/path/test.txt ./downloaded.txt
```

**Planned CLI E2E Testing Components:**
1. **CLI Test Harness**: Execute actual CLI binary using `exec.Command()` 
2. **Output Validation**: Parse and validate CLI output formatting, progress bars, table display
3. **Command Interface Testing**: Validate argument parsing, flag handling, help text
4. **Workflow Testing**: Complete user scenarios from authentication through file operations
5. **Error Handling**: CLI-specific error messages, exit codes, recovery scenarios

**Implementation Strategy:**
- Maintain both SDK and CLI test suites for comprehensive coverage
- Reuse authentication infrastructure from current E2E framework
- Focus on user experience validation that SDK tests cannot cover
- Integrate with existing test isolation and cleanup procedures

#### Token Refresh Handling
The application ensures that users are not unnecessarily logged out due to expired access tokens. It uses the standard features of the `golang.org/x/oauth2` library to provide seamless, automatic token refreshes.

1.  **Correct Expiry Calculation**: The root cause of previous authentication failures was that the token's `Expiry` timestamp was not being calculated from the `expires_in` value provided by the server. This has been fixed. By ensuring the token has the correct expiry, we allow the `oauth2` library's default mechanisms to work as intended.
2.  **Client Initialization**: When the application starts, `internal/app` creates a standard `oauth2.TokenSource`. Because the token now has a correct expiry date, this source will automatically detect when it needs to be refreshed.
3.  **Persistence Wrapper**: This standard `TokenSource` is wrapped in a custom `persistingTokenSource`. This wrapper's only job is to monitor the token. When it detects that the `oauth2` library has refreshed the token (by seeing a new access token), it triggers a callback to save the new, valid token back to `config.json`.
4.  **Transparent Refresh**: With a correct expiry and a persisting wrapper, the `http.Client` provided by the `oauth2` library handles everything transparently. Before a request is made, if the token is expired, the `TokenSource` refreshes it, the wrapper persists it, and the request proceeds with the new token.

This architecture ensures a robust separation of concerns:
-   `pkg/onedrive` (SDK) knows nothing about token storage or expiry calculation. It just makes API calls.
-   `internal/config` knows how to save and load configuration, but knows nothing about OAuth.
-   `internal/app` is the orchestrator that correctly initializes the token and connects the persistence wrapper, creating the intelligent client that the rest of the application uses.

#### Testing Strategy Updates (2025-06-29)
* The E2E test suite is now compiled and executed by default with `go test ./...`. Tests skip automatically if no authenticated `config.json` is present, preventing CI failures while still offering full coverage in developer environments.
* All SDK error returns are standardized. A final `ErrInternal` sentinel was introduced (2025-06-29) so callers can rely exclusively on `errors.Is` for branching logic.

---

## 3. Future Architecture

This section outlines planned improvements and the long-term vision.

### 3.1. Architectural Refinements

#### A. Isolate the SDK into its own Repository (High Priority)
*   **Goal:** Allow other projects to use the OneDrive Go SDK without depending on the `onedrive-client` CLI.
*   **Why:** True modularity. The SDK will have its own versioning, tests, and release cycle, improving maintainability.
*   **How-To:**
    1.  Create a new, public Git repository (e.g., `github.com/your-org/go-onedrive`).
    2.  Move the entire `pkg/onedrive` directory into the root of the new repository.
    3.  In the `onedrive-client` project, delete the `pkg/` directory.
    4.  Run `go get github.com/your-org/go-onedrive` to add it as a dependency.
    5.  Update all import paths. The `SDK` interface in `internal/app/sdk.go` will remain, wrapping the new external module.

#### B. Implement Resumable Downloads (Complete)
*   **Goal:** Support downloading large files without consuming excessive memory and allow downloads to be resumed.
*   **Status:** Complete. The `files download` command now supports chunked, resumable downloads using HTTP Range requests and session files for state management.
*   **Implementation:** 
    1.  The SDK provides `DownloadFileChunk()` function that accepts byte ranges and returns an `io.ReadCloser`.
    2.  The `files download` command manages session files to track download progress and resume interrupted downloads.
    3.  Downloads use the same session management pattern as uploads for consistency.

### 3.2. The Sync Engine (v1.0+ Vision)

To evolve into a full sync client, the application must run as a persistent, background process (a daemon).

*   **High-Level Plan:** The application will shift to a stateful client-server model. A background daemon will handle all synchronization logic, and the `onedrive-client` CLI will become a controller for that daemon (`onedrive-client sync start`, `onedrive-client sync status`).
*   **New Components:**
    *   `internal/sync/`: A new package containing the core sync logic.
        *   `engine.go`: Orchestrates the main sync loop, using the SDK's `delta` functionality to check for remote changes.
        *   `state.go`: Manages a local database (e.g., SQLite) to store the `deltaToken` and track the state of every synced file.
        *   `watcher.go`: Uses a library like `fsnotify` to watch the local filesystem for changes.
        *   `resolver.go`: A crucial component for handling sync conflicts.
    *   `cmd/sync.go`: A new command to `start`, `stop`, and check the `status` of the sync daemon.

## CLI Layer (`cmd/`)

The CLI layer has been extensively refactored for maintainability and separation of concerns:

### Core Commands
- `root.go` - Root command definition and global flag setup
- `auth.go` - Authentication commands (login, logout, status)  
- `drives.go` - Drive management commands (list, quota, get, activities, root, search, delta, special, recent, shared)
- ~~`shared.go`~~ - **REMOVED**: Shared items functionality moved to drives command

### Drive Commands (`cmd/drives.go`)
All drive-level operations correctly mapped to Microsoft Graph API endpoints:
- `drives list` → `GetDrives()` → `GET /me/drives`
- `drives quota` → `GetDefaultDrive()` → `GET /me/drive` 
- `drives get` → `GetDriveByID()` → `GET /drives/{drive-id}`
- `drives activities` → `GetDriveActivities()` → `GET /me/drive/activities`
- `drives root` → `GetRootDriveItems()` → `GET /me/drive/root/children`
- `drives search` → `SearchDriveItemsWithPaging()` → `GET /me/drive/root/search`
- `drives delta` → `GetDelta()` → `GET /me/drive/root/delta`
- `drives special` → `GetSpecialFolder()` → `GET /me/drive/special/{name}`
- `drives recent` → `GetRecentItems()` → `GET /me/drive/recent` *(corrected placement)*
- `drives shared` → `GetSharedWithMe()` → `GET /me/drive/sharedWithMe` *(corrected placement)*

### Items Commands (`cmd/items/`)
The file and folder management commands have been decomposed into focused modules:

- **`items_root.go`** - Main items command registration and flag setup
  - Centralizes command hierarchy and flag configuration
  - Imports and registers all subcommands
  - Handles pagination flag setup for applicable commands

- **`items_helpers.go`** - Shared utility functions
  - `joinRemotePath()` for URL-safe path joining
  - Common helper functions used across multiple commands

- **`items_meta.go`** - Metadata and query operations (~200 LOC)
  - `list` - Directory listing with pagination support
  - `stat` - File/folder metadata retrieval
  - `search` - Content search with folder scoping and pagination
  - `recent` - Recently accessed items
  - `special` - Special folder access (Documents, Photos, etc.)
  - `versions` - File version history
  - `activities` - Item activity tracking with pagination
  - `thumbnails` - Thumbnail generation and retrieval
  - `preview` - Document preview URL generation

- **`items_upload.go`** - Upload operations (~200 LOC)
  - `mkdir` - Folder creation
  - `upload` - Resumable file upload with session management
  - `upload-simple` - Non-resumable upload for small files
  - `cancel-upload` - Upload session cancellation
  - `get-upload-status` - Upload progress monitoring
  - Integrated session management for upload resumption
  - Signal handling for graceful interruption

- **`items_download.go`** - Download operations (~50 LOC)
  - `download` - File download with format conversion support
  - `list-root-deprecated` - Deprecated root listing method

- **`items_manage.go`** - File manipulation operations (~200 LOC)
  - `rm` - File/folder deletion (moves to recycle bin)
  - `copy` - Asynchronous copy operations with monitoring
  - `copy-status` - Copy operation status checking
  - `mv` - File/folder moving
  - `rename` - Item renaming
  - Comprehensive error handling and status reporting

- **`items_permissions.go`** - Sharing and permissions management (~200 LOC)
  - `share` - Sharing link creation with type/scope validation
  - `invite` - User invitation with role-based permissions
  - `permissions list` - Permission enumeration
  - `permissions get` - Individual permission details
  - `permissions update` - Permission modification
  - `permissions delete` - Permission removal
  - Full CRUD operations for sharing and permissions

### Session Management (`internal/session/`)

Completely refactored to use the Manager pattern:

#### Manager Pattern
- **`Manager`** struct with configurable directory support
- `NewManager()` - Creates manager with default config directory
- `NewManagerWithConfigDir(dir)` - Creates manager with custom directory
- Thread-safe operations with file locking
- Eliminates global state and improves testability

#### Session Operations
- `Save(state)` - Persist upload session state
- `Load(localPath, remotePath)` - Retrieve session state with expiration checking
- `Delete(localPath, remotePath)` - Clean up completed/failed sessions
- `GetSessionFilePath()` - Deterministic session file paths

#### Authentication Session Management
- Consistent Manager pattern for auth state persistence
- `SaveAuthState(state)` / `LoadAuthState()` / `DeleteAuthState()`
- Proper file locking to prevent concurrent auth attempts

### UI Layer (`internal/ui/`)

Enhanced with centralized components:

#### Pagination Support (`pagingflags.go`)
- **`AddPagingFlags(cmd)`** - Standardized pagination flag registration
- **`ParsePagingFlags(cmd)`** - Consistent flag parsing with error handling
- **`HandleNextPageInfo(nextLink, fetchAll)`** - Next page information display
- Eliminates code duplication across paginated commands
- Supports `--top`, `--all`, and `--next` flags uniformly

#### Display Functions (`display.go`)
- Consistent formatting for all data types
- Structured output for API responses
- Progress indicators for long-running operations