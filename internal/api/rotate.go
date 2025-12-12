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
		return "", apierrors.NewGeminiErrorWithCause("create rotate request", err)
	}

	// Set headers
	for key, value := range models.RotateCookiesHeaders() {
		req.Header.Set(key, value)
	}

	// Set cookies (using Snapshot for atomic read)
	psid, psidts := cookies.Snapshot()
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: psid})
	if psidts != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: psidts})
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", apierrors.NewNetworkErrorWithEndpoint("rotate cookies", models.EndpointRotateCookies, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode == 401 {
		return "", apierrors.NewAuthErrorWithEndpoint("unauthorized during cookie rotation", models.EndpointRotateCookies)
	}

	if resp.StatusCode != 200 {
		return "", apierrors.NewAPIError(resp.StatusCode, models.EndpointRotateCookies,
			fmt.Sprintf("cookie rotation failed with status: %d", resp.StatusCode))
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

// RotatorErrorCallback is called when a cookie rotation error occurs
type RotatorErrorCallback func(error)

// CookieRotator manages background cookie rotation
type CookieRotator struct {
	client   tls_client.HttpClient
	cookies  *config.Cookies
	interval time.Duration
	stopCh   chan struct{}
	running  bool
	mu       sync.Mutex
	onError  RotatorErrorCallback // Optional callback for rotation errors
}

// RotatorOption configures the CookieRotator
type RotatorOption func(*CookieRotator)

// WithErrorCallback sets a callback for rotation errors
func WithErrorCallback(fn RotatorErrorCallback) RotatorOption {
	return func(r *CookieRotator) {
		r.onError = fn
	}
}

// NewCookieRotator creates a new cookie rotator
func NewCookieRotator(client tls_client.HttpClient, cookies *config.Cookies, interval time.Duration, opts ...RotatorOption) *CookieRotator {
	r := &CookieRotator{
		client:   client,
		cookies:  cookies,
		interval: interval,
		// stopCh will be created in Start()
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Start begins background cookie rotation
func (r *CookieRotator) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return
	}

	// Create new channel in each Start() to allow restart after Stop()
	r.stopCh = make(chan struct{})
	r.running = true

	// Capture values to avoid race with Stop()
	client := r.client
	cookies := r.cookies
	interval := r.interval
	stopCh := r.stopCh
	onError := r.onError

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newToken, err := RotateCookies(client, cookies)
				if err != nil {
					// Report error via callback if configured
					if onError != nil {
						onError(fmt.Errorf("cookie rotation failed: %w", err))
					}
					continue
				}
				if newToken != "" {
					cookies.Update1PSIDTS(newToken)
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// Stop halts background cookie rotation
// Safe to call multiple times - subsequent calls are no-ops
func (r *CookieRotator) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running && r.stopCh != nil {
		close(r.stopCh)
		r.stopCh = nil
		r.running = false
	}
}
