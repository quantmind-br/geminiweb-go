// Package toolexec provides a modular, extensible tool executor architecture
// for registering, discovering, and executing various types of tools with
// support for both synchronous and asynchronous execution patterns.
//
// # Overview
//
// The toolexec package is designed around a set of core interfaces that enable
// flexible tool management and execution:
//
//   - Tool: The interface that all executable tools must implement
//   - Registry: Manages tool registration and discovery
//   - Executor: Handles tool execution with context support and middleware
//   - SecurityPolicy: Validates tool executions against security rules
//   - ConfirmationHandler: Requests user confirmation for dangerous operations
//   - Middleware: Enables cross-cutting concerns like logging, timing, and validation
//
// # Quick Start
//
// Create a tool by implementing the Tool interface:
//
//	type GreetingTool struct{}
//
//	func (t *GreetingTool) Name() string        { return "greeting" }
//	func (t *GreetingTool) Description() string { return "Returns a greeting message" }
//
//	func (t *GreetingTool) Execute(ctx context.Context, input *Input) (*Output, error) {
//	    name := input.GetParamString("name")
//	    if name == "" {
//	        name = "World"
//	    }
//	    return NewOutput().
//	        WithData([]byte("Hello, " + name + "!")).
//	        WithMessage("Greeting generated"), nil
//	}
//
// Register and execute the tool:
//
//	// Register the tool
//	registry := NewRegistry()
//	registry.Register(&GreetingTool{})
//
//	// Create an executor
//	executor := NewExecutor(registry)
//
//	// Execute the tool
//	input := NewInput().WithParam("name", "Alice")
//	output, err := executor.Execute(context.Background(), "greeting", input)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(string(output.Data)) // Output: Hello, Alice!
//
// # Architecture
//
// The package follows these design principles:
//
//   - Interface-Based Design: Small, focused interfaces following Go best practices
//   - Context-Driven Execution: All execution methods accept context.Context for cancellation
//   - Compile-Time Registration: Tools can self-register via init() functions
//   - Middleware Pattern: Cross-cutting concerns are handled via composable middleware
//   - Functional Options: Flexible, backward-compatible executor configuration
//
// # Tool Interface
//
// The Tool interface is the fundamental building block:
//
//	type Tool interface {
//	    Name() string
//	    Description() string
//	    Execute(ctx context.Context, input *Input) (*Output, error)
//	    RequiresConfirmation(args map[string]any) bool
//	}
//
// Tools should:
//   - Return a unique, stable name that doesn't change between versions
//   - Check ctx.Done() before and during long-running operations
//   - Return meaningful errors with context
//   - Use the Input helper methods (GetParamString, GetParamInt, etc.) for type-safe access
//   - Implement RequiresConfirmation() to declare when user consent is needed
//
// # Registry
//
// The Registry provides thread-safe tool registration and discovery:
//
//	// Create a new registry
//	registry := NewRegistry()
//
//	// Register tools
//	registry.Register(&MyTool{})
//
//	// Look up tools
//	tool, err := registry.Get("mytool")
//
//	// List all tools
//	for _, info := range registry.List() {
//	    fmt.Printf("%s: %s\n", info.Name, info.Description)
//	}
//
// For compile-time registration, tools can use the global registry:
//
//	func init() {
//	    toolexec.Register(&MyTool{})
//	}
//
// The Register function panics on error, which is appropriate for init() functions.
//
// # Executor
//
// The Executor handles tool execution with various features:
//
//   - Security validation via SecurityPolicy (blacklist/path checking)
//   - User confirmation via ConfirmationHandler
//   - Timeout enforcement (30 seconds default)
//   - Panic recovery
//   - Middleware chain execution
//   - Synchronous and asynchronous execution modes
//   - Batch execution with concurrency control
//   - Output truncation (100KB default)
//
// Execution flow: security validation → confirmation → middleware → execution
//
// Create an executor with options:
//
//	executor := NewExecutor(registry,
//	    WithTimeout(60*time.Second),
//	    WithMaxConcurrent(4),
//	    WithSecurityPolicy(DefaultSecurityPolicy()),
//	    WithConfirmationHandler(&AutoApproveHandler{}),
//	    WithDefaultMiddleware(),
//	)
//
// Execute tools:
//
//	// Synchronous execution
//	output, err := executor.Execute(ctx, "mytool", input)
//
//	// Asynchronous execution
//	resultCh := executor.ExecuteAsync(ctx, "mytool", input)
//	result := <-resultCh
//
//	// Batch execution
//	executions := []ToolExecution{
//	    {ToolName: "tool1", Input: input1},
//	    {ToolName: "tool2", Input: input2},
//	}
//	results, err := executor.ExecuteMany(ctx, executions)
//
// # Middleware
//
// Middleware allows adding cross-cutting concerns to tool execution:
//
//	// Built-in middlewares
//	chain := NewMiddlewareChain(
//	    NewRecoveryMiddleware(true),   // Panic recovery with stack traces
//	    NewContextCheckMiddleware(),   // Early context cancellation detection
//	    NewInputValidationMiddleware(), // Input nil check
//	    NewTimingMiddleware(),          // Execution timing
//	)
//
//	// Or use the default chain
//	executor := NewExecutor(registry, WithDefaultMiddleware())
//
// Custom middleware can be created by implementing the Middleware interface:
//
//	type Middleware interface {
//	    Name() string
//	    Wrap(next ToolFunc) ToolFunc
//	}
//
// Or using the MiddlewareFunc adapter:
//
//	mw := NewMiddlewareFunc("logger", func(next ToolFunc) ToolFunc {
//	    return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
//	        log.Printf("Executing %s", toolName)
//	        output, err := next(ctx, toolName, input)
//	        log.Printf("Finished %s (error: %v)", toolName, err)
//	        return output, err
//	    }
//	})
//
// # Error Handling
//
// The package provides structured error types for different failure modes:
//
//   - ErrToolNotFound: Tool not registered in the registry
//   - ErrDuplicateTool: Attempting to register a tool that already exists
//   - ErrExecutionFailed: Tool execution failed
//   - ErrValidationFailed: Input validation failed
//   - ErrTimeout: Execution timed out
//   - ErrPanicRecovered: Panic occurred during execution
//   - ErrContextCancelled: Context was cancelled
//   - ErrUserDenied: User denied confirmation for tool execution
//   - ErrSecurityViolation: Tool execution blocked by security policy
//
// Check error types using errors.Is:
//
//	if errors.Is(err, toolexec.ErrToolNotFound) {
//	    // Handle tool not found
//	}
//
// Or use the helper functions:
//
//	if toolexec.IsTimeoutError(err) {
//	    // Handle timeout
//	}
//
// Specific error types provide additional context:
//
//	var timeoutErr *TimeoutError
//	if errors.As(err, &timeoutErr) {
//	    fmt.Printf("Tool %s timed out after %v\n", timeoutErr.ToolName, timeoutErr.Timeout)
//	}
//
// # Input and Output
//
// The Input type provides flexible parameter passing:
//
//	input := NewInput().
//	    WithName("my-request").
//	    WithParam("key", "value").
//	    WithParam("count", 42).
//	    WithData([]byte("raw data")).
//	    WithMetadata("request-id", "abc123")
//
//	// Type-safe access
//	key := input.GetParamString("key")     // "value"
//	count := input.GetParamInt("count")    // 42
//	flag := input.GetParamBool("enabled")  // false (default)
//
// The Output type provides flexible result returning:
//
//	output := NewOutput().
//	    WithData([]byte("result data")).
//	    WithResult("items", []string{"a", "b", "c"}).
//	    WithMetadata("cache-hit", "true").
//	    WithMessage("Operation completed")
//
// For failures:
//
//	output := NewFailedOutput("Operation failed: invalid input")
//
// # Output Truncation
//
// Large outputs can be truncated to prevent memory exhaustion:
//
//	// Truncate to default limit (100KB)
//	output := NewOutput().WithData(largeData).TruncateDefault()
//
//	// Truncate to custom limit
//	output := NewOutput().WithTruncatedData(largeData, 10*1024) // 10KB
//
//	// Check if output was truncated
//	if output.Truncated {
//	    fmt.Println("Output was truncated")
//	}
//
// # Security Policy
//
// SecurityPolicy validates tool executions before they run. The package provides
// built-in validators:
//
//   - BlacklistValidator: Blocks dangerous bash commands (rm -rf /, dd, mkfs)
//   - PathValidator: Blocks access to sensitive files (.env, .ssh/, *.pem)
//   - CompositeSecurityPolicy: Chains multiple validators together
//
// Example:
//
//	// Use default security policy (blacklist + path validation)
//	executor := NewExecutor(registry, WithSecurityPolicy(DefaultSecurityPolicy()))
//
//	// Custom blacklist
//	blacklist := NewBlacklistValidator("rm -rf", "dd if=", "mkfs")
//	executor := NewExecutor(registry, WithSecurityPolicy(blacklist))
//
//	// Handle security violations
//	_, err := executor.Execute(ctx, "bash", input)
//	if errors.Is(err, ErrSecurityViolation) {
//	    // Command was blocked by security policy
//	}
//
// # Confirmation Handler
//
// ConfirmationHandler requests user confirmation before executing dangerous tools.
// Tools declare when confirmation is needed via RequiresConfirmation():
//
//	type DangerousTool struct{}
//
//	func (t *DangerousTool) RequiresConfirmation(args map[string]any) bool {
//	    // Require confirmation for destructive operations
//	    cmd, _ := args["command"].(string)
//	    return strings.Contains(cmd, "delete")
//	}
//
// Built-in handlers:
//
//   - AutoApproveHandler: Automatically approves all requests (non-interactive)
//   - AutoDenyHandler: Automatically denies all requests (highly restricted)
//   - ConfirmationFunc: Adapter for using functions as handlers
//
// Example:
//
//	// Auto-approve in non-interactive mode
//	executor := NewExecutor(registry, WithConfirmationHandler(&AutoApproveHandler{}))
//
//	// Handle user denial
//	_, err := executor.Execute(ctx, "dangerous-tool", input)
//	if errors.Is(err, ErrUserDenied) {
//	    // User declined to confirm execution
//	}
//
// # Thread Safety
//
// The Registry is thread-safe using sync.RWMutex, optimized for read-heavy workloads.
// Multiple goroutines can safely:
//   - Register tools (exclusive lock)
//   - Get/List/Has/Count tools (shared lock)
//
// The Executor is also safe for concurrent use:
//   - Execute, ExecuteAsync, and ExecuteMany can be called concurrently
//   - Results are properly synchronized in batch operations
//
// # Best Practices
//
// When implementing tools:
//   - Always check ctx.Done() for long-running operations
//   - Return descriptive errors with context
//   - Use the comma-ok idiom for type assertions in parameters
//   - Don't panic; return errors instead
//
// When using the executor:
//   - Use timeouts to prevent hanging executions
//   - Use middleware for cross-cutting concerns
//   - Handle errors appropriately for each error type
//   - Clean up resources in deferred functions
//
// For production use:
//   - Enable panic recovery (default)
//   - Set appropriate concurrency limits for batch operations
//   - Use the logging middleware for observability
//   - Consider implementing custom middleware for metrics collection
//   - Enable security policy to block dangerous commands
//   - Configure confirmation handler for user consent on destructive operations
//   - Use output truncation to prevent memory exhaustion from large outputs
package toolexec
