# TASKS.md - Bug Fixes and Concurrency Issues

## Project Briefing

This task involves fixing **27 identified issues** in the geminiweb-go codebase:
- 7 critical issues (race conditions, panics, resource leaks)
- 8 high severity issues
- 12 moderate/low severity issues

**Files Affected:** 12
**Complexity:** High

---

## Implementation Tasks

### Phase 1: Make config.Cookies Thread-Safe
- [x] 1.1 Add `sync.RWMutex` to Cookies struct in `internal/config/cookies.go`
- [x] 1.2 Add thread-safe getters: `GetSecure1PSID()`, `GetSecure1PSIDTS()`, `Snapshot()`, `SetBoth()`
- [x] 1.3 Modify `Update1PSIDTS()` to use lock
- [x] 1.4 Modify `ToMap()` to be thread-safe
- [x] 1.5 Write tests for Cookies thread-safety

### Phase 2: Fix CookieRotator
- [x] 2.1 Add error callback and create new channel in `Start()` in `internal/api/rotate.go`
- [x] 2.2 Modify `Start()` to create new channel (allows restart)
- [x] 2.3 Update NewCookieRotator with functional options
- [x] 2.4 Write tests for CookieRotator restart and double-stop

### Phase 3: Fix ChatSession Race Condition
- [x] 3.1 Add `sync.RWMutex` to ChatSession struct in `internal/api/session.go`
- [x] 3.2 Add `copyMetadata()` helper function
- [x] 3.3 Modify `SendMessage()` to use locks
- [x] 3.4 Rename `updateMetadata()` to `updateMetadataLocked()`
- [x] 3.5 Add locks to `SetMetadata()`, `GetMetadata()`
- [x] 3.6 Add locks to `CID()`, `RID()`, `RCID()`
- [x] 3.7 Add locks to `GetModel()`, `SetModel()`
- [x] 3.8 Add locks to `LastOutput()`, `ChooseCandidate()`
- [x] 3.9 Add locks to `SetGem()`, `GetGemID()`
- [x] 3.10 Write tests for ChatSession thread-safety

### Phase 4: Fix Ignored Errors in batch.go
- [x] 4.1 Handle `url.Parse()` error in `internal/api/batch.go`

### Phase 5: Fix io.ReadAll Errors in upload.go
- [x] 5.1 Handle `io.ReadAll()` error in upload error response (first location)
- [x] 5.2 Handle `io.ReadAll()` error in upload error response (second location)

### Phase 6: Fix Cookie Store Leak in browser.go
- [x] 6.1 Refactor `extractFromBrowser()` to use defer for cleanup in `internal/browser/browser.go`

### Phase 7: Improve Error Handling in gems.go
- [x] 7.1 Collect errors in `FetchGems()` and return if no gems found in `internal/api/gems.go`

### Phase 8: Fix Spinner Double-Close
- [x] 8.1 Add `stopped` flag to spinner struct in `internal/commands/query.go`
- [x] 8.2 Add `stopOnce()` method
- [x] 8.3 Modify `stopWithSuccess()` and `stopWithError()` to use `stopOnce()`

### Phase 9: Refactor Lock in RefreshFromBrowser
- [x] 9.1 Split `RefreshFromBrowser()` into phases to minimize lock duration in `internal/api/client.go`

### Phase 10: Minor Fixes
- [x] 10.1 Handle `filepath.Abs()` error in `internal/api/download.go`
- [x] 10.2 Update cookie usage in `generate.go` to use `Snapshot()`
- [x] 10.3 Update cookie usage in `batch.go` to use `Snapshot()`
- [x] 10.4 Update cookie usage in `token.go` to use `Snapshot()`
- [x] 10.5 Update cookie usage in `rotate.go` to use `Snapshot()`
- [x] 10.6 Update cookie usage in `client.go` to use `SetBoth()`

### Validation & Testing
- [x] V1. Run `go test -race ./internal/api/...`
- [x] V2. Run `go test -race ./internal/config/...`
- [x] V3. Run `go test -race ./internal/browser/...`
- [x] V4. Run `go test -race ./internal/commands/...`
- [x] V5. Run `make test` for full test suite
- [x] V6. Run `make build` to verify build succeeds

---

## Summary

All 27 issues from the bug fix plan have been successfully implemented:

### Key Changes:

**Thread Safety:**
- `config.Cookies` is now thread-safe with RWMutex and atomic accessor methods (`Snapshot()`, `SetBoth()`)
- `api.ChatSession` is now thread-safe for concurrent access
- `api.CookieRotator` now captures values before goroutine to prevent races

**Error Handling:**
- `batch.go`: `url.Parse()` error is now properly handled
- `upload.go`: `io.ReadAll()` errors are now handled gracefully
- `gems.go`: Errors are now collected and reported if no gems found
- `download.go`: `filepath.Abs()` error fallback to relative path

**Resource Management:**
- `browser.go`: Cookie stores are now properly closed using defer pattern
- `rotate.go`: Channel is now created in `Start()` allowing restart after `Stop()`
- `query.go`: Spinner double-close prevented with `stopped` flag and `stopOnce()` method

**Performance:**
- `client.go`: `RefreshFromBrowser()` now uses three-phase locking to minimize lock duration during network operations

### Test Results:
- All tests in `internal/api`, `internal/config`, `internal/browser` pass with race detection enabled
- Build compiles successfully
- One pre-existing test failure in `internal/commands` (missing 'sync' subcommand) is unrelated to these changes

