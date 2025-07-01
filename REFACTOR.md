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

## 6. Structured Logging & Levels

### Goal / Outcome
* Replace ad-hoc `Debug(v ...interface{})` logger with structured, leveled API using Go 1.22 `log/slog`.

### Work Breakdown
1. Define
   ```go
   type Logger interface { Debugf(string, ...any); Infof(...); Warnf(...); Errorf(...) }
   ```
2. Provide default `StdLogger` using `slog` with `TextHandler`.
3. Wire `--debug` and future `--verbose` flags to set log level.
4. Replace `logger.Debug(...)` calls with `Debugf` formatting for consistency.

### Risks
* SDK consumers outside CLI must set logger (provide noop default).

---

## 7. Error-Handling Consistency

### Goal / Outcome
* No raw `fmt.Errorf(...)` leakage from SDK – always wrap with typed sentinel or exported `ErrorKind`.

### Work Breakdown
1. Audit `pkg/onedrive/*.go` for `fmt.Errorf("%w")` with non-sentinel root; introduce new sentinels where missing.
2. Update tests to use `errors.Is`.
3. Document in `ARCHITECTURE.md`.

### Risks
* Might hide useful message – ensure original `err` wrapped.

---

## 8. Security Hardening

### Topics & Steps
* **Path Sanitisation** – refuse `..` segments in remote paths to avoid Graph API oddities.
* **Download Overwrite Protection** – prompt or `--force` flag if local file exists.

---

## 9. Build & CI Enhancements

1. Add `golangci-lint` with default linters + `gci`, `errcheck`, `unused`.
2. Run `go test -race ./...` in CI.
3. Add basic GitHub Action workflow.
4. Produce coverage badge and enforce ≥80 %.

---

## 10. Documentation Updates

* Update `ARCHITECTURE.md` with new package splits, context model, logger interface.
* Add `docs/uploads.md` explaining resumable logic & session files.

---

### Overall Risks / Coordination
* **Parallel Refactors** – perform file-moving (items split, SDK split) first to minimise later merge pain.
* **Test Fragility** – large test suite depends on mocks; update mocks in lock-step.