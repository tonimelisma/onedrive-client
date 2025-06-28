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