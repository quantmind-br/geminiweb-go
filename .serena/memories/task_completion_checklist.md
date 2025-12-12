# Task Completion Checklist

Before considering a task complete, run through this checklist:

## 1. Code Quality

```bash
# Format code
make fmt

# Run linter
make lint
```

Fix any formatting issues or linter warnings.

## 2. Testing

```bash
# Run all tests
make test
```

Ensure all tests pass. If you added new functionality:
- Add corresponding tests in `*_test.go` files
- Follow existing test patterns (see `internal/api/*_test.go`)
- Tests require environment variables: `SECURE_1PSID` and optionally `SECURE_1PSIDTS`

## 3. Build Verification

```bash
# Verify the build succeeds
make check
```

## 4. Module Verification

```bash
# Ensure go.mod is tidy
make verify-mod
```

If this fails, run `go mod tidy` first.

## 5. Manual Testing (if applicable)

```bash
# Build development version
make build-dev

# Test the affected functionality
./build/geminiweb <relevant-command>
```

## 6. Feature-Specific Checklist

When working on specific areas:

### API Client (internal/api/)
- Test with real Gemini cookies
- Verify token extraction works
- Check cookie rotation functionality
- Ensure proper error handling

### Commands (internal/commands/)
- Test CLI args parsing
- Verify help text is accurate
- Check error messages are user-friendly

### TUI (internal/tui/)
- Test on different terminal sizes
- Verify styling is consistent
- Check input handling

### Persona Management
- Test CRUD operations
- Verify storage in config directory
- Check prompt injection prevention

## Quick One-liner

For quick verification of code changes:

```bash
make fmt && make lint && make test && make check
```

## Pre-commit Checks

Before committing changes:

1. Run `make fmt` to format code
2. Run `make lint` to check for issues
3. Run `make test` to ensure tests pass
4. Run `make check` to verify build
5. Check that `SECURE_1PSID` is set if testing API functionality

## File Changes

Check git status for current state of working tree before committing.
