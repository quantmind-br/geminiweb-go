package toolexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileWriteTool writes content to disk.
type FileWriteTool struct {
	maxBytes   int64
	createDirs bool
	filePerm   os.FileMode
	dirPerm    os.FileMode
}

// FileWriteToolOption configures a FileWriteTool.
type FileWriteToolOption func(*FileWriteTool)

// NewFileWriteTool creates a FileWriteTool with optional configuration.
func NewFileWriteTool(opts ...FileWriteToolOption) *FileWriteTool {
	tool := &FileWriteTool{
		maxBytes:   defaultMaxFileBytes,
		createDirs: true,
		filePerm:   0o644,
		dirPerm:    0o755,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(tool)
		}
	}
	tool.maxBytes = normalizeMaxFileBytes(tool.maxBytes)
	return tool
}

// WithFileWriteMaxBytes sets the maximum file size allowed.
func WithFileWriteMaxBytes(limit int64) FileWriteToolOption {
	return func(t *FileWriteTool) {
		t.maxBytes = limit
	}
}

// WithFileWriteCreateDirs toggles parent directory creation.
func WithFileWriteCreateDirs(enabled bool) FileWriteToolOption {
	return func(t *FileWriteTool) {
		t.createDirs = enabled
	}
}

// WithFileWriteFilePerm sets the file permissions for created files.
func WithFileWriteFilePerm(perm os.FileMode) FileWriteToolOption {
	return func(t *FileWriteTool) {
		t.filePerm = perm
	}
}

// WithFileWriteDirPerm sets the permissions for created directories.
func WithFileWriteDirPerm(perm os.FileMode) FileWriteToolOption {
	return func(t *FileWriteTool) {
		t.dirPerm = perm
	}
}

// Name returns the tool name.
func (t *FileWriteTool) Name() string {
	return "file_write"
}

// Description returns a human-readable description.
func (t *FileWriteTool) Description() string {
	return "Writes content to disk"
}

// RequiresConfirmation always returns true for writes.
func (t *FileWriteTool) RequiresConfirmation(args map[string]any) bool {
	return true
}

// Execute writes the content to the target path.
func (t *FileWriteTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	args := argsFromInput(input)
	path, err := requireStringArg(t.Name(), args, "path")
	if err != nil {
		return nil, err
	}
	content, err := requireStringArg(t.Name(), args, "content")
	if err != nil {
		return nil, err
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	data := []byte(content)
	if int64(len(data)) > t.maxBytes {
		return nil, NewValidationErrorForField(t.Name(), "content", "content exceeds size limit")
	}

	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		return nil, NewValidationErrorForField(t.Name(), "path", "path is a directory")
	}

	if t.createDirs {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, t.dirPerm); err != nil {
				return nil, NewExecutionErrorWithCause(t.Name(), err)
			}
		}
	}

	if err := os.WriteFile(path, data, t.filePerm); err != nil {
		return nil, NewExecutionErrorWithCause(t.Name(), err)
	}

	return NewOutput().WithMessage(
		fmt.Sprintf("wrote %d bytes to %s", len(data), path),
	), nil
}
