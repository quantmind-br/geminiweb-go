package api

import (
	"errors"
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/tidwall/gjson"

	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// TestBuildPayload tests the buildPayload function
func TestBuildPayload(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		metadata []string
		images   []*UploadedImage
		wantErr  bool
	}{
		{
			name:     "simple prompt",
			prompt:   "Hello, Gemini!",
			metadata: nil,
			images:   nil,
			wantErr:  false,
		},
		{
			name:     "prompt with metadata",
			prompt:   "Continue the conversation",
			metadata: []string{"cid123", "rid456", "rcid789"},
			images:   nil,
			wantErr:  false,
		},
		{
			name:     "prompt with images",
			prompt:   "Describe this image",
			metadata: nil,
			images: []*UploadedImage{
				{ResourceID: "img_123", FileName: "test.jpg", MIMEType: "image/jpeg"},
			},
			wantErr: false,
		},
		{
			name:     "empty prompt is allowed in buildPayload (validation happens in GenerateContent)",
			prompt:   "",
			metadata: nil,
			images:   nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildPayload(tt.prompt, tt.metadata)

			if tt.wantErr {
				if err == nil {
					t.Errorf("buildPayload() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("buildPayload() unexpected error: %v", err)
				return
			}

			if got == "" {
				t.Errorf("buildPayload() returned empty string")
			}

			// Verify the JSON is valid
			if !gjson.Valid(got) {
				t.Errorf("buildPayload() returned invalid JSON")
			}
		})
	}
}

// TestParseResponse tests the parseResponse function with various scenarios
func TestParseResponse(t *testing.T) {
	// Helper to build test response body with properly escaped JSON
	// Response structure (based on paths.go):
	// - Outer: [[..., ..., bodyString, ...], ...]
	// - Body at index 2 (PathBody) contains JSON string
	// - Body JSON: [?, metadata(1), ?, ?, candidates(4), ...]
	// - Candidate: ["rcid"(0), ["text"](1.0), ...]
	makeBody := func(innerJSON string) []byte {
		escaped := ""
		for _, c := range innerJSON {
			if c == '"' {
				escaped += `\"`
			} else if c == '\\' {
				escaped += `\\`
			} else {
				escaped += string(c)
			}
		}
		// Structure: [[null, null, "bodyJSON"]]
		return []byte(`[[null, null, "` + escaped + `"]]`)
	}

	tests := []struct {
		name      string
		body      []byte
		modelName string
		wantErr   bool
		check     func(*testing.T, *models.ModelOutput)
	}{
		{
			name:      "empty body response",
			body:      []byte("garbage text with no valid json"),
			modelName: "gemini-2.5-flash",
			wantErr:   true,
		},
		{
			name:      "no response body found",
			body:      []byte(`[[null, null, "invalid"]]`),
			modelName: "gemini-2.5-flash",
			wantErr:   true,
		},
		{
			name: "valid response with single candidate",
			// Body JSON: [?, ["cid","rid","rcid"], ?, ?, [candidate]]
			// Candidate: ["rcid", ["Hello"], ...]
			body:      makeBody(`[null,["cid123","rid456","rcid789"],null,null,[["rcid789",["Hello"]]]]`),
			modelName: "gemini-2.5-flash",
			wantErr:   false,
			check: func(t *testing.T, output *models.ModelOutput) {
				if len(output.Candidates) != 1 {
					t.Errorf("expected 1 candidate, got %d", len(output.Candidates))
				}
				cand := output.Candidates[0]
				if cand.RCID != "rcid789" {
					t.Errorf("RCID = %s, want rcid789", cand.RCID)
				}
				if cand.Text != "Hello" {
					t.Errorf("Text = %s, want Hello", cand.Text)
				}
			},
		},
		{
			name:      "response with metadata",
			body:      makeBody(`[null,["mycid","myrid","myrcid"],null,null,[["myrcid",["Response"]]]]`),
			modelName: "gemini-2.5-pro",
			wantErr:   false,
			check: func(t *testing.T, output *models.ModelOutput) {
				if len(output.Metadata) < 2 {
					t.Errorf("expected at least 2 metadata items, got %d", len(output.Metadata))
				}
				if output.Metadata[0] != "mycid" {
					t.Errorf("Metadata[0] = %s, want mycid", output.Metadata[0])
				}
			},
		},
		{
			name: "error code 1037 - usage limit exceeded",
			body: []byte(`[6, 1037]`),
			modelName: "gemini-2.5-flash",
			wantErr: true,
			check: func(t *testing.T, output *models.ModelOutput) {
				t.Error("should return error for usage limit")
			},
		},
		{
			name: "error code 1050 - model inconsistent",
			body: []byte(`[6, 1050]`),
			modelName: "gemini-2.5-pro",
			wantErr: true,
			check: func(t *testing.T, output *models.ModelOutput) {
				t.Error("should return error for model inconsistent")
			},
		},
		{
			name: "error code 1052 - model header invalid",
			body: []byte(`[6, 1052]`),
			modelName: "gemini-2.5-flash",
			wantErr: true,
			check: func(t *testing.T, output *models.ModelOutput) {
				t.Error("should return error for model header invalid")
			},
		},
		{
			name: "error code 1060 - IP blocked",
			body: []byte(`[6, 1060]`),
			modelName: "gemini-2.5-flash",
			wantErr: true,
			check: func(t *testing.T, output *models.ModelOutput) {
				t.Error("should return error for IP blocked")
			},
		},
		{
			name: "unknown error code",
			body: []byte(`[6, 9999]`),
			modelName: "gemini-2.5-flash",
			wantErr: true,
			check: func(t *testing.T, output *models.ModelOutput) {
				t.Error("should return error for unknown code")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResponse(tt.body, tt.modelName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseResponse() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseResponse() unexpected error: %v", err)
				return
			}

			if got == nil {
				t.Errorf("parseResponse() returned nil")
				return
			}

			// Verify structure
			if got.Candidates == nil {
				t.Errorf("parseResponse() returned nil candidates")
			}

			if len(got.Candidates) == 0 {
				t.Errorf("parseResponse() returned empty candidates")
			}

			// Run additional checks if provided
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// TestGenerateContent tests the GenerateContent function
func TestGenerateContent(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	// Helper to build test response body (same as in TestParseResponse)
	makeBody := func(innerJSON string) []byte {
		escaped := ""
		for _, c := range innerJSON {
			if c == '"' {
				escaped += `\"`
			} else if c == '\\' {
				escaped += `\\`
			} else {
				escaped += string(c)
			}
		}
		return []byte(`[[null, null, "` + escaped + `"]]`)
	}

	tests := []struct {
		name         string
		prompt       string
		opts         *GenerateOptions
		setupMock    func(*MockHttpClient)
		expectedErr  bool
		clientClosed bool
	}{
		{
			name:        "empty prompt",
			prompt:      "",
			opts:        nil,
			setupMock:   func(m *MockHttpClient) {},
			expectedErr: true,
		},
		{
			name:         "client closed",
			prompt:       "test",
			opts:         nil,
			setupMock:    func(m *MockHttpClient) {},
			expectedErr:  true,
			clientClosed: true,
		},
		{
			name:   "network error",
			prompt: "test",
			opts:   nil,
			setupMock: func(m *MockHttpClient) {
				m.Err = errors.New("network connection failed")
				m.Response = nil
			},
			expectedErr: true,
		},
		{
			name:   "status code != 200",
			prompt: "test",
			opts:   nil,
			setupMock: func(m *MockHttpClient) {
				body := NewMockResponseBody([]byte(""))
				m.Response = &fhttp.Response{
					StatusCode: 500,
					Body:       body,
					Header:     make(fhttp.Header),
				}
			},
			expectedErr: true,
		},
		{
			name:   "successful generation",
			prompt: "test prompt",
			opts: &GenerateOptions{
				Model: models.Model25Flash,
			},
			setupMock: func(m *MockHttpClient) {
				// Response format: [[null, null, bodyJSON]]
				// Body JSON: [?, ["cid","rid","rcid"], ?, ?, [candidate]]
				innerJSON := `[null,["cid123","rid456","rcid789"],null,null,[["rcid789",["test response"]]]]`
				responseBody := makeBody(innerJSON)
				body := NewMockResponseBody(responseBody)
				m.Response = &fhttp.Response{
					StatusCode: 200,
					Body:       body,
					Header:     make(fhttp.Header),
				}
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHttpClient{}
			tt.setupMock(mockClient)

			// Create a test client with the mock
			client := &GeminiClient{
				httpClient:  mockClient,
				cookies:     validCookies,
				model:       models.Model25Flash,
				accessToken: "test_token",
				closed:      false,
			}

			if tt.clientClosed {
				client.closed = true
			}

			got, err := client.GenerateContent(tt.prompt, tt.opts)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("GenerateContent() expected error but got none")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateContent() unexpected error: %v", err)
				return
			}

			if got == nil {
				t.Errorf("GenerateContent() returned nil")
			}
		})
	}
}

// TestHandleErrorCode tests the handleErrorCode function
func TestHandleErrorCode(t *testing.T) {
	tests := []struct {
		name      string
		code      models.ErrorCode
		modelName string
		errType   error
	}{
		{
			name:      "error 1037 usage limit exceeded",
			code:      models.ErrUsageLimitExceeded,
			modelName: "gemini-2.5-flash",
			errType:   &apierrors.UsageLimitError{},
		},
		{
			name:      "error 1050 model inconsistent",
			code:      models.ErrModelInconsistent,
			modelName: "gemini-2.5-pro",
			errType:   &apierrors.ModelError{},
		},
		{
			name:      "error 1052 model header invalid",
			code:      models.ErrModelHeaderInvalid,
			modelName: "gemini-2.5-flash",
			errType:   &apierrors.ModelError{},
		},
		{
			name:      "error 1060 IP blocked",
			code:      models.ErrIPBlocked,
			modelName: "gemini-2.5-flash",
			errType:   &apierrors.BlockedError{},
		},
		{
			name:      "unknown error code",
			code:      9999,
			modelName: "gemini-2.5-flash",
			errType:   &apierrors.APIError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleErrorCode(tt.code, tt.modelName)

			if got == nil {
				t.Errorf("handleErrorCode() returned nil")
				return
			}

			if tt.errType != nil {
				switch tt.errType.(type) {
				case *apierrors.UsageLimitError:
					if _, ok := got.(*apierrors.UsageLimitError); !ok {
						t.Errorf("handleErrorCode() error type = %T, want %T", got, tt.errType)
					}
				case *apierrors.ModelError:
					if _, ok := got.(*apierrors.ModelError); !ok {
						t.Errorf("handleErrorCode() error type = %T, want %T", got, tt.errType)
					}
				case *apierrors.BlockedError:
					if _, ok := got.(*apierrors.BlockedError); !ok {
						t.Errorf("handleErrorCode() error type = %T, want %T", got, tt.errType)
					}
				case *apierrors.APIError:
					if _, ok := got.(*apierrors.APIError); !ok {
						t.Errorf("handleErrorCode() error type = %T, want %T", got, tt.errType)
					}
				}
			}

			// Verify error message is not empty
			if got.Error() == "" {
				t.Errorf("handleErrorCode() returned empty error message")
			}
		})
	}
}

// TestBuildPayloadWithImagesExtended tests buildPayloadWithImages with various image combinations
func TestBuildPayloadWithImagesExtended(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		metadata []string
		images   []*UploadedImage
	}{
		{
			name:     "single image",
			prompt:   "Describe this image",
			metadata: nil,
			images: []*UploadedImage{
				{ResourceID: "img1", FileName: "test1.jpg", MIMEType: "image/jpeg"},
			},
		},
		{
			name:     "multiple images",
			prompt:   "Compare these images",
			metadata: nil,
			images: []*UploadedImage{
				{ResourceID: "img1", FileName: "test1.jpg", MIMEType: "image/jpeg"},
				{ResourceID: "img2", FileName: "test2.png", MIMEType: "image/png"},
			},
		},
		{
			name:     "no images",
			prompt:   "Just text",
			metadata: []string{"cid"},
			images:   nil,
		},
		{
			name:     "images with metadata",
			prompt:   "Continue the conversation",
			metadata: []string{"cid123", "rid456"},
			images: []*UploadedImage{
				{ResourceID: "img1", FileName: "test.jpg", MIMEType: "image/jpeg"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildPayloadWithImages(tt.prompt, tt.metadata, tt.images)
			if err != nil {
				t.Errorf("buildPayloadWithImages() unexpected error: %v", err)
				return
			}

			if !gjson.Valid(got) {
				t.Errorf("buildPayloadWithImages() returned invalid JSON")
				return
			}

			// Verify the structure
			parsed := gjson.Parse(got)
			if !parsed.IsArray() || len(parsed.Array()) != 2 {
				t.Errorf("buildPayloadWithImages() returned invalid structure")
			}
		})
	}
}

// TestIsAuthError tests the isAuthError function
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "APIError with 401",
			err: &apierrors.APIError{
				StatusCode: 401,
				Message:    "Unauthorized",
			},
			expected: true,
		},
		{
			name: "APIError with 200",
			err: &apierrors.APIError{
				StatusCode: 200,
				Message:    "OK",
			},
			expected: false,
		},
		{
			name: "APIError with 500",
			err: &apierrors.APIError{
				StatusCode: 500,
				Message:    "Internal Server Error",
			},
			expected: false,
		},
		{
			name:     "AuthError",
			err:      &apierrors.AuthError{},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("isAuthError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
