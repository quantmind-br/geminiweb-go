package toolexec

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SearchTool searches for a pattern in files.
type SearchTool struct {
	maxFileBytes  int64
	maxOutputSize int
	defaultPath   string
}

// SearchToolOption configures a SearchTool.
type SearchToolOption func(*SearchTool)

// NewSearchTool creates a SearchTool with optional configuration.
func NewSearchTool(opts ...SearchToolOption) *SearchTool {
	tool := &SearchTool{
		maxFileBytes:  defaultMaxFileBytes,
		maxOutputSize: DefaultMaxOutputSize,
		defaultPath:   ".",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(tool)
		}
	}
	tool.maxFileBytes = normalizeMaxFileBytes(tool.maxFileBytes)
	tool.maxOutputSize = normalizeMaxOutputSize(tool.maxOutputSize)
	if strings.TrimSpace(tool.defaultPath) == "" {
		tool.defaultPath = "."
	}
	return tool
}

// WithSearchMaxFileBytes sets the maximum file size allowed for searching.
func WithSearchMaxFileBytes(limit int64) SearchToolOption {
	return func(t *SearchTool) {
		t.maxFileBytes = limit
	}
}

// WithSearchMaxOutputSize sets the maximum output size before truncation.
func WithSearchMaxOutputSize(limit int) SearchToolOption {
	return func(t *SearchTool) {
		t.maxOutputSize = limit
	}
}

// WithSearchDefaultPath sets the default search path.
func WithSearchDefaultPath(path string) SearchToolOption {
	return func(t *SearchTool) {
		t.defaultPath = path
	}
}

// Name returns the tool name.
func (t *SearchTool) Name() string {
	return "search"
}

// Description returns a human-readable description.
func (t *SearchTool) Description() string {
	return "Searches for a pattern in files"
}

// RequiresConfirmation returns false for searches.
func (t *SearchTool) RequiresConfirmation(args map[string]any) bool {
	return false
}

// Execute searches files for the given pattern.
func (t *SearchTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	args := argsFromInput(input)
	pattern, err := requireStringArg(t.Name(), args, "pattern")
	if err != nil {
		return nil, err
	}

	path := t.defaultPath
	if rawPath, ok := optionalStringArg(args, "path"); ok {
		path = rawPath
	}
	if strings.TrimSpace(path) == "" {
		path = t.defaultPath
	}

	matchType := "literal"
	if rawType, ok := optionalStringArg(args, "type"); ok {
		matchType = strings.ToLower(rawType)
	}

	var matcher func(string) bool
	switch matchType {
	case "literal":
		matcher = func(line string) bool {
			return strings.Contains(line, pattern)
		}
	case "regex", "regexp":
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, NewValidationErrorForField(t.Name(), "pattern", err.Error())
		}
		matcher = re.MatchString
	default:
		return nil, NewValidationErrorForField(t.Name(), "type", "must be 'literal' or 'regex'")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, NewExecutionErrorWithCause(t.Name(), err)
	}

	var buf bytes.Buffer
	truncated := false
	matches := 0
	files := 0
	skipped := 0

	appendLine := func(line string) bool {
		data := []byte(line)
		return appendBytesWithLimit(&buf, data, t.maxOutputSize)
	}

	if !info.IsDir() {
		if info.Size() > t.maxFileBytes {
			return nil, NewValidationErrorForField(t.Name(), "path", "file exceeds size limit")
		}
		fileMatches, err := t.searchFile(ctx, path, matcher, appendLine)
		if err != nil {
			if errors.Is(err, errSearchTruncated) {
				truncated = true
			} else {
				return nil, err
			}
		}
		if fileMatches > 0 {
			matches += fileMatches
			files++
		}
		return buildSearchOutput(&buf, truncated, matches, files, skipped), nil
	}

	err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > t.maxFileBytes {
			skipped++
			return nil
		}

		fileMatches, err := t.searchFile(ctx, current, matcher, appendLine)
		if err != nil {
			if errors.Is(err, errSearchTruncated) {
				truncated = true
				return errSearchTruncated
			}
			return err
		}
		if fileMatches > 0 {
			matches += fileMatches
			files++
		}
		return nil
	})
	if err != nil && !errors.Is(err, errSearchTruncated) {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if IsExecutionError(err) || IsValidationError(err) {
			return nil, err
		}
		return nil, NewExecutionErrorWithCause(t.Name(), err)
	}

	if errors.Is(err, errSearchTruncated) {
		truncated = true
	}

	return buildSearchOutput(&buf, truncated, matches, files, skipped), nil
}

var errSearchTruncated = errors.New("search output truncated")

func (t *SearchTool) searchFile(
	ctx context.Context,
	path string,
	matcher func(string) bool,
	appendLine func(string) bool,
) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, NewExecutionErrorWithCause(t.Name(), err)
	}
	defer func() { _ = file.Close() }()

	reader := bufio.NewReader(file)
	matches := 0
	lineNum := 0

	for {
		if ctx.Err() != nil {
			return matches, ctx.Err()
		}
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return matches, NewExecutionErrorWithCause(t.Name(), err)
		}
		if line == "" && err == io.EOF {
			break
		}
		lineNum++
		if matcher(line) {
			matches++
			trimmed := strings.TrimRight(line, "\r\n")
			if appendLine(filepath.Clean(path) + ":" + strconv.Itoa(lineNum) + ":" + trimmed + "\n") {
				return matches, errSearchTruncated
			}
		}
		if err == io.EOF {
			break
		}
	}

	return matches, nil
}

func buildSearchOutput(buf *bytes.Buffer, truncated bool, matches, files, skipped int) *Output {
	output := NewOutput().WithData(buf.Bytes())
	output.Truncated = truncated
	output.Result["matches"] = matches
	output.Result["files"] = files
	if skipped > 0 {
		output.Result["skipped"] = skipped
	}
	if matches == 0 && buf.Len() == 0 {
		output.Message = "no matches found"
		output.Data = []byte(output.Message)
	}
	return output
}
