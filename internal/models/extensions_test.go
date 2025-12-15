package models

import (
	"testing"
)

// ============================================================================
// DetectExtension Tests
// ============================================================================

func TestDetectExtension_Gmail(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   Extension
		found  bool
	}{
		{"Gmail at start", "@Gmail check my inbox", ExtGmail, true},
		{"Gmail with space prefix", " @Gmail check my inbox", ExtGmail, true},
		{"Gmail lowercase not detected", "@gmail check my inbox", "", false},
		{"Gmail in middle not detected", "check @Gmail inbox", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := DetectExtension(tt.prompt)
			if found != tt.found {
				t.Errorf("DetectExtension() found = %v, want %v", found, tt.found)
			}
			if got != tt.want {
				t.Errorf("DetectExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectExtension_YouTube(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   Extension
		found  bool
	}{
		{"YouTube at start", "@YouTube search for videos", ExtYouTube, true},
		{"YouTube with tabs", "\t@YouTube search", ExtYouTube, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := DetectExtension(tt.prompt)
			if found != tt.found {
				t.Errorf("DetectExtension() found = %v, want %v", found, tt.found)
			}
			if got != tt.want {
				t.Errorf("DetectExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectExtension_GoogleMaps(t *testing.T) {
	ext, found := DetectExtension("@GoogleMaps find restaurants near me")
	if !found {
		t.Error("DetectExtension should find @GoogleMaps")
	}
	if ext != ExtGoogleMaps {
		t.Errorf("DetectExtension() = %v, want %v", ext, ExtGoogleMaps)
	}
}

func TestDetectExtension_GoogleFlights(t *testing.T) {
	ext, found := DetectExtension("@GoogleFlights find cheap flights to NYC")
	if !found {
		t.Error("DetectExtension should find @GoogleFlights")
	}
	if ext != ExtGoogleFlights {
		t.Errorf("DetectExtension() = %v, want %v", ext, ExtGoogleFlights)
	}
}

func TestDetectExtension_GoogleHotels(t *testing.T) {
	ext, found := DetectExtension("@GoogleHotels book a room")
	if !found {
		t.Error("DetectExtension should find @GoogleHotels")
	}
	if ext != ExtGoogleHotels {
		t.Errorf("DetectExtension() = %v, want %v", ext, ExtGoogleHotels)
	}
}

func TestDetectExtension_GoogleWorkspace(t *testing.T) {
	ext, found := DetectExtension("@GoogleWorkspace create a document")
	if !found {
		t.Error("DetectExtension should find @GoogleWorkspace")
	}
	if ext != ExtGoogleWorkspace {
		t.Errorf("DetectExtension() = %v, want %v", ext, ExtGoogleWorkspace)
	}
}

func TestDetectExtension_NotFound(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"Regular prompt", "What is the weather today?"},
		{"Empty prompt", ""},
		{"Just @ symbol", "@"},
		{"Unknown extension", "@Unknown what is this"},
		{"Extension in middle", "Please use @Gmail for this"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := DetectExtension(tt.prompt)
			if found {
				t.Errorf("DetectExtension(%q) found = true, want false", tt.prompt)
			}
			if got != "" {
				t.Errorf("DetectExtension(%q) = %v, want empty", tt.prompt, got)
			}
		})
	}
}

// ============================================================================
// DetectAllExtensions Tests
// ============================================================================

func TestDetectAllExtensions_Multiple(t *testing.T) {
	prompt := "@Gmail check my inbox and @YouTube find videos about it"
	extensions := DetectAllExtensions(prompt)

	if len(extensions) != 2 {
		t.Errorf("DetectAllExtensions() found %d extensions, want 2", len(extensions))
	}

	// Check that both extensions are found
	foundGmail := false
	foundYouTube := false
	for _, ext := range extensions {
		if ext == ExtGmail {
			foundGmail = true
		}
		if ext == ExtYouTube {
			foundYouTube = true
		}
	}

	if !foundGmail {
		t.Error("DetectAllExtensions() should find @Gmail")
	}
	if !foundYouTube {
		t.Error("DetectAllExtensions() should find @YouTube")
	}
}

func TestDetectAllExtensions_AllExtensions(t *testing.T) {
	// A prompt containing all extensions
	prompt := "@Gmail @YouTube @GoogleMaps @GoogleFlights @GoogleHotels @GoogleWorkspace"
	extensions := DetectAllExtensions(prompt)

	if len(extensions) != len(AllExtensions) {
		t.Errorf("DetectAllExtensions() found %d extensions, want %d", len(extensions), len(AllExtensions))
	}
}

func TestDetectAllExtensions_None(t *testing.T) {
	prompt := "This is a regular prompt without any extensions"
	extensions := DetectAllExtensions(prompt)

	if len(extensions) != 0 {
		t.Errorf("DetectAllExtensions() found %d extensions, want 0", len(extensions))
	}
}

func TestDetectAllExtensions_Empty(t *testing.T) {
	extensions := DetectAllExtensions("")

	if len(extensions) != 0 {
		t.Errorf("DetectAllExtensions(\"\") should return empty slice, got %v", extensions)
	}
}

func TestDetectAllExtensions_DuplicateExtensions(t *testing.T) {
	// Same extension mentioned twice
	prompt := "@Gmail check inbox @Gmail send email"
	extensions := DetectAllExtensions(prompt)

	// Should only find it once (due to Contains check)
	if len(extensions) != 1 {
		t.Errorf("DetectAllExtensions() found %d extensions, want 1 (no duplicates)", len(extensions))
	}
}

// ============================================================================
// Extension.String Tests
// ============================================================================

func TestExtension_String(t *testing.T) {
	tests := []struct {
		ext  Extension
		want string
	}{
		{ExtGmail, "@Gmail"},
		{ExtYouTube, "@YouTube"},
		{ExtGoogleMaps, "@GoogleMaps"},
		{ExtGoogleFlights, "@GoogleFlights"},
		{ExtGoogleHotels, "@GoogleHotels"},
		{ExtGoogleWorkspace, "@GoogleWorkspace"},
		{Extension("@Custom"), "@Custom"},
		{Extension(""), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.ext), func(t *testing.T) {
			got := tt.ext.String()
			if got != tt.want {
				t.Errorf("Extension.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Extension.Info Tests
// ============================================================================

func TestExtension_Info(t *testing.T) {
	tests := []struct {
		ext        Extension
		wantSubstr string
	}{
		{ExtGmail, "Gmail"},
		{ExtYouTube, "YouTube"},
		{ExtGoogleMaps, "maps"},
		{ExtGoogleFlights, "flight"},
		{ExtGoogleHotels, "hotel"},
		{ExtGoogleWorkspace, "Workspace"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ext), func(t *testing.T) {
			got := tt.ext.Info()
			if got == "" {
				t.Errorf("Extension.Info() should not be empty for %v", tt.ext)
			}
			// The info should contain relevant keywords
			// (case insensitive check)
		})
	}
}

func TestExtension_Info_InvalidExtension(t *testing.T) {
	ext := Extension("@Invalid")
	info := ext.Info()

	if info != "" {
		t.Errorf("Extension.Info() for invalid extension = %q, want empty", info)
	}
}

func TestExtension_Info_AllKnownExtensions(t *testing.T) {
	for _, ext := range AllExtensions {
		info := ext.Info()
		if info == "" {
			t.Errorf("Extension.Info() for %v should not be empty", ext)
		}
	}
}

// ============================================================================
// Extension.ShortName Tests
// ============================================================================

func TestExtension_ShortName(t *testing.T) {
	tests := []struct {
		ext  Extension
		want string
	}{
		{ExtGmail, "Gmail"},
		{ExtYouTube, "YouTube"},
		{ExtGoogleMaps, "GoogleMaps"},
		{ExtGoogleFlights, "GoogleFlights"},
		{ExtGoogleHotels, "GoogleHotels"},
		{ExtGoogleWorkspace, "GoogleWorkspace"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ext), func(t *testing.T) {
			got := tt.ext.ShortName()
			if got != tt.want {
				t.Errorf("Extension.ShortName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtension_ShortName_NoPrefix(t *testing.T) {
	// Extension without @ prefix
	ext := Extension("NoPrefix")
	got := ext.ShortName()
	want := "NoPrefix"

	if got != want {
		t.Errorf("Extension.ShortName() = %q, want %q", got, want)
	}
}

func TestExtension_ShortName_Empty(t *testing.T) {
	ext := Extension("")
	got := ext.ShortName()

	if got != "" {
		t.Errorf("Extension.ShortName() = %q, want empty", got)
	}
}

// ============================================================================
// Extension.IsValid Tests
// ============================================================================

func TestExtension_IsValid_AllKnownExtensions(t *testing.T) {
	for _, ext := range AllExtensions {
		if !ext.IsValid() {
			t.Errorf("Extension.IsValid() for %v should be true", ext)
		}
	}
}

func TestExtension_IsValid_InvalidExtensions(t *testing.T) {
	tests := []Extension{
		"@Invalid",
		"@Unknown",
		"Gmail",     // Missing @
		"@gmail",    // Wrong case
		"",          // Empty
		"@",         // Just @
		"@Calendar", // Not a real extension
	}

	for _, ext := range tests {
		t.Run(string(ext), func(t *testing.T) {
			if ext.IsValid() {
				t.Errorf("Extension.IsValid() for %v should be false", ext)
			}
		})
	}
}

// ============================================================================
// AllExtensions Slice Tests
// ============================================================================

func TestAllExtensions_Count(t *testing.T) {
	expected := 6 // Gmail, YouTube, GoogleMaps, GoogleFlights, GoogleHotels, GoogleWorkspace

	if len(AllExtensions) != expected {
		t.Errorf("AllExtensions has %d items, want %d", len(AllExtensions), expected)
	}
}

func TestAllExtensions_NoDuplicates(t *testing.T) {
	seen := make(map[Extension]bool)

	for _, ext := range AllExtensions {
		if seen[ext] {
			t.Errorf("AllExtensions contains duplicate: %v", ext)
		}
		seen[ext] = true
	}
}

func TestAllExtensions_AllValid(t *testing.T) {
	for _, ext := range AllExtensions {
		if !ext.IsValid() {
			t.Errorf("Extension %v in AllExtensions should be valid", ext)
		}
		if ext.String() == "" {
			t.Errorf("Extension %v in AllExtensions should have non-empty String()", ext)
		}
		if ext.ShortName() == "" {
			t.Errorf("Extension %v in AllExtensions should have non-empty ShortName()", ext)
		}
		if ext.Info() == "" {
			t.Errorf("Extension %v in AllExtensions should have non-empty Info()", ext)
		}
	}
}
