package models

import (
	"testing"
)

func TestAllModels(t *testing.T) {
	models := AllModels()

	if len(models) == 0 {
		t.Error("Expected at least one model")
	}

	// Check that all models have required fields
	for _, model := range models {
		if model.Name == "" {
			t.Error("Model name should not be empty")
		}
		if model.Header == nil {
			t.Error("Model header should not be nil")
		}
	}
}

func TestModelFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected Model
	}{
		{"gemini-2.5-flash", Model25Flash},
		{"gemini-3.0-pro", Model30Pro},
		{"invalid-model", ModelUnspecified},
		{"", ModelUnspecified},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := ModelFromName(tt.name)

			if model.Name != tt.expected.Name {
				t.Errorf("ModelFromName(%s) = %v, want %v", tt.name, model.Name, tt.expected.Name)
			}
		})
	}
}

func TestDefaultHeaders(t *testing.T) {
	headers := DefaultHeaders()

	if len(headers) == 0 {
		t.Error("Expected at least one default header")
	}

	// Check for required headers
	requiredHeaders := []string{
		"User-Agent",
		"Content-Type",
		"Host",
		"Origin",
		"Referer",
	}

	for _, required := range requiredHeaders {
		if _, exists := headers[required]; !exists {
			t.Errorf("Missing required header: %s", required)
		}
	}
}

func TestRotateCookiesHeaders(t *testing.T) {
	headers := RotateCookiesHeaders()

	if len(headers) == 0 {
		t.Error("Expected at least one rotate cookies header")
	}

	// Check for required headers for cookie rotation
	requiredHeaders := []string{
		"Content-Type",
	}

	for _, required := range requiredHeaders {
		if _, exists := headers[required]; !exists {
			t.Errorf("Missing required header for cookie rotation: %s", required)
		}
	}
}

func TestModelOutput_Text(t *testing.T) {
	tests := []struct {
		name     string
		output   ModelOutput
		expected string
	}{
		{
			name: "single candidate",
			output: ModelOutput{
				Candidates: []Candidate{{Text: "Hello world"}},
				Chosen:     0,
			},
			expected: "Hello world",
		},
		{
			name: "multiple candidates",
			output: ModelOutput{
				Candidates: []Candidate{
					{Text: "First response"},
					{Text: "Second response"},
				},
				Chosen: 1,
			},
			expected: "Second response",
		},
		{
			name: "no candidates",
			output: ModelOutput{
				Candidates: []Candidate{},
				Chosen:     0,
			},
			expected: "",
		},
		{
			name: "chosen index out of bounds",
			output: ModelOutput{
				Candidates: []Candidate{{Text: "Only response"}},
				Chosen:     5,
			},
			expected: "Only response", // Returns first candidate when out of bounds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.Text()
			if result != tt.expected {
				t.Errorf("Text() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestModelOutput_Thoughts(t *testing.T) {
	tests := []struct {
		name     string
		output   ModelOutput
		expected string
	}{
		{
			name: "single candidate with thoughts",
			output: ModelOutput{
				Candidates: []Candidate{{Thoughts: "Thinking..."}},
				Chosen:     0,
			},
			expected: "Thinking...",
		},
		{
			name: "multiple candidates with thoughts",
			output: ModelOutput{
				Candidates: []Candidate{
					{Thoughts: "First thought"},
					{Thoughts: "Second thought"},
				},
				Chosen: 1,
			},
			expected: "Second thought",
		},
		{
			name: "no thoughts",
			output: ModelOutput{
				Candidates: []Candidate{{Thoughts: ""}},
				Chosen:     0,
			},
			expected: "",
		},
		{
			name: "no candidates",
			output: ModelOutput{
				Candidates: []Candidate{},
				Chosen:     0,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.Thoughts()
			if result != tt.expected {
				t.Errorf("Thoughts() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestModelOutput_RCID(t *testing.T) {
	tests := []struct {
		name     string
		output   ModelOutput
		expected string
	}{
		{
			name: "single candidate",
			output: ModelOutput{
				Candidates: []Candidate{{RCID: "rcid123"}},
				Chosen:     0,
			},
			expected: "rcid123",
		},
		{
			name: "multiple candidates",
			output: ModelOutput{
				Candidates: []Candidate{
					{RCID: "rcid456"},
					{RCID: "rcid789"},
				},
				Chosen: 1,
			},
			expected: "rcid789",
		},
		{
			name: "no candidates",
			output: ModelOutput{
				Candidates: []Candidate{},
				Chosen:     0,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.RCID()
			if result != tt.expected {
				t.Errorf("RCID() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestModelOutput_Images(t *testing.T) {
	tests := []struct {
		name     string
		output   ModelOutput
		expected int
	}{
		{
			name: "single candidate with images",
			output: ModelOutput{
				Candidates: []Candidate{
					{
						WebImages:       []WebImage{{URL: "http://example.com/1.jpg"}},
						GeneratedImages: []GeneratedImage{{URL: "http://example.com/gen1.jpg"}},
					},
				},
				Chosen: 0,
			},
			expected: 2,
		},
		{
			name: "multiple candidates",
			output: ModelOutput{
				Candidates: []Candidate{
					{WebImages: []WebImage{{URL: "http://example.com/1.jpg"}}},
					{GeneratedImages: []GeneratedImage{{URL: "http://example.com/gen1.jpg"}}},
				},
				Chosen: 1,
			},
			expected: 1,
		},
		{
			name: "no images",
			output: ModelOutput{
				Candidates: []Candidate{{}},
				Chosen:     0,
			},
			expected: 0,
		},
		{
			name: "no candidates",
			output: ModelOutput{
				Candidates: []Candidate{},
				Chosen:     0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.Images()
			if len(result) != tt.expected {
				t.Errorf("Images() = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestModelOutput_CID(t *testing.T) {
	tests := []struct {
		name     string
		output   ModelOutput
		expected string
	}{
		{
			name: "with metadata",
			output: ModelOutput{
				Metadata: []string{"cid123", "rid456", "rcid789"},
			},
			expected: "cid123",
		},
		{
			name: "empty metadata",
			output: ModelOutput{
				Metadata: []string{},
			},
			expected: "",
		},
		{
			name: "nil metadata",
			output: ModelOutput{
				Metadata: nil,
			},
			expected: "",
		},
		{
			name: "insufficient metadata",
			output: ModelOutput{
				Metadata: []string{"cid123"},
			},
			expected: "cid123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.CID()
			if result != tt.expected {
				t.Errorf("CID() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestModelOutput_RID(t *testing.T) {
	tests := []struct {
		name     string
		output   ModelOutput
		expected string
	}{
		{
			name: "with metadata",
			output: ModelOutput{
				Metadata: []string{"cid123", "rid456", "rcid789"},
			},
			expected: "rid456",
		},
		{
			name: "empty metadata",
			output: ModelOutput{
				Metadata: []string{},
			},
			expected: "",
		},
		{
			name: "nil metadata",
			output: ModelOutput{
				Metadata: nil,
			},
			expected: "",
		},
		{
			name: "insufficient metadata",
			output: ModelOutput{
				Metadata: []string{"cid123"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.RID()
			if result != tt.expected {
				t.Errorf("RID() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestUploadHeaders(t *testing.T) {
	headers := UploadHeaders()

	if len(headers) == 0 {
		t.Error("Expected at least one upload header")
	}

	// Check for Push-ID header which is required for uploads
	if _, exists := headers["Push-ID"]; !exists {
		t.Error("Missing required header: Push-ID")
	}
}
