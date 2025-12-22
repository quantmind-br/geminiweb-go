package api

import (
	"errors"
	"testing"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

// TestChatSession_SendMessage tests the SendMessage function
func TestChatSession_SendMessage(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	tests := []struct {
		name        string
		prompt      string
		setupMock   func(*MockHttpClient)
		expectedErr bool
	}{
		{
			name:        "empty prompt",
			prompt:      "",
			setupMock:   func(m *MockHttpClient) {},
			expectedErr: true,
		},
		{
			name:   "error from GenerateContent",
			prompt: "test",
			setupMock: func(m *MockHttpClient) {
				m.Err = errors.New("network error")
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHttpClient{}
			tt.setupMock(mockClient)

			// Create a real GeminiClient with the mock
			geminiClient, err := NewClient(validCookies)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			geminiClient.httpClient = mockClient

			session := &ChatSession{
				client: geminiClient,
				model:  models.Model25Flash,
			}

			got, err := session.SendMessage(tt.prompt, nil)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("SendMessage() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("SendMessage() unexpected error: %v", err)
				return
			}

			if got == nil {
				t.Errorf("SendMessage() returned nil")
				return
			}
		})
	}
}

// TestChatSession_Getters tests the getter methods
func TestChatSession_Getters(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	geminiClient, err := NewClient(validCookies)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	session := &ChatSession{
		client: geminiClient,
		model:  models.ModelPro,
	}

	t.Run("GetMetadata returns empty initially", func(t *testing.T) {
		metadata := session.GetMetadata()
		if len(metadata) != 0 {
			t.Errorf("GetMetadata() length = %d, want 0", len(metadata))
		}
	})

	t.Run("GetModel returns correct model", func(t *testing.T) {
		model := session.GetModel()
		if model.Name != models.ModelPro.Name {
			t.Errorf("GetModel().Name = %v, want %v", model.Name, models.ModelPro.Name)
		}
	})

	t.Run("LastOutput returns nil initially", func(t *testing.T) {
		last := session.LastOutput()
		if last != nil {
			t.Errorf("LastOutput() = %v, want nil", last)
		}
	})

	t.Run("CID/RID/RCID return empty when no metadata", func(t *testing.T) {
		if session.CID() != "" {
			t.Error("CID() should be empty")
		}
		if session.RID() != "" {
			t.Error("RID() should be empty")
		}
		if session.RCID() != "" {
			t.Error("RCID() should be empty")
		}
	})
}

// TestChatSession_SetMetadata tests the SetMetadata function
func TestChatSession_SetMetadata(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	geminiClient, err := NewClient(validCookies)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	session := &ChatSession{
		client: geminiClient,
		model:  models.Model25Flash,
	}

	t.Run("SetMetadata updates metadata fields", func(t *testing.T) {
		cid, rid, rcid := "cid123", "rid456", "rcid789"
		session.SetMetadata(cid, rid, rcid)

		if session.CID() != cid {
			t.Errorf("CID() = %s, want %s", session.CID(), cid)
		}
		if session.RID() != rid {
			t.Errorf("RID() = %s, want %s", session.RID(), rid)
		}
		if session.RCID() != rcid {
			t.Errorf("RCID() = %s, want %s", session.RCID(), rcid)
		}

		metadata := session.GetMetadata()
		if len(metadata) != 3 {
			t.Errorf("GetMetadata() length = %d, want 3", len(metadata))
		}
		if metadata[0] != cid || metadata[1] != rid || metadata[2] != rcid {
			t.Error("GetMetadata() doesn't match SetMetadata values")
		}
	})
}

// TestChatSession_SetModel tests the SetModel function
func TestChatSession_SetModel(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	geminiClient, err := NewClient(validCookies)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	session := &ChatSession{
		client: geminiClient,
		model:  models.Model25Flash,
	}

	newModel := models.ModelPro
	session.SetModel(newModel)

	if session.GetModel().Name != newModel.Name {
		t.Errorf("GetModel().Name = %v, want %v", session.GetModel().Name, newModel.Name)
	}
}

// TestChatSession_ChooseCandidate tests the ChooseCandidate function
func TestChatSession_ChooseCandidate(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	geminiClient, err := NewClient(validCookies)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	session := &ChatSession{
		client: geminiClient,
		model:  models.Model25Flash,
	}

	// Setup session with output
	output := &models.ModelOutput{
		Metadata: []string{"cid1", "rid1", "rcid1"},
		Candidates: []models.Candidate{
			{RCID: "rcid1", Text: "First response"},
			{RCID: "rcid2", Text: "Second response"},
			{RCID: "rcid3", Text: "Third response"},
		},
		Chosen: 0,
	}
	session.lastOutput = output

	t.Run("ChooseCandidate within bounds", func(t *testing.T) {
		err := session.ChooseCandidate(1)
		if err != nil {
			t.Errorf("ChooseCandidate() unexpected error: %v", err)
		}

		lastOutput := session.LastOutput()
		if lastOutput == nil {
			t.Error("LastOutput() returned nil")
			return
		}

		if lastOutput.Chosen != 1 {
			t.Errorf("Chosen = %d, want 1", lastOutput.Chosen)
		}

		// Verify RCID is updated
		if session.RCID() != "rcid2" {
			t.Errorf("RCID() = %s, want rcid2", session.RCID())
		}
	})

	t.Run("ChooseCandidate out of bounds (index >= len)", func(t *testing.T) {
		err := session.ChooseCandidate(10)
		if err != nil {
			t.Errorf("ChooseCandidate() unexpected error: %v", err)
		}

		lastOutput := session.LastOutput()
		if lastOutput == nil {
			t.Error("LastOutput() returned nil")
			return
		}

		// Should remain at previous index
		if lastOutput.Chosen != 1 {
			t.Errorf("Chosen = %d, want 1", lastOutput.Chosen)
		}
	})

	t.Run("ChooseCandidate with no last output", func(t *testing.T) {
		emptySession := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}
		err := emptySession.ChooseCandidate(0)
		if err != nil {
			t.Errorf("ChooseCandidate() unexpected error: %v", err)
		}
	})
}

// TestChatSession_SendMessageWithFiles tests SendMessage with file attachments
func TestChatSession_SendMessageWithFiles(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	t.Run("SendMessage with nil files", func(t *testing.T) {
		mockClient := &MockHttpClient{}
		mockClient.Err = errors.New("simulated error")

		geminiClient, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		geminiClient.httpClient = mockClient

		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}

		// Call with nil files - should work the same as before
		_, err = session.SendMessage("test prompt", nil)
		// We expect an error from the mock, but it should be called
		if err == nil {
			t.Error("Expected error from mock client")
		}
	})

	t.Run("SendMessage with empty files slice", func(t *testing.T) {
		mockClient := &MockHttpClient{}
		mockClient.Err = errors.New("simulated error")

		geminiClient, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		geminiClient.httpClient = mockClient

		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}

		// Call with empty files slice
		files := []*UploadedFile{}
		_, err = session.SendMessage("test prompt", files)
		if err == nil {
			t.Error("Expected error from mock client")
		}
	})

	t.Run("SendMessage with files slice", func(t *testing.T) {
		mockClient := &MockHttpClient{}
		mockClient.Err = errors.New("simulated error")

		geminiClient, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		geminiClient.httpClient = mockClient

		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}

		// Call with files
		files := []*UploadedFile{
			{ResourceID: "res-1", FileName: "test.jpg", MIMEType: "image/jpeg", Size: 1024},
			{ResourceID: "res-2", FileName: "test2.png", MIMEType: "image/png", Size: 2048},
		}
		_, err = session.SendMessage("describe these images", files)
		if err == nil {
			t.Error("Expected error from mock client")
		}
	})

	t.Run("SendMessage preserves gemID with files", func(t *testing.T) {
		mockClient := &MockHttpClient{}
		mockClient.Err = errors.New("simulated error")

		geminiClient, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		geminiClient.httpClient = mockClient

		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
			gemID:  "test-gem-id",
		}

		files := []*UploadedFile{
			{ResourceID: "res-1", FileName: "test.jpg", MIMEType: "image/jpeg", Size: 1024},
		}

		// SetGem should be preserved
		if session.GetGemID() != "test-gem-id" {
			t.Errorf("GetGemID() = %s, want test-gem-id", session.GetGemID())
		}

		// Call with files - gem ID should still be set
		_, _ = session.SendMessage("test", files)

		if session.GetGemID() != "test-gem-id" {
			t.Errorf("GetGemID() after SendMessage = %s, want test-gem-id", session.GetGemID())
		}
	})

	t.Run("SendMessage preserves metadata with files", func(t *testing.T) {
		mockClient := &MockHttpClient{}
		mockClient.Err = errors.New("simulated error")

		geminiClient, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		geminiClient.httpClient = mockClient

		session := &ChatSession{
			client:   geminiClient,
			model:    models.Model25Flash,
			metadata: []string{"cid1", "rid1", "rcid1"},
		}

		files := []*UploadedFile{
			{ResourceID: "res-1", FileName: "test.jpg", MIMEType: "image/jpeg", Size: 1024},
		}

		// Verify metadata before call
		if session.CID() != "cid1" {
			t.Errorf("CID() = %s, want cid1", session.CID())
		}

		// Call with files - metadata should still be preserved (since we're mocking an error)
		_, _ = session.SendMessage("test", files)

		if session.CID() != "cid1" {
			t.Errorf("CID() after SendMessage = %s, want cid1", session.CID())
		}
	})
}

// TestChatSession_updateMetadata tests the updateMetadata function
func TestChatSession_updateMetadata(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	geminiClient, err := NewClient(validCookies)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("updateMetadata with empty output", func(t *testing.T) {
		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}

		session.updateMetadata(&models.ModelOutput{})

		// Should not panic
		_ = session.GetMetadata()
	})

	t.Run("updateMetadata with full metadata", func(t *testing.T) {
		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}

		output := &models.ModelOutput{
			Metadata:   []string{"cid1", "rid1"},
			Candidates: []models.Candidate{{RCID: "rcid1", Text: "Response"}},
		}

		session.updateMetadata(output)

		metadata := session.GetMetadata()
		if len(metadata) != 3 {
			t.Errorf("metadata length = %d, want 3", len(metadata))
		}

		if metadata[0] != "cid1" || metadata[1] != "rid1" {
			t.Error("metadata not copied correctly")
		}

		if session.RCID() != "rcid1" {
			t.Errorf("RCID() = %s, want rcid1", session.RCID())
		}
	})

	t.Run("updateMetadata when metadata slice exists", func(t *testing.T) {
		session := &ChatSession{
			client:   geminiClient,
			model:    models.Model25Flash,
			metadata: []string{"cid1", "rid1", "old_rcid"},
		}

		newOutput := &models.ModelOutput{
			Metadata:   []string{"cid2", "rid2", "rcid2"},
			Candidates: []models.Candidate{{RCID: "rcid2", Text: "Response"}},
		}

		session.updateMetadata(newOutput)

		metadata := session.GetMetadata()
		if metadata[0] != "cid2" || metadata[1] != "rid2" || metadata[2] != "rcid2" {
			t.Error("metadata not updated correctly")
		}
	})

	t.Run("updateMetadata when metadata has 3 elements updates RCID", func(t *testing.T) {
		session := &ChatSession{
			client:   geminiClient,
			model:    models.Model25Flash,
			metadata: []string{"cid1", "rid1", "old_rcid"},
		}

		newOutput := &models.ModelOutput{
			Metadata:   []string{"cid1", "rid1", "new_rcid"}, // Same CID/RID, different RCID
			Candidates: []models.Candidate{{RCID: "new_rcid", Text: "Response"}},
		}

		session.updateMetadata(newOutput)

		metadata := session.GetMetadata()
		if len(metadata) != 3 {
			t.Errorf("metadata length = %d, want 3", len(metadata))
		}

		if metadata[2] != "new_rcid" {
			t.Errorf("metadata[2] = %s, want new_rcid", metadata[2])
		}
	})
}

// TestChatSession_SetGem tests the SetGem function
func TestChatSession_SetGem(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	geminiClient, err := NewClient(validCookies)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("SetGem updates gemID", func(t *testing.T) {
		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
		}

		session.SetGem("new-gem-id")

		if session.gemID != "new-gem-id" {
			t.Errorf("gemID = %s, want new-gem-id", session.gemID)
		}
		if session.GetGemID() != "new-gem-id" {
			t.Errorf("GetGemID() = %s, want new-gem-id", session.GetGemID())
		}
	})

	t.Run("SetGem clears gemID with empty string", func(t *testing.T) {
		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
			gemID:  "existing-gem",
		}

		session.SetGem("")

		if session.gemID != "" {
			t.Errorf("gemID = %s, want empty", session.gemID)
		}
		if session.GetGemID() != "" {
			t.Errorf("GetGemID() = %s, want empty", session.GetGemID())
		}
	})

	t.Run("SetGem replaces existing gemID", func(t *testing.T) {
		session := &ChatSession{
			client: geminiClient,
			model:  models.Model25Flash,
			gemID:  "old-gem-id",
		}

		session.SetGem("replaced-gem-id")

		if session.gemID != "replaced-gem-id" {
			t.Errorf("gemID = %s, want replaced-gem-id", session.gemID)
		}
	})
}
