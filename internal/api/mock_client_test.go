package api_test

import (
	"testing"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
)

func TestMockGeminiClient(t *testing.T) {
	mock := &api.MockGeminiClient{
		GenerateContentVal: &models.ModelOutput{
			Candidates: []models.Candidate{
				{Text: "Mock response"},
			},
		},
	}

	// Verify interface compliance
	var client api.GeminiClientInterface = mock

	// Test StartChat
	session := client.StartChat()
	if session == nil {
		t.Fatal("StartChat returned nil")
	}

	// Test SendMessage (which calls GenerateContent on the mock)
	resp, err := session.SendMessage("Hello", nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp.Text() != "Mock response" {
		t.Errorf("Expected 'Mock response', got '%s'", resp.Text())
	}

	if !mock.GenerateContentCalled {
		t.Error("GenerateContent was not called on mock")
	}

	if mock.LastPrompt != "Hello" {
		t.Errorf("Expected prompt 'Hello', got '%s'", mock.LastPrompt)
	}
}
