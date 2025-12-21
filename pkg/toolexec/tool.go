// Package toolexec provides a modular, extensible tool executor architecture
// for registering, discovering, and executing various types of tools.
// It supports both synchronous and asynchronous execution patterns with
// proper context handling for cancellation and timeouts.
package toolexec

import (
	"context"
)

// Tool defines the interface that all executable tools must implement.
// Tools are the fundamental building blocks of the executor system.
// Each tool has a unique name, description, and an execution method
// that accepts context for cancellation support.
type Tool interface {
	// Name returns the unique identifier for this tool.
	// The name is used to register and lookup tools in the registry.
	// It should be stable and not change between versions.
	Name() string

	// Description returns a human-readable description of what this tool does.
	// This is used for documentation and discovery purposes.
	Description() string

	// Execute runs the tool with the given input and returns the output.
	// The context should be used for cancellation and deadline propagation.
	// Implementations must check ctx.Done() before and during long-running operations.
	// Returns an error if execution fails or is cancelled.
	Execute(ctx context.Context, input *Input) (*Output, error)
}

// Input represents the input data passed to a tool for execution.
// It provides a flexible structure for passing parameters and metadata.
type Input struct {
	// Name is an optional identifier for this input (useful for logging/tracing).
	Name string

	// Params holds the input parameters as key-value pairs.
	// Keys are parameter names, values can be any type.
	Params map[string]any

	// Data holds arbitrary input data (e.g., file contents, raw bytes).
	Data []byte

	// Metadata holds additional context information (e.g., request ID, user info).
	Metadata map[string]string
}

// NewInput creates a new Input with initialized maps.
func NewInput() *Input {
	return &Input{
		Params:   make(map[string]any),
		Metadata: make(map[string]string),
	}
}

// WithName sets the input name and returns the Input for chaining.
func (i *Input) WithName(name string) *Input {
	i.Name = name
	return i
}

// WithParam adds a parameter and returns the Input for chaining.
func (i *Input) WithParam(key string, value any) *Input {
	if i.Params == nil {
		i.Params = make(map[string]any)
	}
	i.Params[key] = value
	return i
}

// WithData sets the data and returns the Input for chaining.
func (i *Input) WithData(data []byte) *Input {
	i.Data = data
	return i
}

// WithMetadata adds a metadata entry and returns the Input for chaining.
func (i *Input) WithMetadata(key, value string) *Input {
	if i.Metadata == nil {
		i.Metadata = make(map[string]string)
	}
	i.Metadata[key] = value
	return i
}

// GetParam retrieves a parameter by key.
// Returns nil if the parameter does not exist.
func (i *Input) GetParam(key string) any {
	if i.Params == nil {
		return nil
	}
	return i.Params[key]
}

// GetParamString retrieves a string parameter by key.
// Returns empty string if the parameter does not exist or is not a string.
func (i *Input) GetParamString(key string) string {
	v := i.GetParam(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetParamInt retrieves an int parameter by key.
// Returns 0 if the parameter does not exist or is not an int.
func (i *Input) GetParamInt(key string) int {
	v := i.GetParam(key)
	if n, ok := v.(int); ok {
		return n
	}
	return 0
}

// GetParamBool retrieves a bool parameter by key.
// Returns false if the parameter does not exist or is not a bool.
func (i *Input) GetParamBool(key string) bool {
	v := i.GetParam(key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// Output represents the result of a tool execution.
// It provides a flexible structure for returning data and metadata.
type Output struct {
	// Success indicates whether the tool execution succeeded.
	Success bool

	// Data holds the primary output data.
	Data []byte

	// Result holds structured result data as key-value pairs.
	Result map[string]any

	// Metadata holds additional output context (e.g., execution time, resource usage).
	Metadata map[string]string

	// Message is an optional human-readable message describing the result.
	Message string
}

// NewOutput creates a new Output with initialized maps and Success set to true.
func NewOutput() *Output {
	return &Output{
		Success:  true,
		Result:   make(map[string]any),
		Metadata: make(map[string]string),
	}
}

// NewFailedOutput creates a new Output with Success set to false.
func NewFailedOutput(message string) *Output {
	return &Output{
		Success:  false,
		Result:   make(map[string]any),
		Metadata: make(map[string]string),
		Message:  message,
	}
}

// WithData sets the data and returns the Output for chaining.
func (o *Output) WithData(data []byte) *Output {
	o.Data = data
	return o
}

// WithResult adds a result entry and returns the Output for chaining.
func (o *Output) WithResult(key string, value any) *Output {
	if o.Result == nil {
		o.Result = make(map[string]any)
	}
	o.Result[key] = value
	return o
}

// WithMetadata adds a metadata entry and returns the Output for chaining.
func (o *Output) WithMetadata(key, value string) *Output {
	if o.Metadata == nil {
		o.Metadata = make(map[string]string)
	}
	o.Metadata[key] = value
	return o
}

// WithMessage sets the message and returns the Output for chaining.
func (o *Output) WithMessage(message string) *Output {
	o.Message = message
	return o
}

// GetResult retrieves a result value by key.
// Returns nil if the key does not exist.
func (o *Output) GetResult(key string) any {
	if o.Result == nil {
		return nil
	}
	return o.Result[key]
}

// GetResultString retrieves a string result value by key.
// Returns empty string if the key does not exist or is not a string.
func (o *Output) GetResultString(key string) string {
	v := o.GetResult(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ToolInfo provides metadata about a registered tool.
// This is used for tool discovery and documentation.
type ToolInfo struct {
	// Name is the unique identifier for the tool.
	Name string

	// Description is a human-readable description of the tool.
	Description string
}

// ToolInfoFromTool creates a ToolInfo from a Tool interface.
func ToolInfoFromTool(t Tool) ToolInfo {
	return ToolInfo{
		Name:        t.Name(),
		Description: t.Description(),
	}
}
