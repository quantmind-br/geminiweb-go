package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/tui"
)

func TestRunChat_Table(t *testing.T) {
	// Save and restore global state
	oldCreateFunc := createGemsClientFunc
	oldNewFlag := chatNewFlag
	oldFileFlag := chatFileFlag
	oldGemFlag := chatGemFlag
	oldPersonaFlag := chatPersonaFlag

	defer func() {
		createGemsClientFunc = oldCreateFunc
		chatNewFlag = oldNewFlag
		chatFileFlag = oldFileFlag
		chatGemFlag = oldGemFlag
		chatPersonaFlag = oldPersonaFlag
	}()

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	os.WriteFile(promptFile, []byte("file content"), 0644)

	tests := []struct {
		name        string
		newFlag     bool
		fileFlag    string
		gemFlag     string
		personaFlag string
		mockGems    []*api.RPCData // Simplified mock state
		wantErr     bool
		errMsg      string
	}{
		{
			name:    "new chat session",
			newFlag: true,
			wantErr: false,
		},
		{
			name:     "chat with file input",
			newFlag:  true,
			fileFlag: promptFile,
			wantErr:  false,
		},
		{
			name:     "non-existent file",
			newFlag:  true,
			fileFlag: "/non/existent",
			wantErr:  true,
			errMsg:   "file not found",
		},
		{
			name:    "with gem",
			newFlag: true,
			gemFlag: "test-gem",
			wantErr: false,
		},
		{
			name:        "with persona",
			newFlag:     true,
			personaFlag: "coder",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatNewFlag = tt.newFlag
			chatFileFlag = tt.fileFlag
			chatGemFlag = tt.gemFlag
			chatPersonaFlag = tt.personaFlag

			mockClient := &mockGeminiClient{
				closed: false,
				fetchGemsFunc: func(includeHidden bool) (*models.GemJar, error) {
					jar := make(models.GemJar)
					if tt.gemFlag != "" {
						jar["test-gem-id"] = &models.Gem{ID: "test-gem-id", Name: tt.gemFlag}
					}
					return &jar, nil
				},
			}

			mockTUI := &mockTUI{
				historyRes: tui.HistorySelectorResult{Confirmed: true, Conversation: &history.Conversation{ID: "123"}},
			}

			deps := &Dependencies{
				Client: mockClient,
				TUI:    mockTUI,
			}

			// We need to mock history.DefaultStore() or ensure it works in test
			// history.DefaultStore() uses HOME, so we should set it
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpDir)
			defer os.Setenv("HOME", oldHome)

			err := runChat(deps)

			if (err != nil) != tt.wantErr {
				t.Errorf("runChat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runChat() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}
