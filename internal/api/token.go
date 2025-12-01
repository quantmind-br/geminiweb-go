package api

import (
	"fmt"
	"regexp"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// SNlM0e pattern for extracting access token from HTML
var snlm0ePattern = regexp.MustCompile(`"SNlM0e":"([^"]+)"`)

// GetAccessToken fetches the SNlM0e access token from gemini.google.com
func GetAccessToken(client tls_client.HttpClient, cookies *config.Cookies) (string, error) {
	req, err := http.NewRequest(http.MethodGet, models.EndpointInit, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range models.DefaultHeaders() {
		req.Header.Set(key, value)
	}

	// Set cookies
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch access token: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		return "", apierrors.NewAuthError(fmt.Sprintf("failed to fetch access token, status: %d", resp.StatusCode))
	}

	// Read response body
	body := make([]byte, 0)
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

	// Extract SNlM0e token using regex
	matches := snlm0ePattern.FindSubmatch(body)
	if len(matches) < 2 {
		return "", apierrors.NewAuthError("SNlM0e token not found in response. Cookies may be expired.")
	}

	return string(matches[1]), nil
}
