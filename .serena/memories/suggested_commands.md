# Suggested Commands

## Building

```bash
# Production build with version info (requires CGO_ENABLED=1)
make build

# Fast development build (no optimization)
make build-dev

# Install to GOPATH/bin
make install
```

## Running

```bash
# Build and run with arguments
make run ARGS="chat"
make run ARGS="\"your prompt\""
make run ARGS="config"
make run ARGS="persona create MyPersona"
make run ARGS="persona list"

# Direct execution after build
./build/geminiweb chat              # Interactive chat
./build/geminiweb "your prompt"     # Single query
./build/geminiweb config            # Configuration menu
./build/geminiweb import-cookies <path>  # Import browser cookies
./build/geminiweb persona create <name>  # Create a persona
./build/geminiweb persona list      # List personas
./build/geminiweb persona delete <id>    # Delete persona
```

## Testing

```bash
# Run all tests
make test
# or
go test -v ./...

# Run tests with coverage report
make test-coverage

# Run specific test
go test -v ./internal/api -run TestClientInit
go test -v ./internal/commands -run TestPersona
```

**Note**: Tests require environment variables:
```bash
export SECURE_1PSID="your_cookie_value"
export SECURE_1PSIDTS="your_cookie_value"
```

## Code Quality

```bash
# Run linter (requires golangci-lint)
make lint

# Format code (requires gofumpt)
make fmt
```

## Other Commands

```bash
# Download and tidy dependencies
make deps

# Clean build artifacts
make clean

# Verify build would succeed
make check

# Verify go.mod is tidy
make verify-mod

# Build for all platforms (requires goreleaser)
make release-snapshot
```

## Debugging

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o build/geminiweb-debug ./cmd/geminiweb

# Run with verbose output
./build/geminiweb --verbose chat

# Test cookie import
./build/geminiweb import-cookies ~/.config/google-chrome/Default/Cookies
```

## System Utilities

The project runs on Linux. Standard utilities available:
- `git` - Version control
- `ls`, `cd`, `find`, `grep` - File operations
- `go` - Go toolchain
- `make` - Build automation
- `tree` - Directory structure display
- `curl` - HTTP requests for testing
