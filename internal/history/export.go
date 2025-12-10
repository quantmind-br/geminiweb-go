// Package history provides local conversation history storage.
package history

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ExportFormat represents the format for exporting conversations
type ExportFormat string

const (
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatJSON     ExportFormat = "json"
)

// ExportOptions configures how conversations are exported
type ExportOptions struct {
	Format          ExportFormat
	IncludeMetadata bool // Include CID, RID, RCID in JSON export
	IncludeThoughts bool // Include thought/reasoning content
}

// DefaultExportOptions returns sensible defaults for export
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		Format:          ExportFormatMarkdown,
		IncludeMetadata: false,
		IncludeThoughts: true,
	}
}

// ExportToMarkdown exports a conversation to Markdown format
func (s *Store) ExportToMarkdown(id string) (string, error) {
	return s.ExportToMarkdownWithOptions(id, DefaultExportOptions())
}

// ExportToMarkdownWithOptions exports a conversation to Markdown with options
func (s *Store) ExportToMarkdownWithOptions(id string, opts ExportOptions) (string, error) {
	conv, err := s.GetConversation(id)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	// Header
	sb.WriteString("# ")
	sb.WriteString(conv.Title)
	sb.WriteString("\n\n")

	// Metadata
	sb.WriteString("**Model:** ")
	sb.WriteString(conv.Model)
	sb.WriteString("\n")
	sb.WriteString("**Created:** ")
	sb.WriteString(conv.CreatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")
	sb.WriteString("**Updated:** ")
	sb.WriteString(conv.UpdatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")
	sb.WriteString("**Messages:** ")
	sb.WriteString(fmt.Sprintf("%d", len(conv.Messages)))
	sb.WriteString("\n\n---\n\n")

	// Messages
	for i, msg := range conv.Messages {
		// Role header
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}

		sb.WriteString("## ")
		sb.WriteString(role)
		if !msg.Timestamp.IsZero() {
			sb.WriteString(" (")
			sb.WriteString(msg.Timestamp.Format("15:04:05"))
			sb.WriteString(")")
		}
		sb.WriteString("\n\n")

		// Thoughts (if enabled and present)
		if opts.IncludeThoughts && msg.Thoughts != "" {
			sb.WriteString("<details>\n<summary>💭 Thinking</summary>\n\n")
			sb.WriteString(msg.Thoughts)
			sb.WriteString("\n\n</details>\n\n")
		}

		// Content
		sb.WriteString(msg.Content)
		sb.WriteString("\n")

		// Separator between messages (except last)
		if i < len(conv.Messages)-1 {
			sb.WriteString("\n---\n\n")
		}
	}

	return sb.String(), nil
}

// ExportToJSON exports a conversation to JSON format
func (s *Store) ExportToJSON(id string) ([]byte, error) {
	return s.ExportToJSONWithOptions(id, DefaultExportOptions())
}

// ExportToJSONWithOptions exports a conversation to JSON with options
func (s *Store) ExportToJSONWithOptions(id string, opts ExportOptions) ([]byte, error) {
	conv, err := s.GetConversation(id)
	if err != nil {
		return nil, err
	}

	// Create export structure
	type ExportMessage struct {
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		Thoughts  string    `json:"thoughts,omitempty"`
		Timestamp time.Time `json:"timestamp"`
	}

	type ExportConversation struct {
		ID        string          `json:"id"`
		Title     string          `json:"title"`
		Model     string          `json:"model"`
		CreatedAt time.Time       `json:"created_at"`
		UpdatedAt time.Time       `json:"updated_at"`
		Messages  []ExportMessage `json:"messages"`
		// API metadata (optional)
		CID  string `json:"cid,omitempty"`
		RID  string `json:"rid,omitempty"`
		RCID string `json:"rcid,omitempty"`
	}

	export := ExportConversation{
		ID:        conv.ID,
		Title:     conv.Title,
		Model:     conv.Model,
		CreatedAt: conv.CreatedAt,
		UpdatedAt: conv.UpdatedAt,
		Messages:  make([]ExportMessage, len(conv.Messages)),
	}

	// Include API metadata if requested
	if opts.IncludeMetadata {
		export.CID = conv.CID
		export.RID = conv.RID
		export.RCID = conv.RCID
	}

	// Copy messages
	for i, msg := range conv.Messages {
		export.Messages[i] = ExportMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		}
		if opts.IncludeThoughts {
			export.Messages[i].Thoughts = msg.Thoughts
		}
	}

	return json.MarshalIndent(export, "", "  ")
}

// SearchResult represents a search match in conversations
type SearchResult struct {
	Conversation *Conversation
	MatchSnippet string // Snippet where the term was found
	MatchField   string // "title" or "content"
	MatchIndex   int    // Message index if MatchField is "content", -1 for title
}

// SearchConversations searches for a query in conversation titles and optionally content
func (s *Store) SearchConversations(query string, searchContent bool) ([]*SearchResult, error) {
	conversations, err := s.ListConversations()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []*SearchResult

	for _, conv := range conversations {
		// Search in title
		if strings.Contains(strings.ToLower(conv.Title), queryLower) {
			results = append(results, &SearchResult{
				Conversation: conv,
				MatchSnippet: conv.Title,
				MatchField:   "title",
				MatchIndex:   -1,
			})
			continue // Don't search content if title matched
		}

		// Search in content if enabled
		if searchContent {
			for i, msg := range conv.Messages {
				contentLower := strings.ToLower(msg.Content)
				if strings.Contains(contentLower, queryLower) {
					// Extract snippet around match
					snippet := extractSnippet(msg.Content, query, 100)
					results = append(results, &SearchResult{
						Conversation: conv,
						MatchSnippet: snippet,
						MatchField:   "content",
						MatchIndex:   i,
					})
					break // Only one match per conversation
				}
			}
		}
	}

	return results, nil
}

// extractSnippet extracts a snippet around the first occurrence of query
func extractSnippet(content, query string, maxLen int) string {
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)

	idx := strings.Index(contentLower, queryLower)
	if idx == -1 {
		// Shouldn't happen, but fallback to start
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}

	// Calculate start and end positions
	half := maxLen / 2
	start := idx - half
	end := idx + len(query) + half

	if start < 0 {
		start = 0
		end = maxLen
	}
	if end > len(content) {
		end = len(content)
		start = end - maxLen
		if start < 0 {
			start = 0
		}
	}

	snippet := content[start:end]

	// Add ellipsis if truncated
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// FormatRelativeTime formats a time as a relative string like "há 2h" or "ontem"
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "agora"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "há 1 min"
		}
		return fmt.Sprintf("há %d min", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "há 1h"
		}
		return fmt.Sprintf("há %dh", hours)
	case diff < 48*time.Hour:
		return "ontem"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("há %d dias", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "há 1 sem"
		}
		return fmt.Sprintf("há %d sem", weeks)
	default:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "há 1 mês"
		}
		if months < 12 {
			return fmt.Sprintf("há %d meses", months)
		}
		return t.Format("02/01/2006")
	}
}
