package commands

import "testing"

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("expected unchanged, got %s", got)
	}

	if got := truncate("abcdefghijklmnopqrstuvwxyz", 5); got != "abcde..." {
		t.Fatalf("expected truncated with ellipsis, got %s", got)
	}
}
