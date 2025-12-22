package toolexec

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
)

// FileReadTool reads file contents from disk.
type FileReadTool struct {
	maxBytes      int64
	maxOutputSize int
}

// FileReadToolOption configures a FileReadTool.
type FileReadToolOption func(*FileReadTool)

// NewFileReadTool creates a FileReadTool with optional configuration.
func NewFileReadTool(opts ...FileReadToolOption) *FileReadTool {
	tool := &FileReadTool{
		maxBytes:      defaultMaxFileBytes,
		maxOutputSize: DefaultMaxOutputSize,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(tool)
		}
	}
	tool.maxBytes = normalizeMaxFileBytes(tool.maxBytes)
	tool.maxOutputSize = normalizeMaxOutputSize(tool.maxOutputSize)
	return tool
}

// WithFileReadMaxBytes sets the maximum file size allowed.
func WithFileReadMaxBytes(limit int64) FileReadToolOption {
	return func(t *FileReadTool) {
		t.maxBytes = limit
	}
}

// WithFileReadMaxOutputSize sets the maximum output size before truncation.
func WithFileReadMaxOutputSize(limit int) FileReadToolOption {
	return func(t *FileReadTool) {
		t.maxOutputSize = limit
	}
}

// Name returns the tool name.
func (t *FileReadTool) Name() string {
	return "file_read"
}

// Description returns a human-readable description.
func (t *FileReadTool) Description() string {
	return "Reads file contents from disk"
}

// RequiresConfirmation returns false for reads.
func (t *FileReadTool) RequiresConfirmation(args map[string]any) bool {
	return false
}

// Execute reads the file and returns its contents.
func (t *FileReadTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	args := argsFromInput(input)
	path, err := requireStringArg(t.Name(), args, "path")
	if err != nil {
		return nil, err
	}

	lines, hasLines, err := parseIntArg(args["lines"])
	if err != nil {
		return nil, NewValidationErrorForField(t.Name(), "lines", err.Error())
	}
	if hasLines && lines < 0 {
		return nil, NewValidationErrorForField(t.Name(), "lines", "must be >= 0")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, NewExecutionErrorWithCause(t.Name(), err)
	}
	if info.IsDir() {
		return nil, NewValidationErrorForField(t.Name(), "path", "path is a directory")
	}
	if info.Size() > t.maxBytes {
		return nil, NewValidationErrorForField(t.Name(), "path", "file exceeds size limit")
	}

	if hasLines && lines > 0 {
		return t.readLines(ctx, path, lines)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewExecutionErrorWithCause(t.Name(), err)
	}

	return NewOutput().WithTruncatedData(data, t.maxOutputSize), nil
}

func (t *FileReadTool) readLines(ctx context.Context, path string, maxLines int) (*Output, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, NewExecutionErrorWithCause(t.Name(), err)
	}
	defer func() { _ = file.Close() }()

	reader := bufio.NewReader(file)
	var buf bytes.Buffer
	truncated := false

	for line := 0; line < maxLines; line++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		chunk, err := reader.ReadBytes('\n')
		if len(chunk) > 0 {
			if appendBytesWithLimit(&buf, chunk, t.maxOutputSize) {
				truncated = true
				break
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, NewExecutionErrorWithCause(t.Name(), err)
		}
	}

	output := NewOutput().WithData(buf.Bytes())
	output.Truncated = truncated
	return output, nil
}
