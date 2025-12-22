package toolexec

import (
	"context"
	"os"
	"os/exec"
)

// BashTool executes shell commands via bash -c.
type BashTool struct {
	shell         string
	workingDir    string
	env           []string
	maxOutputSize int
}

// BashToolOption configures a BashTool.
type BashToolOption func(*BashTool)

// NewBashTool creates a BashTool with optional configuration.
func NewBashTool(opts ...BashToolOption) *BashTool {
	tool := &BashTool{
		shell:         "bash",
		maxOutputSize: DefaultMaxOutputSize,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(tool)
		}
	}
	if tool.shell == "" {
		tool.shell = "bash"
	}
	tool.maxOutputSize = normalizeMaxOutputSize(tool.maxOutputSize)
	return tool
}

// WithBashToolShell overrides the shell binary used for execution.
func WithBashToolShell(shell string) BashToolOption {
	return func(t *BashTool) {
		t.shell = shell
	}
}

// WithBashToolWorkingDir sets the working directory for command execution.
func WithBashToolWorkingDir(dir string) BashToolOption {
	return func(t *BashTool) {
		t.workingDir = dir
	}
}

// WithBashToolEnv sets extra environment variables for command execution.
func WithBashToolEnv(env []string) BashToolOption {
	return func(t *BashTool) {
		if len(env) == 0 {
			t.env = nil
			return
		}
		t.env = append([]string(nil), env...)
	}
}

// WithBashToolMaxOutputSize sets the maximum output size before truncation.
func WithBashToolMaxOutputSize(limit int) BashToolOption {
	return func(t *BashTool) {
		t.maxOutputSize = limit
	}
}

// Name returns the tool name.
func (t *BashTool) Name() string {
	return "bash"
}

// Description returns a human-readable description.
func (t *BashTool) Description() string {
	return "Executes shell commands via bash"
}

// RequiresConfirmation always returns true for bash execution.
func (t *BashTool) RequiresConfirmation(args map[string]any) bool {
	return true
}

// Execute runs the bash command and returns combined output.
func (t *BashTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	args := argsFromInput(input)
	command, err := requireStringArg(t.Name(), args, "command")
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, t.shell, "-c", command)
	if t.workingDir != "" {
		cmd.Dir = t.workingDir
	}
	if len(t.env) > 0 {
		cmd.Env = append(os.Environ(), t.env...)
	}

	data, err := cmd.CombinedOutput()
	output := NewOutput().WithTruncatedData(data, t.maxOutputSize)
	if err != nil {
		output.Success = false
		if ctx.Err() != nil {
			return output, ctx.Err()
		}
		return output, NewExecutionErrorWithCause(t.Name(), err)
	}

	return output, nil
}
