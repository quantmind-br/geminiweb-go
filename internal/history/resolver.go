// Package history provides local conversation history storage.
package history

import (
	"fmt"
	"strconv"
	"strings"
)

// Resolver resolves user-friendly references to conversation IDs
type Resolver struct {
	store *Store
}

// NewResolver creates a new alias resolver
func NewResolver(store *Store) *Resolver {
	return &Resolver{store: store}
}

// Resolve converts a user-friendly reference to a conversation ID
//
// Supported references:
//   - "@last" - most recently modified conversation
//   - "@first" - first conversation in the list
//   - "1", "2", "3" - by index (1-based)
//   - "substring" - fuzzy match on title (error if multiple matches)
//   - "conv-..." - direct ID
func (r *Resolver) Resolve(ref string) (string, error) {
	ref = strings.TrimSpace(ref)

	if ref == "" {
		return "", fmt.Errorf("empty reference")
	}

	// Get all conversations
	conversations, err := r.store.ListConversations()
	if err != nil {
		return "", fmt.Errorf("failed to list conversations: %w", err)
	}

	if len(conversations) == 0 {
		return "", fmt.Errorf("no conversations found")
	}

	// Handle special aliases
	switch strings.ToLower(ref) {
	case "@last":
		// Already sorted by UpdatedAt descending
		return conversations[0].ID, nil
	case "@first":
		return conversations[len(conversations)-1].ID, nil
	}

	// Handle numeric index (1-based)
	if index, err := strconv.Atoi(ref); err == nil {
		if index < 1 || index > len(conversations) {
			return "", fmt.Errorf("index %d out of range (1-%d)", index, len(conversations))
		}
		return conversations[index-1].ID, nil
	}

	// Handle direct ID (starts with conv-)
	if strings.HasPrefix(ref, "conv-") {
		for _, conv := range conversations {
			if conv.ID == ref {
				return conv.ID, nil
			}
		}
		return "", fmt.Errorf("conversation not found: %s", ref)
	}

	// Handle substring match on title (case-insensitive)
	refLower := strings.ToLower(ref)
	var matches []*Conversation
	for _, conv := range conversations {
		if strings.Contains(strings.ToLower(conv.Title), refLower) {
			matches = append(matches, conv)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no conversation matching '%s'", ref)
	case 1:
		return matches[0].ID, nil
	default:
		// Multiple matches - show them to user
		var titles []string
		for _, m := range matches {
			titles = append(titles, fmt.Sprintf("'%s'", m.Title))
		}
		return "", fmt.Errorf("multiple conversations match '%s': %s. Use ID or be more specific",
			ref, strings.Join(titles, ", "))
	}
}

// MustResolve is like Resolve but panics on error (for testing)
func (r *Resolver) MustResolve(ref string) string {
	id, err := r.Resolve(ref)
	if err != nil {
		panic(err)
	}
	return id
}

// ResolveWithInfo resolves a reference and returns the conversation info
func (r *Resolver) ResolveWithInfo(ref string) (*Conversation, error) {
	id, err := r.Resolve(ref)
	if err != nil {
		return nil, err
	}

	conv, err := r.store.GetConversation(id)
	if err != nil {
		return nil, err
	}

	return conv, nil
}

// ValidateRef checks if a reference is valid without resolving it
func (r *Resolver) ValidateRef(ref string) error {
	_, err := r.Resolve(ref)
	return err
}

// ListAliases returns information about supported aliases
func ListAliases() string {
	return `Supported references:
  @last          Most recently modified conversation
  @first         First conversation in the list
  1, 2, 3        By index (1-based, from most recent)
  "text"         Search by title substring
  conv-...       Direct conversation ID`
}
