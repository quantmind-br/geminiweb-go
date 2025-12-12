package api

import (
	"fmt"
	"regexp"
	"strings"

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
		return "", apierrors.NewGeminiErrorWithCause("create access token request", err)
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
		return "", apierrors.NewNetworkErrorWithEndpoint("fetch access token", models.EndpointInit, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		// Check for redirect (302) - often indicates blocking or auth issues
		if resp.StatusCode == 302 || resp.StatusCode == 301 {
			location := resp.Header.Get("Location")
			if location != "" {
				// Check if it's a Google blocking page
				if strings.Contains(location, "/sorry/") || strings.Contains(location, "sorry/index") {
					authErr := apierrors.NewAuthErrorWithEndpoint(
						"Google has temporarily blocked access (too many requests)",
						models.EndpointInit,
					)
					authErr.HTTPStatus = resp.StatusCode
					_ = authErr.WithBody(fmt.Sprintf(
						"Redirect to blocking page: %s\n\nTo resolve:\n1. Open your browser and visit: https://gemini.google.com/app\n2. Solve any CAPTCHA if presented\n3. Try again after a few minutes",
						location,
					))
					return "", authErr
				}
				// Generic redirect
				authErr := apierrors.NewAuthErrorWithEndpoint(
					fmt.Sprintf("unexpected redirect (status: %d)", resp.StatusCode),
					models.EndpointInit,
				)
				authErr.HTTPStatus = resp.StatusCode
				_ = authErr.WithBody(fmt.Sprintf("Redirect to: %s", location))
				return "", authErr
			}
		}

		// Read response body for diagnostics
		errorBody := make([]byte, 0, 2048)
		buf := make([]byte, 512)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				errorBody = append(errorBody, buf[:n]...)
				if len(errorBody) >= 2048 {
					break
				}
			}
			if readErr != nil {
				break
			}
		}

		authErr := apierrors.NewAuthErrorWithEndpoint(
			fmt.Sprintf("failed to fetch access token, status: %d", resp.StatusCode),
			models.EndpointInit,
		)
		authErr.HTTPStatus = resp.StatusCode
		_ = authErr.WithBody(string(errorBody))
		return "", authErr
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
		return "", apierrors.NewAuthErrorWithEndpoint(
			"SNlM0e token not found in response. Cookies may be expired.",
			models.EndpointInit,
		)
	}

	return string(matches[1]), nil
}
