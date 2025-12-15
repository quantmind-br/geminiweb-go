package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

// Minimal valid image fixtures
var (
	// Minimal 1x1 PNG (67 bytes)
	minimalPNG = []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
		0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59,
		0xE7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Minimal 1x1 JPEG (134 bytes)
	minimalJPEG = []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
		0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
		0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x1F, 0x00, 0x00,
		0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F,
		0x00, 0x7F, 0xFF, 0xD9,
	}

	// Minimal 1x1 WebP (30 bytes)
	minimalWebP = []byte{
		0x52, 0x49, 0x46, 0x46, 0x1A, 0x00, 0x00, 0x00, // RIFF header
		0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x4C, // WEBP VP8L
		0x0D, 0x00, 0x00, 0x00, 0x2F, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// Minimal 1x1 GIF (26 bytes)
	minimalGIF = []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a
		0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, // Logical screen descriptor
		0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // Image descriptor
		0x02, 0x02, 0x44, 0x01, 0x00, // Image data
		0x3B, // Trailer
	}
)

// validDownloadCookies returns a valid config.Cookies for download tests
func validDownloadCookies() *config.Cookies {
	return &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
}

// createTestDownloadClient creates a client with a mock HTTP client for download tests
func createTestDownloadClient(t *testing.T, mockClient *DynamicMockHttpClient) *GeminiClient {
	t.Helper()
	client, err := NewClient(validDownloadCookies(), WithAutoRefresh(false))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.httpClient = mockClient
	client.initialized = true // Mark as initialized to skip Init() requirement
	return client
}

// ============================================================================
// DefaultDownloadOptions Tests
// ============================================================================

func TestDefaultDownloadOptions(t *testing.T) {
	opts := DefaultDownloadOptions()

	// Should have a valid directory
	if opts.Directory == "" {
		t.Error("DefaultDownloadOptions().Directory should not be empty")
	}

	// Should contain .geminiweb/images
	if !strings.Contains(opts.Directory, ".geminiweb") || !strings.Contains(opts.Directory, "images") {
		t.Errorf("DefaultDownloadOptions().Directory = %s, should contain .geminiweb/images", opts.Directory)
	}

	// FullSize should be true by default
	if !opts.FullSize {
		t.Error("DefaultDownloadOptions().FullSize should be true")
	}

	// Filename should be empty (auto-generated)
	if opts.Filename != "" {
		t.Errorf("DefaultDownloadOptions().Filename = %s, want empty", opts.Filename)
	}
}

// ============================================================================
// generateFilename Tests
// ============================================================================

func TestGenerateFilename_FromContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantExt     string
	}{
		{"PNG", "image/png", ".png"},
		{"JPEG", "image/jpeg", ".jpg"},
		{"JPG variation", "image/jpg", ".jpg"},
		{"WebP", "image/webp", ".webp"},
		{"GIF", "image/gif", ".gif"},
		{"Unknown defaults to jpg", "image/unknown", ".jpg"},
		{"Empty defaults to jpg", "", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a URL without extension so content type is used
			filename := generateFilename("http://example.com/image", "", tt.contentType)
			if !strings.HasSuffix(filename, tt.wantExt) {
				t.Errorf("generateFilename() = %s, want suffix %s", filename, tt.wantExt)
			}
		})
	}
}

func TestGenerateFilename_FromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantName string
	}{
		{
			"URL with extension",
			"http://example.com/photo.png",
			"photo.png",
		},
		{
			"URL with query params",
			"http://example.com/photo.jpg?size=large",
			"photo.jpg",
		},
		{
			"URL with complex path",
			"http://example.com/path/to/image.webp",
			"image.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := generateFilename(tt.url, "", "image/jpeg")
			if filename != tt.wantName {
				t.Errorf("generateFilename() = %s, want %s", filename, tt.wantName)
			}
		})
	}
}

func TestGenerateFilename_FromTitle(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		contentType string
		wantPrefix  string
		wantExt     string
	}{
		{
			"Simple title",
			"My Photo",
			"image/png",
			"My Photo",
			".png",
		},
		{
			"Title with special chars",
			"Photo: Test/Image",
			"image/jpeg",
			"Photo_ Test_Image",
			".jpg",
		},
		{
			"Long title truncated",
			"This is a very long title that should be truncated to prevent extremely long filenames",
			"image/webp",
			"This is a very long title that should be truncated",
			".webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use URL without extension so title is used
			filename := generateFilename("http://example.com/img", tt.title, tt.contentType)

			if !strings.HasPrefix(filename, tt.wantPrefix) {
				t.Errorf("generateFilename() = %s, want prefix %s", filename, tt.wantPrefix)
			}
			if !strings.HasSuffix(filename, tt.wantExt) {
				t.Errorf("generateFilename() = %s, want suffix %s", filename, tt.wantExt)
			}
		})
	}
}

func TestGenerateFilename_Fallback(t *testing.T) {
	// No URL extension, no title -> timestamp fallback
	filename := generateFilename("http://example.com/img", "", "image/jpeg")

	if !strings.HasPrefix(filename, "image_") {
		t.Errorf("generateFilename() = %s, want prefix 'image_'", filename)
	}
	if !strings.HasSuffix(filename, ".jpg") {
		t.Errorf("generateFilename() = %s, want suffix '.jpg'", filename)
	}

	// Should contain a timestamp format
	if len(filename) < len("image_20060102_150405.jpg") {
		t.Errorf("generateFilename() = %s, too short for timestamp format", filename)
	}
}

// ============================================================================
// sanitizeFilename Tests
// ============================================================================

func TestSanitizeFilename_RemovesInvalidChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"file<>.png", "file__.png"},
		{"file:\"/\\.png", "file____.png"}, // : " / \ -> 4 underscores
		{"file|?*.png", "file___.png"},
		{"path/to/file.png", "path_to_file.png"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_PreservesValidChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.png", "normal.png"},
		{"file-name_123.jpg", "file-name_123.jpg"},
		{"My Photo (2024).webp", "My Photo (2024).webp"},
		{"файл.png", "файл.png"}, // Unicode characters preserved
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_TrimSpaces(t *testing.T) {
	got := sanitizeFilename("  spaced.png  ")
	want := "spaced.png"
	if got != want {
		t.Errorf("sanitizeFilename() = %q, want %q", got, want)
	}
}

// ============================================================================
// DownloadImage Tests
// ============================================================================

func TestDownloadImage_Success(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
		Filename:  "test.png",
	}

	img := models.WebImage{
		URL:   "http://example.com/image.png",
		Title: "Test Image",
	}

	path, err := client.DownloadImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Downloaded file does not exist: %s", path)
	}

	// Verify file content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if len(content) != len(minimalPNG) {
		t.Errorf("Downloaded file size = %d, want %d", len(content), len(minimalPNG))
	}
}

func TestDownloadImage_NetworkError(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			return nil, errors.New("network error: connection refused")
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
	}

	img := models.WebImage{URL: "http://example.com/image.png"}

	_, err := client.DownloadImage(img, opts)
	if err == nil {
		t.Error("DownloadImage() should return error on network failure")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Errorf("Error should mention network, got: %v", err)
	}
}

func TestDownloadImage_NonImageContentType(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "text/html")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody([]byte("<html></html>")),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{Directory: tempDir}
	img := models.WebImage{URL: "http://example.com/image.png"}

	_, err := client.DownloadImage(img, opts)
	if err == nil {
		t.Error("DownloadImage() should return error for non-image content type")
	}
	if !strings.Contains(err.Error(), "not an image") {
		t.Errorf("Error should mention 'not an image', got: %v", err)
	}
}

func TestDownloadImage_HTTPError404(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			return &fhttp.Response{
				StatusCode: 404,
				Body:       NewMockResponseBody([]byte("Not Found")),
				Header:     make(fhttp.Header),
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{Directory: tempDir}
	img := models.WebImage{URL: "http://example.com/notfound.png"}

	_, err := client.DownloadImage(img, opts)
	if err == nil {
		t.Error("DownloadImage() should return error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Error should mention 404, got: %v", err)
	}
}

func TestDownloadImage_HTTPError500(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			return &fhttp.Response{
				StatusCode: 500,
				Body:       NewMockResponseBody([]byte("Internal Server Error")),
				Header:     make(fhttp.Header),
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{Directory: tempDir}
	img := models.WebImage{URL: "http://example.com/error.png"}

	_, err := client.DownloadImage(img, opts)
	if err == nil {
		t.Error("DownloadImage() should return error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error should mention 500, got: %v", err)
	}
}

func TestDownloadImage_ClientClosed(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{}
	client := createTestDownloadClient(t, mockClient)

	// Close the client
	client.Close()

	opts := ImageDownloadOptions{Directory: tempDir}
	img := models.WebImage{URL: "http://example.com/image.png"}

	_, err := client.DownloadImage(img, opts)
	if err == nil {
		t.Error("DownloadImage() should return error when client is closed")
	}
}

// ============================================================================
// DownloadGeneratedImage Tests
// ============================================================================

func TestDownloadGeneratedImage_Success(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/jpeg")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalJPEG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
		Filename:  "generated.jpg",
		FullSize:  false,
	}

	img := models.GeneratedImage{
		URL:   "http://example.com/generated",
		Title: "AI Generated",
	}

	path, err := client.DownloadGeneratedImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadGeneratedImage() error = %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Downloaded file does not exist: %s", path)
	}
}

func TestDownloadGeneratedImage_FullSizeParameter(t *testing.T) {
	tempDir := t.TempDir()

	var capturedURL string
	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			capturedURL = req.URL.String()
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
		FullSize:  true, // Should append =s2048
	}

	img := models.GeneratedImage{
		URL: "http://example.com/generated",
	}

	_, err := client.DownloadGeneratedImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadGeneratedImage() error = %v", err)
	}

	// URL should have =s2048 appended
	if !strings.Contains(capturedURL, "=s2048") {
		t.Errorf("URL should contain =s2048 when FullSize is true, got: %s", capturedURL)
	}
}

func TestDownloadGeneratedImage_WithoutFullSize(t *testing.T) {
	tempDir := t.TempDir()

	var capturedURL string
	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			capturedURL = req.URL.String()
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
		FullSize:  false, // Should NOT append =s2048
	}

	img := models.GeneratedImage{
		URL: "http://example.com/generated",
	}

	_, err := client.DownloadGeneratedImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadGeneratedImage() error = %v", err)
	}

	// URL should NOT have =s2048 appended
	if strings.Contains(capturedURL, "=s2048") {
		t.Errorf("URL should not contain =s2048 when FullSize is false, got: %s", capturedURL)
	}
}

func TestDownloadGeneratedImage_AlreadyHasSizeParam(t *testing.T) {
	tempDir := t.TempDir()

	var capturedURL string
	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			capturedURL = req.URL.String()
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
		FullSize:  true, // Should NOT add =s2048 since URL already has =s
	}

	img := models.GeneratedImage{
		URL: "http://example.com/generated=s1024", // Already has size param
	}

	_, err := client.DownloadGeneratedImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadGeneratedImage() error = %v", err)
	}

	// Should not double-append size parameter
	if strings.Contains(capturedURL, "=s2048") {
		t.Errorf("URL should not have =s2048 when already has =s, got: %s", capturedURL)
	}
}

// ============================================================================
// downloadImageURL Tests
// ============================================================================

func TestDownloadImageURL_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "subdir", "nested")

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: newDir,
		Filename:  "test.png",
	}

	img := models.WebImage{URL: "http://example.com/image.png"}

	path, err := client.DownloadImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Errorf("Directory was not created: %s", newDir)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("File was not created: %s", path)
	}
}

func TestDownloadImageURL_CustomFilename(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	customName := "my_custom_filename.png"
	opts := ImageDownloadOptions{
		Directory: tempDir,
		Filename:  customName,
	}

	img := models.WebImage{URL: "http://example.com/original.png"}

	path, err := client.DownloadImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	if !strings.HasSuffix(path, customName) {
		t.Errorf("Downloaded file path = %s, want suffix %s", path, customName)
	}
}

func TestDownloadImageURL_AutoFilename(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/jpeg")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalJPEG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{
		Directory: tempDir,
		// No filename specified - should auto-generate
	}

	img := models.WebImage{
		URL:   "http://example.com/photo.jpg",
		Title: "",
	}

	path, err := client.DownloadImage(img, opts)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	// Should use URL filename
	if !strings.HasSuffix(path, "photo.jpg") {
		t.Errorf("Downloaded file path = %s, want suffix photo.jpg", path)
	}
}

// ============================================================================
// DownloadAllImages Tests
// ============================================================================

func TestDownloadAllImages_WebImages(t *testing.T) {
	tempDir := t.TempDir()

	callCount := 0
	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			callCount++
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/img1.png", Title: "Image 1"},
					{URL: "http://example.com/img2.png", Title: "Image 2"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	paths, err := client.DownloadAllImages(output, opts)
	if err != nil {
		t.Fatalf("DownloadAllImages() error = %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("DownloadAllImages() returned %d paths, want 2", len(paths))
	}

	// Verify all files exist
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Downloaded file does not exist: %s", path)
		}
	}
}

func TestDownloadAllImages_GeneratedImages(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/jpeg")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalJPEG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				GeneratedImages: []models.GeneratedImage{
					{URL: "http://example.com/gen1", Title: "Generated 1"},
					{URL: "http://example.com/gen2", Title: "Generated 2"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	paths, err := client.DownloadAllImages(output, opts)
	if err != nil {
		t.Fatalf("DownloadAllImages() error = %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("DownloadAllImages() returned %d paths, want 2", len(paths))
	}
}

func TestDownloadAllImages_MixedImages(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/web.png"},
				},
				GeneratedImages: []models.GeneratedImage{
					{URL: "http://example.com/gen"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	paths, err := client.DownloadAllImages(output, opts)
	if err != nil {
		t.Fatalf("DownloadAllImages() error = %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("DownloadAllImages() returned %d paths, want 2", len(paths))
	}
}

func TestDownloadAllImages_NilOutput(t *testing.T) {
	mockClient := &DynamicMockHttpClient{}
	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{Directory: t.TempDir()}

	paths, err := client.DownloadAllImages(nil, opts)
	if err != nil {
		t.Errorf("DownloadAllImages(nil) should not return error, got: %v", err)
	}
	if paths != nil {
		t.Errorf("DownloadAllImages(nil) should return nil paths, got: %v", paths)
	}
}

func TestDownloadAllImages_NilCandidate(t *testing.T) {
	mockClient := &DynamicMockHttpClient{}
	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{}, // Empty candidates
	}

	opts := ImageDownloadOptions{Directory: t.TempDir()}

	paths, err := client.DownloadAllImages(output, opts)
	if err != nil {
		t.Errorf("DownloadAllImages() should not return error for empty candidates, got: %v", err)
	}
	if paths != nil {
		t.Errorf("DownloadAllImages() should return nil paths for empty candidates, got: %v", paths)
	}
}

func TestDownloadAllImages_PartialFailure(t *testing.T) {
	tempDir := t.TempDir()

	callCount := 0
	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			callCount++
			// First request succeeds, second fails
			if callCount == 2 {
				return nil, errors.New("network error")
			}
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/img1.png"},
					{URL: "http://example.com/img2.png"}, // This will fail
					{URL: "http://example.com/img3.png"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	paths, err := client.DownloadAllImages(output, opts)
	// Should not return error if some images succeeded
	if err != nil {
		t.Errorf("DownloadAllImages() should not return error for partial failure, got: %v", err)
	}

	// Should have 2 successful downloads
	if len(paths) != 2 {
		t.Errorf("DownloadAllImages() returned %d paths, want 2", len(paths))
	}
}

// ============================================================================
// DownloadSelectedImages Tests
// ============================================================================

func TestDownloadSelectedImages_ValidIndices(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/img0.png"},
					{URL: "http://example.com/img1.png"},
					{URL: "http://example.com/img2.png"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	// Select only indices 0 and 2
	paths, err := client.DownloadSelectedImages(output, []int{0, 2}, opts)
	if err != nil {
		t.Fatalf("DownloadSelectedImages() error = %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("DownloadSelectedImages() returned %d paths, want 2", len(paths))
	}
}

func TestDownloadSelectedImages_InvalidIndices(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/img0.png"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	// Mix of valid (0) and invalid (-1, 100) indices
	paths, err := client.DownloadSelectedImages(output, []int{-1, 0, 100}, opts)
	if err != nil {
		t.Fatalf("DownloadSelectedImages() error = %v", err)
	}

	// Should only download the valid index
	if len(paths) != 1 {
		t.Errorf("DownloadSelectedImages() returned %d paths, want 1", len(paths))
	}
}

func TestDownloadSelectedImages_OutOfBounds(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{}
	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/img0.png"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	// All indices out of bounds
	paths, err := client.DownloadSelectedImages(output, []int{10, 20, 30}, opts)
	if err != nil {
		t.Errorf("DownloadSelectedImages() should not error for out-of-bounds, got: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("DownloadSelectedImages() returned %d paths, want 0", len(paths))
	}
}

func TestDownloadSelectedImages_EmptyIndices(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{}
	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/img0.png"},
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	paths, err := client.DownloadSelectedImages(output, []int{}, opts)
	if err != nil {
		t.Errorf("DownloadSelectedImages([]) should not error, got: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("DownloadSelectedImages([]) returned %d paths, want 0", len(paths))
	}
}

func TestDownloadSelectedImages_GeneratedImageIndex(t *testing.T) {
	tempDir := t.TempDir()

	mockClient := &DynamicMockHttpClient{
		DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			header := make(fhttp.Header)
			header.Set("Content-Type", "image/png")
			return &fhttp.Response{
				StatusCode: 200,
				Body:       NewMockResponseBody(minimalPNG),
				Header:     header,
			}, nil
		},
	}

	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	output := &models.ModelOutput{
		Candidates: []models.Candidate{
			{
				WebImages: []models.WebImage{
					{URL: "http://example.com/web0.png"}, // index 0
					{URL: "http://example.com/web1.png"}, // index 1
				},
				GeneratedImages: []models.GeneratedImage{
					{URL: "http://example.com/gen0"}, // index 2
					{URL: "http://example.com/gen1"}, // index 3
				},
			},
		},
	}

	opts := ImageDownloadOptions{Directory: tempDir}

	// Select a generated image (index 3 = second generated image)
	paths, err := client.DownloadSelectedImages(output, []int{3}, opts)
	if err != nil {
		t.Fatalf("DownloadSelectedImages() error = %v", err)
	}

	if len(paths) != 1 {
		t.Errorf("DownloadSelectedImages() returned %d paths, want 1", len(paths))
	}
}

func TestDownloadSelectedImages_NilOutput(t *testing.T) {
	mockClient := &DynamicMockHttpClient{}
	client := createTestDownloadClient(t, mockClient)
	defer client.Close()

	opts := ImageDownloadOptions{Directory: t.TempDir()}

	paths, err := client.DownloadSelectedImages(nil, []int{0}, opts)
	if err != nil {
		t.Errorf("DownloadSelectedImages(nil) should not error, got: %v", err)
	}
	if paths != nil {
		t.Errorf("DownloadSelectedImages(nil) should return nil, got: %v", paths)
	}
}

// ============================================================================
// Integration/Edge Case Tests
// ============================================================================

func TestDownload_DifferentImageFormats(t *testing.T) {
	testCases := []struct {
		name        string
		imageData   []byte
		contentType string
		wantExt     string
	}{
		{"PNG", minimalPNG, "image/png", ".png"},
		{"JPEG", minimalJPEG, "image/jpeg", ".jpg"},
		{"WebP", minimalWebP, "image/webp", ".webp"},
		{"GIF", minimalGIF, "image/gif", ".gif"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			mockClient := &DynamicMockHttpClient{
				DoFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
					header := make(fhttp.Header)
					header.Set("Content-Type", tc.contentType)
					return &fhttp.Response{
						StatusCode: 200,
						Body:       NewMockResponseBody(tc.imageData),
						Header:     header,
					}, nil
				},
			}

			client := createTestDownloadClient(t, mockClient)
			defer client.Close()

			opts := ImageDownloadOptions{Directory: tempDir}
			img := models.WebImage{URL: "http://example.com/image"}

			path, err := client.DownloadImage(img, opts)
			if err != nil {
				t.Fatalf("DownloadImage() error = %v", err)
			}

			if !strings.HasSuffix(path, tc.wantExt) {
				t.Errorf("Downloaded file path = %s, want suffix %s", path, tc.wantExt)
			}

			// Verify content matches
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			if len(content) != len(tc.imageData) {
				t.Errorf("File size = %d, want %d", len(content), len(tc.imageData))
			}
		})
	}
}
