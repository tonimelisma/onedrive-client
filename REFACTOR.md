# Refactor Roadmap (v2 → v2.1)

This document describes **all remaining structural improvements** identified during the recent code-review sessions.  Items are grouped by theme, each with:

* **Goal / Outcome** – what success looks like.
* **Work Breakdown** – concrete steps, file locations, APIs to touch.
* **Risks / Soft-Spots** – things that can break, unknowns that must be investigated.

---

## 1. Decompose `cmd/files.go` (a.k.a. `items`)

### Goal / Outcome
* Reduce a 1 200-line monolith into focused source files ≤200 LOC each.
* Improve readability, reviewability and future change isolation.
* Zero functional change.

### Work Breakdown
1. Create `cmd/items/` folder (still in `package cmd`).
2. Split by concern:
   * `items_upload.go` – upload/resumable-upload/session logic.
   * `items_download.go` – download, download-as-format, resumable downloads (when added).
   * `items_manage.go` – mkdir, rm, mv, rename, copy, copy-status.
   * `items_meta.go` – list, stat, recent, special, search, versions, thumbnails, preview.
   * `items_permissions.go` – share, invite, permissions *subtree*.
3. Move **only** command registration + helper logic into each file; leave shared helpers (`joinRemotePath`, progress bar etc.) in a new `items_helpers.go`.
4. Update `init()` registration – keep it in **one** file (recommend `items_root.go`) to avoid duplicate registration.
5. Adjust tests: either
   * leave in `cmd/files_test.go`, or
   * split tests mirroring new layout.  Both okay.

### Risks / Soft-Spots
* Accidental duplicate `init()` registering the same command.
* Circular imports if helper code slips outside `cmd`.
* Build tags in tests referencing files that moved.

---

## 2. Remove Legacy Session Helpers

### Goal / Outcome
* Eliminate deprecated free-functions (`session.Save/Load/Delete`, `GetSessionFilePath`, `GetConfigDir`, etc.).
* Rely exclusively on `session.Manager` (thread-safe, locked, 0600 perms).

### Work Breakdown
1. Replace helper usage in **tests** (`cmd/files_test.go`, `cmd/auth_test.go`) with `session.ManagerWithConfigDir`.
2. Delete legacy symbols from `internal/session/session.go`.
3. Run `go vet` & `go test ./...` to confirm no lingering references.

### Risks
* Tests that relied on mutable package-level functions must inject manager instead – may require shared test util.

---

## 3. Context Propagation

### Goal / Outcome
* Every SDK call is cancellable.
* Ctrl-C or context timeout aborts long uploads/downloads gracefully.

### Work Breakdown
1. Add `context.Context` first parameter to **all** SDK interface methods (breaking change – bump major).
2. Thread `cmd.Context()` (Cobra) into app layer, then into SDK.
3. Update `pkg/onedrive.Client` methods to accept context and call `c.httpClient.Do(req.WithContext(ctx))`.
4. Provide default `context.Background()` wrappers (`UploadFileCtx` etc.) for backward compatibility where needed.

### Risks
* Large surface breaking change – must migrate tests, mocks.
* Long-running loops (upload chunks) need periodic `<-ctx.Done()` checks.

---

## 4. Central Pagination Flags Helper

### Goal / Outcome
* DRY logic for `--top`, `--all`, `--next` flags across list / search / activities commands.

### Work Breakdown
1. `internal/ui/pagingflags.go` with `AddPagingFlags(cmd *cobra.Command)` and `ParsePagingFlags(cmd)` returning `onedrive.Paging`.
2. Remove flag duplication from commands.
3. Unit-test helper for mutually exclusive flag validation.

### Risks
* Accidental behaviour change if default values differ.

---

## 5. Split `pkg/onedrive/client.go` - ✅ COMPLETED

### Goal / Outcome
* ✅ **ACHIEVED**: Each responsibility in its own file for maintainability; shortened giant file from 1018 LOC to 461 LOC.

### Completed Implementation
**Successfully created focused modules (completed 2025-06-30):**
* ✅ `client.go` (461 LOC) – Core client, authentication, shared utilities (`apiCall`, `collectAllPages`), cross-cutting concerns
* ✅ `upload.go` (90 LOC) – Upload session methods (CreateUploadSession, UploadChunk, GetUploadSessionStatus, CancelUploadSession)
* ✅ `download.go` (162 LOC) – Download operations (DownloadFile, DownloadFileByItem, DownloadFileChunk, DownloadFileAsFormat, format conversion)
* ✅ `search.go` (80 LOC) – Search functionality (SearchDriveItems, SearchDriveItemsInFolder, SearchDriveItemsWithPaging)
* ✅ `activity.go` (35 LOC) – Activity tracking (GetItemActivities)
* ✅ `permissions.go` (158 LOC) – Sharing and permissions (CreateSharingLink, InviteUsers, ListPermissions, GetPermission, UpdatePermission, DeletePermission)
* ✅ `thumbnails.go` (85 LOC) – Thumbnail and preview operations (GetThumbnails, GetThumbnailBySize, PreviewItem)

### Implementation Details Completed
1. ✅ **Mechanical cut-and-paste with preserved functionality**: All methods moved with identical signatures and behavior
2. ✅ **Import management**: Proper `goimports` applied to each file with correct dependencies
3. ✅ **Cross-file dependencies resolved**: Shared utilities (`apiCall`, `collectAllPages`, `GetDriveItemByPath`) properly accessible across modules
4. ✅ **Zero functional changes**: All existing functionality preserved and verified through comprehensive test suite
5. ✅ **Build verification**: Project builds successfully with no errors
6. ✅ **Test verification**: All unit tests and E2E tests pass, confirming no regressions

### Benefits Achieved
- ✅ **Improved maintainability**: 57% reduction in client.go size (1018 → 461 LOC)
- ✅ **Better code organization**: Each file has focused, single responsibility 
- ✅ **Easier navigation**: Developers can quickly locate specific functionality
- ✅ **Enhanced readability**: Smaller files are more approachable for code review
- ✅ **Future extensibility**: New features can be added to appropriate focused files

---

## 6. Structured Logging & Levels - ✅ COMPLETED

### Goal / Outcome
* ✅ **ACHIEVED**: Replace ad-hoc `Debug(v ...interface{})` logger with structured, leveled API using Go 1.22 `log/slog`.

### Completed Implementation
**Successfully implemented structured logging (completed 2025-07-01):**
1. ✅ **Logger Interface**: Defined comprehensive `Logger` interface with `Debugf`, `Infof`, `Warnf`, `Errorf` methods
2. ✅ **Slog Integration**: Implemented `SlogLogger` using Go 1.22 `log/slog` with `TextHandler`
3. ✅ **NoopLogger**: Created no-op logger for production/silent operation
4. ✅ **Flag Integration**: Wired `--debug` flag to control log level throughout the application
5. ✅ **SDK Integration**: All SDK calls now use structured logging with proper context

### Benefits Achieved
- ✅ **Structured Output**: JSON-compatible logging format for better parsing
- ✅ **Level Control**: Debug, Info, Warn, Error levels with runtime configuration
- ✅ **Performance**: Noop logger ensures zero overhead in production
- ✅ **Consistency**: Standardized logging format across all components
- ✅ **Backward Compatibility**: Existing functionality preserved

---

## 6a. HTTP Client & Polling Configuration - ✅ COMPLETED

### Goal / Outcome
* ✅ **ACHIEVED**: Centralized HTTP client configuration with consistent timeouts and retry behavior across all operations.

### Completed Implementation
**Successfully implemented centralized HTTP configuration (completed 2025-07-01):**
1. ✅ **HTTP Configuration**: Added `HTTPConfig` struct with timeout, retry attempts, retry delay, and max retry delay
2. ✅ **Polling Configuration**: Added `PollingConfig` struct for configurable copy operation monitoring
3. ✅ **Centralized Factory**: Created `NewConfiguredHTTPClient` functions for consistent client creation
4. ✅ **OAuth Preservation**: Maintained OAuth2 transport layer while applying timeout configuration
5. ✅ **Backward Compatibility**: Existing configurations automatically receive sensible defaults
6. ✅ **All Operations**: SDK, upload, download, auth, and copy monitoring use configured timeouts

### Configuration Defaults
- ✅ **HTTP Timeout**: 30 seconds (was previously inconsistent)
- ✅ **Retry Attempts**: 3 attempts with exponential backoff
- ✅ **Retry Delays**: 1 second initial, 10 second maximum
- ✅ **Polling Interval**: 2 seconds initial, 30 seconds maximum, 1.5x multiplier

### Benefits Achieved
- ✅ **Consistency**: All HTTP operations use the same timeout and retry configuration
- ✅ **Reliability**: Improved handling of network issues and rate limiting (429 responses)
- ✅ **User Experience**: Copy operations show progress with smart polling intervals
- ✅ **Resource Efficiency**: Exponential backoff reduces server load
- ✅ **Configurability**: Users can customize timeouts and retry behavior

---

## 6b. UI and Code Quality Improvements - ✅ COMPLETED

### Goal / Outcome
* ✅ **ACHIEVED**: Enhanced user interface functionality and resolved technical debt items.

### Completed Implementation
**Successfully implemented UI and quality improvements (completed 2025-07-01):**
1. ✅ **DisplayDriveItemsWithTitle**: Added flexible title function for context-specific directory listings
2. ✅ **TODO Resolution**: Fixed misleading title issue in `internal/ui/display.go`
3. ✅ **Backward Compatibility**: Existing `DisplayDriveItems` function preserved with default behavior
4. ✅ **Test Coverage**: Comprehensive test suite for UI functionality
5. ✅ **Code Organization**: Improved separation of concerns in display logic

### Benefits Achieved
- ✅ **User Experience**: Custom titles provide better context (root folder vs. subfolders vs. search results)
- ✅ **Technical Debt**: Eliminated identified TODO items that could cause confusion
- ✅ **Maintainability**: Better structured UI code with proper testing
- ✅ **Flexibility**: Easy to customize titles for different use cases

---

## 6c. Error-Handling Consistency ✅ COMPLETED

**Goal**: Create a consistent error handling strategy throughout the OneDrive SDK with clear, testable error types.

**Implementation Summary**:
- **Comprehensive Sentinel Error System**: Added 14 standardized error types covering all OneDrive operation categories
- **Complete Error Wrapping**: Replaced all raw `fmt.Errorf` patterns with proper `%w` verb wrapping across 8 SDK files
- **Error Type Categorization**: Systematic categorization of all error conditions:
  - Authentication errors: `ErrReauthRequired`, `ErrAccessDenied`, `ErrTokenExpired`
  - Operation errors: `ErrRetryLater`, `ErrInvalidRequest`, `ErrOperationFailed`
  - Resource errors: `ErrResourceNotFound`, `ErrConflict`, `ErrQuotaExceeded`
  - Flow errors: `ErrAuthorizationPending`, `ErrAuthorizationDeclined`
  - Technical errors: `ErrInternal`, `ErrDecodingFailed`, `ErrNetworkFailed`
- **Universal Error Testing**: All error sentinels support reliable `errors.Is()` testing
- **Comprehensive Test Coverage**: Dedicated test suite validating error sentinel behavior
- **Enhanced Developer Experience**: Predictable error types enable proper error handling strategies

**Files Updated**:
- `pkg/onedrive/client.go`: User info, shared items, recent items, special folders, delta sync, pagination, versions (20+ error sites)
- `pkg/onedrive/drive.go`: Drive operations, default drive, root items listing (3 error sites)
- `pkg/onedrive/auth.go`: OAuth2 authorization flow, device code flow, token exchange (4 error sites)
- `pkg/onedrive/item.go`: File/folder operations, metadata, children listing, CRUD operations (8 error sites)
- `pkg/onedrive/upload.go`: Upload session management, chunk upload progress (2 error sites)
- `pkg/onedrive/thumbnails.go`: Thumbnail and preview generation (3 error sites)
- `pkg/onedrive/permissions.go`: Sharing links, invitations, permission management (6 error sites)
- `pkg/onedrive/search.go`: Search result processing (1 error site)
- `pkg/onedrive/client_test.go`: New error sentinel testing framework

**Technical Achievements**:
- **Zero Breaking Changes**: All existing error handling patterns preserved while adding type safety
- **Consistent Error Chains**: All SDK errors now provide proper error wrapping for debugging
- **Production Ready**: Enhanced error categorization improves reliability and debuggability
- **Test Validated**: Comprehensive test suite ensures error sentinel behavior works correctly

**Status**: ✅ COMPLETED - All SDK error handling now uses consistent sentinel errors with proper wrapping and testing support.

---

## 7. Security Hardening

### Topics & Steps
* **Path Sanitisation** – refuse `..` segments in remote paths to avoid Graph API oddities.
* **Download Overwrite Protection** – prompt or `--force` flag if local file exists.

---

## 8. Build & CI Enhancements

1. Add `golangci-lint` with default linters + `gci`, `errcheck`, `unused`.
2. Run `go test -race ./...` in CI.
3. Add basic GitHub Action workflow.
4. Produce coverage badge and enforce ≥80 %.

---

## 9. Documentation Updates

* Update `ARCHITECTURE.md`