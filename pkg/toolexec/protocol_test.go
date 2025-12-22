// Package toolexec provides a modular, extensible tool executor architecture.
// This file contains tests for the tool call protocol parsing.
package toolexec

import (
	"strings"
	"testing"
	"time"
)

// TestToolCall_Validate tests the ToolCall.Validate method.
func TestToolCall_Validate(t *testing.T) {
	tests := []struct {
		name      string
		call      ToolCall
		expectErr bool
	}{
		{
			name: "valid call",
			call: ToolCall{
				Name: "bash",
				Args: map[string]any{"command": "ls"},
			},
			expectErr: false,
		},
		{
			name: "valid call with reason",
			call: ToolCall{
				Name:   "bash",
				Args:   map[string]any{"command": "ls"},
				Reason: "List directory contents",
			},
			expectErr: false,
		},
		{
			name: "missing name",
			call: ToolCall{
				Args: map[string]any{"command": "ls"},
			},
			expectErr: true,
		},
		{
			name: "missing args",
			call: ToolCall{
				Name: "bash",
			},
			expectErr: true,
		},
		{
			name: "empty name",
			call: ToolCall{
				Name: "",
				Args: map[string]any{"command": "ls"},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call.Validate()
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestToolCall_ToInput tests the ToolCall.ToInput method.
func TestToolCall_ToInput(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		call := ToolCall{
			Name: "bash",
			Args: map[string]any{"command": "ls -la"},
		}

		input := call.ToInput()
		if input == nil {
			t.Fatal("ToInput returned nil")
		}
		if input.Name != "bash" {
			t.Errorf("expected Name 'bash', got %q", input.Name)
		}
		if input.GetParamString("command") != "ls -la" {
			t.Errorf("expected command 'ls -la', got %q", input.GetParamString("command"))
		}
	})

	t.Run("with reason", func(t *testing.T) {
		call := ToolCall{
			Name:   "file_read",
			Args:   map[string]any{"path": "/tmp/test.txt"},
			Reason: "Read test file",
		}

		input := call.ToInput()
		reason, ok := input.Metadata["reason"]
		if !ok {
			t.Error("expected reason in metadata")
		}
		if reason != "Read test file" {
			t.Errorf("expected reason 'Read test file', got %q", reason)
		}
	})

	t.Run("without reason", func(t *testing.T) {
		call := ToolCall{
			Name: "bash",
			Args: map[string]any{"command": "ls"},
		}

		input := call.ToInput()
		_, ok := input.Metadata["reason"]
		if ok {
			t.Error("expected no reason in metadata when not provided")
		}
	})
}

// TestParseToolCalls tests the ParseToolCalls function.
func TestParseToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		expectErr bool
	}{
		{
			name:      "no tool calls",
			input:     "Just regular text without any tool calls",
			wantCount: 0,
			expectErr: false,
		},
		{
			name: "single tool call",
			input: `Here's what I'll do:
` + "```tool\n" + `{"name": "bash", "args": {"command": "ls"}}
` + "```",
			wantCount: 1,
			expectErr: false,
		},
		{
			name: "multiple tool calls",
			input: `First:
` + "```tool\n" + `{"name": "bash", "args": {"command": "ls"}}
` + "```" + `
Then:
` + "```tool\n" + `{"name": "file_read", "args": {"path": "/tmp/test.txt"}}
` + "```",
			wantCount: 2,
			expectErr: false,
		},
		{
			name: "tool call with reason",
			input: "```tool\n" + `{"name": "bash", "args": {"command": "ls -la"}, "reason": "List files"}
` + "```",
			wantCount: 1,
			expectErr: false,
		},
		{
			name: "invalid json",
			input: "```tool\n" + `{invalid json}
` + "```",
			wantCount: 0,
			expectErr: true,
		},
		{
			name: "missing name",
			input: "```tool\n" + `{"args": {"command": "ls"}}
` + "```",
			wantCount: 0,
			expectErr: true,
		},
		{
			name: "missing args",
			input: "```tool\n" + `{"name": "bash"}
` + "```",
			wantCount: 0,
			expectErr: true,
		},
		{
			name:      "empty tool block",
			input:     "```tool\n\n```",
			wantCount: 0,
			expectErr: false, // Empty blocks don't match the regex, so no error is returned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := ParseToolCalls(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(calls) != tt.wantCount {
				t.Errorf("expected %d calls, got %d", tt.wantCount, len(calls))
			}
		})
	}
}

// TestParseToolCallsLenient tests the lenient parsing function.
func TestParseToolCallsLenient(t *testing.T) {
	t.Run("skips invalid json", func(t *testing.T) {
		input := "```tool\n" + `{invalid}
` + "```" + `
` + "```tool\n" + `{"name": "bash", "args": {"command": "ls"}}
` + "```"

		calls := ParseToolCallsLenient(input)
		if len(calls) != 1 {
			t.Errorf("expected 1 valid call, got %d", len(calls))
		}
	})

	t.Run("skips invalid calls", func(t *testing.T) {
		input := "```tool\n" + `{"name": "bash"}
` + "```" + `
` + "```tool\n" + `{"name": "bash", "args": {"command": "ls"}}
` + "```"

		calls := ParseToolCallsLenient(input)
		if len(calls) != 1 {
			t.Errorf("expected 1 valid call, got %d", len(calls))
		}
	})

	t.Run("returns empty for no valid calls", func(t *testing.T) {
		input := "```tool\n" + `{invalid}
` + "```"

		calls := ParseToolCallsLenient(input)
		if len(calls) != 0 {
			t.Errorf("expected 0 calls, got %d", len(calls))
		}
	})

	t.Run("returns empty for no calls", func(t *testing.T) {
		input := "Just text"
		calls := ParseToolCallsLenient(input)
		if len(calls) != 0 {
			t.Errorf("expected 0 calls, got %d", len(calls))
		}
	})
}

// TestHasToolCall tests the HasToolCall function.
func TestHasToolCall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "no tool call",
			input:    "Just text",
			expected: false,
		},
		{
			name:     "has tool call",
			input:    "```tool\n{}\n```",
			expected: true,
		},
		{
			name:     "code block but not tool",
			input:    "```python\nprint('hello')\n```",
			expected: false,
		},
		{
			name:     "multiple tool calls",
			input:    "```tool\n{}\n```\n```tool\n{}\n```",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasToolCall(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCountToolCalls tests the CountToolCalls function.
func TestCountToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "no tool calls",
			input:    "Just text",
			expected: 0,
		},
		{
			name:     "one tool call",
			input:    "```tool\n{}\n```",
			expected: 1,
		},
		{
			name:     "three tool calls",
			input:    "```tool\n{}\n```\n```tool\n{}\n```\n```tool\n{}\n```",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountToolCalls(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestToolCallResult tests the ToolCallResult type.
func TestToolCallResult(t *testing.T) {
	t.Run("from successful result", func(t *testing.T) {
		result := &Result{
			ToolName:  "bash",
			Output:    NewOutput().WithMessage("Command executed").WithData([]byte("file1 file2")),
			Error:     nil,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(100 * time.Millisecond),
			Duration:  100 * time.Millisecond,
		}

		tcr := NewToolCallResult(result)

		if tcr.ToolName != "bash" {
			t.Errorf("expected ToolName 'bash', got %q", tcr.ToolName)
		}
		if !tcr.Success {
			t.Error("expected Success to be true")
		}
		if tcr.Output != "file1 file2" {
			t.Errorf("expected Output 'file1 file2', got %q", tcr.Output)
		}
		if tcr.Error != "" {
			t.Errorf("expected no Error, got %q", tcr.Error)
		}
		if tcr.ExecutionTimeMs != 100 {
			t.Errorf("expected ExecutionTimeMs 100, got %d", tcr.ExecutionTimeMs)
		}
	})

	t.Run("from failed result", func(t *testing.T) {
		result := &Result{
			ToolName: "bash",
			Output:   nil,
			Error:    NewExecutionError("bash", "command failed"),
			Duration: 50 * time.Millisecond,
		}

		tcr := NewToolCallResult(result)

		if tcr.Success {
			t.Error("expected Success to be false")
		}
		if tcr.Error == "" {
			t.Error("expected Error to be set")
		}
	})

	t.Run("from result with message only", func(t *testing.T) {
		result := &Result{
			ToolName: "echo",
			Output:   NewOutput().WithMessage("Hello World"),
			Duration: 10 * time.Millisecond,
		}

		tcr := NewToolCallResult(result)

		if tcr.Output != "Hello World" {
			t.Errorf("expected Output 'Hello World', got %q", tcr.Output)
		}
	})
}

// TestToolCallResult_ToJSON tests JSON serialization.
func TestToolCallResult_ToJSON(t *testing.T) {
	tcr := &ToolCallResult{
		ToolName:        "bash",
		Success:         true,
		Output:          "output",
		ExecutionTimeMs: 100,
	}

	json, err := tcr.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(json, `"tool_name":"bash"`) {
		t.Errorf("JSON should contain tool_name: %s", json)
	}
	if !strings.Contains(json, `"success":true`) {
		t.Errorf("JSON should contain success: %s", json)
	}
}

// TestToolCallResult_FormatAsBlock tests block formatting.
func TestToolCallResult_FormatAsBlock(t *testing.T) {
	tcr := &ToolCallResult{
		ToolName: "bash",
		Success:  true,
		Output:   "file1 file2",
	}

	block := tcr.FormatAsBlock()

	if !strings.HasPrefix(block, "```result\n") {
		t.Error("block should start with ```result")
	}
	if !strings.HasSuffix(block, "\n```") {
		t.Error("block should end with ```")
	}
	if !strings.Contains(block, `"tool_name":"bash"`) {
		t.Error("block should contain tool_name")
	}
}

// TestParseToolCalls_ComplexScenarios tests complex parsing scenarios.
func TestParseToolCalls_ComplexScenarios(t *testing.T) {
	t.Run("nested json in args", func(t *testing.T) {
		input := "```tool\n" + `{"name": "api_call", "args": {"body": {"nested": {"deep": true}}}}
` + "```"

		calls, err := ParseToolCalls(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(calls))
		}

		body, ok := calls[0].Args["body"].(map[string]any)
		if !ok {
			t.Error("body should be a nested object")
		}
		nested, ok := body["nested"].(map[string]any)
		if !ok {
			t.Error("nested should be an object")
		}
		if nested["deep"] != true {
			t.Error("deep should be true")
		}
	})

	t.Run("array in args", func(t *testing.T) {
		input := "```tool\n" + `{"name": "multi_file", "args": {"files": ["a.txt", "b.txt"]}}
` + "```"

		calls, err := ParseToolCalls(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		files, ok := calls[0].Args["files"].([]any)
		if !ok {
			t.Error("files should be an array")
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d", len(files))
		}
	})

	t.Run("special characters in args", func(t *testing.T) {
		input := "```tool\n" + `{"name": "bash", "args": {"command": "echo \"hello\nworld\""}}
` + "```"

		calls, err := ParseToolCalls(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cmd := calls[0].Args["command"].(string)
		if !strings.Contains(cmd, "hello") {
			t.Error("command should contain 'hello'")
		}
	})

	t.Run("unicode in args", func(t *testing.T) {
		input := "```tool\n" + `{"name": "echo", "args": {"message": "Hello, World!"}}
` + "```"

		calls, err := ParseToolCalls(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := calls[0].Args["message"].(string)
		if msg != "Hello, World!" {
			t.Errorf("message should be 'Hello, World!', got %q", msg)
		}
	})
}
