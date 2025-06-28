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

## 3. Release Plan

### 3.1. Release v0.1: Core Operations (Complete)

This initial release focuses on providing the most essential file management commands. It establishes the core authentication flow and command structure.

**Features:**
*   User authentication via OAuth 2.0 Device Code Flow. `[x]`
*   Ability to list files and folders. `[x]`
*   Ability to list all available drives and check quota. `[x]`
*   Ability to view metadata for items (`stat`). `[x]`
*   Ability to download files (`download`). `[x]`
*   Ability to upload files, including large files via resumable sessions (`upload`). `[x]`
*   Ability to create folders (`mkdir`). `[x]`

### 3.2. Release v0.2: Advanced Management and Sharing (In Progress)

This release will build on the core by adding destructive operations, resumable downloads, and sharing capabilities.

**Features:**
*   Ability to download large files resiliently. `[x]`
*   Ability to delete files and folders (`rm`). `[x]`
*   Ability to move files and folders (`mv`). `[x]`
*   Ability to rename items (`rename`). `[x]`
*   Ability to copy files and folders (`copy`) with progress monitoring. `[x]`
*   Ability to search for items within the drive (`search`). `[ ]`
*   Ability to view items shared with the user. `[ ]`

### 3.3. Future Releases (Post-v0.2)

*   **v1.0: The Sync Engine:** Introduction of a persistent background process to automatically keep a local directory in sync with a remote OneDrive folder.
*   **Post-v1.0:** Performance enhancements, support for shared drive management, and other advanced features based on user feedback.

## 4. Epics, User Stories, and CLI Commands

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

### `[/]` Epic 2: File and Folder Management

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
*   **[ ] User Story 2.12:** I want to search for files and folders across my entire drive by a query string.
    *   **Command:** `onedrive-client files search "<query>"`

---

### `[ ]` Epic 3: Sharing Management

As a user, I want to see and manage content that has been shared with me.

*   **[ ] User Story 3.1:** I want to list all the files and folders that have been shared with me.
    *   **Command:** `onedrive-client shared list`

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
