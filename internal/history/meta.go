// Package history provides local conversation history storage.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	metaFileName    = "meta.json"
	metaVersion     = 1
	metaFileComment = "// This file is auto-generated. Manual edits may be overwritten."
)

// ConversationMeta stores global metadata per conversation
type ConversationMeta struct {
	ID         string `json:"id"`
	Title      string `json:"title"`       // Cached title for quick listing
	IsFavorite bool   `json:"is_favorite"`
}

// HistoryMeta stores the order and favorites for all conversations
type HistoryMeta struct {
	Version int                          `json:"version"` // For future migration
	Order   []string                     `json:"order"`   // IDs in display order
	Meta    map[string]*ConversationMeta `json:"meta"`    // Metadata per ID
}

// newHistoryMeta creates a new empty HistoryMeta
func newHistoryMeta() *HistoryMeta {
	return &HistoryMeta{
		Version: metaVersion,
		Order:   []string{},
		Meta:    make(map[string]*ConversationMeta),
	}
}

// metaPath returns the path to the meta.json file
func (s *Store) metaPath() string {
	return filepath.Join(s.baseDir, metaFileName)
}

// loadMeta loads the metadata from meta.json
// If the file doesn't exist, returns a new empty HistoryMeta
func (s *Store) loadMeta() (*HistoryMeta, error) {
	path := s.metaPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newHistoryMeta(), nil
		}
		return nil, fmt.Errorf("failed to read meta file: %w", err)
	}

	var meta HistoryMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse meta file: %w", err)
	}

	// Ensure maps are initialized
	if meta.Meta == nil {
		meta.Meta = make(map[string]*ConversationMeta)
	}
	if meta.Order == nil {
		meta.Order = []string{}
	}

	return &meta, nil
}

// saveMeta saves the metadata to meta.json
func (s *Store) saveMeta(meta *HistoryMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meta: %w", err)
	}

	path := s.metaPath()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write meta file: %w", err)
	}

	return nil
}

// removeFromMeta removes a conversation from metadata
func (s *Store) removeFromMeta(id string) error {
	meta, err := s.loadMeta()
	if err != nil {
		return err
	}

	// Remove from order
	newOrder := make([]string, 0, len(meta.Order))
	for _, oid := range meta.Order {
		if oid != id {
			newOrder = append(newOrder, oid)
		}
	}
	meta.Order = newOrder

	// Remove from meta map
	delete(meta.Meta, id)

	return s.saveMeta(meta)
}

// updateTitleInMeta updates the cached title in metadata
func (s *Store) updateTitleInMeta(id, title string) error {
	meta, err := s.loadMeta()
	if err != nil {
		return err
	}

	if m, exists := meta.Meta[id]; exists {
		m.Title = title
		return s.saveMeta(meta)
	}

	return nil
}

// IsFavorite returns whether a conversation is marked as favorite
func (s *Store) IsFavorite(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, err := s.loadMeta()
	if err != nil {
		return false, err
	}

	if m, exists := meta.Meta[id]; exists {
		return m.IsFavorite, nil
	}

	return false, nil
}

// ToggleFavorite toggles the favorite status of a conversation
// Returns the new favorite status
func (s *Store) ToggleFavorite(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify conversation exists
	if _, err := s.loadConversation(id); err != nil {
		return false, err
	}

	meta, err := s.loadMeta()
	if err != nil {
		return false, err
	}

	// Ensure conversation is in meta
	if _, exists := meta.Meta[id]; !exists {
		conv, _ := s.loadConversation(id)
		meta.Meta[id] = &ConversationMeta{
			ID:         id,
			Title:      conv.Title,
			IsFavorite: false,
		}
		// Also add to order if not present
		found := false
		for _, oid := range meta.Order {
			if oid == id {
				found = true
				break
			}
		}
		if !found {
			meta.Order = append(meta.Order, id)
		}
	}

	// Toggle favorite
	meta.Meta[id].IsFavorite = !meta.Meta[id].IsFavorite
	newStatus := meta.Meta[id].IsFavorite

	if err := s.saveMeta(meta); err != nil {
		return false, err
	}

	return newStatus, nil
}

// SetFavorite sets the favorite status of a conversation to a specific value
func (s *Store) SetFavorite(id string, isFavorite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify conversation exists
	if _, err := s.loadConversation(id); err != nil {
		return err
	}

	meta, err := s.loadMeta()
	if err != nil {
		return err
	}

	// Ensure conversation is in meta
	if _, exists := meta.Meta[id]; !exists {
		conv, _ := s.loadConversation(id)
		meta.Meta[id] = &ConversationMeta{
			ID:         id,
			Title:      conv.Title,
			IsFavorite: false,
		}
	}

	meta.Meta[id].IsFavorite = isFavorite

	return s.saveMeta(meta)
}

// MoveConversation moves a conversation to a new position in the order
// newIndex is 0-based
func (s *Store) MoveConversation(id string, newIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, err := s.loadMeta()
	if err != nil {
		return err
	}

	// Find current position
	currentIndex := -1
	for i, oid := range meta.Order {
		if oid == id {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return fmt.Errorf("conversation not found in order: %s", id)
	}

	// Validate new index
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(meta.Order) {
		newIndex = len(meta.Order) - 1
	}

	// No change needed
	if currentIndex == newIndex {
		return nil
	}

	// Remove from current position
	meta.Order = append(meta.Order[:currentIndex], meta.Order[currentIndex+1:]...)

	// Insert at new position
	meta.Order = append(meta.Order[:newIndex], append([]string{id}, meta.Order[newIndex:]...)...)

	return s.saveMeta(meta)
}

// SwapConversations swaps the positions of two conversations
func (s *Store) SwapConversations(id1, id2 string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, err := s.loadMeta()
	if err != nil {
		return err
	}

	// Find positions
	idx1, idx2 := -1, -1
	for i, oid := range meta.Order {
		if oid == id1 {
			idx1 = i
		}
		if oid == id2 {
			idx2 = i
		}
	}

	if idx1 == -1 {
		return fmt.Errorf("conversation not found: %s", id1)
	}
	if idx2 == -1 {
		return fmt.Errorf("conversation not found: %s", id2)
	}

	// Swap
	meta.Order[idx1], meta.Order[idx2] = meta.Order[idx2], meta.Order[idx1]

	return s.saveMeta(meta)
}

// GetOrderIndex returns the position of a conversation in the order (0-based)
// Returns -1 if not found
func (s *Store) GetOrderIndex(id string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, err := s.loadMeta()
	if err != nil {
		return -1, err
	}

	for i, oid := range meta.Order {
		if oid == id {
			return i, nil
		}
	}

	return -1, nil
}

// cleanOrphanedMeta removes entries from meta that don't have corresponding conversation files
// This is called during ListConversations to maintain consistency
func (s *Store) cleanOrphanedMeta(meta *HistoryMeta, existingIDs map[string]bool) bool {
	changed := false

	// Clean order
	newOrder := make([]string, 0, len(meta.Order))
	for _, id := range meta.Order {
		if existingIDs[id] {
			newOrder = append(newOrder, id)
		} else {
			changed = true
		}
	}
	meta.Order = newOrder

	// Clean meta map
	for id := range meta.Meta {
		if !existingIDs[id] {
			delete(meta.Meta, id)
			changed = true
		}
	}

	return changed
}
