# Specification: Modular Tool Executor Architecture

## Overview

This task involves designing and implementing a modular, extensible tool executor architecture in Go for the geminiweb-go project. The architecture will provide a clean, interface-based system for registering, discovering, and executing various types of tools (CLI commands, HTTP APIs, file operations, etc.) with support for both synchronous and asynchronous execution patterns. The design will leverage Go's strengths in concurrency, interface composition, and context management while avoiding common pitfalls like the unreliable plugin package.

## Workflow Type

**Type**: feature

**Rationale**: This is a new architectural component being added to the codebase. It involves designing core abstractions, implementing new interfaces, and establishing patterns for tool execution. This is greenfield development with no legacy dependencies to maintain, allowing for modern best practices from the start.

## Task Scope

### Services Involved
- **main** (primary) - Go service at /home/diogo/dev/geminiweb-go where the tool executor architecture will be implemented

### This Task Will:
- [ ] Design and implement core interfaces: `Tool`, `Executor`, `Registry`, `Result`
- [ ] Create a tool registration mechanism using compile-time registration pattern
- [ ] Implement context-driven execution with support for cancellation and timeouts
- [ ] Build dual execution modes (synchronous and asynchronous using errgroup)
- [ ] Design security layer with `SecurityPolicy`, blacklist validation, path validation, and timeout enforcement
- [ ] Define TUI integration interfaces for confirmations (`ConfirmationHandler`) and result rendering
- [ ] Specify tool call protocol (JSON format for ```tool blocks from AI responses)
- [ ] Establish middleware/hook system for cross-cutting concerns (logging, validation, metrics)
- [ ] Define structured error handling patterns with proper error wrapping
- [ ] Create comprehensive tests using go.uber.org/mock framework
- [ ] Document architecture decisions and usage patterns

### Out of Scope:
- Runtime dynamic plugin loading (avoiding Go's plugin package due to version/CGO constraints)
- Full TUI implementation (this task defines interfaces; separate task implements Bubble Tea components)
- Specific tool implementations beyond example reference tool (BashTool, FileReadTool, etc. in separate tasks)
- Persistence/state management for tool execution history

## Service Context

### main

**Tech Stack:**
- Language: Go 1.24.1
- Framework: None (standard library + selected dependencies)
- Package Manager: go mod
- Key directories:
  - `cmd/` - Command-line entry points
  - `internal/` - Internal packages
  - `pkg/` - Public API packages (likely location for tool executor)

**Entry Point:** `cmd/` directory contains application entry points

**How to Run:**
```bash
go run ./cmd/...
go test ./...
```

**Available Dependencies:**
- `github.com/spf13/cobra` (v1.8.1) - CLI framework
- `github.com/charmbracelet/bubbletea` (v1.3.4) - TUI framework
- `github.com/charmbracelet/glamour` (v0.10.0) - Markdown rendering
- `bogdanfinn/tls-client` (v1.11.2) - HTTP client
- `golang.org/x/sync` (v0.19.0) - errgroup for concurrency
- `go.uber.org/mock` (v0.5.0) - Testing framework

## Files to Modify

| File | Service | What to Change |
|------|---------|---------------|
| `pkg/toolexec/tool.go` (NEW) | main | Define core `Tool` interface and related types |
| `pkg/toolexec/executor.go` (NEW) | main | Implement `Executor` with sync/async execution |
| `pkg/toolexec/registry.go` (NEW) | main | Create `Registry` pattern for tool discovery |
| `pkg/toolexec/result.go` (NEW) | main | Define `Result` and error types |
| `pkg/toolexec/security.go` (NEW) | main | Define `SecurityPolicy`, blacklist/path validation, timeout enforcement |
| `pkg/toolexec/confirmation.go` (NEW) | main | Define `ConfirmationHandler` interface for TUI integration |
| `pkg/toolexec/protocol.go` (NEW) | main | Define tool call JSON protocol and parsing patterns |
| `pkg/toolexec/middleware.go` (NEW) | main | Implement middleware/hook system |
| `pkg/toolexec/options.go` (NEW) | main | Functional options for configuration |
| `pkg/toolexec/executor_test.go` (NEW) | main | Comprehensive unit tests |
| `pkg/toolexec/example_tool.go` (NEW) | main | Example tool implementation demonstrating patterns |

## Files to Reference

These files show patterns to follow:

| File | Pattern to Copy |
|------|----------------|
| Existing `internal/` packages | Go project structure, package organization patterns |
| GitHub Actions workflows | CI/CD patterns for running tests |
| Go module dependencies | How external packages are integrated |

## Patterns to Follow

### 1. Interface-Based Design (Core Pattern)

```go
// Small, focused interfaces following Go best practices
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args map[string]any) (*Result, error)
    RequiresConfirmation(args map[string]any) bool
}

type Executor interface {
    Execute(ctx context.Context, toolName string, args map[string]any) (*Result, error)
    ExecuteAsync(ctx context.Context, toolName string, args map[string]any) <-chan *Result
}

type Registry interface {
    Register(tool Tool) error
    Get(name string) (Tool, error)
    List() []ToolInfo
}

type SecurityPolicy interface {
    Validate(ctx context.Context, toolName string, args map[string]any) error
}

type ConfirmationHandler interface {
    RequestConfirmation(ctx context.Context, tool Tool, args map[string]any) (bool, error)
}
```

**Key Points:**
- Keep interfaces minimal and focused
- Use `map[string]any` for flexible argument passing (aligns with JSON protocol)
- `RequiresConfirmation()` allows tools to declare security requirements based on specific arguments
- `SecurityPolicy` enables multi-layered validation (blacklists, path checks, etc.)
- `ConfirmationHandler` abstracts TUI integration for user consent
- Use composition over inheritance
- Return errors explicitly (no panic in library code)
- All execution methods accept `context.Context` as first parameter

### 2. Registry Pattern with Compile-Time Registration

```go
var defaultRegistry = NewRegistry()

func Register(tool Tool) {
    if err := defaultRegistry.Register(tool); err != nil {
        panic(fmt.Sprintf("failed to register tool %s: %v", tool.Name(), err))
    }
}

// Tool implementations can self-register in init()
func init() {
    Register(&MyTool{})
}
```

**Key Points:**
- Avoid Go's plugin package (version/CGO issues)
- Use init() functions for automatic registration
- Panic only during initialization, never at runtime
- Support both default registry and custom registries

### 3. Context-Driven Execution (MANDATORY)

```go
func (e *executor) Execute(ctx context.Context, toolName string, args map[string]any) (*Result, error) {
    tool, err := e.registry.Get(toolName)
    if err != nil {
        return nil, fmt.Errorf("tool not found: %w", err)
    }

    // Security validation
    if e.securityPolicy != nil {
        if err := e.securityPolicy.Validate(ctx, toolName, args); err != nil {
            return nil, fmt.Errorf("security validation failed: %w", err)
        }
    }

    // Confirmation if required
    if tool.RequiresConfirmation(args) && e.confirmHandler != nil {
        confirmed, err := e.confirmHandler.RequestConfirmation(ctx, tool, args)
        if err != nil {
            return nil, fmt.Errorf("confirmation failed: %w", err)
        }
        if !confirmed {
            return nil, ErrUserDenied
        }
    }

    // Always check context before execution
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    return tool.Execute(ctx, args)
}
```

**Key Points:**
- Always use `context.Context` for cancellation support
- Check `ctx.Done()` before long-running operations
- Use `defer cancel()` to prevent goroutine leaks
- Pass context to all downstream calls

### 4. Error Wrapping Strategy

```go
// Use %w verb to maintain error chain
// Custom error types for specific failures
var (
    ErrToolNotFound       = errors.New("tool not found")
    ErrUserDenied         = errors.New("user denied confirmation")
    ErrSecurityViolation  = errors.New("security policy violation")
    ErrTimeout            = errors.New("execution timeout")
)

func (e *executor) Execute(ctx context.Context, toolName string, args map[string]any) (*Result, error) {
    tool, err := e.registry.Get(toolName)
    if err != nil {
        return nil, fmt.Errorf("%w: %s", ErrToolNotFound, toolName)
    }

    result, err := tool.Execute(ctx, args)
    if err != nil {
        return nil, fmt.Errorf("tool %s execution failed: %w", toolName, err)
    }

    return result, nil
}

// Check error type
if errors.Is(err, ErrToolNotFound) {
    // Handle tool not found
}
```

**Key Points:**
- Use `fmt.Errorf` with `%w` to wrap errors
- Define custom sentinel errors (ErrToolNotFound, ErrUserDenied, ErrSecurityViolation, ErrTimeout)
- Use errors.Is() and errors.As() for type checking
- Avoid `errors.New()` for wrapping - loses context
- Preserve error chains for debugging
- Include tool name and context in error messages

### 5. Async Execution with errgroup

```go
import "golang.org/x/sync/errgroup"

func (e *executor) ExecuteAsync(ctx context.Context, toolName string, args map[string]any) <-chan *Result {
    resultCh := make(chan *Result, 1)

    go func() {
        defer close(resultCh)

        result, err := e.Execute(ctx, toolName, args)
        if err != nil {
            resultCh <- &Result{
                ToolName: toolName,
                Success:  false,
                Error:    err.Error(),
            }
        } else {
            resultCh <- result
        }
    }()

    return resultCh
}

// For multiple tools
type ToolExecution struct {
    ToolName string
    Args     map[string]any
}

func (e *executor) ExecuteMany(ctx context.Context, tools []ToolExecution) ([]*Result, error) {
    g, gctx := errgroup.WithContext(ctx)
    results := make([]*Result, len(tools))

    for i, te := range tools {
        i, te := i, te // Capture loop variables
        g.Go(func() error {
            result, err := e.Execute(gctx, te.ToolName, te.Args)
            if err != nil {
                results[i] = &Result{ToolName: te.ToolName, Success: false, Error: err.Error()}
                return err // Fail fast on first error
            }
            results[i] = result
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return results, fmt.Errorf("batch execution failed: %w", err)
    }

    return results, nil
}
```

**Key Points:**
- Use `errgroup.WithContext` for coordinated cancellation
- Capture loop variables when launching goroutines (i, te := i, te)
- Always close channels to prevent receiver hangs
- Use buffered channels (size 1) for single-result scenarios
- Fail-fast behavior in ExecuteMany: first error stops all remaining executions
- Populate results array even on error (partial results available)

### 6. Security Policy Pattern (CRITICAL)

```go
// Multi-layered security validation
type SecurityPolicy interface {
    Validate(ctx context.Context, toolName string, args map[string]any) error
}

// Blacklist validator prevents dangerous commands
type BlacklistValidator struct {
    blockedPatterns []string
}

func (v *BlacklistValidator) Validate(ctx context.Context, toolName string, args map[string]any) error {
    if toolName != "bash" {
        return nil // Only validate bash commands
    }

    cmd, ok := args["command"].(string)
    if !ok {
        return fmt.Errorf("invalid command argument")
    }

    for _, pattern := range v.blockedPatterns {
        if strings.Contains(cmd, pattern) {
            return fmt.Errorf("blocked command pattern: %s", pattern)
        }
    }
    return nil
}

// Path validator prevents access to sensitive files
type PathValidator struct {
    blockedPaths []string
}

func (v *PathValidator) Validate(ctx context.Context, toolName string, args map[string]any) error {
    if toolName != "file_read" && toolName != "file_write" {
        return nil
    }

    path, ok := args["path"].(string)
    if !ok {
        return fmt.Errorf("invalid path argument")
    }

    cleanPath := filepath.Clean(path)
    for _, blocked := range v.blockedPaths {
        if matched, _ := filepath.Match(blocked, cleanPath); matched {
            return fmt.Errorf("access denied to sensitive path: %s", path)
        }
    }
    return nil
}

// Composite security policy chains multiple validators
type CompositeSecurityPolicy struct {
    validators []SecurityPolicy
}

func (p *CompositeSecurityPolicy) Validate(ctx context.Context, toolName string, args map[string]any) error {
    for _, v := range p.validators {
        if err := v.Validate(ctx, toolName, args); err != nil {
            return err
        }
    }
    return nil
}
```

**Key Points:**
- Defense in depth: multiple validation layers
- Blacklist patterns for bash commands (rm -rf /, dd, mkfs, etc.)
- Path validation for file operations (.env, .ssh/, *.pem, etc.)
- Composite pattern for chaining validators
- Tool-specific validation logic
- Always return descriptive errors for security denials

### 7. Tool Call Protocol (JSON Format)

```go
// Protocol for AI to invoke tools via ```tool blocks
// Example from AI response:
// ```tool
// {"name": "bash", "args": {"command": "ls -la"}, "reason": "List directory contents"}
// ```

type ToolCall struct {
    Name   string         `json:"name"`
    Args   map[string]any `json:"args"`
    Reason string         `json:"reason,omitempty"`
}

type Result struct {
    ToolName      string        `json:"tool_name"`
    Success       bool          `json:"success"`
    Output        string        `json:"output"`
    Error         string        `json:"error,omitempty"`
    Truncated     bool          `json:"truncated"`
    ExecutionTime time.Duration `json:"execution_time_ms"`
}

// Parsing tool calls from AI responses
var toolBlockRegex = regexp.MustCompile(`(?s)` + "```tool\\n(.+?)\\n```")

func ParseToolCalls(text string) ([]ToolCall, error) {
    matches := toolBlockRegex.FindAllStringSubmatch(text, -1)
    calls := make([]ToolCall, 0, len(matches))

    for _, match := range matches {
        var call ToolCall
        if err := json.Unmarshal([]byte(match[1]), &call); err != nil {
            return nil, fmt.Errorf("failed to parse tool call: %w", err)
        }
        calls = append(calls, call)
    }

    return calls, nil
}
```

**Key Points:**
- AI generates tool calls in fenced code blocks (```tool)
- JSON format: name (required), args (required), reason (optional)
- Use regexp to extract blocks from streaming responses
- Non-greedy matching (.+?) to handle multiple tool blocks
- (?s) flag allows . to match newlines
- Streaming consideration: buffer text until complete block received
- Result format includes timing and truncation info for AI feedback

### 8. Functional Options Pattern

```go
type ExecutorOption func(*executorConfig)

func WithTimeout(d time.Duration) ExecutorOption {
    return func(c *executorConfig) {
        c.timeout = d
    }
}

func WithSecurityPolicy(policy SecurityPolicy) ExecutorOption {
    return func(c *executorConfig) {
        c.securityPolicy = policy
    }
}

func WithConfirmationHandler(handler ConfirmationHandler) ExecutorOption {
    return func(c *executorConfig) {
        c.confirmHandler = handler
    }
}

func WithMiddleware(mw Middleware) ExecutorOption {
    return func(c *executorConfig) {
        c.middlewares = append(c.middlewares, mw)
    }
}

func WithMaxConcurrent(n int) ExecutorOption {
    return func(c *executorConfig) {
        c.maxConcurrent = n
    }
}

func NewExecutor(registry Registry, opts ...ExecutorOption) *executor {
    config := &executorConfig{
        timeout:         30 * time.Second,
        maxConcurrent:   1, // Safe default
        middlewares:     []Middleware{},
        securityPolicy:  nil, // Optional
        confirmHandler:  nil, // Optional
    }

    for _, opt := range opts {
        opt(config)
    }

    return &executor{
        registry: registry,
        config:   config,
    }
}
```

**Key Points:**
- Use for backward-compatible configuration
- Security policy and confirmation handler are optional (nil-safe execution)
- Default timeout: 30 seconds (matches research recommendation)
- Default max concurrent: 1 (conservative for safety)
- Provide sensible defaults
- Allow option composition
- Keep option functions simple and focused

## Requirements

### Functional Requirements

1. **Tool Interface Abstraction**
   - Description: Define a minimal, focused `Tool` interface that all tools must implement
   - Acceptance: Interface includes Name(), Description(), Execute(), and RequiresConfirmation() methods; Execute accepts context.Context and map[string]any; returns *Result and error

2. **Tool Registry**
   - Description: Implement a registry pattern for tool registration, discovery, and retrieval
   - Acceptance: Registry supports Register(), Get(), and List() operations; prevents duplicate registrations; thread-safe

3. **Security Policy System**
   - Description: Multi-layered security validation before tool execution
   - Acceptance: SecurityPolicy interface validates tool calls; supports blacklist checking (rm -rf /, dd, mkfs), path validation (.env, .ssh/, *.pem), composite policy chaining; returns descriptive errors

4. **Confirmation System**
   - Description: Request user confirmation for dangerous operations
   - Acceptance: ConfirmationHandler interface abstracts TUI integration; tools declare if confirmation required via RequiresConfirmation(); confirmation happens after security validation, before execution

5. **Tool Call Protocol**
   - Description: Parse tool calls from AI responses in ```tool JSON format
   - Acceptance: ParseToolCalls() extracts JSON from fenced code blocks; handles multiple blocks; validates JSON schema; supports streaming (partial blocks)

6. **Synchronous Execution**
   - Description: Support blocking tool execution with context for cancellation
   - Acceptance: Executor.Execute() runs security validation → confirmation → execution; blocks until completion or context cancellation; returns Result or error

7. **Asynchronous Execution**
   - Description: Support non-blocking tool execution using goroutines and channels
   - Acceptance: ExecuteAsync() returns channel that receives result; supports cancellation via context

8. **Batch Execution**
   - Description: Execute multiple tools concurrently using errgroup pattern
   - Acceptance: ExecuteMany() runs tools in parallel; fails fast on first error; respects context cancellation

9. **Timeout Enforcement**
   - Description: Enforce maximum execution time per tool
   - Acceptance: Default 30 second timeout; configurable via options; context.WithTimeout() applied; timeout errors distinguishable

10. **Output Truncation**
    - Description: Prevent memory exhaustion from large tool outputs
    - Acceptance: Result includes Truncated boolean; default 100KB limit; truncated output still returned with flag set

11. **Middleware System**
    - Description: Allow pre/post execution hooks for logging, validation, metrics
    - Acceptance: Middleware can wrap tool execution; chain multiple middlewares; access input/output/errors

12. **Error Handling**
    - Description: Structured error types with proper wrapping and context preservation
    - Acceptance: Errors use %w for wrapping; custom error types (ErrToolNotFound, ErrUserDenied, ErrSecurityViolation, ErrTimeout); errors.Is() and errors.As() work correctly

13. **Configuration Options**
    - Description: Functional options pattern for executor configuration
    - Acceptance: Options for timeout, security policy, confirmation handler, middleware, concurrency limits; backward-compatible; composable

### Edge Cases

1. **Context Cancellation During Execution** - Tool must check ctx.Done() and return ctx.Err(); no goroutine leaks
2. **Duplicate Tool Registration** - Registry returns error; does not overwrite existing tool silently
3. **Tool Not Found** - Executor returns typed error (ErrToolNotFound) with tool name
4. **Security Violation** - Returns ErrSecurityViolation with specific reason (blacklist match, blocked path, etc.)
5. **User Denies Confirmation** - Returns ErrUserDenied; does not execute tool
6. **Timeout Exceeded** - Returns ErrTimeout; context cancellation propagates to tool; process cleanup ensured
7. **Malformed Tool Call JSON** - ParseToolCalls() returns descriptive error; does not panic
8. **Partial Tool Block in Stream** - Parser buffers until complete block; handles incomplete blocks gracefully
9. **Type Assertion Failures** - Always use comma-ok idiom: `cmd, ok := args["command"].(string)` to prevent panics
10. **Output Exceeds Limit** - Truncates output; sets Truncated flag; returns partial output (not error)
11. **Concurrent Registry Access** - Registry is thread-safe; uses RWMutex for high read concurrency
12. **Middleware Panic Recovery** - Executor recovers from panics in middleware and tools; converts to error
13. **Nil SecurityPolicy/ConfirmationHandler** - Executor handles nil gracefully (skips validation/confirmation)
14. **Zero-Value Executor** - Document that zero-value executor is NOT usable; must use NewExecutor()

## Implementation Notes

### DO
- Follow interface-based design with small, focused interfaces
- Use `context.Context` for ALL execution paths
- Implement multi-layered security (blacklist + path validation + confirmation + timeout)
- Use comma-ok idiom for type assertions: `cmd, ok := args["command"].(string)`
- Validate all inputs before execution (security-first mindset)
- Handle nil SecurityPolicy and ConfirmationHandler gracefully
- Implement proper error wrapping with `%w` verb
- Define custom error types (ErrToolNotFound, ErrUserDenied, ErrSecurityViolation, ErrTimeout)
- Use `errgroup.WithContext` for concurrent execution
- Truncate output at configurable limit (default 100KB)
- Write comprehensive unit tests with mock framework
- Test security policies with attack scenarios (command injection, path traversal)
- Add package-level documentation with examples
- Use functional options for backward-compatible configuration
- Check `ctx.Done()` before long-running operations
- Implement thread-safe registry with RWMutex (not mutex - optimize for reads)
- Recover from panics in tool execution and middleware
- Use non-greedy regex for parsing tool blocks: `.+?` not `.+`
- Buffer streaming responses until complete tool block received
- Return descriptive errors for security violations (which pattern matched, which path blocked)

### DON'T
- Use Go's plugin package (version/CGO constraints)
- Panic at runtime (only acceptable in init() for registration)
- Use `errors.New()` - loses error context
- Trust AI-generated tool calls without validation
- Execute bash commands without blacklist checking
- Access paths without validation (.env, .ssh/, *.pem)
- Execute tools requiring confirmation without user consent
- Create goroutines without context cancellation checks
- Forget `defer cancel()` after context.WithTimeout()
- Block indefinitely without respecting context
- Use greedy regex matching (`.+` captures too much)
- Mutate shared state without synchronization
- Return interface{} when concrete types work (use map[string]any for flexibility)
- Ignore errors or return nil errors on failure
- Skip type assertions (will panic on wrong types)
- Silently drop tool output that exceeds limit (set Truncated flag)

## Development Environment

### Start Services

```bash
# Run tests
go test ./pkg/toolexec/... -v

# Run tests with coverage
go test ./pkg/toolexec/... -cover -coverprofile=coverage.out

# View coverage
go tool cover -html=coverage.out

# Run linter (if configured)
golangci-lint run ./pkg/toolexec/...

# Build
go build ./cmd/...
```

### Service URLs
- N/A (library package, no HTTP service)

### Required Environment Variables
- None for core tool executor library
- Individual tool implementations may require specific env vars

## Success Criteria

The task is complete when:

1. [ ] All core interfaces defined (Tool, Executor, Registry, Result, SecurityPolicy, ConfirmationHandler)
2. [ ] Tool interface includes RequiresConfirmation() method
3. [ ] Registry pattern implemented with compile-time registration support
4. [ ] SecurityPolicy with blacklist and path validation implemented
5. [ ] ConfirmationHandler interface defined for TUI integration
6. [ ] Tool call protocol (ToolCall, Result types) and ParseToolCalls() implemented
7. [ ] Synchronous execution with security validation → confirmation → execution flow works
8. [ ] Asynchronous execution with channels works
9. [ ] Batch execution with errgroup implemented
10. [ ] Timeout enforcement (30s default) with ErrTimeout error type works
11. [ ] Output truncation (100KB default) with Truncated flag works
12. [ ] Middleware system supports pre/post execution hooks
13. [ ] Structured error types defined (ErrToolNotFound, ErrUserDenied, ErrSecurityViolation, ErrTimeout)
14. [ ] Functional options pattern for configuration implemented (WithSecurityPolicy, WithConfirmationHandler, WithTimeout, WithMaxConcurrent)
15. [ ] Unit tests achieve >80% coverage
16. [ ] Security tests verify blacklist and path validation work correctly
17. [ ] All tests pass with `go test ./pkg/toolexec/...`
18. [ ] No race conditions with `go test -race ./pkg/toolexec/...`
19. [ ] Package documentation includes usage examples
20. [ ] No goroutine leaks (verified with leak detector or manual review)
21. [ ] Thread-safety verified for registry (RWMutex for concurrent reads)
22. [ ] Example tool implementation demonstrating patterns (including RequiresConfirmation)
23. [ ] Regex-based tool call parsing handles multiple blocks and streaming correctly

## QA Acceptance Criteria

**CRITICAL**: These criteria must be verified by the QA Agent before sign-off.

### Unit Tests

| Test | File | What to Verify |
|------|------|----------------|
| TestToolInterface | `pkg/toolexec/tool_test.go` | Tool interface can be implemented by mock; includes RequiresConfirmation() |
| TestRegistryRegister | `pkg/toolexec/registry_test.go` | Register() adds tool; duplicate returns error |
| TestRegistryGet | `pkg/toolexec/registry_test.go` | Get() retrieves registered tool; missing returns ErrToolNotFound |
| TestRegistryList | `pkg/toolexec/registry_test.go` | List() returns all registered tools |
| TestSecurityPolicyBlacklist | `pkg/toolexec/security_test.go` | Blacklist validator blocks rm -rf /, dd, mkfs commands |
| TestSecurityPolicyPath | `pkg/toolexec/security_test.go` | Path validator blocks .env, .ssh/, *.pem files |
| TestSecurityPolicyComposite | `pkg/toolexec/security_test.go` | Composite policy chains validators correctly |
| TestConfirmationHandler | `pkg/toolexec/confirmation_test.go` | ConfirmationHandler interface can be implemented |
| TestToolCallProtocol | `pkg/toolexec/protocol_test.go` | ParseToolCalls() extracts JSON from ```tool blocks |
| TestToolCallMultipleBlocks | `pkg/toolexec/protocol_test.go` | Parser handles multiple tool blocks in same text |
| TestToolCallMalformed | `pkg/toolexec/protocol_test.go` | Parser returns error for invalid JSON |
| TestExecutorExecuteSync | `pkg/toolexec/executor_test.go` | Synchronous execution: security → confirmation → execution |
| TestExecutorSecurityViolation | `pkg/toolexec/executor_test.go` | Executor returns ErrSecurityViolation on blacklist match |
| TestExecutorUserDenied | `pkg/toolexec/executor_test.go` | Executor returns ErrUserDenied when confirmation rejected |
| TestExecutorExecuteContext | `pkg/toolexec/executor_test.go` | Context cancellation stops execution; returns ctx.Err() |
| TestExecutorTimeout | `pkg/toolexec/executor_test.go` | Timeout enforcement works; returns ErrTimeout |
| TestExecutorExecuteAsync | `pkg/toolexec/executor_test.go` | Async execution returns channel with result |
| TestExecutorExecuteMany | `pkg/toolexec/executor_test.go` | Batch execution runs concurrently; fails fast |
| TestOutputTruncation | `pkg/toolexec/result_test.go` | Large output truncated at 100KB; Truncated flag set |
| TestMiddleware | `pkg/toolexec/middleware_test.go` | Middleware wraps execution; can access input/output |
| TestMiddlewareChain | `pkg/toolexec/middleware_test.go` | Multiple middlewares chain correctly |
| TestErrorWrapping | `pkg/toolexec/result_test.go` | Errors wrap with %w; errors.Is/As work |
| TestErrorTypes | `pkg/toolexec/result_test.go` | Custom errors (ErrToolNotFound, ErrUserDenied, ErrSecurityViolation, ErrTimeout) defined |
| TestFunctionalOptions | `pkg/toolexec/options_test.go` | Options apply configuration correctly |
| TestConcurrency | `pkg/toolexec/executor_test.go` | Registry is thread-safe under concurrent access (RWMutex) |
| TestPanicRecovery | `pkg/toolexec/executor_test.go` | Panics in tools/middleware convert to errors |
| TestNilSecurityPolicy | `pkg/toolexec/executor_test.go` | Executor handles nil SecurityPolicy gracefully (skips validation) |
| TestNilConfirmationHandler | `pkg/toolexec/executor_test.go` | Executor handles nil ConfirmationHandler gracefully (skips confirmation) |

### Integration Tests

| Test | Services | What to Verify |
|------|----------|----------------|
| TestExampleTool | main | Example tool implementation works end-to-end with RequiresConfirmation |
| TestMultipleTools | main | Multiple tools can be registered and executed |
| TestSecurityIntegration | main | Security policy blocks dangerous commands in real execution |
| TestConfirmationIntegration | main | Confirmation handler integrates with tool execution flow |
| TestToolCallParsing | main | ParseToolCalls() works with real AI response text |
| TestMiddlewareIntegration | main | Middleware works with real tool execution |

### End-to-End Tests

| Flow | Steps | Expected Outcome |
|------|-------|------------------|
| Register and Execute Tool | 1. Create tool implementation 2. Register with registry 3. Execute via executor | Tool executes successfully; returns Result with output |
| Security Violation Flow | 1. Create executor with security policy 2. Attempt dangerous command (rm -rf /) 3. Check error | Returns ErrSecurityViolation; tool not executed |
| Confirmation Flow | 1. Create executor with confirmation handler 2. Execute tool requiring confirmation 3. User approves 4. Check result | Confirmation requested; tool executes after approval |
| Confirmation Denial Flow | 1. Create executor with confirmation handler 2. Execute tool requiring confirmation 3. User denies 4. Check error | Returns ErrUserDenied; tool not executed |
| Context Cancellation | 1. Start long-running tool 2. Cancel context 3. Check result | Execution stops; returns ctx.Err() |
| Timeout Flow | 1. Execute tool with 1s timeout 2. Tool runs for 5s 3. Check error | Returns ErrTimeout after 1s |
| Batch Execution | 1. Register multiple tools 2. ExecuteMany() 3. Verify results | All tools execute; results collected; fast failure on error |
| Tool Call Parsing | 1. AI response with ```tool block 2. ParseToolCalls() 3. Execute parsed call | JSON extracted; tool executed with correct args |

### Browser Verification (if frontend)
N/A - This is a backend library package

### Database Verification (if applicable)
N/A - No database interaction in core tool executor

### Code Quality Checks

| Check | Command | Expected |
|-------|---------|----------|
| Test Coverage | `go test ./pkg/toolexec/... -cover` | Coverage >80% |
| No Race Conditions | `go test ./pkg/toolexec/... -race` | PASS with no race warnings |
| Build Success | `go build ./pkg/toolexec/...` | Successful compilation |
| Vet Passes | `go vet ./pkg/toolexec/...` | No issues reported |
| Gofmt Check | `gofmt -l pkg/toolexec/` | No files need formatting |

### Documentation Verification

| Check | File | Expected |
|-------|------|----------|
| Package Documentation | `pkg/toolexec/doc.go` | Overview, usage examples, architecture |
| Interface Documentation | `pkg/toolexec/tool.go` | All interfaces have godoc comments |
| Example Code | `pkg/toolexec/example_test.go` | Working example demonstrating usage |
| README | `pkg/toolexec/README.md` (optional) | Architecture decisions, patterns, gotchas |

### QA Sign-off Requirements
- [ ] All unit tests pass (28+ tests covering core functionality, security, protocol)
- [ ] Integration tests pass (6+ tests for real-world usage including security)
- [ ] End-to-end tests pass (8+ flows including security violations and confirmations)
- [ ] Test coverage >80% across all packages
- [ ] Security tests verify blacklist blocks dangerous commands (rm -rf /, dd, mkfs)
- [ ] Security tests verify path validation blocks sensitive files (.env, .ssh/, *.pem)
- [ ] No race conditions detected with `-race` flag
- [ ] No goroutine leaks (manual review or leak detector)
- [ ] Thread-safety verified for registry under concurrent access (RWMutex)
- [ ] Code follows Go best practices (interfaces, error handling, naming)
- [ ] All interfaces defined (Tool with RequiresConfirmation, Executor, Registry, SecurityPolicy, ConfirmationHandler)
- [ ] Tool call protocol (ParseToolCalls) handles ```tool blocks correctly
- [ ] Tool call parsing handles multiple blocks and malformed JSON gracefully
- [ ] Timeout enforcement works (30s default, configurable)
- [ ] Output truncation works (100KB default, Truncated flag set)
- [ ] Custom error types defined (ErrToolNotFound, ErrUserDenied, ErrSecurityViolation, ErrTimeout)
- [ ] Nil SecurityPolicy and ConfirmationHandler handled gracefully
- [ ] Package documentation complete with examples
- [ ] All code formatted with gofmt
- [ ] go vet reports no issues
- [ ] No security vulnerabilities (validation before execution, proper error handling)
- [ ] Example tool implementation works end-to-end (demonstrates RequiresConfirmation)
- [ ] Context cancellation works correctly in all execution paths
- [ ] Security validation → confirmation → execution flow works
- [ ] Middleware system functions as designed
- [ ] Error wrapping preserves context (errors.Is/As work)
- [ ] Type assertions use comma-ok idiom (no panics on wrong types)
- [ ] Regex parsing uses non-greedy matching (.+? not .+)
