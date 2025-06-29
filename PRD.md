# Product Requirements Document: onedrive-client (v2)

## 1. Overview and Project Aims

### 1.1. Project Aim

The primary goal of the `onedrive-client` project is to create a powerful, fast, and intuitive command-line interface (CLI) for interacting with Microsoft OneDrive. The long-term vision is to build a full-fledged, bi-directional sync client. The immediate focus is on providing a robust set of command-line primitives for manual file management, including resilient large file transfers.

This tool serves users who prefer to work in a terminal environment, enabling them to manage their OneDrive files and folders without needing a graphical user interface.

### 1.2. Target Audience

*   Developers, System Administrators, and Power Users.
*   Users who need to script interactions with their OneDrive storage.
*   Users who want a lightweight, non-GUI alternative for managing their cloud files, especially for large file transfers.

## 2. Technical Implementation Notes

The client is written in Go and uses the [Cobra](https://cobra.dev/) library for its command-line interface.

All interactions with the Microsoft Graph API are handled by an integrated **SDK package** located at `pkg/onedrive`. This package is responsible for authentication, request signing, and API call logic. The main application (`cmd/` and `internal/`) focuses on user interaction, command parsing, and orchestrating calls to this SDK.

**Future Goal:** For better reusability, the SDK in `pkg/onedrive` will be extracted into its own version-controlled Git repository. This will allow other applications to use the Go SDK for OneDrive without depending on the `onedrive-client` CLI application itself.

### Technical Implementation Details
#### 2025-06-29 OAuth Hardening
* Expiry parsing and automatic refresh implemented (see CHANGELOG).
* Atomic config persistence and refresh‐token preservation.

## 3. Epics, User Stories, and CLI Commands

**Status Legend:**
*   `[ ]` - **Queued**: The task has not been started.
*   `[/]` - **Active**: The task is in progress.
*   `[x]` - **Done**: The task is complete.

---

### `[x]` Epic 1: Core Account and Drive Information

As a user, I want to get basic information about my OneDrive account and storage.

*   **[x] User Story 1.1:** I want to list all OneDrive drives (personal, business, etc.) available to my account so I can see what I can interact with.
    *   **Command:** `onedrive-client drives list`
*   **[x] User Story 1.2:** I want to check my storage quota (total, used, remaining space) so I can manage my storage consumption.
    *   **Command:** `onedrive-client drives quota`

---

### `[x]` Epic 2: File and Folder Management

As a user, I want to perform standard file and folder operations from the command line.

*   **[x] User Story 2.1:** I want to list the files and folders within a specific directory in my OneDrive.
    *   **Command:** `onedrive-client files list [remote-path]`
*   **[x] User Story 2.2:** I want to view detailed metadata (like size, creation date) for a specific file or folder.
    *   **Command:** `onedrive-client files stat <remote-path>`
*   **[x] User Story 2.3:** I want to download a file from my OneDrive to my local machine.
    *   **Command:** `onedrive-client files download <remote-path> [local-path]`
*   **[x] User Story 2.4:** I want to upload a file from my local machine to a specific folder in my OneDrive.
    *   **Command:** `onedrive-client files upload <local-file> [remote-path]`
*   **[x] User Story 2.5:** I want to create a new, empty folder at a specified path in my OneDrive.
    *   **Command:** `onedrive-client files mkdir <remote-path>`
*   **[x] User Story 2.6 (Large Files):** I want to upload files larger than 4MB, so that I can transfer large assets to my cloud storage. The upload must be resumable.
    *   **Status:** Complete. The `upload` command automatically uses a resumable session.
*   **[x] User Story 2.7 (Large Files):** I want to download files larger than 4MB without consuming excessive memory, and the download should be resumable.
    *   **Status:** Complete. The `download` command uses resumable chunked downloads.
*   **[x] User Story 2.7a (Upload Session Management):** I want to cancel an active resumable upload session if I no longer need it.
    *   **Command:** `onedrive-client files cancel-upload <upload-url>`
*   **[x] User Story 2.7b (Upload Session Management):** I want to check the status of a resumable upload session to see its progress.
    *   **Command:** `onedrive-client files get-upload-status <upload-url>`
*   **[x] User Story 2.7c (Simple Upload):** I want to upload small files using a simple, non-resumable method for better performance.
    *   **Command:** `onedrive-client files upload-simple <local-file> <remote-path>`
*   **[x] User Story 2.7d (Legacy Support):** I want to access the deprecated root listing method for compatibility.
    *   **Command:** `onedrive-client files list-root-deprecated`
*   **[x] User Story 2.8:** I want to delete a file or folder from my OneDrive.
    *   **Command:** `onedrive-client files rm <remote-path>`
*   **[x] User Story 2.9:** I want to move a file or folder from one location to another.
    *   **Command:** `onedrive-client files mv <source-path> <destination-path>`
*   **[x] User Story 2.10:** I want to rename a file or folder.
    *   **Command:** `onedrive-client files rename <remote-path> <new-name>`
*   **[x] User Story 2.11:** I want to copy a file or folder to another location.
    *   **Command:** `onedrive-client files copy <source-path> <destination-path> [new-name]`
    *   **Advanced:** `onedrive-client files copy --wait <source-path> <destination-path> [new-name]` for blocking operation
    *   **Monitoring:** `onedrive-client files copy-status <monitor-url>` to check progress
*   **[x] User Story 2.12:** I want to search for files and folders across my entire drive by a query string.
    *   **Command:** `onedrive-client files search "<query>"`
    *   **Implementation:** `SearchDriveItems()` (see Epic 7)

---

### `[x]` Epic 3: Sharing Management

As a user, I want to see and manage content that has been shared with me, and create sharing links for my own content.

*   **[x] User Story 3.1:** I want to list all the files and folders that have been shared with me.
    *   **Command:** `onedrive-client shared list`
*   **[x] User Story 3.2:** I want to create sharing links for my files and folders to share them with others.
    *   **Command:** `onedrive-client files share <remote-path> <link-type> <scope>`
    *   **Link Types:** "view" (read-only), "edit" (read-write), "embed" (embeddable for web pages)
    *   **Scopes:** "anonymous" (anyone with link), "organization" (organization members only)
    *   **Features:** Input validation, comprehensive link details display, proper error handling

---

### `[x]` Epic 4: Authentication Management

As a user, I want to explicitly manage my authentication state.

*   **[x] User Story 4.1:** I want to log in using a non-interactive flow suitable for CLI applications.
    *   **Command:** `onedrive-client auth login`
*   **[x] User Story 4.2:** I want to check if I am currently logged in and see my user information.
    *   **Command:** `onedrive-client auth status`
*   **[x] User Story 4.3:** I want to log out, clearing my local credentials.
    *   **Command:** `onedrive-client auth logout`

---

### `[x]` Epic 5: E2E Testing Infrastructure

As a developer, I want comprehensive end-to-end testing against real OneDrive accounts to ensure API integration reliability and catch regressions.

*   **[x] User Story 5.1:** I want automated tests that authenticate using the same device code flow as the CLI to ensure authentication consistency.
    *   **Implementation:** Tests use existing `config.json` from CLI login process
*   **[x] User Story 5.2:** I want test isolation to prevent data conflicts and protect user data during testing.
    *   **Implementation:** Tests run in unique timestamped directories (`/E2E-Tests/test-{timestamp}`)
*   **[x] User Story 5.3:** I want comprehensive test coverage of file operations against real OneDrive API.
    *   **Test Coverage:** File uploads, directory creation, metadata retrieval, drive operations, URL construction verification
*   **[x] User Story 5.4:** I want automatic token refresh during long-running test suites.
    *   **Implementation:** Tests handle token expiration and refresh automatically
*   **[x] User Story 5.5:** I want proper test cleanup and safety measures to protect user data.
    *   **Implementation:** Tests use isolated directories and include cleanup procedures
*   **[x] User Story 5.6:** I want API bug detection through real endpoint testing.
    *   **Achievement:** Discovered and fixed critical URL construction bugs in Microsoft Graph API calls

**Setup Process:**
1. Authenticate with CLI: `./onedrive-client auth login`
2. Copy config to project: `cp ~/.config/onedrive-client/config.json ./config.json`
3. Run E2E tests: `go test -tags=e2e -v ./e2e/...`

**Current Status:** Framework operational with 100% test pass rate. All previously known limitations have been successfully resolved, achieving comprehensive coverage of all available SDK functionality with robust error handling and test isolation.

---

### `[ ]` Epic 6: True CLI End-to-End Testing (Future Enhancement)

As a developer, I want comprehensive end-to-end testing of the actual CLI commands to ensure the complete user experience works correctly, including command parsing, output formatting, and error handling.

*   **[ ] User Story 6.1:** I want automated tests that execute the actual CLI binary to test the complete command execution path.
    *   **Implementation:** Tests using `exec.Command()` to run `./onedrive-client` commands
*   **[ ] User Story 6.2:** I want CLI output format validation to ensure consistent and correct user interface behavior.
    *   **Test Coverage:** Command help text, progress bars, table formatting, success/error messages
*   **[ ] User Story 6.3:** I want CLI argument and flag parsing validation to catch command-line interface regressions.
    *   **Test Coverage:** Flag validation, argument parsing, error handling for invalid inputs
*   **[ ] User Story 6.4:** I want CLI workflow testing to validate complete user scenarios from start to finish.
    *   **Test Coverage:** Authentication flows, file operations, command chaining, error recovery
*   **[ ] User Story 6.5:** I want CLI-specific error handling validation to ensure proper error messages and exit codes.
    *   **Implementation:** Tests for network failures, authentication errors, file system errors

**Distinction from Current E2E Tests:**
- **Current E2E Tests:** Test SDK functionality directly (`helper.App.SDK.UploadFile()`)
- **Future CLI E2E Tests:** Test actual CLI commands (`./onedrive-client files upload test.txt /remote/`)

**Implementation Plan:**
1. Create CLI test harness using `exec.Command()` 
2. Implement output parsing and validation utilities
3. Add CLI-specific test scenarios covering all commands
4. Integrate with existing E2E test infrastructure for authentication
5. Maintain both SDK and CLI test suites for comprehensive coverage

---

### `[x]` Epic 7: Comprehensive Microsoft Graph OneDrive API Coverage

As a developer, I want comprehensive coverage of Microsoft Graph OneDrive APIs to provide users with full OneDrive functionality through the CLI.

#### Drive-Level Operations

*   **[x] User Story 7.1:** I want to retrieve a list of all drives available to the user.
    *   **API:** `GET /drives`
    *   **Implementation:** `GetDrives()`
    *   **Command:** `onedrive-client drives list`

*   **[x] User Story 7.2:** I want to get metadata for the user's default drive including quota information.
    *   **API:** `GET /drive` (equivalent to `GET /drive/root` for metadata)
    *   **Implementation:** `GetDefaultDrive()`
    *   **Command:** `onedrive-client drives quota`

*   **[x] User Story 7.3:** I want to retrieve metadata for a specific drive by its ID.
    *   **API:** `GET /drives/{drive-id}`
    *   **Implementation:** `GetDriveByID()`
    *   **Command:** `onedrive-client drives get <drive-id>`

*   **[x] User Story 7.4:** I want to list children of the drive root folder.
    *   **API:** `GET /drive/root/children`
    *   **Implementation:** `GetRootDriveItems()`, `GetDriveItemChildrenByPath("/")`
    *   **Command:** `onedrive-client files list` or `onedrive-client files list /`

*   **[x] User Story 7.5:** I want to view recent activity and changes across the entire drive.
    *   **API:** `GET /drive/activities`
    *   **Implementation:** `GetDriveActivities()`
    *   **Command:** `onedrive-client drives activities`

*   **[x] User Story 7.6:** I want to track changes and get delta information for all items in the drive.
    *   **API:** `GET /drive/root/delta`
    *   **Implementation:** `GetDelta()`
    *   **Command:** `onedrive-client delta [delta-token]`

*   **[x] User Story 7.7:** I want to search for items across the entire drive.
    *   **API:** `GET /drive/root/search(q='query')`
    *   **Implementation:** `SearchDriveItems()`
    *   **Command:** `onedrive-client files search "<query>"`

*   **[x] User Story 7.8:** I want to access well-known special folders.
    *   **API:** `GET /drive/special/{name}`
    *   **Implementation:** `GetSpecialFolder()`
    *   **Command:** `onedrive-client files special <folder-name>`

#### Item Operations

*   **[x] User Story 7.9:** I want to retrieve metadata for a specific item by its path.
    *   **API:** `GET /drive/items/{item-id}` (implemented via path-based addressing)
    *   **Implementation:** `GetDriveItemByPath()`
    *   **Command:** `onedrive-client files stat <remote-path>`

*   **[x] User Story 7.10:** I want to list children of a specific folder.
    *   **API:** `GET /drive/items/{item-id}/children`
    *   **Implementation:** `GetDriveItemChildrenByPath()`
    *   **Command:** `onedrive-client files list <remote-path>`

*   **[x] User Story 7.11:** I want to view activity history for a specific item.
    *   **API:** `GET /drive/items/{item-id}/activities`
    *   **Implementation:** `GetItemActivities()`
    *   **Command:** `onedrive-client files activities <remote-path>`

*   **[x] User Story 7.12:** I want to list all versions of a specific file.
    *   **API:** `GET /drive/items/{item-id}/versions`
    *   **Implementation:** `GetFileVersions()`
    *   **Command:** `onedrive-client files versions <remote-path>`

*   **[x] User Story 7.13:** I want to create new folders and items.
    *   **API:** `POST /drive/items/{item-id}/children`
    *   **Implementation:** `CreateFolder()`
    *   **Command:** `onedrive-client files mkdir <remote-path>`

*   **[x] User Story 7.14:** I want to update item properties like name.
    *   **API:** `PATCH /drive/items/{item-id}`
    *   **Implementation:** `UpdateDriveItem()`, `MoveDriveItem()`
    *   **Commands:** `onedrive-client files rename <remote-path> <new-name>`, `onedrive-client files mv <source> <dest>`

*   **[x] User Story 7.15:** I want to upload file content to items.
    *   **API:** `PUT /drive/items/{item-id}/content`
    *   **Implementation:** `UploadFile()` + upload session APIs
    *   **Commands:** `onedrive-client files upload <local-file> [remote-path]`

*   **[x] User Story 7.16:** I want to download file content from items.
    *   **API:** `GET /drive/items/{item-id}/content`
    *   **Implementation:** `DownloadFile()`, `DownloadFileByItem()`, `DownloadFileChunk()`
    *   **Command:** `onedrive-client files download <remote-path> [local-path]`

*   **[x] User Story 7.17:** I want to download files in specific formats.
    *   **API:** `GET /drive/items/{item-id}/content?format={format}`
    *   **Implementation:** `DownloadFileAsFormat()`
    *   **Command:** `onedrive-client files download <remote-path> [local-path] --format <format>`

*   **[x] User Story 7.18:** I want to delete items.
    *   **API:** `DELETE /drive/items/{item-id}`
    *   **Implementation:** `DeleteDriveItem()`
    *   **Command:** `onedrive-client files rm <remote-path>`

*   **[x] User Story 7.19:** I want to copy items to new locations.
    *   **API:** `POST /drive/items/{item-id}/copy`
    *   **Implementation:** `CopyDriveItem()`, `MonitorCopyOperation()`
    *   **Commands:** `onedrive-client files copy <source> <dest> [new-name]`, `onedrive-client files copy-status <monitor-url>`

*   **[x] User Story 7.20:** I want to search within specific folders.
    *   **API:** `GET /drive/items/{item-id}/search(q='text')`
    *   **Implementation:** `SearchDriveItemsInFolder()`
    *   **Command:** `onedrive-client files search "<query>" --in <remote-path>`

*   **[x] User Story 7.21:** I want to get thumbnail images for items.
    *   **API:** `GET /drive/items/{item-id}/thumbnails`
    *   **Implementation:** `GetThumbnails()`, `GetThumbnailBySize()`
    *   **Command:** `onedrive-client files thumbnails <remote-path>`

*   **[x] User Story 7.22:** I want to preview items without downloading.
    *   **API:** `POST /drive/items/{item-id}/preview`
    *   **Implementation:** `PreviewItem()`
    *   **Command:** `onedrive-client files preview <remote-path> [--page <page>] [--zoom <zoom>]`

#### Sharing and Permissions

*   **[x] User Story 7.23:** I want to create sharing links for items.
    *   **API:** `POST /drive/items/{item-id}/createLink`
    *   **Implementation:** `CreateSharingLink()`
    *   **Command:** `onedrive-client files share <remote-path> <link-type> <scope>`

*   **[x] User Story 7.24:** I want to invite people and add permissions to items.
    *   **API:** `POST /drive/items/{item-id}/invite`
    *   **Implementation:** `InviteUsers()`
    *   **Command:** `onedrive-client files invite <remote-path> <email> [additional-emails...] [--message <msg>] [--roles <roles>]`

*   **[x] User Story 7.25:** I want to list all permissions on an item.
    *   **API:** `GET /drive/items/{item-id}/permissions`
    *   **Implementation:** `ListPermissions()`
    *   **Command:** `onedrive-client files permissions list <remote-path>`

*   **[x] User Story 7.26:** I want to get details of a specific permission.
    *   **API:** `GET /drive/items/{item-id}/permissions/{id}`
    *   **Implementation:** `GetPermission()`
    *   **Command:** `onedrive-client files permissions get <remote-path> <permission-id>`

*   **[x] User Story 7.27:** I want to update existing permissions.
    *   **API:** `PATCH /drive/items/{item-id}/permissions/{id}`
    *   **Implementation:** `UpdatePermission()`
    *   **Command:** `onedrive-client files permissions update <remote-path> <permission-id> [--roles <roles>] [--expiration <date>]`

*   **[x] User Story 7.28:** I want to remove permissions from items.
    *   **API:** `DELETE /drive/items/{item-id}/permissions/{perm-id}`
    *   **Implementation:** `DeletePermission()`
    *   **Command:** `onedrive-client files permissions delete <remote-path> <permission-id>`

#### Additional Features

*   **[x] User Story 7.29:** I want to view items that have been shared with me.
    *   **API:** `GET /drive/sharedWithMe` (non-standard endpoint)
    *   **Implementation:** `GetSharedWithMe()`
    *   **Command:** `onedrive-client shared list`

*   **[x] User Story 7.30:** I want to view recently accessed items.
    *   **API:** `GET /drive/recent` (non-standard endpoint)
    *   **Implementation:** `GetRecentItems()`
    *   **Command:** `onedrive-client files recent`

**Implementation Status Summary:**
- **Implemented:** 30/30 API endpoints (100% COMPLETE)
- **Core file operations:** Complete coverage
- **Drive management:** Enhanced coverage with drive-specific access
- **Activities tracking:** Complete coverage for both drive-level and item-level activities
- **Advanced search:** Complete coverage including folder-scoped search with pagination
- **Download capabilities:** Complete coverage including format conversion
- **Sharing:** Complete coverage with link creation and comprehensive permissions management
- **Advanced features:** Delta tracking, version control, activities, thumbnails, preview, and permissions fully implemented
- **Synchronization:** Delta API enables efficient sync operations
- **Permissions management:** Complete CRUD operations for permissions with invitation system
- **Media features:** Thumbnail retrieval and file preview capabilities

**Epic 7 Achievement:** All Microsoft Graph OneDrive API coverage goals have been successfully achieved. The CLI now provides comprehensive OneDrive functionality including advanced features for thumbnails, previews, and complete permissions management system.

---

### `[ ]` Epic 8: Sync Engine (Future v1.0)

As a user, I want to automatically keep a local directory synchronized with a remote OneDrive folder to eliminate manual file management overhead.

*   **[ ] User Story 8.1:** I want to initialize a sync relationship between a local directory and a OneDrive folder.
    *   **Command:** `onedrive-client sync init <local-path> <remote-path>` (proposed)
*   **[ ] User Story 8.2:** I want a persistent background process that monitors for changes and automatically syncs files.
    *   **Command:** `onedrive-client sync daemon start` (proposed)
*   **[ ] User Story 8.3:** I want to view the status of all sync relationships and their current state.
    *   **Command:** `onedrive-client sync status` (proposed)
*   **[ ] User Story 8.4:** I want to pause and resume sync operations when needed.
    *   **Commands:** `onedrive-client sync pause`, `onedrive-client sync resume` (proposed)
*   **[ ] User Story 8.5:** I want to handle sync conflicts intelligently with user-configurable resolution strategies.
    *   **Features:** Conflict detection, resolution strategies (newest wins, manual resolution, etc.)
*   **[ ] User Story 8.6:** I want to exclude certain files or patterns from synchronization.
    *   **Command:** `onedrive-client sync exclude <pattern>` (proposed)
*   **[ ] User Story 8.7:** I want to monitor sync progress and receive notifications about sync events.
    *   **Command:** `onedrive-client sync logs` (proposed)

**Technical Requirements:**
- Bi-directional synchronization using delta API for efficient change detection
- File system watcher integration for real-time local changes
- Robust conflict resolution with user preferences
- Persistent sync state management
- Cross-platform compatibility (Windows, macOS, Linux)

**Future Enhancements (Post-v1.0):**
- Performance optimizations for large directories
- Shared drive management and team synchronization
- Advanced filtering and selective sync capabilities
- Integration with cloud-native applications
