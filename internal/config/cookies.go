package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Cookies represents the authentication cookies
type Cookies struct {
	mu            sync.RWMutex `json:"-"` // Not serialized
	Secure1PSID   string       `json:"__Secure-1PSID"`
	Secure1PSIDTS string       `json:"__Secure-1PSIDTS,omitempty"`
}

// GetSecure1PSID returns the __Secure-1PSID cookie in a thread-safe manner
func (c *Cookies) GetSecure1PSID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Secure1PSID
}

// GetSecure1PSIDTS returns the __Secure-1PSIDTS cookie in a thread-safe manner
func (c *Cookies) GetSecure1PSIDTS() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Secure1PSIDTS
}

// Snapshot returns both cookies atomically (for serialization or HTTP requests)
func (c *Cookies) Snapshot() (psid, psidts string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Secure1PSID, c.Secure1PSIDTS
}

// SetBoth updates both cookies atomically
func (c *Cookies) SetBoth(psid, psidts string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Secure1PSID = psid
	c.Secure1PSIDTS = psidts
}

// CookieListItem represents a cookie in browser export format
type CookieListItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// LoadCookies loads cookies from the cookies file
func LoadCookies() (*Cookies, error) {
	cookiesPath, err := GetCookiesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cookiesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no cookies found. Please import cookies first:\n  geminiweb import-cookies <path-to-cookies.json>")
		}
		return nil, fmt.Errorf("failed to read cookies file: %w", err)
	}

	return parseCookies(data)
}

// parseCookies parses cookies from JSON data
// Supports both list format [{name, value}] and dict format {name: value}
func parseCookies(data []byte) (*Cookies, error) {
	// Try dict format first
	var dictFormat map[string]string
	if err := json.Unmarshal(data, &dictFormat); err == nil {
		psid, ok := dictFormat["__Secure-1PSID"]
		if !ok {
			return nil, fmt.Errorf("missing required cookie: __Secure-1PSID")
		}
		return &Cookies{
			Secure1PSID:   psid,
			Secure1PSIDTS: dictFormat["__Secure-1PSIDTS"],
		}, nil
	}

	// Try list format (browser export)
	var listFormat []CookieListItem
	if err := json.Unmarshal(data, &listFormat); err == nil {
		cookies := &Cookies{}
		for _, item := range listFormat {
			switch item.Name {
			case "__Secure-1PSID":
				cookies.Secure1PSID = item.Value
			case "__Secure-1PSIDTS":
				cookies.Secure1PSIDTS = item.Value
			}
		}

		if cookies.Secure1PSID == "" {
			return nil, fmt.Errorf("missing required cookie: __Secure-1PSID")
		}
		return cookies, nil
	}

	return nil, fmt.Errorf("invalid cookies format: expected list [{name, value}] or dict {name: value}")
}

// SaveCookies saves cookies to the cookies file
func SaveCookies(cookies *Cookies) error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	cookiesPath := configDir + "/cookies.json"

	// Save in list format for compatibility
	listFormat := []CookieListItem{
		{Name: "__Secure-1PSID", Value: cookies.Secure1PSID},
	}
	if cookies.Secure1PSIDTS != "" {
		listFormat = append(listFormat, CookieListItem{
			Name:  "__Secure-1PSIDTS",
			Value: cookies.Secure1PSIDTS,
		})
	}

	data, err := json.MarshalIndent(listFormat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Save with restrictive permissions (owner read/write only)
	if err := os.WriteFile(cookiesPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	return nil
}

// ImportCookies imports cookies from a source file
func ImportCookies(sourcePath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source file not found: %s", sourcePath)
		}
		return fmt.Errorf("could not read file: %w", err)
	}

	cookies, err := parseCookies(data)
	if err != nil {
		return err
	}

	return SaveCookies(cookies)
}

// ValidateCookies checks if cookies are valid
func ValidateCookies(cookies *Cookies) error {
	if cookies == nil {
		return fmt.Errorf("cookies are nil")
	}
	if cookies.Secure1PSID == "" {
		return fmt.Errorf("missing required cookie: __Secure-1PSID")
	}
	return nil
}

// ToMap converts cookies to a map for HTTP requests (thread-safe)
func (c *Cookies) ToMap() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := map[string]string{
		"__Secure-1PSID": c.Secure1PSID,
	}
	if c.Secure1PSIDTS != "" {
		m["__Secure-1PSIDTS"] = c.Secure1PSIDTS
	}
	return m
}

// Update1PSIDTS updates the PSIDTS cookie value (thread-safe)
func (c *Cookies) Update1PSIDTS(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Secure1PSIDTS = value
}
