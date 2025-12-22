package commands

import (
	"testing"
	"time"
)

func TestSpinnerLifecycle_StopWithSuccess(t *testing.T) {
	s := newSpinner("Connecting")
	s.start()
	// Let it spin briefly
	time.Sleep(50 * time.Millisecond)
	// Should stop cleanly and print success
	s.stopWithSuccess("done")
}

func TestSpinnerLifecycle_StopWithError(t *testing.T) {
	s := newSpinner("Connecting")
	s.start()
	time.Sleep(30 * time.Millisecond)
	// Should stop cleanly on error (no panic)
	s.stopWithError()
}
