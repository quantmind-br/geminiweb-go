package history

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExportToMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create a conversation with messages
	conv, _ := store.CreateConversation("gemini-2.5-flash")
	// Note: AddMessage with role="user" and len(messages)==1 updates the title
	// So we add messages first, then set the title we want
	_ = store.AddMessage(conv.ID, "user", "Hello, how are you?", "")
	_ = store.AddMessage(conv.ID, "assistant", "I'm doing well, thank you!", "Thinking about the response...")
	_ = store.UpdateTitle(conv.ID, "Test Conversation") // Set title after messages

	// Export to Markdown
	md, err := store.ExportToMarkdown(conv.ID)
	if err != nil {
		t.Fatalf("ExportToMarkdown failed: %v", err)
	}

	// Verify content
	if !strings.Contains(md, "# Test Conversation") {
		t.Error("markdown should contain title as header")
	}
	if !strings.Contains(md, "**Model:** gemini-2.5-flash") {
		t.Error("markdown should contain model info")
	}
	if !strings.Contains(md, "## User") {
		t.Error("markdown should contain User header")
	}
	if !strings.Contains(md, "## Assistant") {
		t.Error("markdown should contain Assistant header")
	}
	if !strings.Contains(md, "Hello, how are you?") {
		t.Error("markdown should contain user message")
	}
	if !strings.Contains(md, "I'm doing well") {
		t.Error("markdown should contain assistant message")
	}
	// Default includes thoughts
	if !strings.Contains(md, "Thinking about the response") {
		t.Error("markdown should contain thoughts by default")
	}
}

func TestExportToMarkdown_WithoutThoughts(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	_ = store.AddMessage(conv.ID, "assistant", "Response", "Secret thinking...")

	// Export without thoughts
	opts := DefaultExportOptions()
	opts.IncludeThoughts = false
	md, err := store.ExportToMarkdownWithOptions(conv.ID, opts)
	if err != nil {
		t.Fatalf("ExportToMarkdownWithOptions failed: %v", err)
	}

	if strings.Contains(md, "Secret thinking") {
		t.Error("markdown should NOT contain thoughts when disabled")
	}
}

func TestExportToJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("gemini-2.5-flash")
	_ = store.UpdateMetadata(conv.ID, "cid123", "rid456", "rcid789")
	_ = store.AddMessage(conv.ID, "user", "Test message", "")
	_ = store.UpdateTitle(conv.ID, "JSON Test") // Set title after first message

	// Export to JSON
	jsonData, err := store.ExportToJSON(conv.ID)
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	// Parse and verify
	var exported map[string]interface{}
	if err := json.Unmarshal(jsonData, &exported); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if exported["title"] != "JSON Test" {
		t.Errorf("title = %v, want JSON Test", exported["title"])
	}
	if exported["model"] != "gemini-2.5-flash" {
		t.Errorf("model = %v, want gemini-2.5-flash", exported["model"])
	}

	// By default, API metadata is NOT included
	if exported["cid"] != nil && exported["cid"] != "" {
		t.Error("CID should not be included by default")
	}
}

func TestExportToJSON_WithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	_ = store.UpdateMetadata(conv.ID, "cid123", "rid456", "rcid789")

	// Export with metadata
	opts := DefaultExportOptions()
	opts.IncludeMetadata = true
	jsonData, err := store.ExportToJSONWithOptions(conv.ID, opts)
	if err != nil {
		t.Fatalf("ExportToJSONWithOptions failed: %v", err)
	}

	var exported map[string]interface{}
	_ = json.Unmarshal(jsonData, &exported)

	if exported["cid"] != "cid123" {
		t.Errorf("cid = %v, want cid123", exported["cid"])
	}
	if exported["rid"] != "rid456" {
		t.Errorf("rid = %v, want rid456", exported["rid"])
	}
}

func TestExportToJSON_Messages(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	_ = store.AddMessage(conv.ID, "user", "Question", "")
	_ = store.AddMessage(conv.ID, "assistant", "Answer", "Thinking...")

	jsonData, _ := store.ExportToJSON(conv.ID)

	var exported struct {
		Messages []struct {
			Role     string `json:"role"`
			Content  string `json:"content"`
			Thoughts string `json:"thoughts,omitempty"`
		} `json:"messages"`
	}
	_ = json.Unmarshal(jsonData, &exported)

	if len(exported.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(exported.Messages))
	}

	if exported.Messages[0].Role != "user" {
		t.Errorf("first message role = %s, want user", exported.Messages[0].Role)
	}
	if exported.Messages[0].Content != "Question" {
		t.Errorf("first message content = %s, want Question", exported.Messages[0].Content)
	}
	if exported.Messages[1].Thoughts != "Thinking..." {
		t.Errorf("second message thoughts = %s, want Thinking...", exported.Messages[1].Thoughts)
	}
}

func TestSearchConversations_TitleMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv1, _ := store.CreateConversation("model-1")
	conv2, _ := store.CreateConversation("model-2")
	_ = store.UpdateTitle(conv1.ID, "API Development")
	_ = store.UpdateTitle(conv2.ID, "Database Design")

	// Search for "API" (title only)
	results, err := store.SearchConversations("API", false)
	if err != nil {
		t.Fatalf("SearchConversations failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Conversation.ID != conv1.ID {
		t.Errorf("result ID = %s, want %s", results[0].Conversation.ID, conv1.ID)
	}
	if results[0].MatchField != "title" {
		t.Errorf("MatchField = %s, want title", results[0].MatchField)
	}
}

func TestSearchConversations_ContentMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	// Add a message that doesn't contain "endpoint" first
	_ = store.AddMessage(conv.ID, "user", "Starting a general chat", "")
	// Then add a message that contains "endpoint"
	_ = store.AddMessage(conv.ID, "assistant", "How do I use the API endpoint?", "")
	_ = store.UpdateTitle(conv.ID, "General Chat") // Title without "endpoint"

	// Search in titles only - should not find "endpoint"
	results, _ := store.SearchConversations("endpoint", false)
	if len(results) != 0 {
		t.Errorf("expected 0 results for title-only search, got %d", len(results))
	}

	// Search in content - should find
	results, err := store.SearchConversations("endpoint", true)
	if err != nil {
		t.Fatalf("SearchConversations failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].MatchField != "content" {
		t.Errorf("MatchField = %s, want content", results[0].MatchField)
	}
	if !strings.Contains(results[0].MatchSnippet, "endpoint") {
		t.Error("MatchSnippet should contain the search term")
	}
}

func TestSearchConversations_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	_ = store.UpdateTitle(conv.ID, "API Development")

	// Search with different cases
	tests := []string{"api", "API", "Api", "aPi"}
	for _, query := range tests {
		results, err := store.SearchConversations(query, false)
		if err != nil {
			t.Errorf("SearchConversations(%s) failed: %v", query, err)
			continue
		}
		if len(results) != 1 {
			t.Errorf("SearchConversations(%s) expected 1 result, got %d", query, len(results))
		}
	}
}

func TestSearchConversations_NoResults(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	_ = store.UpdateTitle(conv.ID, "General Chat")

	results, err := store.SearchConversations("xyz123nonexistent", true)
	if err != nil {
		t.Fatalf("SearchConversations failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchConversations_TitleMatchPriority(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	_ = store.UpdateTitle(conv.ID, "API Chat")
	_ = store.AddMessage(conv.ID, "user", "Tell me about the API", "")

	// Title matches - should stop there, not search content
	results, _ := store.SearchConversations("API", true)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].MatchField != "title" {
		t.Errorf("should match title, not content")
	}
}

func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"now", 30 * time.Second, "agora"},
		{"1 min", time.Minute, "há 1 min"},
		{"5 mins", 5 * time.Minute, "há 5 min"},
		{"1 hour", time.Hour, "há 1h"},
		{"3 hours", 3 * time.Hour, "há 3h"},
		{"yesterday", 30 * time.Hour, "ontem"},
		{"3 days", 3 * 24 * time.Hour, "há 3 dias"},
		{"1 week", 7 * 24 * time.Hour, "há 1 sem"},
		{"2 weeks", 14 * 24 * time.Hour, "há 2 sem"},
		{"1 month", 32 * 24 * time.Hour, "há 1 mês"},
		{"3 months", 90 * 24 * time.Hour, "há 3 meses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Now().Add(-tt.duration)
			result := FormatRelativeTime(testTime)
			if result != tt.expected {
				t.Errorf("FormatRelativeTime(%s) = %s, want %s", tt.name, result, tt.expected)
			}
		})
	}
}

func TestFormatRelativeTime_OldDate(t *testing.T) {
	// Very old date should show full date
	oldTime := time.Now().AddDate(-2, 0, 0) // 2 years ago
	result := FormatRelativeTime(oldTime)

	// Should be in DD/MM/YYYY format
	if !strings.Contains(result, "/") {
		t.Errorf("old date should show full date format, got: %s", result)
	}
}

func TestExtractSnippet(t *testing.T) {
	content := "This is a long piece of text that contains the word API somewhere in the middle of it."

	snippet := extractSnippet(content, "API", 40)

	if !strings.Contains(snippet, "API") {
		t.Error("snippet should contain the search term")
	}

	// Should be around the search term
	if len(snippet) > 50 { // 40 + some ellipsis allowance
		t.Errorf("snippet too long: %d chars", len(snippet))
	}
}

func TestExtractSnippet_AtStart(t *testing.T) {
	content := "API is at the very beginning of this text."

	snippet := extractSnippet(content, "API", 30)

	if !strings.HasPrefix(snippet, "API") {
		t.Error("snippet should start with API")
	}
}

func TestExtractSnippet_AtEnd(t *testing.T) {
	content := "This text ends with API"

	snippet := extractSnippet(content, "API", 30)

	if !strings.HasSuffix(snippet, "API") {
		t.Errorf("snippet should end with API, got: %s", snippet)
	}
}

func TestDefaultExportOptions(t *testing.T) {
	opts := DefaultExportOptions()

	if opts.Format != ExportFormatMarkdown {
		t.Errorf("default format = %v, want markdown", opts.Format)
	}
	if opts.IncludeMetadata {
		t.Error("default IncludeMetadata should be false")
	}
	if !opts.IncludeThoughts {
		t.Error("default IncludeThoughts should be true")
	}
}
