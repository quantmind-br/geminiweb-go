package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"

	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// ImageDownloadOptions configures image download behavior
type ImageDownloadOptions struct {
	// Directory is the destination directory (default: ~/.geminiweb/images)
	Directory string
	// Filename is the output filename (auto-generated if empty)
	Filename string
	// FullSize downloads the image at maximum resolution (only for GeneratedImage)
	FullSize bool
}

// DefaultDownloadOptions returns the default download options
func DefaultDownloadOptions() ImageDownloadOptions {
	homeDir, _ := os.UserHomeDir()
	return ImageDownloadOptions{
		Directory: filepath.Join(homeDir, ".geminiweb", "images"),
		FullSize:  true,
	}
}

// DownloadImage downloads a WebImage to disk
func (c *GeminiClient) DownloadImage(img models.WebImage, opts ImageDownloadOptions) (string, error) {
	return c.downloadImageURL(img.URL, img.Title, opts)
}

// DownloadGeneratedImage downloads a GeneratedImage to disk
// If opts.FullSize is true, appends =s2048 to the URL for maximum resolution
func (c *GeminiClient) DownloadGeneratedImage(img models.GeneratedImage, opts ImageDownloadOptions) (string, error) {
	url := img.URL

	// Add size parameter for full-size images
	if opts.FullSize && !strings.Contains(url, "=s") {
		url += "=s2048"
	}

	return c.downloadImageURL(url, img.Title, opts)
}

// downloadImageURL is the internal implementation for downloading images
func (c *GeminiClient) downloadImageURL(url, title string, opts ImageDownloadOptions) (string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(opts.Directory, 0755); err != nil {
		return "", apierrors.NewDownloadError("failed to create directory: "+err.Error(), url)
	}

	// Create request
	req, err := fhttp.NewRequest(fhttp.MethodGet, url, nil)
	if err != nil {
		return "", apierrors.NewDownloadError("failed to create request: "+err.Error(), url)
	}

	// Set browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", apierrors.NewDownloadNetworkError(url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Validate status code
	if resp.StatusCode != 200 {
		return "", apierrors.NewDownloadErrorWithStatus(url, resp.StatusCode)
	}

	// Validate content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image") {
		return "", apierrors.NewDownloadError("response is not an image: "+contentType, url)
	}

	// Determine filename
	filename := opts.Filename
	if filename == "" {
		filename = generateFilename(url, title, contentType)
	}

	// Build full path
	destPath := filepath.Join(opts.Directory, filename)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apierrors.NewDownloadError("failed to read response: "+err.Error(), url)
	}

	// Write file
	if err := os.WriteFile(destPath, body, 0644); err != nil {
		return "", apierrors.NewDownloadError("failed to save file: "+err.Error(), url)
	}

	// Return absolute path (fallback to relative path if Abs fails)
	absPath, err := filepath.Abs(destPath)
	if err != nil {
		return destPath, nil
	}
	return absPath, nil
}

// generateFilename creates a filename based on URL, title, and content type
func generateFilename(url, title, contentType string) string {
	// Determine extension from content type
	ext := ".jpg"
	switch {
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	}

	// Try to extract filename from URL
	urlParts := strings.Split(strings.Split(url, "?")[0], "/")
	if len(urlParts) > 0 {
		lastPart := urlParts[len(urlParts)-1]
		if matched, _ := regexp.MatchString(`\.\w+$`, lastPart); matched {
			return sanitizeFilename(lastPart)
		}
	}

	// Use title if available
	if title != "" {
		safe := sanitizeFilename(title)
		if len(safe) > 50 {
			safe = safe[:50]
		}
		return safe + ext
	}

	// Fallback: timestamp
	return fmt.Sprintf("image_%s%s", time.Now().Format("20060102_150405"), ext)
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	// Remove characters not allowed in filenames
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := reg.ReplaceAllString(name, "_")
	return strings.TrimSpace(safe)
}

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	Path  string
	Error error
}

// DownloadAllImages downloads all images from a ModelOutput
// Returns a slice of paths for successfully downloaded images
func (c *GeminiClient) DownloadAllImages(output *models.ModelOutput, opts ImageDownloadOptions) ([]string, error) {
	if output == nil {
		return nil, nil
	}

	candidate := output.ChosenCandidate()
	if candidate == nil {
		return nil, nil
	}

	var paths []string
	var lastError error

	// Download web images
	for i, img := range candidate.WebImages {
		imgOpts := opts
		if imgOpts.Filename == "" {
			// Generate unique filename for each image
			imgOpts.Filename = ""
		}

		path, err := c.DownloadImage(img, imgOpts)
		if err != nil {
			lastError = err
			continue
		}
		paths = append(paths, path)

		// Small delay between downloads to avoid rate limiting
		if i < len(candidate.WebImages)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Download generated images
	for i, img := range candidate.GeneratedImages {
		imgOpts := opts
		if imgOpts.Filename == "" {
			imgOpts.Filename = ""
		}

		path, err := c.DownloadGeneratedImage(img, imgOpts)
		if err != nil {
			lastError = err
			continue
		}
		paths = append(paths, path)

		// Small delay between downloads
		if i < len(candidate.GeneratedImages)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Return paths even if some failed, report last error
	if len(paths) == 0 && lastError != nil {
		return nil, lastError
	}

	return paths, nil
}

// DownloadSelectedImages downloads specific images by their indices
// indices refers to the combined list (WebImages first, then GeneratedImages)
func (c *GeminiClient) DownloadSelectedImages(output *models.ModelOutput, indices []int, opts ImageDownloadOptions) ([]string, error) {
	if output == nil {
		return nil, nil
	}

	candidate := output.ChosenCandidate()
	if candidate == nil {
		return nil, nil
	}

	allImages := output.Images()
	var paths []string
	var lastError error

	for _, idx := range indices {
		if idx < 0 || idx >= len(allImages) {
			continue
		}

		img := allImages[idx]
		path, err := c.DownloadImage(img, opts)
		if err != nil {
			lastError = err
			continue
		}
		paths = append(paths, path)
	}

	if len(paths) == 0 && lastError != nil {
		return nil, lastError
	}

	return paths, nil
}
