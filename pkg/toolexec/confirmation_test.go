// Package toolexec provides a modular, extensible tool executor architecture.
// This file contains tests for the confirmation handler components.
package toolexec

import (
	"context"
	"errors"
	"testing"
)

// TestAutoApproveHandler tests the AutoApproveHandler.
func TestAutoApproveHandler(t *testing.T) {
	handler := &AutoApproveHandler{}
	tool := NewMockTool("test", "A test tool")

	approved, err := handler.RequestConfirmation(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("AutoApproveHandler should always approve")
	}

	// Test with args
	approved, err = handler.RequestConfirmation(context.Background(), tool, map[string]any{"key": "value"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("AutoApproveHandler should always approve even with args")
	}
}

// TestAutoDenyHandler tests the AutoDenyHandler.
func TestAutoDenyHandler(t *testing.T) {
	handler := &AutoDenyHandler{}
	tool := NewMockTool("test", "A test tool")

	approved, err := handler.RequestConfirmation(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if approved {
		t.Error("AutoDenyHandler should always deny")
	}

	// Test with args
	approved, err = handler.RequestConfirmation(context.Background(), tool, map[string]any{"key": "value"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if approved {
		t.Error("AutoDenyHandler should always deny even with args")
	}
}

// TestConfirmationFunc tests the ConfirmationFunc adapter.
func TestConfirmationFunc(t *testing.T) {
	t.Run("approving function", func(t *testing.T) {
		handler := ConfirmationFunc(func(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
			return true, nil
		})

		approved, err := handler.RequestConfirmation(context.Background(), nil, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !approved {
			t.Error("expected approval")
		}
	})

	t.Run("denying function", func(t *testing.T) {
		handler := ConfirmationFunc(func(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
			return false, nil
		})

		approved, err := handler.RequestConfirmation(context.Background(), nil, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if approved {
			t.Error("expected denial")
		}
	})

	t.Run("error returning function", func(t *testing.T) {
		expectedErr := errors.New("confirmation error")
		handler := ConfirmationFunc(func(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
			return false, expectedErr
		})

		approved, err := handler.RequestConfirmation(context.Background(), nil, nil)
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if approved {
			t.Error("should not approve on error")
		}
	})

	t.Run("receives correct arguments", func(t *testing.T) {
		tool := NewMockTool("test-tool", "Test tool")
		args := map[string]any{"key": "value"}
		var receivedTool Tool
		var receivedArgs map[string]any

		handler := ConfirmationFunc(func(ctx context.Context, t Tool, a map[string]any) (bool, error) {
			receivedTool = t
			receivedArgs = a
			return true, nil
		})

		_, _ = handler.RequestConfirmation(context.Background(), tool, args)

		if receivedTool != tool {
			t.Error("received wrong tool")
		}
		if receivedArgs["key"] != "value" {
			t.Error("received wrong args")
		}
	})
}

// TestCallbackConfirmationHandler tests the CallbackConfirmationHandler.
func TestCallbackConfirmationHandler(t *testing.T) {
	t.Run("calls OnRequest before confirmation", func(t *testing.T) {
		var order []string

		handler := &CallbackConfirmationHandler{
			Handler: ConfirmationFunc(func(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
				order = append(order, "handler")
				return true, nil
			}),
			OnRequest: func(ctx context.Context, tool Tool, args map[string]any) error {
				order = append(order, "onRequest")
				return nil
			},
			OnResponse: func(ctx context.Context, tool Tool, args map[string]any, approved bool, err error) {
				order = append(order, "onResponse")
			},
		}

		tool := NewMockTool("test", "Test")
		_, _ = handler.RequestConfirmation(context.Background(), tool, nil)

		expected := []string{"onRequest", "handler", "onResponse"}
		if len(order) != len(expected) {
			t.Errorf("expected %d calls, got %d", len(expected), len(order))
		}
		for i, v := range expected {
			if i < len(order) && order[i] != v {
				t.Errorf("order[%d] = %s, want %s", i, order[i], v)
			}
		}
	})

	t.Run("OnRequest error skips handler", func(t *testing.T) {
		handlerCalled := false
		expectedErr := errors.New("request error")

		handler := &CallbackConfirmationHandler{
			Handler: ConfirmationFunc(func(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
				handlerCalled = true
				return true, nil
			}),
			OnRequest: func(ctx context.Context, tool Tool, args map[string]any) error {
				return expectedErr
			},
		}

		approved, err := handler.RequestConfirmation(context.Background(), nil, nil)

		if handlerCalled {
			t.Error("handler should not be called when OnRequest errors")
		}
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if approved {
			t.Error("should not approve when OnRequest errors")
		}
	})

	t.Run("OnResponse receives approval result", func(t *testing.T) {
		var receivedApproved bool
		var receivedErr error

		handler := &CallbackConfirmationHandler{
			Handler: &AutoApproveHandler{},
			OnResponse: func(ctx context.Context, tool Tool, args map[string]any, approved bool, err error) {
				receivedApproved = approved
				receivedErr = err
			},
		}

		_, _ = handler.RequestConfirmation(context.Background(), nil, nil)

		if !receivedApproved {
			t.Error("OnResponse should receive approved=true")
		}
		if receivedErr != nil {
			t.Error("OnResponse should receive nil error")
		}
	})

	t.Run("OnResponse receives denial result", func(t *testing.T) {
		var receivedApproved bool

		handler := &CallbackConfirmationHandler{
			Handler: &AutoDenyHandler{},
			OnResponse: func(ctx context.Context, tool Tool, args map[string]any, approved bool, err error) {
				receivedApproved = approved
			},
		}

		_, _ = handler.RequestConfirmation(context.Background(), nil, nil)

		if receivedApproved {
			t.Error("OnResponse should receive approved=false")
		}
	})

	t.Run("works without callbacks", func(t *testing.T) {
		handler := &CallbackConfirmationHandler{
			Handler: &AutoApproveHandler{},
		}

		approved, err := handler.RequestConfirmation(context.Background(), nil, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !approved {
			t.Error("expected approval")
		}
	})
}

// TestUserDeniedError tests the UserDeniedError type.
func TestUserDeniedError(t *testing.T) {
	t.Run("creates error correctly", func(t *testing.T) {
		err := NewUserDeniedError("bash")
		if err.ToolName != "bash" {
			t.Errorf("expected ToolName 'bash', got %q", err.ToolName)
		}
	})

	t.Run("Error() returns message", func(t *testing.T) {
		err := NewUserDeniedError("bash")
		msg := err.Error()
		if msg == "" {
			t.Error("Error() should not return empty string")
		}
		// Should contain tool name
		if msg != "user denied confirmation for tool 'bash'" {
			t.Errorf("unexpected error message: %s", msg)
		}
	})

	t.Run("Is works with sentinel", func(t *testing.T) {
		err := NewUserDeniedError("bash")
		if !errors.Is(err, ErrUserDenied) {
			t.Error("expected Is(ErrUserDenied) to be true")
		}
	})

	t.Run("helper function works", func(t *testing.T) {
		err := NewUserDeniedError("bash")
		if !IsUserDeniedError(err) {
			t.Error("expected IsUserDeniedError to be true")
		}
		if IsUserDeniedError(nil) {
			t.Error("expected IsUserDeniedError(nil) to be false")
		}
		if IsUserDeniedError(errors.New("other error")) {
			t.Error("expected IsUserDeniedError for other error to be false")
		}
	})
}

// TestConfirmationHandlerInterface verifies interface implementations.
func TestConfirmationHandlerInterface(t *testing.T) {
	// This test verifies compile-time interface satisfaction
	var _ ConfirmationHandler = (*AutoApproveHandler)(nil)
	var _ ConfirmationHandler = (*AutoDenyHandler)(nil)
	var _ ConfirmationHandler = ConfirmationFunc(nil)
	var _ ConfirmationHandler = (*CallbackConfirmationHandler)(nil)
}
