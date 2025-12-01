# Recent Changes and Project Status

## Git Status Overview

As of the current workspace snapshot, the following changes have been made:

### Deleted Files (D)
- `internal/api/mock_test.go` - Removed mock test file
- `internal/api/stream.go` - Removed SSE streaming implementation
- `internal/api/stream_test.go` - Removed streaming tests

### Modified Files (M)
The following files have uncommitted modifications:
- `internal/api/client_test.go`
- `internal/api/generate.go`
- `internal/api/session.go`
- `internal/commands/chat.go`
- `internal/commands/query.go`
- `internal/tui/model.go`
- `api.test`
- `cookies.json`

### New Files
The following new files were created:
- `.gitignore`
- `.serena/`
- `CLAUDE.md`
- `internal/browser/browser.go` - Browser cookie extraction module
- `internal/browser/browser_test.go` - Browser module tests
- `internal/commands/autologin.go` - Auto-login command
- `internal/api/generate_test.go`
- `internal/api/session_test.go`
- `internal/commands/query_test.go`
- `internal/commands/root_test.go`
- `internal/errors/errors_test.go`
- `internal/models/models_test.go`
- `internal/tui/model_test.go`
- `removesteaming.md`

## Key Architectural Changes

### Removal of Streaming Support
The project has removed Server-Sent Events (SSE) streaming functionality:
- **File**: `internal/api/stream.go` - Deleted
- **File**: `internal/api/stream_test.go` - Deleted
- **Impact**: Client now returns complete responses synchronously instead of streaming
- **Reason**: Streamed responses were causing issues with the Gemini web API

### Testing Infrastructure
- Enhanced test coverage with new `_test.go` files
- Separated unit tests into dedicated files per module
- Tests require `SECURE_1PSID` environment variable
- Some tests require `SECURE_1PSIDTS` depending on account type

### Commands Refactoring
- `internal/commands/chat.go` - Modified for improved chat experience
- `internal/commands/query.go` - Enhanced query handling
- `internal/commands/root_test.go` - New test suite for root command

### TUI Enhancements
- `internal/tui/model.go` - Modified to improve user interface
- Added `internal/tui/model_test.go` - New test suite for TUI

### Persona System
- New feature for custom persona management
- CRUD operations for persona creation and management
- Storage in `~/.geminiweb/personas.json`

### Browser Cookie Extraction (NEW)
- **Package**: `internal/browser/` - New module for browser cookie extraction
- **Library**: Uses `browserutils/kooky` for cross-browser cookie access
- **Supported browsers**: Chrome, Chromium, Firefox, Edge, Opera
- **Features**:
  - `auto-login` command: Extract cookies directly from browser
  - `--browser-refresh` flag: Auto-refresh cookies on 401 errors
  - Multi-profile support: Scans all browser profiles
  - Rate limiting: 30 second minimum between refresh attempts
- **Note**: Browser must be closed to avoid SQLite database locks

## Documentation Updates

### CLAUDE.md
Project documentation has been updated with:
- Current architecture details
- Client lifecycle patterns
- Persona management documentation
- Recent changes tracking

## Testing Strategy

Current testing approach:
1. **Unit Tests**: Individual module testing
   - `internal/api/*_test.go` - API client tests
   - `internal/commands/*_test.go` - Command tests
   - `internal/tui/model_test.go` - TUI tests
   - `internal/errors/errors_test.go` - Error handling tests
   - `internal/models/models_test.go` - Model tests

2. **Integration Tests**: End-to-end testing with real Gemini cookies
   - Requires valid `SECURE_1PSID` cookie
   - Optionally requires `SECURE_1PSIDTS`

3. **Manual Testing**: 
   - Build and run: `make build-dev && ./build/geminiweb chat`
   - Test new features interactively

## Recommended Next Steps

1. Review and commit test files
2. Run `make test` to verify all tests pass
3. Verify the removal of streaming functionality doesn't break existing workflows
4. Test persona management features
5. Update documentation if needed
6. Clean up temporary files (`cookies.json`, etc.)

## Known Issues

- No known critical issues at this time
- Cookie rotation should be tested with long-running sessions
- Persona system should be tested for prompt injection vulnerabilities
