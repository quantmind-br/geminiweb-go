package toolexec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const defaultMaxFileBytes int64 = 10 * 1024 * 1024

func argsFromInput(input *Input) map[string]any {
	if input == nil || input.Params == nil {
		return map[string]any{}
	}
	return input.Params
}

func requireStringArg(toolName string, args map[string]any, field string) (string, error) {
	raw, ok := args[field]
	if !ok {
		return "", NewValidationErrorForField(toolName, field, "required")
	}
	value, ok := raw.(string)
	if !ok {
		return "", NewValidationErrorForField(toolName, field, "must be a string")
	}
	if strings.TrimSpace(value) == "" {
		return "", NewValidationErrorForField(toolName, field, "cannot be empty")
	}
	return value, nil
}

func optionalStringArg(args map[string]any, field string) (string, bool) {
	raw, ok := args[field]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}

func parseIntArg(value any) (int, bool, error) {
	if value == nil {
		return 0, false, nil
	}

	switch v := value.(type) {
	case int:
		return v, true, nil
	case int8:
		return int(v), true, nil
	case int16:
		return int(v), true, nil
	case int32:
		return int(v), true, nil
	case int64:
		return int(v), true, nil
	case float32:
		f := float64(v)
		if math.Trunc(f) != f {
			return 0, true, fmt.Errorf("expected integer, got %v", v)
		}
		return int(f), true, nil
	case float64:
		if math.Trunc(v) != v {
			return 0, true, fmt.Errorf("expected integer, got %v", v)
		}
		return int(v), true, nil
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true, nil
		}
		f, err := v.Float64()
		if err != nil {
			return 0, true, err
		}
		if math.Trunc(f) != f {
			return 0, true, fmt.Errorf("expected integer, got %v", v)
		}
		return int(f), true, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, false, nil
		}
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, true, err
		}
		return i, true, nil
	default:
		return 0, true, fmt.Errorf("expected integer, got %T", value)
	}
}

func normalizeMaxOutputSize(limit int) int {
	if limit <= 0 {
		return DefaultMaxOutputSize
	}
	return limit
}

func normalizeMaxFileBytes(limit int64) int64 {
	if limit <= 0 {
		return defaultMaxFileBytes
	}
	return limit
}

func appendBytesWithLimit(buf *bytes.Buffer, data []byte, limit int) bool {
	if limit <= 0 {
		_, _ = buf.Write(data)
		return false
	}

	remaining := limit - buf.Len()
	if remaining <= 0 {
		return true
	}
	if len(data) > remaining {
		_, _ = buf.Write(data[:remaining])
		return true
	}
	_, _ = buf.Write(data)
	return false
}
