package models

// Message represents a chat message for TUI display
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}
