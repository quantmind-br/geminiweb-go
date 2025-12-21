package toolexec_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diogogmt/geminiweb-go/pkg/toolexec"
)

// GreetingTool is a simple tool that returns a greeting message.
type GreetingTool struct{}

func (t *GreetingTool) Name() string        { return "greeting" }
func (t *GreetingTool) Description() string { return "Returns a greeting message" }

func (t *GreetingTool) Execute(ctx context.Context, input *toolexec.Input) (*toolexec.Output, error) {
	// Check context before processing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get name parameter with type-safe access
	name := input.GetParamString("name")
	if name == "" {
		name = "World"
	}

	return toolexec.NewOutput().
		WithData([]byte("Hello, " + name + "!")).
		WithMessage("Greeting generated"), nil
}

// CalculatorTool performs basic arithmetic operations.
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string        { return "calculator" }
func (t *CalculatorTool) Description() string { return "Performs arithmetic operations" }

func (t *CalculatorTool) Execute(ctx context.Context, input *toolexec.Input) (*toolexec.Output, error) {
	a := input.GetParamInt("a")
	b := input.GetParamInt("b")
	op := input.GetParamString("operation")

	var result int
	switch op {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}

	return toolexec.NewOutput().
		WithResult("value", result).
		WithMessage(fmt.Sprintf("%d %s %d = %d", a, op, b, result)), nil
}

// SlowTool simulates a slow operation for timeout testing.
type SlowTool struct {
	duration time.Duration
}

func (t *SlowTool) Name() string        { return "slow" }
func (t *SlowTool) Description() string { return "Simulates a slow operation" }

func (t *SlowTool) Execute(ctx context.Context, input *toolexec.Input) (*toolexec.Output, error) {
	select {
	case <-time.After(t.duration):
		return toolexec.NewOutput().WithMessage("Slow operation completed"), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Example demonstrates basic tool registration and execution.
func Example() {
	// Create a registry and register tools
	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	// Create an executor
	executor := toolexec.NewExecutor(registry)

	// Execute the tool
	input := toolexec.NewInput().WithParam("name", "Gopher")
	output, err := executor.Execute(context.Background(), "greeting", input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(output.Data))
	// Output: Hello, Gopher!
}

// Example_registry demonstrates registry operations.
func Example_registry() {
	registry := toolexec.NewRegistry()

	// Register tools
	_ = registry.Register(&GreetingTool{})
	_ = registry.Register(&CalculatorTool{})

	// Check if a tool exists
	fmt.Println("Has greeting:", registry.Has("greeting"))
	fmt.Println("Has unknown:", registry.Has("unknown"))

	// Count registered tools
	fmt.Println("Tool count:", registry.Count())

	// List all tools (sorted alphabetically)
	fmt.Println("Registered tools:")
	for _, info := range registry.List() {
		fmt.Printf("  - %s: %s\n", info.Name, info.Description)
	}
	// Output:
	// Has greeting: true
	// Has unknown: false
	// Tool count: 2
	// Registered tools:
	//   - calculator: Performs arithmetic operations
	//   - greeting: Returns a greeting message
}

// Example_input demonstrates input creation and parameter access.
func Example_input() {
	// Create input with fluent API
	input := toolexec.NewInput().
		WithName("test-input").
		WithParam("name", "Alice").
		WithParam("count", 42).
		WithParam("enabled", true).
		WithData([]byte("raw data")).
		WithMetadata("request-id", "abc123")

	// Type-safe parameter access
	fmt.Println("Name:", input.GetParamString("name"))
	fmt.Println("Count:", input.GetParamInt("count"))
	fmt.Println("Enabled:", input.GetParamBool("enabled"))

	// Missing parameters return zero values
	fmt.Println("Missing string:", input.GetParamString("missing"))
	fmt.Println("Missing int:", input.GetParamInt("missing"))
	fmt.Println("Missing bool:", input.GetParamBool("missing"))
	// Output:
	// Name: Alice
	// Count: 42
	// Enabled: true
	// Missing string:
	// Missing int: 0
	// Missing bool: false
}

// Example_output demonstrates output creation.
func Example_output() {
	// Create successful output
	output := toolexec.NewOutput().
		WithData([]byte("result")).
		WithResult("items", []string{"a", "b", "c"}).
		WithMetadata("cache-hit", "true").
		WithMessage("Operation completed")

	fmt.Println("Success:", output.Success)
	fmt.Println("Message:", output.Message)
	fmt.Println("Data:", string(output.Data))

	// Create failed output
	failed := toolexec.NewFailedOutput("Something went wrong")
	fmt.Println("Failed success:", failed.Success)
	fmt.Println("Failed message:", failed.Message)
	// Output:
	// Success: true
	// Message: Operation completed
	// Data: result
	// Failed success: false
	// Failed message: Something went wrong
}

// Example_executor demonstrates executor configuration and execution.
func Example_executor() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&CalculatorTool{})

	// Create executor with options
	executor := toolexec.NewExecutor(registry,
		toolexec.WithTimeout(5*time.Second),
		toolexec.WithMaxConcurrent(2),
	)

	// Execute a calculation
	input := toolexec.NewInput().
		WithParam("a", 10).
		WithParam("b", 5).
		WithParam("operation", "multiply")

	output, err := executor.Execute(context.Background(), "calculator", input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Message:", output.Message)
	// Output: Message: 10 multiply 5 = 50
}

// Example_asyncExecution demonstrates asynchronous tool execution.
func Example_asyncExecution() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	executor := toolexec.NewExecutor(registry)

	// Start async execution
	input := toolexec.NewInput().WithParam("name", "Async")
	resultCh := executor.ExecuteAsync(context.Background(), "greeting", input)

	// Do other work here...
	fmt.Println("Waiting for result...")

	// Get the result
	result := <-resultCh
	if result.Error != nil {
		fmt.Println("Error:", result.Error)
		return
	}

	fmt.Println(string(result.Output.Data))
	// Output:
	// Waiting for result...
	// Hello, Async!
}

// Example_batchExecution demonstrates batch execution of multiple tools.
func Example_batchExecution() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&CalculatorTool{})

	// Allow parallel execution
	executor := toolexec.NewExecutor(registry,
		toolexec.WithMaxConcurrent(3),
	)

	// Define batch operations
	executions := []toolexec.ToolExecution{
		{
			ToolName: "calculator",
			Input: toolexec.NewInput().
				WithParam("a", 1).
				WithParam("b", 2).
				WithParam("operation", "add"),
		},
		{
			ToolName: "calculator",
			Input: toolexec.NewInput().
				WithParam("a", 10).
				WithParam("b", 3).
				WithParam("operation", "subtract"),
		},
		{
			ToolName: "calculator",
			Input: toolexec.NewInput().
				WithParam("a", 4).
				WithParam("b", 5).
				WithParam("operation", "multiply"),
		},
	}

	results, err := executor.ExecuteMany(context.Background(), executions)
	if err != nil {
		fmt.Println("Batch error:", err)
	}

	// Results are in the same order as inputs
	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("Operation %d: error - %v\n", i+1, result.Error)
		} else {
			fmt.Printf("Operation %d: %s\n", i+1, result.Output.Message)
		}
	}
	// Output:
	// Operation 1: 1 add 2 = 3
	// Operation 2: 10 subtract 3 = 7
	// Operation 3: 4 multiply 5 = 20
}

// Example_middleware demonstrates middleware usage.
func Example_middleware() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	// Create executor with middleware
	executor := toolexec.NewExecutor(registry,
		toolexec.WithMiddleware(toolexec.NewTimingMiddleware()),
		toolexec.WithMiddleware(toolexec.NewInputValidationMiddleware()),
	)

	// Execute - middleware adds timing metadata
	input := toolexec.NewInput().WithParam("name", "Middleware")
	output, err := executor.Execute(context.Background(), "greeting", input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(output.Data))
	// Timing metadata is available
	if _, hasTime := output.Metadata["execution_time_ms"]; hasTime {
		fmt.Println("Timing metadata: present")
	}
	// Output:
	// Hello, Middleware!
	// Timing metadata: present
}

// Example_customMiddleware demonstrates creating custom middleware.
func Example_customMiddleware() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	// Create custom logging middleware using MiddlewareFunc
	loggingMw := toolexec.NewMiddlewareFunc("custom-logger",
		func(next toolexec.ToolFunc) toolexec.ToolFunc {
			return func(ctx context.Context, toolName string, input *toolexec.Input) (*toolexec.Output, error) {
				fmt.Printf(">>> Executing: %s\n", toolName)
				output, err := next(ctx, toolName, input)
				if err != nil {
					fmt.Printf("<<< Error: %v\n", err)
				} else {
					fmt.Printf("<<< Success: %s\n", output.Message)
				}
				return output, err
			}
		})

	executor := toolexec.NewExecutor(registry,
		toolexec.WithMiddleware(loggingMw),
	)

	input := toolexec.NewInput().WithParam("name", "Custom")
	_, _ = executor.Execute(context.Background(), "greeting", input)
	// Output:
	// >>> Executing: greeting
	// <<< Success: Greeting generated
}

// Example_defaultMiddleware demonstrates using the default middleware chain.
func Example_defaultMiddleware() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	// Use default middleware (recovery, context check, validation, timing)
	executor := toolexec.NewExecutor(registry,
		toolexec.WithDefaultMiddleware(),
	)

	input := toolexec.NewInput().WithParam("name", "Default")
	output, err := executor.Execute(context.Background(), "greeting", input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(output.Data))
	// Output: Hello, Default!
}

// Example_errorHandling demonstrates error handling patterns.
func Example_errorHandling() {
	registry := toolexec.NewRegistry()
	executor := toolexec.NewExecutor(registry)

	// Try to execute a non-existent tool
	_, err := executor.Execute(context.Background(), "nonexistent", toolexec.NewInput())
	if err != nil {
		// Check error type
		if errors.Is(err, toolexec.ErrToolNotFound) {
			fmt.Println("Tool not found (errors.Is)")
		}

		// Or use helper function
		if toolexec.IsToolNotFoundError(err) {
			fmt.Println("Tool not found (helper)")
		}

		// Extract tool name from error
		toolName := toolexec.GetToolName(err)
		fmt.Println("Tool name:", toolName)
	}
	// Output:
	// Tool not found (errors.Is)
	// Tool not found (helper)
	// Tool name: nonexistent
}

// Example_timeout demonstrates timeout handling.
func Example_timeout() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&SlowTool{duration: 5 * time.Second})

	// Create executor with short timeout
	executor := toolexec.NewExecutor(registry,
		toolexec.WithTimeout(100*time.Millisecond),
	)

	_, err := executor.Execute(context.Background(), "slow", toolexec.NewInput())
	if err != nil {
		if toolexec.IsTimeoutError(err) {
			fmt.Println("Operation timed out")
		}

		// Get detailed timeout info
		var timeoutErr *toolexec.TimeoutError
		if errors.As(err, &timeoutErr) {
			fmt.Printf("Tool '%s' timed out\n", timeoutErr.ToolName)
		}
	}
	// Output:
	// Operation timed out
	// Tool 'slow' timed out
}

// Example_contextCancellation demonstrates context cancellation handling.
func Example_contextCancellation() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&SlowTool{duration: 5 * time.Second})

	executor := toolexec.NewExecutor(registry,
		toolexec.WithNoTimeout(), // Rely on context cancellation
	)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := executor.Execute(ctx, "slow", toolexec.NewInput())
	if err != nil {
		fmt.Println("Execution cancelled")
	}
	// Output: Execution cancelled
}

// Example_globalRegistry demonstrates using the global registry.
func Example_globalRegistry() {
	// Note: In a real application, you would register tools in init() functions
	// like this:
	//
	//   func init() {
	//       toolexec.Register(&MyTool{})
	//   }
	//
	// For this example, we'll use a fresh registry to avoid global state issues.

	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	// Use DefaultRegistry() to access the global registry
	// Here we use our local registry for demonstration
	executor := toolexec.NewExecutor(registry)

	output, _ := executor.Execute(context.Background(), "greeting",
		toolexec.NewInput().WithParam("name", "Global"))

	fmt.Println(string(output.Data))
	// Output: Hello, Global!
}

// Example_result demonstrates working with async Result.
func Example_result() {
	registry := toolexec.NewRegistry()
	_ = registry.Register(&GreetingTool{})

	executor := toolexec.NewExecutor(registry)
	resultCh := executor.ExecuteAsync(context.Background(), "greeting",
		toolexec.NewInput().WithParam("name", "Result"))

	result := <-resultCh

	// Check success
	if result.IsSuccess() {
		fmt.Println("Tool executed successfully")
		fmt.Println("Tool name:", result.ToolName)
		fmt.Println("Output:", string(result.Output.Data))
	} else {
		fmt.Println("Tool failed:", result.Error)
	}

	// Duration is available for timing analysis
	if result.Duration > 0 {
		fmt.Println("Duration: recorded")
	}
	// Output:
	// Tool executed successfully
	// Tool name: greeting
	// Output: Hello, Result!
	// Duration: recorded
}

// Example_middlewareChain demonstrates building a middleware chain.
func Example_middlewareChain() {
	// Build a chain manually
	chain := toolexec.NewMiddlewareChain()
	chain.Add(toolexec.NewRecoveryMiddleware(true))
	chain.Add(toolexec.NewContextCheckMiddleware())
	chain.Add(toolexec.NewInputValidationMiddleware())
	chain.Add(toolexec.NewTimingMiddleware())

	fmt.Println("Chain length:", chain.Len())

	// Or create with middlewares directly
	chain2 := toolexec.ChainMiddleware(
		toolexec.NewRecoveryMiddleware(true),
		toolexec.NewTimingMiddleware(),
	)

	fmt.Println("Chain2 length:", chain2.Len())

	// Or use the default chain
	defaultChain := toolexec.DefaultMiddlewareChain()
	fmt.Println("Default chain length:", defaultChain.Len())
	// Output:
	// Chain length: 4
	// Chain2 length: 2
	// Default chain length: 4
}

// Example_executorConfig demonstrates inspecting executor configuration.
func Example_executorConfig() {
	registry := toolexec.NewRegistry()

	executor := toolexec.NewExecutor(registry,
		toolexec.WithTimeout(45*time.Second),
		toolexec.WithMaxConcurrent(8),
		toolexec.WithDefaultMiddleware(),
	)

	// Inspect configuration
	config := executor.Config()

	fmt.Println("Timeout:", config.Timeout)
	fmt.Println("Max concurrent:", config.MaxConcurrent)
	fmt.Println("Recover panics:", config.RecoverPanics)
	fmt.Println("Has middleware:", config.HasMiddleware)
	fmt.Println("Middleware count:", config.MiddlewareCount)
	// Output:
	// Timeout: 45s
	// Max concurrent: 8
	// Recover panics: true
	// Has middleware: true
	// Middleware count: 4
}
