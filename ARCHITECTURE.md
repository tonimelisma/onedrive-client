# Architecture Document: onedrive-client (v2)

## 1. Purpose and Guiding Principles

This document outlines the technical architecture for the `onedrive-client` application. Its purpose is to provide a clear and consistent guide for developers, ensuring that the project remains maintainable, testable, and extensible as it evolves.

All development should adhere to these core principles:

1.  **Separation of Concerns:** Each component has a single, well-defined responsibility.
2.  **Testability:** Application logic should be testable without live network calls.
3.  **Extensibility:** Adding new commands should be simple and not require major refactoring.

This document is divided into two parts:
*   **Current Architecture:** Describes the application as it is currently built.
*   **Future Architecture:** Outlines the vision for improving the design and adding a background sync engine.

---

## 2. Current Architecture: CLI Primitives

The application functions as a stateless, command-driven utility. The user executes a command, the action is performed, and the application exits.

### 2.1. High-Level Data Flow

The flow for any given command is as follows:

```
[User] -> [main.go] -> [Cobra CLI (cmd/)] -> [App Core (internal/app)] -> [SDK (pkg/onedrive)] -> [MS Graph API]
  ^
  |                                                                                             |
  +----------------------- [UI (internal/ui)] <----------------- [Cobra CLI (cmd/)] <-----------+
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
    ├── app/              // Core application logic (initialization, auth).
    │   └── app.go
    ├── config/           // Configuration loading and saving.
    │   └── config.go
    └── ui/               // User interface formatting and output.
        └── display.go
└── pkg/
    └── onedrive/         // The Go SDK for interacting with the OneDrive API.
        ├── onedrive.go
        └── models.go
```

#### `cmd/` (The Command Layer)
*   **Technology:** [Cobra](https://cobra.dev/).
*   **Responsibility:** Defines the command structure (`drives`, etc.), arguments, and flags. The `Run` function for each command is responsible for:
    1.  Initializing the App Core (`internal/app`).
    2.  Calling the appropriate SDK function via the App Core client.
    3.  Passing the results (or errors) to the UI Layer (`internal/ui`) for display.
*   **Constraint:** This layer contains **no business logic**. It only orchestrates calls.

#### `internal/app/` (The App Core)
*   **Responsibility:** Acts as the central hub for the application. It initializes the configuration and the OneDrive HTTP client (handling the authentication flow if necessary). It provides a fully configured client to the command layer.
*   **Implementation:** The `app.NewApp()` function returns an `App` struct containing the loaded configuration and a ready-to-use `*http.Client`. The `NewApp` function is now also responsible for checking for and completing any pending authentication flows.

#### `internal/config/` (Configuration Management)
*   **Responsibility:** Handles all logic for loading, parsing, and saving the `config.json` file, which stores the final OAuth tokens.

#### `internal/session/` (Session Management)
*   **Responsibility:** Manages temporary state files. For authentication, this includes the `auth_session.json` file, which stores the details of a pending Device Code Flow login. It uses file-locking to prevent race conditions from concurrent CLI invocations.

#### `internal/ui/` (The Presentation Layer)
*   **Responsibility:** Handles all user-facing output. This includes printing tables of files, success messages, and formatted errors. Separating this logic ensures display format changes don't affect other layers.

#### `pkg/onedrive/` (The SDK Layer)
*   **Responsibility:** This package is the **only** component that knows how to communicate with the Microsoft Graph API. It handles creating API requests, parsing responses, and defining the data models (`DriveItem`, etc.).
*   **Authentication**: Implements the OAuth 2.0 Device Code Flow. It has no awareness of the higher-level application's stateful, non-blocking flow; it simply provides the functions to initiate the flow and verify a device code.

### 2.3. How to Add a New Command (Example: `files list`)

An intern should follow these steps:

1.  **SDK Layer (`pkg/onedrive`):** Check if a function already exists to get the data you need (e.g., `GetDrives`). If not, add a new function that calls the required Microsoft Graph API endpoint.
2.  **Command Layer (`cmd/`):** Create a new file, e.g., `cmd/mynewcommand.go`. Define the new command using Cobra.
3.  **Command Layer:** In the `Run` function for the command, call `app.NewApp()` to get the initialized application struct. The `PersistentPreRunE` hook on the root command will automatically handle any pending login flows, so the command's own `Run` function does not need to worry about it. It can assume that if it runs, the user is authenticated.
4.  **UI Layer (`internal/ui`):** Pass the data returned from the SDK to a display function in the `ui` package.

---

## 3. Future Architecture

This section outlines planned improvements to the architecture and the long-term vision for a sync client.

### 3.1. Architectural Refinements for Testability and Reusability

To make the application more robust and the SDK truly reusable, two key changes should be implemented.

#### A. Isolate the SDK into its own Repository
*   **Goal:** Allow other projects to use the OneDrive Go SDK without depending on the `onedrive-client` CLI.
*   **Why:** True modularity. The SDK gets its own versioning, tests, and release cycle.
*   **How-To:**
    1.  Create a new, public Git repository (e.g., `github.com/your-org/onedrive-sdk-go`).
    2.  Move the entire `pkg/onedrive` directory into the root of this new repository.
    3.  In the `onedrive-client` project, delete the `pkg/` directory.
    4.  Run `go get github.com/your-org/onedrive-sdk-go` to add the new, external SDK as a dependency.
    5.  Update all import paths from `.../pkg/onedrive` to `github.com/your-org/onedrive-sdk-go`.

#### B. Abstract the SDK for Testability (Implemented)
*   **Goal:** Enable true unit testing of CLI commands without making live network calls.
*   **Why:** Previously, testing a command like `drives` required a live, authenticated `http.Client`. By using an interface, we can provide a "mock" SDK during tests that returns predefined data.
*   **How-To:**
    1.  **Define an Interface:** In `internal/app`, a new `sdk.go` file was created with an `SDK` interface:
        ```go
        package app
        
        type SDK interface {
            GetDriveItemChildrenByPath(client *http.Client, path string) (onedrive.DriveItemList, error)
            // ... other SDK methods
        }
        ```
    2.  **Create a Concrete Wrapper:** A struct `LiveSDK` was created that implements this interface and wraps the real SDK calls.
        ```go
        type LiveSDK struct{}

        func (s *LiveSDK) GetDriveItemChildrenByPath(client *http.Client) (onedrive.DriveItemList, error) {
            return onedrive.GetDriveItemChildrenByPath(client) // The real call
        }
        ```
    3.  **Update App Core:** The `App` struct in `internal/app/app.go` now holds an instance of the `SDK` interface.
    4.  **Update Commands:** Commands now call `a.SDK.GetDriveItemChildrenByPath()` instead of the package-level function. In tests, an `App` can be created with a mock SDK implementation.

### 3.2. The Sync Engine (v1.0+ Vision)

To evolve into a full sync client, the application must run as a persistent, background process (a daemon).

*   **High-Level Plan:** The application will shift to a stateful client-server model. A background daemon will handle all synchronization logic, and the `onedrive-client` CLI will become a controller for that daemon (`onedrive-client sync start`, `onedrive-client sync status`).
*   **New Components:**
    *   `internal/sync/`: A new package containing the core sync logic.
        *   `engine.go`: Orchestrates the main sync loop, using the SDK's `delta` functionality to check for remote changes.
        *   `state.go`: Manages a local database (e.g., SQLite) to store the `deltaToken` and track the state of every synced file.
        *   `watcher.go`: Uses a library like `fsnotify` to watch the local filesystem for changes, triggering the sync engine.
        *   `resolver.go`: A crucial component for handling sync conflicts (e.g., a file modified in both places).
    *   `cmd/sync.go`: A new command to `start`, `stop`, and check the `status` of the sync daemon.

### 3.3. Authentication Flow (Device Code)

The application uses a non-blocking, stateful implementation of the OAuth 2.0 Device Code Flow. This provides a seamless and user-friendly experience for a CLI.

1.  **`auth login`**: The user runs this command to start the process.
    - The CLI calls the Microsoft Identity endpoint to get a `user_code` and a `device_code`.
    - It saves these details, along with the verification URL, into a temporary `auth_session.json` file.
    - It displays the code and URL to the user and immediately exits.
2.  **User Action**: The user opens the verification URL on any browser-enabled device and enters the `user_code` to approve the sign-in.
3.  **Any Subsequent Command**: When the user runs *any* command (e.g., `files list` or `auth status`):
    - The application starts and, in `app.NewApp`, detects the `auth_session.json` file.
    - It automatically makes a request to the token endpoint with the stored `device_code`.
    - If the user has approved the sign-in, the CLI receives the final access and refresh tokens, saves them to the main `config.json`, deletes the `auth_session.json` file, and proceeds with executing the command.
    - If the user has not yet approved, the API returns a pending status, and the CLI informs the user to complete the step in their browser. All commands (except for `auth` itself) are blocked from running by a `PersistentPreRunE` hook on the root command.

This non-blocking flow allows the user to initiate a login and continue working in their terminal without being locked into a polling loop.

The `onedrive` package is a self-contained SDK for interacting with the Microsoft Graph API. It has no dependencies on the rest of the application. It handles API calls, error wrapping, and OAuth2 logic. All models specific to the OneDrive API are defined here.

### Session State Management

For features that require state to be maintained across multiple command invocations (e.g., resumable file uploads), a session management system is used.

- **Location**: Session files are stored in `~/.config/onedrive-client/sessions/`.
- **Mechanism**: When a resumable operation (like a file upload or a pending login) begins, a session file is created. This file contains the necessary information to resume the operation. The auth session is stored in `auth_session.json`, while file uploads are named using a SHA256 hash of the file paths.
- **Lifecycle**: The session file is created when the operation starts and deleted upon successful completion. If the operation is interrupted, the file remains, and the application will detect and use it to resume the next time the same command is run. File locking is used to prevent race conditions.

## UI

The `ui` package is responsible for all console output. It provides functions for displaying tables, progress bars, and formatted success/error messages. It also contains the logger implementation that can be passed to the `onedrive` SDK for debug output.
