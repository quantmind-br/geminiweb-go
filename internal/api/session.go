package api

import (
	"sync"

	"github.com/diogo/geminiweb/internal/models"
)

// ChatSession maintains conversation context across messages
type ChatSession struct {
	client     *GeminiClient
	mu         sync.RWMutex // Protects metadata, lastOutput, gemID, model
	model      models.Model
	metadata   []string // [cid, rid, rcid]
	lastOutput *models.ModelOutput
	gemID      string // ID do gem associado à sessão (server-side persona)
}

// copyMetadata creates a copy of the metadata slice to avoid races
func copyMetadata(m []string) []string {
	if m == nil {
		return nil
	}
	result := make([]string, len(m))
	copy(result, m)
	return result
}

// SendMessage sends a message in the chat session and updates context
// files is optional - pass nil when no files are attached
func (s *ChatSession) SendMessage(prompt string, files []*UploadedFile) (*models.ModelOutput, error) {
	// Read current state with read lock
	s.mu.RLock()
	opts := &GenerateOptions{
		Model:    s.model,
		Metadata: copyMetadata(s.metadata), // Copy to avoid race
		GemID:    s.gemID,
		Files:    files,
	}
	s.mu.RUnlock()

	// GenerateContent is thread-safe, no lock needed
	output, err := s.client.GenerateContent(prompt, opts)
	if err != nil {
		return nil, err
	}

	// Update state with write lock
	s.mu.Lock()
	s.lastOutput = output
	s.updateMetadataLocked(output)
	s.mu.Unlock()

	return output, nil
}

// updateMetadataLocked updates the session metadata from the response
// MUST be called with s.mu.Lock() held
func (s *ChatSession) updateMetadataLocked(output *models.ModelOutput) {
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

// updateMetadata is a convenience method that acquires the lock and calls updateMetadataLocked
// Exported for testing purposes
func (s *ChatSession) updateMetadata(output *models.ModelOutput) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateMetadataLocked(output)
}

// SetMetadata allows setting metadata directly (for resuming conversations)
func (s *ChatSession) SetMetadata(cid, rid, rcid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata = []string{cid, rid, rcid}
}

// GetMetadata returns the current session metadata (returns a copy)
func (s *ChatSession) GetMetadata() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMetadata(s.metadata)
}

// CID returns the conversation ID
func (s *ChatSession) CID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.metadata) > 0 {
		return s.metadata[0]
	}
	return ""
}

// RID returns the reply ID
func (s *ChatSession) RID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.metadata) > 1 {
		return s.metadata[1]
	}
	return ""
}

// RCID returns the reply candidate ID
func (s *ChatSession) RCID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.metadata) > 2 {
		return s.metadata[2]
	}
	return ""
}

// GetModel returns the session's model
func (s *ChatSession) GetModel() models.Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.model
}

// SetModel changes the session's model
func (s *ChatSession) SetModel(model models.Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
}

// LastOutput returns the last response from the session
func (s *ChatSession) LastOutput() *models.ModelOutput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastOutput
}

// ChooseCandidate selects a different candidate from the last output
func (s *ChatSession) ChooseCandidate(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastOutput == nil {
		return nil
	}
	if index >= len(s.lastOutput.Candidates) {
		return nil
	}

	s.lastOutput.Chosen = index
	s.updateMetadataLocked(s.lastOutput)
	return nil
}

// SetGem define o gem para a sessão
func (s *ChatSession) SetGem(gemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gemID = gemID
}

// GetGemID retorna o gem ID da sessão
func (s *ChatSession) GetGemID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gemID
}
