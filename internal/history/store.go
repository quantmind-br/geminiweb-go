// Package history provides local conversation history storage.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Thoughts  string    `json:"thoughts,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation represents a complete chat conversation
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`

	// Gemini API metadata for resuming
	CID  string `json:"cid,omitempty"`
	RID  string `json:"rid,omitempty"`
	RCID string `json:"rcid,omitempty"`

	// Computed fields (populated from HistoryMeta, not saved in conversation JSON)
	IsFavorite bool `json:"-"` // Populated by ListConversations
	OrderIndex int  `json:"-"` // Position in list (0-based, populated by ListConversations)
}

// Store manages conversation history persistence
type Store struct {
	baseDir string
	mu      sync.RWMutex
}

// NewStore creates a new history store
func NewStore(baseDir string) (*Store, error) {
	historyDir := filepath.Join(baseDir, "history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	return &Store{
		baseDir: historyDir,
	}, nil
}

// CreateConversation creates a new conversation
func (s *Store) CreateConversation(model string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	conv := &Conversation{
		ID:        generateConvID(),
		Title:     fmt.Sprintf("Chat %s", now.Format("2006-01-02 15:04")),
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}

	if err := s.saveConversation(conv); err != nil {
		return nil, err
	}

	// Add to meta.json at the beginning (most recent first)
	meta, err := s.loadMeta()
	if err != nil {
		// Don't fail if meta can't be loaded, conversation is already saved
		return conv, nil
	}

	meta.Order = append([]string{conv.ID}, meta.Order...)
	meta.Meta[conv.ID] = &ConversationMeta{
		ID:         conv.ID,
		Title:      conv.Title,
		IsFavorite: false,
	}
	_ = s.saveMeta(meta) // Ignore error, conversation is already saved

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *Store) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadConversation(id)
}

// ListConversations returns all conversations ordered by meta.json
// If no meta.json exists, falls back to sorting by UpdatedAt descending
// Populates computed fields IsFavorite and OrderIndex
func (s *Store) ListConversations() ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listConversationsLocked()
}

// listConversationsLocked is the internal implementation without locking
func (s *Store) listConversationsLocked() ([]*Conversation, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	// Load all conversations into a map
	convMap := make(map[string]*Conversation)
	existingIDs := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		// Skip meta.json
		if entry.Name() == metaFileName {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // Remove .json
		conv, err := s.loadConversation(id)
		if err != nil {
			continue // Skip corrupted files
		}
		convMap[id] = conv
		existingIDs[id] = true
	}

	// Load meta
	meta, err := s.loadMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to load meta: %w", err)
	}

	// Clean orphaned entries from meta
	if s.cleanOrphanedMeta(meta, existingIDs) {
		// Save cleaned meta (ignore error, it's just cleanup)
		_ = s.saveMeta(meta)
	}

	// Add any new conversations not in meta to the beginning of the order
	for id := range existingIDs {
		found := false
		for _, oid := range meta.Order {
			if oid == id {
				found = true
				break
			}
		}
		if !found {
			// Prepend new conversations (most recent at top)
			meta.Order = append([]string{id}, meta.Order...)
			meta.Meta[id] = &ConversationMeta{
				ID:         id,
				Title:      convMap[id].Title,
				IsFavorite: false,
			}
		}
	}

	// Build result in order
	var conversations []*Conversation
	for i, id := range meta.Order {
		if conv, exists := convMap[id]; exists {
			// Populate computed fields
			if m, ok := meta.Meta[id]; ok {
				conv.IsFavorite = m.IsFavorite
			}
			conv.OrderIndex = i
			conversations = append(conversations, conv)
		}
	}

	// If meta was empty, sort by UpdatedAt as fallback
	if len(meta.Order) == 0 && len(conversations) > 0 {
		sort.Slice(conversations, func(i, j int) bool {
			return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
		})
		// Update order indices
		for i := range conversations {
			conversations[i].OrderIndex = i
		}
	}

	return conversations, nil
}

// AddMessage adds a message to a conversation
func (s *Store) AddMessage(id, role, content, thoughts string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, err := s.loadConversation(id)
	if err != nil {
		return err
	}

	msg := Message{
		Role:      role,
		Content:   content,
		Thoughts:  thoughts,
		Timestamp: time.Now(),
	}

	conv.Messages = append(conv.Messages, msg)
	conv.UpdatedAt = time.Now()

	// Update title from first user message if still default
	titleUpdated := false
	if role == "user" && len(conv.Messages) == 1 {
		title := content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv.Title = title
		titleUpdated = true
	}

	if err := s.saveConversation(conv); err != nil {
		return err
	}

	// Sync title to meta.json if it was updated
	if titleUpdated {
		_ = s.updateTitleInMeta(id, conv.Title)
	}

	return nil
}

// UpdateMetadata updates the Gemini API metadata for a conversation
func (s *Store) UpdateMetadata(id, cid, rid, rcid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, err := s.loadConversation(id)
	if err != nil {
		return err
	}

	conv.CID = cid
	conv.RID = rid
	conv.RCID = rcid
	conv.UpdatedAt = time.Now()

	return s.saveConversation(conv)
}

// DeleteConversation removes a conversation
func (s *Store) DeleteConversation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.conversationPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("conversation not found: %s", id)
		}
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	// Remove from meta.json
	_ = s.removeFromMeta(id) // Ignore error, file is already deleted

	return nil
}

// UpdateTitle updates the title of a conversation
func (s *Store) UpdateTitle(id, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, err := s.loadConversation(id)
	if err != nil {
		return err
	}

	conv.Title = title
	conv.UpdatedAt = time.Now()

	if err := s.saveConversation(conv); err != nil {
		return err
	}

	// Update cached title in meta.json
	_ = s.updateTitleInMeta(id, title)

	return nil
}

// ClearAll deletes all conversations
func (s *Store) ClearAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read history directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.baseDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete %s: %w", entry.Name(), err)
		}
	}

	// Reset meta.json to empty
	_ = s.saveMeta(newHistoryMeta())

	return nil
}

// Internal methods

func (s *Store) conversationPath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}

func (s *Store) loadConversation(id string) (*Conversation, error) {
	path := s.conversationPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("conversation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read conversation: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("failed to parse conversation: %w", err)
	}

	return &conv, nil
}

func (s *Store) saveConversation(conv *Conversation) error {
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	path := s.conversationPath(conv.ID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write conversation: %w", err)
	}

	return nil
}

func generateConvID() string {
	return fmt.Sprintf("conv-%d", time.Now().UnixNano())
}

// GetHistoryDir returns the default history directory path
func GetHistoryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".geminiweb"), nil
}

// DefaultStore creates a store using the default location
func DefaultStore() (*Store, error) {
	dir, err := GetHistoryDir()
	if err != nil {
		return nil, err
	}
	return NewStore(dir)
}
