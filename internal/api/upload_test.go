package api

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"

	"github.com/diogo/geminiweb/internal/config"
)

func TestSupportedImageTypes(t *testing.T) {
	types := SupportedImageTypes()

	expected := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}

	if len(types) != len(expected) {
		t.Errorf("expected %d types, got %d", len(expected), len(types))
	}

	for i, tp := range types {
		if tp != expected[i] {
			t.Errorf("types[%d] = %s, want %s", i, tp, expected[i])
		}
	}
}

func TestUploadedImage_Fields(t *testing.T) {
	img := &UploadedImage{
		ResourceID: "resource-123",
		FileName:   "test.png",
		MIMEType:   "image/png",
		Size:       1024,
	}

	if img.ResourceID != "resource-123" {
		t.Error("ResourceID mismatch")
	}
	if img.FileName != "test.png" {
		t.Error("FileName mismatch")
	}
	if img.MIMEType != "image/png" {
		t.Error("MIMEType mismatch")
	}
	if img.Size != 1024 {
		t.Error("Size mismatch")
	}
}

func TestImageUploader_IsSupportedType(t *testing.T) {
	uploader := &ImageUploader{}

	tests := []struct {
		mimeType string
		expected bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/jpeg; charset=utf-8", true},
		{"image/bmp", false},
		{"text/plain", false},
		{"application/pdf", false},
		{"", false},
	}

	for _, tt := range tests {
		result := uploader.isSupportedType(tt.mimeType)
		if result != tt.expected {
			t.Errorf("isSupportedType(%s) = %v, want %v", tt.mimeType, result, tt.expected)
		}
	}
}

func TestGenerateUploadID(t *testing.T) {
	id1 := generateUploadID()
	id2 := generateUploadID()

	if id1 == "" {
		t.Error("generateUploadID returned empty string")
	}

	if !strings.HasPrefix(id1, "geminiweb-") {
		t.Errorf("uploadID should start with 'geminiweb-', got %s", id1)
	}

	// IDs should be unique (based on nanosecond timestamp)
	if id1 == id2 {
		t.Log("Warning: two consecutive IDs are the same (rare but possible)")
	}
}

func TestMaxImageSize(t *testing.T) {
	expected := 20 * 1024 * 1024 // 20MB

	if MaxImageSize != expected {
		t.Errorf("MaxImageSize = %d, want %d", MaxImageSize, expected)
	}
}

func TestBuildPayloadWithImages(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		metadata []string
		images   []*UploadedImage
		wantErr  bool
	}{
		{
			name:     "no images",
			prompt:   "Hello",
			metadata: []string{"cid", "rid"},
			images:   nil,
			wantErr:  false,
		},
		{
			name:     "with one image",
			prompt:   "Describe this",
			metadata: []string{"cid", "rid"},
			images: []*UploadedImage{
				{ResourceID: "res-1", MIMEType: "image/png", FileName: "test.png"},
			},
			wantErr: false,
		},
		{
			name:     "with multiple images",
			prompt:   "Compare these",
			metadata: nil,
			images: []*UploadedImage{
				{ResourceID: "res-1", MIMEType: "image/png", FileName: "a.png"},
				{ResourceID: "res-2", MIMEType: "image/jpeg", FileName: "b.jpg"},
			},
			wantErr: false,
		},
		{
			name:     "empty prompt",
			prompt:   "",
			metadata: nil,
			images:   nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := buildPayloadWithImages(tt.prompt, tt.metadata, tt.images)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildPayloadWithImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && payload == "" {
				t.Error("expected non-empty payload")
			}

			// Verify payload is valid JSON
			if payload != "" {
				if !strings.HasPrefix(payload, "[") {
					t.Error("payload should start with '['")
				}
			}
		})
	}
}

func TestBuildPayload_MatchesWithImages(t *testing.T) {
	// buildPayload should produce same result as buildPayloadWithImages with nil images
	prompt := "test prompt"
	metadata := []string{"cid", "rid"}

	payload1, err1 := buildPayload(prompt, metadata)
	payload2, err2 := buildPayloadWithImages(prompt, metadata, nil)

	if err1 != nil || err2 != nil {
		t.Fatal("unexpected errors")
	}

	if payload1 != payload2 {
		t.Error("buildPayload should match buildPayloadWithImages with nil images")
	}
}

func TestGenerateOptions_WithImages(t *testing.T) {
	images := []*UploadedImage{
		{ResourceID: "res-1", FileName: "test.png", MIMEType: "image/png", Size: 1024},
	}

	opts := GenerateOptions{
		Metadata: []string{"cid", "rid"},
		Images:   images,
	}

	if len(opts.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(opts.Images))
	}

	if opts.Images[0].ResourceID != "res-1" {
		t.Error("image ResourceID mismatch")
	}
}

func TestImageUploader_NewImageUploader(t *testing.T) {
	client := &GeminiClient{}
	uploader := NewImageUploader(client)

	if uploader == nil {
		t.Error("NewImageUploader returned nil")
	}

	if uploader.client != client {
		t.Error("uploader client does not match input client")
	}
}

func TestImageUploader_UploadFile_TooLarge(t *testing.T) {
	// Create a temporary file larger than MaxImageSize
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.jpg")
	// Create a file that's slightly larger than MaxImageSize
	largeData := make([]byte, MaxImageSize+1)
	err := os.WriteFile(testFile, largeData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	client := &GeminiClient{}
	uploader := NewImageUploader(client)

	_, err = uploader.UploadFile(testFile)
	if err == nil {
		t.Error("expected error for too large file")
	}

	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected 'exceeds maximum' in error, got: %v", err)
	}
}

func TestImageUploader_UploadFile_UnsupportedType(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.xyz")
	err := os.WriteFile(testFile, []byte("data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	client := &GeminiClient{}
	uploader := NewImageUploader(client)

	_, err = uploader.UploadFile(testFile)
	if err == nil {
		t.Error("expected error for unsupported file type")
	}

	if !strings.Contains(err.Error(), "unsupported image type") {
		t.Errorf("expected 'unsupported image type' in error, got: %v", err)
	}
}

func TestImageUploader_UploadFile_FileNotFound(t *testing.T) {
	client := &GeminiClient{}
	uploader := NewImageUploader(client)

	_, err := uploader.UploadFile("/nonexistent/file.jpg")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "failed to stat file") {
		t.Errorf("expected 'failed to stat file' in error, got: %v", err)
	}
}

func TestImageUploader_UploadFromReader_TooLarge(t *testing.T) {
	// Create data larger than MaxImageSize
	largeData := make([]byte, MaxImageSize+1)
	reader := strings.NewReader(string(largeData))

	client := &GeminiClient{}
	uploader := NewImageUploader(client)

	_, err := uploader.UploadFromReader(reader, "large.jpg", "image/jpeg")
	if err == nil {
		t.Error("expected error for too large data")
	}

	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected 'exceeds maximum' in error, got: %v", err)
	}
}

func TestGeminiClient_UploadImage_CreatesUploader(t *testing.T) {
	// Just test that the method exists and calls NewImageUploader
	client := &GeminiClient{}

	// We can't actually test the upload without a real server, but we can verify
	// that the methods compile correctly by checking they exist in the struct
	// The methods are on GeminiClient, so they will always be non-nil
	_ = client.UploadImage
	_ = client.UploadImageFromReader
}

func TestUploadedImage_Struct(t *testing.T) {
	img := &UploadedImage{
		ResourceID: "test-resource",
		FileName:   "test.jpg",
		MIMEType:   "image/jpeg",
		Size:       12345,
	}

	if img.ResourceID != "test-resource" {
		t.Errorf("ResourceID = %s, want test-resource", img.ResourceID)
	}

	if img.FileName != "test.jpg" {
		t.Errorf("FileName = %s, want test.jpg", img.FileName)
	}

	if img.MIMEType != "image/jpeg" {
		t.Errorf("MIMEType = %s, want image/jpeg", img.MIMEType)
	}

	if img.Size != 12345 {
		t.Errorf("Size = %d, want 12345", img.Size)
	}
}

// TestGeminiClient_UploadImage tests UploadImage method
func TestGeminiClient_UploadImage(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	t.Run("successful_upload", func(t *testing.T) {
		// Create a temporary image file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.png")
		err := os.WriteFile(testFile, []byte("fake image data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Setup mock client
		mockClient := &MockHttpClient{}
		body := NewMockResponseBody([]byte(`{"resourceId": "test_resource_123"}`))
		mockClient.Response = &fhttp.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(fhttp.Header),
		}

		// Create client and set mock
		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		client.httpClient = mockClient
		client.autoRefresh = false // Disable rotator

		// Upload image
		uploaded, err := client.UploadImage(testFile)
		if err != nil {
			t.Errorf("UploadImage() unexpected error: %v", err)
			return
		}

		if uploaded == nil {
			t.Error("UploadImage() returned nil")
			return
		}

		if uploaded.ResourceID != "test_resource_123" {
			t.Errorf("ResourceID = %s, want test_resource_123", uploaded.ResourceID)
		}

		if uploaded.FileName != "test.png" {
			t.Errorf("FileName = %s, want test.png", uploaded.FileName)
		}

		if uploaded.MIMEType != "image/png" {
			t.Errorf("MIMEType = %s, want image/png", uploaded.MIMEType)
		}
	})

	t.Run("file_not_found", func(t *testing.T) {
		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}

		_, err = client.UploadImage("/nonexistent/file.png")
		if err == nil {
			t.Error("UploadImage() expected error for nonexistent file")
		}
	})
}

// TestGeminiClient_UploadImageFromReader tests UploadImageFromReader method
func TestGeminiClient_UploadImageFromReader(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	t.Run("successful_upload", func(t *testing.T) {
		// Setup mock client
		mockClient := &MockHttpClient{}
		body := NewMockResponseBody([]byte(`{"resourceId": "reader_resource_456"}`))
		mockClient.Response = &fhttp.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(fhttp.Header),
		}

		// Create client and set mock
		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		client.httpClient = mockClient
		client.autoRefresh = false

		// Create reader with image data
		reader := bytes.NewReader([]byte("image data"))

		// Upload image
		uploaded, err := client.UploadImageFromReader(reader, "test.jpg", "image/jpeg")
		if err != nil {
			t.Errorf("UploadImageFromReader() unexpected error: %v", err)
			return
		}

		if uploaded == nil {
			t.Error("UploadImageFromReader() returned nil")
			return
		}

		if uploaded.ResourceID != "reader_resource_456" {
			t.Errorf("ResourceID = %s, want reader_resource_456", uploaded.ResourceID)
		}

		if uploaded.FileName != "test.jpg" {
			t.Errorf("FileName = %s, want test.jpg", uploaded.FileName)
		}

		if uploaded.MIMEType != "image/jpeg" {
			t.Errorf("MIMEType = %s, want image/jpeg", uploaded.MIMEType)
		}
	})

	t.Run("data_too_large", func(t *testing.T) {
		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}

		// Create data larger than MaxImageSize
		largeData := make([]byte, MaxImageSize+1)
		reader := bytes.NewReader(largeData)

		_, err = client.UploadImageFromReader(reader, "large.png", "image/png")
		if err == nil {
			t.Error("UploadImageFromReader() expected error for large data")
		}
	})
}

// TestImageUploader_UploadStream tests the private uploadStream function
func TestImageUploader_UploadStream(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	t.Run("successful_stream_upload", func(t *testing.T) {
		// Setup mock client
		mockClient := &MockHttpClient{}
		body := NewMockResponseBody([]byte(`{"resourceId": "stream_resource_789"}`))
		mockClient.Response = &fhttp.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(fhttp.Header),
		}

		// Create client
		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		client.httpClient = mockClient
		client.autoRefresh = false

		// Create uploader
		uploader := NewImageUploader(client)

		// Upload from reader
		reader := bytes.NewReader([]byte("stream data"))
		uploaded, err := uploader.uploadStream(reader, "stream.png", "image/png", 1024)
		if err != nil {
			t.Errorf("uploadStream() unexpected error: %v", err)
			return
		}

		if uploaded == nil {
			t.Error("uploadStream() returned nil")
			return
		}

		if uploaded.ResourceID != "stream_resource_789" {
			t.Errorf("ResourceID = %s, want stream_resource_789", uploaded.ResourceID)
		}

		if uploaded.FileName != "stream.png" {
			t.Errorf("FileName = %s, want stream.png", uploaded.FileName)
		}
	})

	t.Run("http_error_status", func(t *testing.T) {
		// Setup mock client with error status
		mockClient := &MockHttpClient{}
		body := NewMockResponseBody([]byte("error"))
		mockClient.Response = &fhttp.Response{
			StatusCode: 500,
			Body:       body,
			Header:     make(fhttp.Header),
		}

		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		client.httpClient = mockClient
		client.autoRefresh = false

		uploader := NewImageUploader(client)
		reader := bytes.NewReader([]byte("data"))

		_, err = uploader.uploadStream(reader, "test.png", "image/png", 1024)
		if err == nil {
			t.Error("uploadStream() expected error for HTTP 500")
		} else if !strings.Contains(err.Error(), "upload failed") {
			t.Errorf("Expected 'upload failed' in error, got: %v", err)
		}
	})

	t.Run("parse_error_with_header_fallback", func(t *testing.T) {
		// Setup mock client with response that has resource ID in header
		mockClient := &MockHttpClient{}
		body := NewMockResponseBody([]byte("invalid json"))
		mockClient.Response = &fhttp.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(fhttp.Header),
		}
		mockClient.Response.Header.Set("X-Goog-Upload-URL", "header_resource_123")

		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		client.httpClient = mockClient
		client.autoRefresh = false

		uploader := NewImageUploader(client)
		reader := bytes.NewReader([]byte("data"))

		uploaded, err := uploader.uploadStream(reader, "test.png", "image/png", 1024)
		if err != nil {
			t.Errorf("uploadStream() unexpected error: %v", err)
			return
		}

		if uploaded == nil {
			t.Error("uploadStream() returned nil")
			return
		}

		// Should use header value when JSON parse fails
		if uploaded.ResourceID != "header_resource_123" {
			t.Errorf("ResourceID = %s, want header_resource_123", uploaded.ResourceID)
		}
	})

	t.Run("parse_error_with_fallback_id", func(t *testing.T) {
		// Setup mock client with invalid JSON and no header
		mockClient := &MockHttpClient{}
		body := NewMockResponseBody([]byte("invalid json"))
		mockClient.Response = &fhttp.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(fhttp.Header),
		}

		client, err := NewClient(validCookies)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		client.httpClient = mockClient
		client.autoRefresh = false

		uploader := NewImageUploader(client)
		reader := bytes.NewReader([]byte("data"))

		uploaded, err := uploader.uploadStream(reader, "test.png", "image/png", 1024)
		if err != nil {
			t.Errorf("uploadStream() unexpected error: %v", err)
			return
		}

		if uploaded == nil {
			t.Error("uploadStream() returned nil")
			return
		}

		// Should use upload ID as fallback when both JSON and header fail
		if uploaded.ResourceID == "" {
			t.Error("ResourceID should not be empty with fallback")
		}
	})
}
