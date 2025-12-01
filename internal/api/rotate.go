package api

import (
	"fmt"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// Rate limiting for cookie rotation (1 call per minute max)
var (
	lastRotateTime time.Time
	rotateMutex    sync.Mutex
)

// RotateCookies refreshes the __Secure-1PSIDTS cookie
func RotateCookies(client tls_client.HttpClient, cookies *config.Cookies) (string, error) {
	rotateMutex.Lock()
	defer rotateMutex.Unlock()

	// Rate limit: don't call more than once per minute
	if time.Since(lastRotateTime) < time.Minute {
		return "", nil // Skip if called too recently
	}

	req, err := http.NewRequest(
		http.MethodPost,
		models.EndpointRotateCookies,
		strings.NewReader(`[000,"-0000000000000000000"]`),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create rotate request: %w", err)
	}

	// Set headers
	for key, value := range models.RotateCookiesHeaders() {
		req.Header.Set(key, value)
	}

	// Set cookies
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to rotate cookies: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode == 401 {
		return "", apierrors.NewAuthError("unauthorized during cookie rotation")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("cookie rotation failed with status: %d", resp.StatusCode)
	}

	// Update last rotate time
	lastRotateTime = time.Now()

	// Extract new __Secure-1PSIDTS from response cookies
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "__Secure-1PSIDTS" {
			return cookie.Value, nil
		}
	}

	return "", nil
}

// CookieRotator manages background cookie rotation
type CookieRotator struct {
	client   tls_client.HttpClient
	cookies  *config.Cookies
	interval time.Duration
	stopCh   chan struct{}
	running  bool
	mu       sync.Mutex
}

// NewCookieRotator creates a new cookie rotator
func NewCookieRotator(client tls_client.HttpClient, cookies *config.Cookies, interval time.Duration) *CookieRotator {
	return &CookieRotator{
		client:   client,
		cookies:  cookies,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins background cookie rotation
func (r *CookieRotator) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newToken, err := RotateCookies(r.client, r.cookies)
				if err != nil {
					// Log error but continue
					continue
				}
				if newToken != "" {
					r.cookies.Update1PSIDTS(newToken)
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop halts background cookie rotation
func (r *CookieRotator) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		close(r.stopCh)
		r.running = false
	}
}
