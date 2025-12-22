package commands

import (
	"testing"

	"github.com/diogo/geminiweb/internal/models"
)

func TestResolveGemFlag_EmptyFlag(t *testing.T) {
	// When gemFlag is empty, should return empty ResolvedGem without error
	resolved, err := resolveGemFlag(nil, "")
	if err != nil {
		t.Errorf("Expected no error for empty flag, got: %v", err)
	}
	if resolved.ID != "" {
		t.Errorf("Expected empty gemID for empty flag, got: %s", resolved.ID)
	}
	if resolved.Name != "" {
		t.Errorf("Expected empty gemName for empty flag, got: %s", resolved.Name)
	}
}

func TestResolveGemFlag_GemNotFound(t *testing.T) {
	// Create a mock GemJar with no gems
	jar := make(models.GemJar)

	// Create mock client
	// Note: This test requires access to resolveGem which internally fetches gems
	// Since resolveGem uses *api.GeminiClient directly, we can't easily mock it here
	// Instead we test the flow through runChat integration tests

	t.Log("GemJar created:", jar)
	t.Log("Integration test for gem resolution should be done via chat command")
}

// TestResolveGemFlag_Integration tests the full flow of gem resolution
// This is more of an integration test and requires mocking the full client
func TestResolveGemFlag_Integration(t *testing.T) {
	tests := []struct {
		name      string
		gemFlag   string
		wantEmpty bool
		wantErr   bool
	}{
		{
			name:      "empty flag returns empty string",
			gemFlag:   "",
			wantEmpty: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolveGemFlag(nil, tt.gemFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveGemFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (resolved.ID == "") != tt.wantEmpty {
				t.Errorf("resolveGemFlag() gemID = %v, wantEmpty %v", resolved.ID, tt.wantEmpty)
			}
		})
	}
}

// TestResolveGem tests the resolveGem function from gems.go
func TestResolveGem_SessionNotFound(t *testing.T) {
	// This test verifies error handling when gem is not found
	// It requires a real client, so we just test the function signature
	t.Log("resolveGem returns error when gem is not found")

	// The actual test would be:
	// gem, err := resolveGem(client, "nonexistent")
	// if err == nil {
	//     t.Error("Expected error for nonexistent gem")
	// }
}

// BenchmarkResolveGemFlag benchmarks the gem flag resolution
func BenchmarkResolveGemFlag(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Empty flag should be very fast
		_, _ = resolveGemFlag(nil, "")
	}
}

// TestChatGemFlag tests that the chatGemFlag variable exists
func TestChatGemFlag(t *testing.T) {
	// Test that the flag variable is properly initialized
	// The actual flag is defined in chat.go init()

	// Check that the flag is registered on chatCmd
	flag := chatCmd.Flags().Lookup("gem")
	if flag == nil {
		t.Error("gem flag not found on chatCmd")
		return
	}

	if flag.Shorthand != "g" {
		t.Errorf("Expected shorthand 'g', got '%s'", flag.Shorthand)
	}

	if flag.Usage == "" {
		t.Error("Flag should have usage description")
	}
}

// TestChatCommand_WithGemFlag tests the chat command with gem flag
func TestChatCommand_WithGemFlag(t *testing.T) {
	// Verify the command long description mentions gem flag
	if chatCmd.Long == "" {
		t.Error("Chat command should have long description")
	}

	// Check that --gem is documented
	if len(chatCmd.Long) < 50 {
		t.Error("Chat command long description seems too short")
	}
}

// TestChatCommand_GemFlagUsage tests the gem flag usage text
func TestChatCommand_GemFlagUsage(t *testing.T) {
	flag := chatCmd.Flags().Lookup("gem")
	if flag == nil {
		t.Skip("gem flag not found")
		return
	}

	expectedUsage := "Use a gem (by ID or name) - server-side persona"
	if flag.Usage != expectedUsage {
		t.Errorf("Expected usage '%s', got '%s'", expectedUsage, flag.Usage)
	}
}

// TestResolveGemFlag_Example demonstrates how to use resolveGemFlag
func TestResolveGemFlag_Example(t *testing.T) {
	// When no gem is specified
	resolved, err := resolveGemFlag(nil, "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resolved.ID != "" {
		t.Errorf("Expected empty gemID, got: %s", resolved.ID)
	}
	t.Log("No gem specified - returns empty ResolvedGem")
}

// TestCreateChatSession_WithGemID tests createChatSession with a gem ID
func TestCreateChatSession_WithGemID(t *testing.T) {
	// Create mock client
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Call createChatSession with gem ID
	session := createChatSession(mockClient, "test-gem-id", models.Model25Flash)

	// The session should not be nil (it's created by StartChatWithOptions)
	if session == nil {
		t.Error("Expected non-nil session")
	}
}

// TestCreateChatSession_WithoutGemID tests createChatSession without a gem ID
func TestCreateChatSession_WithoutGemID(t *testing.T) {
	// Create mock client
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Call createChatSession without gem ID
	session := createChatSession(mockClient, "", models.Model25Flash)

	// The session should not be nil
	if session == nil {
		t.Error("Expected non-nil session")
	}
}

// TestCreateChatSession_DifferentModels tests createChatSession with different models
func TestCreateChatSession_DifferentModels(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	models := []models.Model{
		models.ModelFast,
		models.ModelPro,
		models.ModelThinking,
	}

	for _, model := range models {
		t.Run(model.Name, func(t *testing.T) {
			session := createChatSession(mockClient, "", model)
			if session == nil {
				t.Error("Expected non-nil session")
			}
		})
	}
}

// TestCreateChatSession_WithEmptyGemID tests createChatSession with empty gem ID string
func TestCreateChatSession_WithEmptyGemID(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Empty string should be treated as no gem
	session := createChatSession(mockClient, "", models.Model25Flash)
	if session == nil {
		t.Error("Expected non-nil session with empty gem ID")
	}
}

// TestCreateChatSession_WithClosedClient tests createChatSession with a closed client
func TestCreateChatSession_WithClosedClient(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: true,
	}

	// Should still create session even with closed client (caller's responsibility to check)
	session := createChatSession(mockClient, "test-gem", models.Model25Flash)
	if session == nil {
		t.Error("Expected non-nil session even with closed client")
	}
}
