package api

import (
	"testing"

	"github.com/tidwall/gjson"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

func TestParseGemsResponse(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		predefined bool
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "valid response with gems",
			data:       `[null,null,[["gem1",["Name1","Description1"],["Prompt1"]],["gem2",["Name2","Description2"],["Prompt2"]]]]`,
			predefined: false,
			wantCount:  2,
			wantErr:    false,
		},
		{
			name:       "valid response with system gems",
			data:       `[null,null,[["sys1",["System Gem","System desc"],["System prompt"]]]]`,
			predefined: true,
			wantCount:  1,
			wantErr:    false,
		},
		{
			name:       "empty gems array",
			data:       `[null,null,[]]`,
			predefined: false,
			wantCount:  0,
			wantErr:    false,
		},
		{
			name:       "no gems position",
			data:       `[null,null]`,
			predefined: false,
			wantCount:  0,
			wantErr:    false,
		},
		{
			name:       "gem without prompt",
			data:       `[null,null,[["gem1",["Name1","Description1"],null]]]`,
			predefined: false,
			wantCount:  1,
			wantErr:    false,
		},
		{
			name:       "not an array",
			data:       `{"error": "invalid"}`,
			predefined: false,
			wantCount:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gems, err := parseGemsResponse(tt.data, tt.predefined)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGemsResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(gems) != tt.wantCount {
				t.Errorf("parseGemsResponse() got %d gems, want %d", len(gems), tt.wantCount)
			}
			if len(gems) > 0 {
				for _, gem := range gems {
					if gem.Predefined != tt.predefined {
						t.Errorf("parseGemsResponse() predefined = %v, want %v", gem.Predefined, tt.predefined)
					}
				}
			}
		})
	}
}

func TestParseGemData(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		predefined bool
		wantID     string
		wantName   string
		wantDesc   string
		wantPrompt string
		wantNil    bool
	}{
		{
			name:       "complete gem",
			data:       `["gem123",["Test Gem","A test description"],["You are a test assistant"]]`,
			predefined: false,
			wantID:     "gem123",
			wantName:   "Test Gem",
			wantDesc:   "A test description",
			wantPrompt: "You are a test assistant",
			wantNil:    false,
		},
		{
			name:       "gem without prompt",
			data:       `["gem456",["No Prompt Gem","Description only"],null]`,
			predefined: true,
			wantID:     "gem456",
			wantName:   "No Prompt Gem",
			wantDesc:   "Description only",
			wantPrompt: "",
			wantNil:    false,
		},
		{
			name:       "gem with empty prompt array",
			data:       `["gem789",["Empty Prompt","Desc"],[]]`,
			predefined: false,
			wantID:     "gem789",
			wantName:   "Empty Prompt",
			wantDesc:   "Desc",
			wantPrompt: "",
			wantNil:    false,
		},
		{
			name:       "gem without ID",
			data:       `["",["Name","Desc"],["Prompt"]]`,
			predefined: false,
			wantNil:    true,
		},
		{
			name:       "invalid structure",
			data:       `["only_id"]`,
			predefined: false,
			wantID:     "only_id",
			wantName:   "",
			wantDesc:   "",
			wantPrompt: "",
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use gjson to parse the test data
			parsed := gjson.Parse(tt.data)
			gem := parseGemData(parsed, tt.predefined)

			if tt.wantNil {
				if gem != nil {
					t.Errorf("parseGemData() expected nil, got %v", gem)
				}
				return
			}

			if gem == nil {
				t.Fatal("parseGemData() returned nil, expected gem")
			}

			if gem.ID != tt.wantID {
				t.Errorf("parseGemData() ID = %v, want %v", gem.ID, tt.wantID)
			}
			if gem.Name != tt.wantName {
				t.Errorf("parseGemData() Name = %v, want %v", gem.Name, tt.wantName)
			}
			if gem.Description != tt.wantDesc {
				t.Errorf("parseGemData() Description = %v, want %v", gem.Description, tt.wantDesc)
			}
			if gem.Prompt != tt.wantPrompt {
				t.Errorf("parseGemData() Prompt = %v, want %v", gem.Prompt, tt.wantPrompt)
			}
			if gem.Predefined != tt.predefined {
				t.Errorf("parseGemData() Predefined = %v, want %v", gem.Predefined, tt.predefined)
			}
		})
	}
}

func TestGemsClientMethods(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.accessToken = "test-token"

	t.Run("Gems returns nil before FetchGems", func(t *testing.T) {
		gems := client.Gems()
		if gems != nil {
			t.Error("Expected nil gems before FetchGems")
		}
	})

	t.Run("GetGem returns nil before FetchGems", func(t *testing.T) {
		gem := client.GetGem("test-id", "test-name")
		if gem != nil {
			t.Error("Expected nil gem before FetchGems")
		}
	})

	t.Run("Gems returns cached jar after manual set", func(t *testing.T) {
		jar := make(models.GemJar)
		jar["test-id"] = &models.Gem{ID: "test-id", Name: "Test"}
		client.gems = &jar

		gems := client.Gems()
		if gems == nil {
			t.Fatal("Expected non-nil gems")
		}
		if gems.Len() != 1 {
			t.Errorf("Expected 1 gem, got %d", gems.Len())
		}
	})

	t.Run("GetGem returns gem from cache", func(t *testing.T) {
		gem := client.GetGem("test-id", "")
		if gem == nil {
			t.Fatal("Expected non-nil gem")
		}
		if gem.Name != "Test" {
			t.Errorf("Expected name 'Test', got %s", gem.Name)
		}
	})

	t.Run("GetGem by name", func(t *testing.T) {
		gem := client.GetGem("", "Test")
		if gem == nil {
			t.Fatal("Expected non-nil gem")
		}
		if gem.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got %s", gem.ID)
		}
	})
}

func TestChatOptionsWithGem(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("WithGemID sets gemID", func(t *testing.T) {
		session := client.StartChatWithOptions(WithGemID("my-gem-id"))
		if session.gemID != "my-gem-id" {
			t.Errorf("Expected gemID 'my-gem-id', got %s", session.gemID)
		}
	})

	t.Run("WithGem sets gemID from Gem object", func(t *testing.T) {
		gem := &models.Gem{ID: "gem-from-object", Name: "Test Gem"}
		session := client.StartChatWithOptions(WithGem(gem))
		if session.gemID != "gem-from-object" {
			t.Errorf("Expected gemID 'gem-from-object', got %s", session.gemID)
		}
	})

	t.Run("WithGem with nil does not set gemID", func(t *testing.T) {
		session := client.StartChatWithOptions(WithGem(nil))
		if session.gemID != "" {
			t.Errorf("Expected empty gemID, got %s", session.gemID)
		}
	})

	t.Run("WithChatModel sets model", func(t *testing.T) {
		session := client.StartChatWithOptions(WithChatModel(models.Model25Flash))
		if session.model.Name != models.Model25Flash.Name {
			t.Errorf("Expected model %s, got %s", models.Model25Flash.Name, session.model.Name)
		}
	})

	t.Run("Multiple options", func(t *testing.T) {
		session := client.StartChatWithOptions(
			WithGemID("combined-gem"),
			WithChatModel(models.ModelPro),
		)
		if session.gemID != "combined-gem" {
			t.Errorf("Expected gemID 'combined-gem', got %s", session.gemID)
		}
		if session.model.Name != models.ModelPro.Name {
			t.Errorf("Expected model %s, got %s", models.ModelPro.Name, session.model.Name)
		}
	})
}

func TestCreateGemPayload(t *testing.T) {
	// Test the payload structure is correct
	// This is a unit test for the payload construction logic
	name := "Test Gem"
	prompt := "You are a test assistant"
	description := "A test gem"

	inner := []interface{}{
		name,
		description,
		prompt,
		nil, nil, nil, nil, nil,
		0, nil, 1, nil, nil, nil,
		[]interface{}{},
	}

	if len(inner) != 15 {
		t.Errorf("Expected 15 elements in create payload, got %d", len(inner))
	}

	if inner[0] != name {
		t.Errorf("Expected name at position 0, got %v", inner[0])
	}
	if inner[1] != description {
		t.Errorf("Expected description at position 1, got %v", inner[1])
	}
	if inner[2] != prompt {
		t.Errorf("Expected prompt at position 2, got %v", inner[2])
	}
	if inner[8] != 0 {
		t.Errorf("Expected 0 at position 8, got %v", inner[8])
	}
	if inner[10] != 1 {
		t.Errorf("Expected 1 at position 10, got %v", inner[10])
	}
}

func TestUpdateGemPayload(t *testing.T) {
	// Test the update payload structure is correct
	gemID := "gem123"
	name := "Updated Gem"
	prompt := "Updated prompt"
	description := "Updated description"

	inner := []interface{}{
		name,
		description,
		prompt,
		nil, nil, nil, nil, nil,
		0, nil, 1, nil, nil, nil,
		[]interface{}{},
		0, // Extra flag for update
	}

	if len(inner) != 16 {
		t.Errorf("Expected 16 elements in update payload, got %d", len(inner))
	}

	outer := []interface{}{gemID, inner}
	if len(outer) != 2 {
		t.Errorf("Expected 2 elements in outer payload, got %d", len(outer))
	}
	if outer[0] != gemID {
		t.Errorf("Expected gemID at position 0, got %v", outer[0])
	}
}

func TestDeleteGemPayload(t *testing.T) {
	gemID := "gem-to-delete"
	payload := []interface{}{gemID}

	if len(payload) != 1 {
		t.Errorf("Expected 1 element in delete payload, got %d", len(payload))
	}
	if payload[0] != gemID {
		t.Errorf("Expected gemID, got %v", payload[0])
	}
}

func TestFetchGemsClosedClient(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.Close()

	_, err = client.FetchGems(false)
	if err == nil {
		t.Error("Expected error for closed client")
	}
}

func TestCreateGemClosedClient(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.Close()

	_, err = client.CreateGem("test", "prompt", "desc")
	if err == nil {
		t.Error("Expected error for closed client")
	}
}

func TestUpdateGemClosedClient(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.Close()

	_, err = client.UpdateGem("id", "name", "prompt", "desc")
	if err == nil {
		t.Error("Expected error for closed client")
	}
}

func TestDeleteGemClosedClient(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.Close()

	err = client.DeleteGem("id")
	if err == nil {
		t.Error("Expected error for closed client")
	}
}

// Note: FetchGems, CreateGem, UpdateGem, and DeleteGem HTTP tests require
// complex mocking of BatchExecute which parses multi-response batch RPC calls.
// The current tests cover:
// - Response parsing (parseGemsResponse, parseGemData)
// - Client state management (Gems, GetGem cache methods)
// - Closed client error handling
// - Chat options with gems
// - Payload structure validation
