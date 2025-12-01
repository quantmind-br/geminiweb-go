package api

import (
	"github.com/diogo/geminiweb/internal/models"
)

// ChatSession maintains conversation context across messages
type ChatSession struct {
	client     *GeminiClient
	model      models.Model
	metadata   []string // [cid, rid, rcid]
	lastOutput *models.ModelOutput
}

// SendMessage sends a message in the chat session and updates context
func (s *ChatSession) SendMessage(prompt string) (*models.ModelOutput, error) {
	opts := &GenerateOptions{
		Model:    s.model,
		Metadata: s.metadata,
	}

	output, err := s.client.GenerateContent(prompt, opts)
	if err != nil {
		return nil, err
	}

	// Update session state
	s.lastOutput = output
	s.updateMetadata(output)

	return output, nil
}

// updateMetadata updates the session metadata from the response
func (s *ChatSession) updateMetadata(output *models.ModelOutput) {
	if len(output.Metadata) > 0 {
		s.metadata = make([]string, len(output.Metadata))
		copy(s.metadata, output.Metadata)
	}

	// Update rcid with the chosen candidate's RCID
	if len(s.metadata) >= 3 {
		s.metadata[2] = output.RCID()
	} else if len(s.metadata) == 2 {
		s.metadata = append(s.metadata, output.RCID())
	}
}

// SetMetadata allows setting metadata directly (for resuming conversations)
func (s *ChatSession) SetMetadata(cid, rid, rcid string) {
	s.metadata = []string{cid, rid, rcid}
}

// GetMetadata returns the current session metadata
func (s *ChatSession) GetMetadata() []string {
	return s.metadata
}

// CID returns the conversation ID
func (s *ChatSession) CID() string {
	if len(s.metadata) > 0 {
		return s.metadata[0]
	}
	return ""
}

// RID returns the reply ID
func (s *ChatSession) RID() string {
	if len(s.metadata) > 1 {
		return s.metadata[1]
	}
	return ""
}

// RCID returns the reply candidate ID
func (s *ChatSession) RCID() string {
	if len(s.metadata) > 2 {
		return s.metadata[2]
	}
	return ""
}

// GetModel returns the session's model
func (s *ChatSession) GetModel() models.Model {
	return s.model
}

// SetModel changes the session's model
func (s *ChatSession) SetModel(model models.Model) {
	s.model = model
}

// LastOutput returns the last response from the session
func (s *ChatSession) LastOutput() *models.ModelOutput {
	return s.lastOutput
}

// ChooseCandidate selects a different candidate from the last output
func (s *ChatSession) ChooseCandidate(index int) error {
	if s.lastOutput == nil {
		return nil
	}
	if index >= len(s.lastOutput.Candidates) {
		return nil
	}

	s.lastOutput.Chosen = index
	s.updateMetadata(s.lastOutput)
	return nil
}
