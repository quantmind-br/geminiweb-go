package models

// Candidate represents a single response candidate from Gemini
type Candidate struct {
	RCID            string
	Text            string
	Thoughts        string // Only populated for thinking models
	WebImages       []WebImage
	GeneratedImages []GeneratedImage
}

// WebImage represents an image from web search results
type WebImage struct {
	URL   string
	Title string
	Alt   string
}

// GeneratedImage represents an AI-generated image
type GeneratedImage struct {
	URL   string
	Title string
	Alt   string
}

// ModelOutput represents the complete API response from Gemini
type ModelOutput struct {
	Metadata            []string // [cid, rid, rcid]
	Candidates          []Candidate
	Chosen              int  // Index of selected candidate
	IsExtensionResponse bool // True if response came from an extension (@Gmail, @YouTube, etc.)
}

// Text returns the chosen candidate's text
func (m *ModelOutput) Text() string {
	if len(m.Candidates) == 0 {
		return ""
	}
	if m.Chosen >= len(m.Candidates) {
		return m.Candidates[0].Text
	}
	return m.Candidates[m.Chosen].Text
}

// Thoughts returns the chosen candidate's thoughts
func (m *ModelOutput) Thoughts() string {
	if len(m.Candidates) == 0 {
		return ""
	}
	if m.Chosen >= len(m.Candidates) {
		return m.Candidates[0].Thoughts
	}
	return m.Candidates[m.Chosen].Thoughts
}

// RCID returns the chosen candidate's RCID
func (m *ModelOutput) RCID() string {
	if len(m.Candidates) == 0 {
		return ""
	}
	if m.Chosen >= len(m.Candidates) {
		return m.Candidates[0].RCID
	}
	return m.Candidates[m.Chosen].RCID
}

// ChosenCandidate returns a pointer to the chosen candidate
func (m *ModelOutput) ChosenCandidate() *Candidate {
	if len(m.Candidates) == 0 {
		return nil
	}
	if m.Chosen >= len(m.Candidates) {
		return &m.Candidates[0]
	}
	return &m.Candidates[m.Chosen]
}

// Images returns all images from the chosen candidate (web + generated)
func (m *ModelOutput) Images() []WebImage {
	if len(m.Candidates) == 0 {
		return nil
	}
	candidate := m.Candidates[m.Chosen]

	images := make([]WebImage, 0, len(candidate.WebImages)+len(candidate.GeneratedImages))
	images = append(images, candidate.WebImages...)

	// Convert generated images to WebImage format
	for _, img := range candidate.GeneratedImages {
		images = append(images, WebImage(img))
	}

	return images
}

// CID returns the conversation ID from metadata
func (m *ModelOutput) CID() string {
	if len(m.Metadata) > 0 {
		return m.Metadata[0]
	}
	return ""
}

// RID returns the reply ID from metadata
func (m *ModelOutput) RID() string {
	if len(m.Metadata) > 1 {
		return m.Metadata[1]
	}
	return ""
}
