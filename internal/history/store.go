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

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *Store) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadConversation(id)
}

// ListConversations returns all conversations, sorted by most recent
func (s *Store) ListConversations() ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	var conversations []*Conversation
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // Remove .json
		conv, err := s.loadConversation(id)
		if err != nil {
			continue // Skip corrupted files
		}
		conversations = append(conversations, conv)
	}

	// Sort by UpdatedAt descending
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

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
	if role == "user" && len(conv.Messages) == 1 {
		title := content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv.Title = title
	}

	return s.saveConversation(conv)
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

	return s.saveConversation(conv)
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
