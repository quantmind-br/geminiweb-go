package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/tidwall/gjson"

	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// GenerateOptions contains options for content generation
type GenerateOptions struct {
	Model    models.Model
	Metadata []string         // [cid, rid, rcid] for chat context
	Images   []*UploadedImage // Images to include in the prompt
	GemID    string           // ID do gem a usar (server-side persona)
}

// GenerateContent sends a prompt to Gemini and returns the response
func (c *GeminiClient) GenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error) {
	result, err := c.doGenerateContent(prompt, opts)

	// If auth error and browser refresh is enabled, try to refresh and retry
	if err != nil && c.IsBrowserRefreshEnabled() && isAuthError(err) {
		// Use injected refresh function if available (for testing)
		var refreshed bool
		var refreshErr error

		if c.refreshFunc != nil {
			// Use injected function for testing
			refreshed, refreshErr = c.refreshFunc()
		} else {
			// Use default implementation
			refreshed, refreshErr = c.RefreshFromBrowser()
		}

		if refreshErr == nil && refreshed {
			// Retry the request with new cookies
			return c.doGenerateContent(prompt, opts)
		}
	}

	return result, err
}

// isAuthError checks if an error is an authentication error
// using the centralized error checking function
func isAuthError(err error) bool {
	return apierrors.IsAuthError(err)
}

// doGenerateContent performs the actual content generation request
func (c *GeminiClient) doGenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	if c.IsClosed() {
		return nil, fmt.Errorf("client is closed")
	}

	model := c.GetModel()
	var metadata []string
	var images []*UploadedImage
	var gemID string

	if opts != nil {
		if opts.Model.Name != "" {
			model = opts.Model
		}
		metadata = opts.Metadata
		images = opts.Images
		gemID = opts.GemID
	}

	// Build the request payload
	payload, err := buildPayloadWithGem(prompt, metadata, images, gemID)
	if err != nil {
		return nil, fmt.Errorf("failed to build payload: %w", err)
	}

	// Create form data
	form := url.Values{}
	form.Set("at", c.GetAccessToken())
	form.Set("f.req", payload)

	req, err := http.NewRequest(
		http.MethodPost,
		models.EndpointGenerate,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range models.DefaultHeaders() {
		req.Header.Set(key, value)
	}

	// Set model-specific headers
	for key, value := range model.Header {
		req.Header.Set(key, value)
	}

	// Set cookies
	cookies := c.GetCookies()
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewNetworkErrorWithEndpoint("generate content", models.EndpointGenerate, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		// Read response body for error diagnostics
		errorBody := make([]byte, 0, 4096)
		buf := make([]byte, 1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				errorBody = append(errorBody, buf[:n]...)
				// Limit error body to 4KB for safety
				if len(errorBody) >= 4096 {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
		return nil, apierrors.NewAPIErrorWithBody(resp.StatusCode, models.EndpointGenerate, "generate content failed", string(errorBody))
	}

	// Read response body
	body := make([]byte, 0, 65536)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return parseResponse(body, model.Name)
}

// buildPayload creates the f.req payload for the generate request
func buildPayload(prompt string, metadata []string) (string, error) {
	return buildPayloadWithGem(prompt, metadata, nil, "")
}

// buildPayloadWithImages creates the f.req payload including file references
// Based on the Python Gemini-API implementation
func buildPayloadWithImages(prompt string, metadata []string, images []*UploadedImage) (string, error) {
	return buildPayloadWithGem(prompt, metadata, images, "")
}

// buildPayloadWithGem creates the f.req payload including file references and gem
// Based on the Python Gemini-API implementation
func buildPayloadWithGem(prompt string, metadata []string, images []*UploadedImage, gemID string) (string, error) {
	// Inner payload structure depends on whether files are included
	var inner []interface{}

	if len(images) > 0 {
		// Build file parts: [[file_id], filename] for each file
		var fileParts []interface{}
		for _, img := range images {
			fileParts = append(fileParts, []interface{}{
				[]interface{}{img.ResourceID}, // File ID wrapped in array
				img.FileName,                  // Original filename
			})
		}

		// With files: [prompt, 0, None, files_array], None, metadata
		inner = []interface{}{
			[]interface{}{
				prompt, // Prompt directly (not in array)
				0,      // Flags/mode
				nil,    // Reserved
				fileParts,
			},
			nil,      // Reserved
			metadata, // Chat metadata [cid, rid, rcid]
		}
	} else {
		// Without files: [[prompt]], None, metadata
		inner = []interface{}{
			[]interface{}{prompt},
			nil,
			metadata,
		}
	}

	// Add gem_id if provided
	// Format: 16 nulls followed by gem_id (position 19 total)
	if gemID != "" {
		for i := 0; i < 16; i++ {
			inner = append(inner, nil)
		}
		inner = append(inner, gemID)
	}

	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", err
	}

	// Outer payload: [null, innerJSON]
	outer := []interface{}{
		nil,
		string(innerJSON),
	}

	outerJSON, err := json.Marshal(outer)
	if err != nil {
		return "", err
	}

	return string(outerJSON), nil
}

// parseResponse parses the Gemini API response
func parseResponse(body []byte, modelName string) (*models.ModelOutput, error) {
	// Response has garbage prefix - find first valid JSON line
	var jsonLine string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if gjson.Valid(line) {
			jsonLine = line
			break
		}
	}

	if jsonLine == "" {
		return nil, apierrors.NewParseError("no valid JSON found in response", "")
	}

	parsed := gjson.Parse(jsonLine)

	// Check for alternative error format first
	// Format: [["wrb.fr",null,null,null,null,[3]],...]
	// Error code at position 0.5.0 (first element of the array at position 5)
	altErrorCode := parsed.Get(PathAltErrorCode)
	if altErrorCode.Exists() && !altErrorCode.IsArray() && altErrorCode.Int() > 0 {
		return nil, handleErrorCode(models.ErrorCode(altErrorCode.Int()), modelName)
	}

	// Find the response body
	var responseBody gjson.Result
	var bodyIndex int

	parsed.ForEach(func(key, value gjson.Result) bool {
		bodyData := value.Get(PathBody)
		if !bodyData.Exists() {
			return true
		}

		// Try to parse the body data as JSON
		bodyJSON := gjson.Parse(bodyData.String())
		if bodyJSON.Get(PathCandList).Exists() {
			responseBody = bodyJSON
			bodyIndex = int(key.Int())
			return false
		}
		return true
	})

	if !responseBody.Exists() {
		// Check for error codes in the standard path
		errorCode := parsed.Get(PathErrorCode)
		if errorCode.Exists() {
			return nil, handleErrorCode(models.ErrorCode(errorCode.Int()), modelName)
		}
		return nil, apierrors.NewParseError("no response body found", PathBody)
	}

	// Extract metadata
	metadataResult := responseBody.Get(PathMetadata)
	var metadata []string
	if metadataResult.IsArray() {
		metadataResult.ForEach(func(_, v gjson.Result) bool {
			metadata = append(metadata, v.String())
			return true
		})
	}

	// Extract candidates
	candidateList := responseBody.Get(PathCandList)
	if !candidateList.Exists() || !candidateList.IsArray() {
		return nil, apierrors.NewParseError("no candidates found", PathCandList)
	}

	candidates := []models.Candidate{}
	candidateList.ForEach(func(candIdx, candValue gjson.Result) bool {
		rcid := candValue.Get(PathCandRCID).String()
		if rcid == "" {
			return true // Skip candidates without RCID
		}

		// Extract text
		text := candValue.Get(PathCandText).String()

		// Handle special URL-based text
		if matched, _ := regexp.MatchString(`^http://googleusercontent\.com/card_content/\d+`, text); matched {
			altText := candValue.Get(PathCandTextAlt).String()
			if altText != "" {
				text = altText
			}
		}

		// Extract thoughts
		thoughts := candValue.Get(PathCandThoughts).String()

		// Extract web images
		var webImages []models.WebImage
		candValue.Get(PathCandWebImages).ForEach(func(_, imgValue gjson.Result) bool {
			imgURL := imgValue.Get(PathWebImgURL).String()
			if imgURL == "" {
				return true
			}
			webImages = append(webImages, models.WebImage{
				URL:   imgURL,
				Title: imgValue.Get(PathWebImgTitle).String(),
				Alt:   imgValue.Get(PathWebImgAlt).String(),
			})
			return true
		})

		// Extract generated images
		var generatedImages []models.GeneratedImage
		candValue.Get(PathCandGenImages).ForEach(func(imgIdx, imgValue gjson.Result) bool {
			imgURL := imgValue.Get(PathGenImgURL).String()
			if imgURL == "" {
				return true
			}

			imgNum := imgValue.Get(PathGenImgNum).String()
			title := "[Generated Image]"
			if imgNum != "" {
				title = fmt.Sprintf("[Generated Image %s]", imgNum)
			}

			alts := imgValue.Get(PathGenImgAlts)
			alt := ""
			if alts.IsArray() {
				if altVal := alts.Get(fmt.Sprintf("%d", imgIdx.Int())); altVal.Exists() {
					alt = altVal.String()
				} else if altVal := alts.Get("0"); altVal.Exists() {
					alt = altVal.String()
				}
			}

			generatedImages = append(generatedImages, models.GeneratedImage{
				URL:   imgURL,
				Title: title,
				Alt:   alt,
			})
			return true
		})

		candidates = append(candidates, models.Candidate{
			RCID:            rcid,
			Text:            text,
			Thoughts:        thoughts,
			WebImages:       webImages,
			GeneratedImages: generatedImages,
		})
		return true
	})

	if len(candidates) == 0 {
		return nil, apierrors.NewParseError("no valid candidates found", PathCandList)
	}

	_ = bodyIndex // Used for generated image parsing in extended version

	return &models.ModelOutput{
		Metadata:   metadata,
		Candidates: candidates,
		Chosen:     0,
	}, nil
}

// handleErrorCode converts API error codes to appropriate errors
// using the centralized error handling function
func handleErrorCode(code models.ErrorCode, modelName string) error {
	return apierrors.HandleErrorCode(code, models.EndpointGenerate, modelName)
}
