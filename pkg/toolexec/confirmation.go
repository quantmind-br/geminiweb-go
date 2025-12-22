// Package toolexec provides a modular, extensible tool executor architecture.
// This file defines the ConfirmationHandler interface for requesting user
// confirmation before executing potentially dangerous tool operations.
package toolexec

import (
	"context"
)

// ConfirmationHandler defines the interface for requesting user confirmation
// before tool execution. This is typically implemented by a TUI component
// that displays a confirmation dialog to the user.
type ConfirmationHandler interface {
	// RequestConfirmation asks the user to confirm a tool execution.
	// Returns (true, nil) if the user approves the execution.
	// Returns (false, nil) if the user denies the execution.
	// Returns (false, error) if an error occurs during the confirmation process.
	//
	// The context can be used for cancellation (e.g., user presses Ctrl+C).
	// The tool parameter provides information about the tool being executed.
	// The args parameter contains the arguments that will be passed to the tool.
	RequestConfirmation(ctx context.Context, tool Tool, args map[string]any) (bool, error)
}

// AutoApproveHandler is a ConfirmationHandler that automatically approves
// all execution requests. Use this for non-interactive or trusted environments.
type AutoApproveHandler struct{}

// RequestConfirmation always returns (true, nil) - auto-approves all requests.
func (h *AutoApproveHandler) RequestConfirmation(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
	return true, nil
}

// AutoDenyHandler is a ConfirmationHandler that automatically denies
// all execution requests. Use this for highly restricted environments.
type AutoDenyHandler struct{}

// RequestConfirmation always returns (false, nil) - auto-denies all requests.
func (h *AutoDenyHandler) RequestConfirmation(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
	return false, nil
}

// ConfirmationFunc is a function type that implements ConfirmationHandler.
// This allows using simple functions as confirmation handlers.
type ConfirmationFunc func(ctx context.Context, tool Tool, args map[string]any) (bool, error)

// RequestConfirmation implements ConfirmationHandler.
func (f ConfirmationFunc) RequestConfirmation(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
	return f(ctx, tool, args)
}

// CallbackConfirmationHandler wraps callbacks for pre and post confirmation.
// This is useful for logging or monitoring confirmation requests.
type CallbackConfirmationHandler struct {
	// Handler is the underlying confirmation handler.
	Handler ConfirmationHandler

	// OnRequest is called before the confirmation request is made.
	// If it returns an error, the confirmation is skipped and the error is returned.
	OnRequest func(ctx context.Context, tool Tool, args map[string]any) error

	// OnResponse is called after the confirmation response is received.
	OnResponse func(ctx context.Context, tool Tool, args map[string]any, approved bool, err error)
}

// RequestConfirmation implements ConfirmationHandler with callbacks.
func (h *CallbackConfirmationHandler) RequestConfirmation(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
	// Call OnRequest callback if set
	if h.OnRequest != nil {
		if err := h.OnRequest(ctx, tool, args); err != nil {
			if h.OnResponse != nil {
				h.OnResponse(ctx, tool, args, false, err)
			}
			return false, err
		}
	}

	// Request confirmation from underlying handler
	approved, err := h.Handler.RequestConfirmation(ctx, tool, args)

	// Call OnResponse callback if set
	if h.OnResponse != nil {
		h.OnResponse(ctx, tool, args, approved, err)
	}

	return approved, err
}

// Ensure all handlers implement ConfirmationHandler.
var (
	_ ConfirmationHandler = (*AutoApproveHandler)(nil)
	_ ConfirmationHandler = (*AutoDenyHandler)(nil)
	_ ConfirmationHandler = ConfirmationFunc(nil)
	_ ConfirmationHandler = (*CallbackConfirmationHandler)(nil)
)
