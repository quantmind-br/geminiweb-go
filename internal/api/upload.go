package api

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"

	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

const (
	MaxImageSize = 20 * 1024 * 1024 // 20MB
	MaxFileSize  = 50 * 1024 * 1024 // 50MB for text files
)

// SupportedImageTypes returns the list of supported MIME types for image upload
func SupportedImageTypes() []string {
	return []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
}

// SupportedTextTypes returns the list of supported MIME types for text file upload
func SupportedTextTypes() []string {
	return []string{
		"text/plain",
		"text/markdown",
		"text/x-markdown",
		"application/json",
		"text/csv",
		"text/html",
		"text/xml",
		"application/xml",
	}
}

// UploadedFile represents an uploaded file ready for use in prompts
// This can be an image or text file - the API treats them similarly
type UploadedFile struct {
	ResourceID string
	FileName   string
	MIMEType   string
	Size       int64
}

// UploadedImage represents an uploaded image ready for use in prompts
// Deprecated: Use UploadedFile instead
type UploadedImage = UploadedFile

// FileUploader handles file uploads to Gemini (images, text, etc.)
type FileUploader struct {
	client *GeminiClient
}

// NewFileUploader creates a new file uploader
func NewFileUploader(client *GeminiClient) *FileUploader {
	return &FileUploader{
		client: client,
	}
}

// UploadFile uploads any supported file from disk (images or text)
func (u *FileUploader) UploadFile(filePath string) (*UploadedFile, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Detect MIME type
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Determine max size based on file type
	maxSize := int64(MaxFileSize)
	if u.isImageType(mimeType) {
		maxSize = MaxImageSize
	}

	if fileInfo.Size() > maxSize {
		return nil, fmt.Errorf("file size (%d bytes) exceeds maximum (%d bytes)", fileInfo.Size(), maxSize)
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

// UploadText uploads text content as a file
func (u *FileUploader) UploadText(content string, fileName string) (*UploadedFile, error) {
	if fileName == "" {
		fileName = "prompt.txt"
	}

	// Ensure .txt extension for proper MIME detection
	if filepath.Ext(fileName) == "" {
		fileName += ".txt"
	}

	data := []byte(content)
	if int64(len(data)) > MaxFileSize {
		return nil, fmt.Errorf("content size (%d bytes) exceeds maximum (%d bytes)", len(data), MaxFileSize)
	}

	mimeType := "text/plain"
	ext := filepath.Ext(fileName)
	if detectedType := mime.TypeByExtension(ext); detectedType != "" {
		mimeType = detectedType
	}

	return u.uploadStream(bytes.NewReader(data), fileName, mimeType, int64(len(data)))
}

// uploadStream executes the actual upload using Google's content-push service
// Based on the Python Gemini-API implementation
func (u *FileUploader) uploadStream(
	reader io.Reader,
	fileName string,
	mimeType string,
	size int64,
) (*UploadedFile, error) {
	// Create multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create form file: %v", err))
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to write file data: %v", err))
	}

	_ = writer.Close()

	// Simple POST to upload endpoint (no URL parameters)
	req, err := fhttp.NewRequest(fhttp.MethodPost, models.EndpointUpload, &body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create request: %v", err))
	}

	// Headers - only Content-Type and Push-ID are needed
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range models.UploadHeaders() {
		req.Header.Set(key, value)
	}

	// No cookies needed for upload endpoint

	resp, err := u.client.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewUploadNetworkError(fileName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		bodyStr := "(unable to read response body)"
		if readErr == nil {
			bodyStr = string(bodyBytes)
		}
		return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, bodyStr)
	}

	// Response is plain text containing the file identifier
	// Example: /contrib_service/ttl_1d/1709764705i7wdlyx3mdzndme3a767pluckv4flj
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to read response: %v", err))
	}

	resourceID := strings.TrimSpace(string(respBody))
	if resourceID == "" {
		return nil, apierrors.NewUploadError(fileName, "empty resource ID in upload response")
	}

	return &UploadedFile{
		ResourceID: resourceID,
		FileName:   fileName,
		MIMEType:   mimeType,
		Size:       size,
	}, nil
}

func (u *FileUploader) isImageType(mimeType string) bool {
	for _, supported := range SupportedImageTypes() {
		if strings.HasPrefix(mimeType, supported) {
			return true
		}
	}
	return false
}

func (u *FileUploader) isTextType(mimeType string) bool {
	for _, supported := range SupportedTextTypes() {
		if strings.HasPrefix(mimeType, supported) {
			return true
		}
	}
	return false
}

// ImageUploader handles image uploads to Gemini
// Deprecated: Use FileUploader instead
type ImageUploader struct {
	client *GeminiClient
}

// NewImageUploader creates a new image uploader
// Deprecated: Use NewFileUploader instead
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

// uploadStream executes the actual upload using Google's content-push service
// Based on the Python Gemini-API implementation
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
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create form file: %v", err))
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to write file data: %v", err))
	}

	_ = writer.Close()

	// Simple POST to upload endpoint (no URL parameters)
	req, err := fhttp.NewRequest(fhttp.MethodPost, models.EndpointUpload, &body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create request: %v", err))
	}

	// Headers - only Content-Type and Push-ID are needed
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range models.UploadHeaders() {
		req.Header.Set(key, value)
	}

	// No cookies needed for upload endpoint

	resp, err := u.client.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewUploadNetworkError(fileName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		bodyStr := "(unable to read response body)"
		if readErr == nil {
			bodyStr = string(bodyBytes)
		}
		return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, bodyStr)
	}

	// Response is plain text containing the file identifier
	// Example: /contrib_service/ttl_1d/1709764705i7wdlyx3mdzndme3a767pluckv4flj
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to read response: %v", err))
	}

	resourceID := strings.TrimSpace(string(respBody))
	if resourceID == "" {
		return nil, apierrors.NewUploadError(fileName, "empty resource ID in upload response")
	}

	return &UploadedImage{
		ResourceID: resourceID,
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
	// Ensure client is running (may re-init if auto-closed)
	if err := c.ensureRunning(); err != nil {
		return nil, err
	}
	// Reset idle timer to indicate activity
	c.resetIdleTimer()

	uploader := NewImageUploader(c)
	return uploader.UploadFile(filePath)
}

// UploadImageFromReader is a convenience method for uploading from a reader
func (c *GeminiClient) UploadImageFromReader(
	reader io.Reader,
	fileName string,
	mimeType string,
) (*UploadedImage, error) {
	// Ensure client is running (may re-init if auto-closed)
	if err := c.ensureRunning(); err != nil {
		return nil, err
	}
	// Reset idle timer to indicate activity
	c.resetIdleTimer()

	uploader := NewImageUploader(c)
	return uploader.UploadFromReader(reader, fileName, mimeType)
}

// UploadFile is a convenience method on GeminiClient for uploading any file
func (c *GeminiClient) UploadFile(filePath string) (*UploadedFile, error) {
	// Ensure client is running (may re-init if auto-closed)
	if err := c.ensureRunning(); err != nil {
		return nil, err
	}
	// Reset idle timer to indicate activity
	c.resetIdleTimer()

	uploader := NewFileUploader(c)
	return uploader.UploadFile(filePath)
}

// UploadText is a convenience method for uploading text content as a file
func (c *GeminiClient) UploadText(content string, fileName string) (*UploadedFile, error) {
	// Ensure client is running (may re-init if auto-closed)
	if err := c.ensureRunning(); err != nil {
		return nil, err
	}
	// Reset idle timer to indicate activity
	c.resetIdleTimer()

	uploader := NewFileUploader(c)
	return uploader.UploadText(content, fileName)
}

// LargePromptThreshold is the size (in bytes) above which prompts should be uploaded as files
const LargePromptThreshold = 100 * 1024 // 100KB
