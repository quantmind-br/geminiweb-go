package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/tui"
)

type mockTUI struct {
	runGemsResult tui.GemsTUIResult
	runGemsErr    error
	runChatErr    error
	historyRes    tui.HistorySelectorResult
	historyErr    error
}

func (m *mockTUI) RunGemsTUI(client api.GeminiClientInterface, includeHidden bool) (tui.GemsTUIResult, error) {
	return m.runGemsResult, m.runGemsErr
}

func (m *mockTUI) RunChatWithSession(client api.GeminiClientInterface, session tui.ChatSessionInterface, modelName string) error {
	return m.runChatErr
}

func (m *mockTUI) RunHistorySelector(store tui.HistoryStore, modelName string) (tui.HistorySelectorResult, error) {
	return m.historyRes, m.historyErr
}

func (m *mockTUI) RunChatWithInitialPrompt(client api.GeminiClientInterface, session tui.ChatSessionInterface, modelName string, conv *history.Conversation, store tui.HistoryStoreInterface, gemName string, persona *config.Persona, initialPrompt string) error {
	return m.runChatErr
}

func TestRunGemsCreate_Table(t *testing.T) {
	// Save and restore global state
	oldCreateFunc := createGemsClientFunc
	oldPrompt := gemPrompt
	oldPromptFile := gemPromptFile
	oldDesc := gemDescription
	defer func() {
		createGemsClientFunc = oldCreateFunc
		gemPrompt = oldPrompt
		gemPromptFile = oldPromptFile
		gemDescription = oldDesc
	}()

	tests := []struct {
		name          string
		args          []string
		prompt        string
		promptFile    string
		description   string
		mockCreateErr error
		wantErr       bool
		errMsg        string
	}{
		{
			name:        "success with prompt flag",
			args:        []string{"test-gem"},
			prompt:      "system prompt",
			description: "desc",
			wantErr:     false,
		},
		{
			name:    "missing prompt",
			args:    []string{"test-gem"},
			prompt:  "",
			wantErr: true,
			errMsg:  "prompt is required",
		},
		{
			name:          "client error",
			args:          []string{"test-gem"},
			prompt:        "prompt",
			mockCreateErr: fmt.Errorf("api error"),
			wantErr:       true,
			errMsg:        "failed to create gem: api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gemPrompt = tt.prompt
			gemPromptFile = tt.promptFile
			gemDescription = tt.description

			mockClient := &mockGeminiClientForGems{
				mockGeminiClient: &mockGeminiClient{
					createGemFunc: func(name, prompt, description string) (*models.Gem, error) {
						if tt.mockCreateErr != nil {
							return nil, tt.mockCreateErr
						}
						return &models.Gem{ID: "gem-123", Name: name}, nil
					},
				},
			}

			createGemsClientFunc = func() (api.GeminiClientInterface, error) {
				return mockClient, nil
			}

			err := runGemsCreate(nil, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("runGemsCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runGemsCreate() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}

func TestRunGemsDelete_Table(t *testing.T) {
	oldCreateFunc := createGemsClientFunc
	defer func() { createGemsClientFunc = oldCreateFunc }()

	tests := []struct {
		name          string
		args          []string
		gems          []*models.Gem
		mockDeleteErr error
		wantErr       bool
		errMsg        string
	}{
		{
			name: "success delete by ID",
			args: []string{"gem-123"},
			gems: []*models.Gem{
				{ID: "gem-123", Name: "My Gem"},
			},
			wantErr: false,
		},
		{
			name:    "gem not found",
			args:    []string{"missing"},
			gems:    []*models.Gem{},
			wantErr: true,
			errMsg:  "gem 'missing' not found",
		},
		{
			name: "cannot delete system gem",
			args: []string{"system-gem"},
			gems: []*models.Gem{
				{ID: "system-gem", Name: "System", Predefined: true},
			},
			wantErr: true,
			errMsg:  "cannot delete system gems",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGeminiClientForGems{
				mockGeminiClient: &mockGeminiClient{
					fetchGemsFunc: func(includeHidden bool) (*models.GemJar, error) {
						jar := make(models.GemJar)
						for _, g := range tt.gems {
							jar[g.ID] = g
						}
						return &jar, nil
					},
					deleteGemFunc: func(id string) error {
						return tt.mockDeleteErr
					},
				},
			}

			createGemsClientFunc = func() (api.GeminiClientInterface, error) {
				return mockClient, nil
			}

			err := runGemsDelete(nil, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("runGemsDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runGemsDelete() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}

func TestRunGemsUpdate_Table(t *testing.T) {
	oldCreateFunc := createGemsClientFunc
	oldPrompt := gemPrompt
	oldDesc := gemDescription
	oldName := gemName
	defer func() {
		createGemsClientFunc = oldCreateFunc
		gemPrompt = oldPrompt
		gemDescription = oldDesc
		gemName = oldName
	}()

	tests := []struct {
		name          string
		args          []string
		prompt        string
		description   string
		newName       string
		gems          []*models.Gem
		mockUpdateErr error
		wantErr       bool
		errMsg        string
	}{
		{
			name:    "success update name and prompt",
			args:    []string{"gem-1"},
			prompt:  "new prompt",
			newName: "New Name",
			gems: []*models.Gem{
				{ID: "gem-1", Name: "Old Name", Prompt: "old prompt"},
			},
			wantErr: false,
		},
		{
			name:    "gem not found",
			args:    []string{"missing"},
			gems:    []*models.Gem{},
			wantErr: true,
			errMsg:  "gem 'missing' not found",
		},
		{
			name: "cannot update system gem",
			args: []string{"sys-1"},
			gems: []*models.Gem{
				{ID: "sys-1", Name: "System", Predefined: true},
			},
			wantErr: true,
			errMsg:  "cannot update system gems",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gemPrompt = tt.prompt
			gemDescription = tt.description
			gemName = tt.newName

			mockClient := &mockGeminiClientForGems{
				mockGeminiClient: &mockGeminiClient{
					fetchGemsFunc: func(includeHidden bool) (*models.GemJar, error) {
						jar := make(models.GemJar)
						for _, g := range tt.gems {
							jar[g.ID] = g
						}
						return &jar, nil
					},
					updateGemFunc: func(id, name, prompt, description string) (*models.Gem, error) {
						if tt.mockUpdateErr != nil {
							return nil, tt.mockUpdateErr
						}
						return &models.Gem{ID: id, Name: name}, nil
					},
				},
			}

			createGemsClientFunc = func() (api.GeminiClientInterface, error) {
				return mockClient, nil
			}

			err := runGemsUpdate(nil, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("runGemsUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runGemsUpdate() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}

func TestRunGemsList_Table(t *testing.T) {
	oldCreateFunc := createGemsClientFunc
	defer func() { createGemsClientFunc = oldCreateFunc }()

	tests := []struct {
		name        string
		mockGemsRes tui.GemsTUIResult
		mockGemsErr error
		mockChatErr error
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "success list and quit",
			mockGemsRes: tui.GemsTUIResult{GemID: ""},
			wantErr:     false,
		},
		{
			name:        "success list and start chat",
			mockGemsRes: tui.GemsTUIResult{GemID: "gem-123"},
			wantErr:     false,
		},
		{
			name:        "TUI error",
			mockGemsErr: fmt.Errorf("tui failed"),
			wantErr:     true,
			errMsg:      "tui failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGeminiClientForGems{
				mockGeminiClient: &mockGeminiClient{},
			}

			createGemsClientFunc = func() (api.GeminiClientInterface, error) {
				return mockClient, nil
			}

			mockTUI := &mockTUI{
				runGemsResult: tt.mockGemsRes,
				runGemsErr:    tt.mockGemsErr,
				runChatErr:    tt.mockChatErr,
			}

			deps := &Dependencies{
				TUI: mockTUI,
			}

			err := runGemsList(deps, []string{})

			if (err != nil) != tt.wantErr {
				t.Errorf("runGemsList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runGemsList() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}

func TestRunGemsShow_Table(t *testing.T) {
	oldCreateFunc := createGemsClientFunc
	defer func() { createGemsClientFunc = oldCreateFunc }()

	tests := []struct {
		name    string
		args    []string
		gems    []*models.Gem
		wantErr bool
		errMsg  string
	}{
		{
			name: "success show custom gem",
			args: []string{"gem-123"},
			gems: []*models.Gem{
				{ID: "gem-123", Name: "My Gem", Description: "Desc", Prompt: "Prompt"},
			},
			wantErr: false,
		},
		{
			name: "success show system gem",
			args: []string{"sys-1"},
			gems: []*models.Gem{
				{ID: "sys-1", Name: "System", Predefined: true},
			},
			wantErr: false,
		},
		{
			name:    "gem not found",
			args:    []string{"missing"},
			gems:    []*models.Gem{},
			wantErr: true,
			errMsg:  "gem 'missing' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGeminiClientForGems{
				mockGeminiClient: &mockGeminiClient{
					fetchGemsFunc: func(includeHidden bool) (*models.GemJar, error) {
						jar := make(models.GemJar)
						for _, g := range tt.gems {
							jar[g.ID] = g
						}
						return &jar, nil
					},
				},
			}

			createGemsClientFunc = func() (api.GeminiClientInterface, error) {
				return mockClient, nil
			}

			err := runGemsShow(nil, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("runGemsShow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runGemsShow() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}

// TestNewGemsDeleteCmd tests the command constructor
func TestNewGemsDeleteCmd(t *testing.T) {
	deps := &Dependencies{}
	cmd := NewGemsDeleteCmd(deps)

	if cmd == nil {
		t.Fatal("NewGemsDeleteCmd() returned nil")
	}

	if cmd.Use != "delete <id-or-name>" {
		t.Errorf("expected Use 'delete <id-or-name>', got '%s'", cmd.Use)
	}

	if cmd.Short != "Delete a gem" {
		t.Errorf("expected Short 'Delete a gem', got '%s'", cmd.Short)
	}

	if cmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

// TestNewGemsShowCmd tests the command constructor
func TestNewGemsShowCmd(t *testing.T) {
	deps := &Dependencies{}
	cmd := NewGemsShowCmd(deps)

	if cmd == nil {
		t.Fatal("NewGemsShowCmd() returned nil")
	}

	if cmd.Use != "show <id-or-name>" {
		t.Errorf("expected Use 'show <id-or-name>', got '%s'", cmd.Use)
	}

	if cmd.Short != "Show gem details" {
		t.Errorf("expected Short 'Show gem details', got '%s'", cmd.Short)
	}

	if cmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

// TestNewGemsCreateCmd tests the command constructor
func TestNewGemsCreateCmd(t *testing.T) {
	deps := &Dependencies{}
	cmd := NewGemsCreateCmd(deps)

	if cmd == nil {
		t.Fatal("NewGemsCreateCmd() returned nil")
	}

	if cmd.Use != "create <name>" {
		t.Errorf("expected Use 'create <name>', got '%s'", cmd.Use)
	}

	if cmd.Short != "Create a new gem" {
		t.Errorf("expected Short 'Create a new gem', got '%s'", cmd.Short)
	}

	if cmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Verify flags
	flag := cmd.Flags().Lookup("prompt")
	if flag == nil {
		t.Error("missing 'prompt' flag")
	}

	flag = cmd.Flags().Lookup("file")
	if flag == nil {
		t.Error("missing 'file' flag")
	}

	flag = cmd.Flags().Lookup("description")
	if flag == nil {
		t.Error("missing 'description' flag")
	}
}

// TestNewGemsUpdateCmd tests the command constructor
func TestNewGemsUpdateCmd(t *testing.T) {
	deps := &Dependencies{}
	cmd := NewGemsUpdateCmd(deps)

	if cmd == nil {
		t.Fatal("NewGemsUpdateCmd() returned nil")
	}

	if cmd.Use != "update <id-or-name>" {
		t.Errorf("expected Use 'update <id-or-name>', got '%s'", cmd.Use)
	}

	if cmd.Short != "Update an existing gem" {
		t.Errorf("expected Short 'Update an existing gem', got '%s'", cmd.Short)
	}

	if cmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Verify flags
	flag := cmd.Flags().Lookup("prompt")
	if flag == nil {
		t.Error("missing 'prompt' flag")
	}

	flag = cmd.Flags().Lookup("file")
	if flag == nil {
		t.Error("missing 'file' flag")
	}

	flag = cmd.Flags().Lookup("description")
	if flag == nil {
		t.Error("missing 'description' flag")
	}

	flag = cmd.Flags().Lookup("name")
	if flag == nil {
		t.Error("missing 'name' flag")
	}
}
