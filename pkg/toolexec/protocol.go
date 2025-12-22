// Package toolexec provides a modular, extensible tool executor architecture.
// This file defines the tool call protocol for parsing tool invocations from
// AI responses. AI models generate tool calls in fenced code blocks that can
// be extracted and executed.
package toolexec

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ToolCall represents a parsed tool invocation from an AI response.
// Tool calls are typically embedded in ```tool code blocks in the response.
//
// Example AI response:
//
//	```tool
//	{"name": "bash", "args": {"command": "ls -la"}, "reason": "List directory contents"}
//	```
type ToolCall struct {
	// Name is the name of the tool to execute (required).
	Name string `json:"name"`

	// Args contains the arguments to pass to the tool (required).
	// This maps directly to the tool's Input.Params.
	Args map[string]any `json:"args"`

	// Reason is an optional human-readable explanation of why
	// the tool is being called. Useful for logging and debugging.
	Reason string `json:"reason,omitempty"`
}

// Validate checks if the ToolCall has required fields.
func (tc *ToolCall) Validate() error {
	if tc.Name == "" {
		return fmt.Errorf("tool call missing required field: name")
	}
	if tc.Args == nil {
		return fmt.Errorf("tool call missing required field: args")
	}
	return nil
}

// ToInput converts the ToolCall's Args to an Input for execution.
func (tc *ToolCall) ToInput() *Input {
	input := NewInput()
	input.Params = tc.Args
	input.Name = tc.Name
	if tc.Reason != "" {
		input.Metadata["reason"] = tc.Reason
	}
	return input
}

// toolBlockRegex matches ```tool ... ```, ```json ... ```, or unlabeled fenced code blocks.
// Uses non-greedy matching (.+?) to handle multiple blocks correctly.
// The (?s) flag allows . to match newlines within the block.
var toolBlockRegex = regexp.MustCompile("(?is)```\\s*(?:tool|json)?\\s*\\n(.+?)\\n```")

// ParseToolCalls extracts and parses all tool calls from a text string.
// It finds all ```tool code blocks and parses their JSON content.
//
// Example input:
//
//	Here's what I'll do:
//	```tool
//	{"name": "bash", "args": {"command": "ls"}}
//	```
//	And then:
//	```tool
//	{"name": "file_read", "args": {"path": "/tmp/test.txt"}}
//	```
//
// Returns a slice of ToolCall structs, one for each valid tool block found.
// Returns an error if any tool block contains invalid JSON.
func ParseToolCalls(text string) ([]ToolCall, error) {
	matches := toolBlockRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []ToolCall{}, nil
	}

	calls := make([]ToolCall, 0, len(matches))
	for i, match := range matches {
		if len(match) < 2 {
			continue
		}

		jsonContent := match[1]
		var call ToolCall
		if err := json.Unmarshal([]byte(jsonContent), &call); err != nil {
			return nil, fmt.Errorf("failed to parse tool call %d: %w", i+1, err)
		}

		// Validate the parsed call
		if err := call.Validate(); err != nil {
			return nil, fmt.Errorf("invalid tool call %d: %w", i+1, err)
		}

		calls = append(calls, call)
	}

	return calls, nil
}

// ParseToolCallsLenient extracts tool calls, skipping invalid blocks.
// Unlike ParseToolCalls, this function does not return an error for
// malformed tool blocks - it simply skips them.
//
// This is useful for streaming scenarios where partial blocks may be
// present in the text.
func ParseToolCallsLenient(text string) []ToolCall {
	matches := toolBlockRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []ToolCall{}
	}

	calls := make([]ToolCall, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		jsonContent := match[1]
		var call ToolCall
		if err := json.Unmarshal([]byte(jsonContent), &call); err != nil {
			continue // Skip invalid JSON
		}

		if err := call.Validate(); err != nil {
			continue // Skip invalid calls
		}

		calls = append(calls, call)
	}

	return calls
}

// ExtractToolCallsLenient extracts tool calls and returns the cleaned text.
// Tool blocks that parse successfully are removed from the returned text.
// Invalid tool blocks are preserved in the text.
func ExtractToolCallsLenient(text string) ([]ToolCall, string) {
	matches := toolBlockRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return []ToolCall{}, strings.TrimSpace(text)
	}

	calls := make([]ToolCall, 0, len(matches))
	var clean strings.Builder
	last := 0

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		start, end := match[0], match[1]
		jsonStart, jsonEnd := match[2], match[3]
		jsonContent := text[jsonStart:jsonEnd]

		var call ToolCall
		if err := json.Unmarshal([]byte(jsonContent), &call); err == nil {
			if err := call.Validate(); err == nil {
				calls = append(calls, call)
				clean.WriteString(text[last:start])
				last = end
				continue
			}
		}

		// Keep invalid block in the cleaned text.
		clean.WriteString(text[last:end])
		last = end
	}

	clean.WriteString(text[last:])
	return calls, strings.TrimSpace(clean.String())
}

// HasToolCall checks if the text contains at least one tool call block.
// This is a quick check that doesn't fully parse the JSON.
func HasToolCall(text string) bool {
	return toolBlockRegex.MatchString(text)
}

// CountToolCalls returns the number of tool call blocks in the text.
// This counts all ```tool blocks, even if they contain invalid JSON.
func CountToolCalls(text string) int {
	return len(toolBlockRegex.FindAllStringIndex(text, -1))
}

// ToolCallResult represents the result of executing a tool call.
// This is used for formatting results back into a format that AI can understand.
type ToolCallResult struct {
	// ToolName is the name of the tool that was executed.
	ToolName string `json:"tool_name"`

	// Success indicates whether the tool execution succeeded.
	Success bool `json:"success"`

	// Output is the tool's output as a string.
	Output string `json:"output"`

	// Error contains the error message if execution failed.
	Error string `json:"error,omitempty"`

	// Truncated indicates if the output was truncated.
	Truncated bool `json:"truncated,omitempty"`

	// ExecutionTimeMs is the execution time in milliseconds.
	ExecutionTimeMs int64 `json:"execution_time_ms,omitempty"`
}

// NewToolCallResult creates a ToolCallResult from a Result.
func NewToolCallResult(result *Result) *ToolCallResult {
	tcr := &ToolCallResult{
		ToolName:        result.ToolName,
		Success:         result.Error == nil,
		ExecutionTimeMs: result.Duration.Milliseconds(),
	}

	if result.Error != nil {
		tcr.Error = result.Error.Error()
	}

	if result.Output != nil {
		tcr.Output = result.Output.Message
		if result.Output.Data != nil {
			// If there's data, use it as the output
			tcr.Output = string(result.Output.Data)
		}
		tcr.Truncated = result.Output.Truncated
	}

	return tcr
}

// ToJSON returns the result as a JSON string.
func (r *ToolCallResult) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool call result: %w", err)
	}
	return string(data), nil
}

// FormatAsBlock formats the result as a ```result block for AI consumption.
func (r *ToolCallResult) FormatAsBlock() string {
	jsonStr, err := r.ToJSON()
	if err != nil {
		return fmt.Sprintf("```result\n{\"error\": \"failed to format result: %s\"}\n```", err.Error())
	}
	return fmt.Sprintf("```result\n%s\n```", jsonStr)
}
