package models

import "strings"

// Extension represents a Gemini extension
type Extension string

const (
	// ExtGmail provides access to Gmail inbox and emails
	ExtGmail Extension = "@Gmail"
	// ExtYouTube provides access to YouTube videos and activity
	ExtYouTube Extension = "@YouTube"
	// ExtGoogleMaps provides access to location and maps data
	ExtGoogleMaps Extension = "@GoogleMaps"
	// ExtGoogleFlights provides access to flight search and booking
	ExtGoogleFlights Extension = "@GoogleFlights"
	// ExtGoogleHotels provides access to hotel search and booking
	ExtGoogleHotels Extension = "@GoogleHotels"
	// ExtGoogleWorkspace provides access to Docs, Sheets, and other Workspace apps
	ExtGoogleWorkspace Extension = "@GoogleWorkspace"
)

// AllExtensions lists all supported extensions
var AllExtensions = []Extension{
	ExtGmail,
	ExtYouTube,
	ExtGoogleMaps,
	ExtGoogleFlights,
	ExtGoogleHotels,
	ExtGoogleWorkspace,
}

// DetectExtension checks if the prompt contains an extension trigger
// Returns the extension and true if found, empty string and false otherwise
func DetectExtension(prompt string) (Extension, bool) {
	trimmed := strings.TrimSpace(prompt)
	for _, ext := range AllExtensions {
		if strings.HasPrefix(trimmed, string(ext)) {
			return ext, true
		}
	}
	return "", false
}

// DetectAllExtensions returns all extensions found in the prompt
func DetectAllExtensions(prompt string) []Extension {
	var found []Extension
	for _, ext := range AllExtensions {
		if strings.Contains(prompt, string(ext)) {
			found = append(found, ext)
		}
	}
	return found
}

// String returns the extension name
func (e Extension) String() string {
	return string(e)
}

// Info returns a description of what the extension does
func (e Extension) Info() string {
	switch e {
	case ExtGmail:
		return "Access Gmail inbox and emails"
	case ExtYouTube:
		return "Search YouTube videos and activity"
	case ExtGoogleMaps:
		return "Access location and maps data"
	case ExtGoogleFlights:
		return "Search and book flights"
	case ExtGoogleHotels:
		return "Search and book hotels"
	case ExtGoogleWorkspace:
		return "Access Docs, Sheets, and other Workspace apps"
	default:
		return ""
	}
}

// ShortName returns a short display name without the @ prefix
func (e Extension) ShortName() string {
	return strings.TrimPrefix(string(e), "@")
}

// IsValid returns true if the extension is a known extension
func (e Extension) IsValid() bool {
	for _, ext := range AllExtensions {
		if e == ext {
			return true
		}
	}
	return false
}
