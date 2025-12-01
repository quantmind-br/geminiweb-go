package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"

	"github.com/diogo/geminiweb/internal/models"
)

const (
	MaxImageSize = 20 * 1024 * 1024 // 20MB
)

// SupportedImageTypes returns the list of supported MIME types for upload
func SupportedImageTypes() []string {
	return []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
}

// UploadedImage represents an uploaded image ready for use in prompts
type UploadedImage struct {
	ResourceID string
	FileName   string
	MIMEType   string
	Size       int64
}

// ImageUploader handles image uploads to Gemini
type ImageUploader struct {
	client *GeminiClient
}

// NewImageUploader creates a new image uploader
func NewImageUploader(client *GeminiClient) *ImageUploader {
	return &ImageUploader{
		client: client,
	}
}

// UploadFile uploads an image file from disk
func (u *ImageUploader) UploadFile(filePath string) (*UploadedImage, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > MaxImageSize {
		return nil, fmt.Errorf("file size exceeds maximum %d bytes", MaxImageSize)
	}

	// Detect MIME type
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	if !u.isSupportedType(mimeType) {
		return nil, fmt.Errorf("unsupported image type: %s", mimeType)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	return u.uploadStream(file, filepath.Base(filePath), mimeType, fileInfo.Size())
}

// UploadFromReader uploads from an io.Reader
func (u *ImageUploader) UploadFromReader(
	reader io.Reader,
	fileName string,
	mimeType string,
) (*UploadedImage, error) {
	// Read all content into buffer (needed for multipart)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	if int64(len(data)) > MaxImageSize {
		return nil, fmt.Errorf("data size exceeds maximum %d bytes", MaxImageSize)
	}

	return u.uploadStream(bytes.NewReader(data), fileName, mimeType, int64(len(data)))
}

// uploadStream executes the actual upload
func (u *ImageUploader) uploadStream(
	reader io.Reader,
	fileName string,
	mimeType string,
	size int64,
) (*UploadedImage, error) {
	// Create multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	_ = writer.Close()

	// Build URL with parameters
	uploadID := generateUploadID()
	uploadURL := fmt.Sprintf("%s?upload_id=%s&upload_protocol=resumable",
		models.EndpointUpload,
		uploadID,
	)

	req, err := fhttp.NewRequest(fhttp.MethodPost, uploadURL, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	req.Header.Set("X-Goog-Upload-Offset", "0")

	// Cookies
	cookies := u.client.GetCookies()
	req.AddCookie(&fhttp.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&fhttp.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := u.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response to get resource ID
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var uploadResp struct {
		ResourceID string `json:"resourceId"`
	}

	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		// Try to extract from header if no JSON body
		resourceID := resp.Header.Get("X-Goog-Upload-URL")
		if resourceID == "" {
			// Use upload ID as fallback
			uploadResp.ResourceID = uploadID
		} else {
			uploadResp.ResourceID = resourceID
		}
	}

	return &UploadedImage{
		ResourceID: uploadResp.ResourceID,
		FileName:   fileName,
		MIMEType:   mimeType,
		Size:       size,
	}, nil
}

func (u *ImageUploader) isSupportedType(mimeType string) bool {
	for _, supported := range SupportedImageTypes() {
		if strings.HasPrefix(mimeType, supported) {
			return true
		}
	}
	return false
}

func generateUploadID() string {
	return fmt.Sprintf("geminiweb-%d", time.Now().UnixNano())
}

// UploadImage is a convenience method on GeminiClient for uploading images
func (c *GeminiClient) UploadImage(filePath string) (*UploadedImage, error) {
	uploader := NewImageUploader(c)
	return uploader.UploadFile(filePath)
}

// UploadImageFromReader is a convenience method for uploading from a reader
func (c *GeminiClient) UploadImageFromReader(
	reader io.Reader,
	fileName string,
	mimeType string,
) (*UploadedImage, error) {
	uploader := NewImageUploader(c)
	return uploader.UploadFromReader(reader, fileName, mimeType)
}
