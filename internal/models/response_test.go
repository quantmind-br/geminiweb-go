package models

import (
	"testing"
)

// ============================================================================
// ChosenCandidate Tests
// ============================================================================

func TestModelOutput_ChosenCandidate_FirstCandidate(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{RCID: "rcid1", Text: "First candidate"},
			{RCID: "rcid2", Text: "Second candidate"},
		},
		Chosen: 0,
	}

	candidate := output.ChosenCandidate()
	if candidate == nil {
		t.Fatal("ChosenCandidate() returned nil")
	}

	if candidate.RCID != "rcid1" {
		t.Errorf("ChosenCandidate().RCID = %s, want rcid1", candidate.RCID)
	}

	if candidate.Text != "First candidate" {
		t.Errorf("ChosenCandidate().Text = %s, want 'First candidate'", candidate.Text)
	}
}

func TestModelOutput_ChosenCandidate_SecondCandidate(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{RCID: "rcid1", Text: "First candidate"},
			{RCID: "rcid2", Text: "Second candidate"},
		},
		Chosen: 1,
	}

	candidate := output.ChosenCandidate()
	if candidate == nil {
		t.Fatal("ChosenCandidate() returned nil")
	}

	if candidate.RCID != "rcid2" {
		t.Errorf("ChosenCandidate().RCID = %s, want rcid2", candidate.RCID)
	}
}

func TestModelOutput_ChosenCandidate_EmptyCandidates(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{},
		Chosen:     0,
	}

	candidate := output.ChosenCandidate()
	if candidate != nil {
		t.Errorf("ChosenCandidate() should return nil for empty candidates, got %v", candidate)
	}
}

func TestModelOutput_ChosenCandidate_OutOfBounds(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{RCID: "rcid1", Text: "Only candidate"},
		},
		Chosen: 5, // Out of bounds
	}

	candidate := output.ChosenCandidate()
	if candidate == nil {
		t.Fatal("ChosenCandidate() returned nil for out of bounds Chosen")
	}

	// Should fallback to first candidate
	if candidate.RCID != "rcid1" {
		t.Errorf("ChosenCandidate() should fallback to first candidate, got RCID=%s", candidate.RCID)
	}
}

// Note: Negative Chosen values cause a panic (intentional - invalid input).
// Users should never set Chosen to a negative value.

// ============================================================================
// Text Tests
// ============================================================================

func TestModelOutput_Text_ChosenCandidate(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{Text: "First"},
			{Text: "Second"},
		},
		Chosen: 1,
	}

	text := output.Text()
	if text != "Second" {
		t.Errorf("Text() = %s, want 'Second'", text)
	}
}

func TestModelOutput_Text_EmptyCandidates(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{},
	}

	text := output.Text()
	if text != "" {
		t.Errorf("Text() = %s, want empty string", text)
	}
}

func TestModelOutput_Text_OutOfBounds(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{Text: "First"},
		},
		Chosen: 10,
	}

	text := output.Text()
	if text != "First" {
		t.Errorf("Text() = %s, want 'First' (fallback)", text)
	}
}

// ============================================================================
// Thoughts Tests
// ============================================================================

func TestModelOutput_Thoughts_ChosenCandidate(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{Thoughts: "Thinking 1"},
			{Thoughts: "Thinking 2"},
		},
		Chosen: 0,
	}

	thoughts := output.Thoughts()
	if thoughts != "Thinking 1" {
		t.Errorf("Thoughts() = %s, want 'Thinking 1'", thoughts)
	}
}

func TestModelOutput_Thoughts_EmptyCandidates(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{},
	}

	thoughts := output.Thoughts()
	if thoughts != "" {
		t.Errorf("Thoughts() = %s, want empty string", thoughts)
	}
}

// ============================================================================
// RCID Tests
// ============================================================================

func TestModelOutput_RCID_ChosenCandidate(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{RCID: "rcid-1"},
			{RCID: "rcid-2"},
		},
		Chosen: 1,
	}

	rcid := output.RCID()
	if rcid != "rcid-2" {
		t.Errorf("RCID() = %s, want 'rcid-2'", rcid)
	}
}

func TestModelOutput_RCID_EmptyCandidates(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{},
	}

	rcid := output.RCID()
	if rcid != "" {
		t.Errorf("RCID() = %s, want empty string", rcid)
	}
}

// ============================================================================
// Images Tests
// ============================================================================

func TestModelOutput_Images_WebOnly(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{
				WebImages: []WebImage{
					{URL: "http://example.com/1.png"},
					{URL: "http://example.com/2.png"},
				},
			},
		},
		Chosen: 0,
	}

	images := output.Images()
	if len(images) != 2 {
		t.Errorf("Images() returned %d images, want 2", len(images))
	}
}

func TestModelOutput_Images_GeneratedOnly(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{
				GeneratedImages: []GeneratedImage{
					{URL: "http://example.com/gen1.png"},
				},
			},
		},
		Chosen: 0,
	}

	images := output.Images()
	if len(images) != 1 {
		t.Errorf("Images() returned %d images, want 1", len(images))
	}
}

func TestModelOutput_Images_Mixed(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{
			{
				WebImages: []WebImage{
					{URL: "http://example.com/web.png"},
				},
				GeneratedImages: []GeneratedImage{
					{URL: "http://example.com/gen.png"},
				},
			},
		},
		Chosen: 0,
	}

	images := output.Images()
	if len(images) != 2 {
		t.Errorf("Images() returned %d images, want 2", len(images))
	}
}

func TestModelOutput_Images_EmptyCandidates(t *testing.T) {
	output := &ModelOutput{
		Candidates: []Candidate{},
	}

	images := output.Images()
	if images != nil {
		t.Errorf("Images() should return nil for empty candidates, got %v", images)
	}
}

// CID and RID tests are in models_test.go

// ============================================================================
// Candidate Type Tests
// ============================================================================

func TestCandidate_Fields(t *testing.T) {
	candidate := Candidate{
		RCID:     "test-rcid",
		Text:     "Test text",
		Thoughts: "Test thoughts",
		WebImages: []WebImage{
			{URL: "http://example.com/img.png", Title: "Test", Alt: "Alt text"},
		},
		GeneratedImages: []GeneratedImage{
			{URL: "http://example.com/gen.png", Title: "Generated", Alt: "Generated alt"},
		},
	}

	if candidate.RCID != "test-rcid" {
		t.Errorf("Candidate.RCID = %s, want test-rcid", candidate.RCID)
	}
	if candidate.Text != "Test text" {
		t.Errorf("Candidate.Text = %s, want 'Test text'", candidate.Text)
	}
	if len(candidate.WebImages) != 1 {
		t.Errorf("Candidate.WebImages has %d items, want 1", len(candidate.WebImages))
	}
	if len(candidate.GeneratedImages) != 1 {
		t.Errorf("Candidate.GeneratedImages has %d items, want 1", len(candidate.GeneratedImages))
	}
}

// ============================================================================
// WebImage and GeneratedImage Type Tests
// ============================================================================

func TestWebImage_Fields(t *testing.T) {
	img := WebImage{
		URL:   "http://example.com/test.png",
		Title: "Test Title",
		Alt:   "Alt Text",
	}

	if img.URL != "http://example.com/test.png" {
		t.Errorf("WebImage.URL = %s", img.URL)
	}
	if img.Title != "Test Title" {
		t.Errorf("WebImage.Title = %s", img.Title)
	}
	if img.Alt != "Alt Text" {
		t.Errorf("WebImage.Alt = %s", img.Alt)
	}
}

func TestGeneratedImage_Fields(t *testing.T) {
	img := GeneratedImage{
		URL:   "http://example.com/generated.png",
		Title: "Generated Title",
		Alt:   "Generated Alt",
	}

	if img.URL != "http://example.com/generated.png" {
		t.Errorf("GeneratedImage.URL = %s", img.URL)
	}
	if img.Title != "Generated Title" {
		t.Errorf("GeneratedImage.Title = %s", img.Title)
	}
	if img.Alt != "Generated Alt" {
		t.Errorf("GeneratedImage.Alt = %s", img.Alt)
	}
}

// Test that GeneratedImage can be converted to WebImage
func TestGeneratedImageToWebImage(t *testing.T) {
	gen := GeneratedImage{
		URL:   "http://example.com/gen.png",
		Title: "Generated",
		Alt:   "Alt",
	}

	// This conversion is used in Images()
	web := WebImage(gen)

	if web.URL != gen.URL {
		t.Errorf("Converted WebImage.URL = %s, want %s", web.URL, gen.URL)
	}
	if web.Title != gen.Title {
		t.Errorf("Converted WebImage.Title = %s, want %s", web.Title, gen.Title)
	}
	if web.Alt != gen.Alt {
		t.Errorf("Converted WebImage.Alt = %s, want %s", web.Alt, gen.Alt)
	}
}
