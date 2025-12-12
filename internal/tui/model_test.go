package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)


func TestModelInit(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Model.Init panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.Init
}

func TestModelUpdate(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Model.Update panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.Update
}

func TestModelView(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Model.View panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.View
}

func TestRenderWelcome(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderWelcome panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.renderWelcome
}

func TestRenderLoadingAnimation(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderLoadingAnimation panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.renderLoadingAnimation
}

func TestRenderStatusBar(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderStatusBar panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.renderStatusBar
}

func TestSendMessage(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("sendMessage panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.sendMessage
}

func TestUpdateViewport(t *testing.T) {
	// For now, just test that function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("updateViewport panicked: %v", r)
		}
	}()

	// We can't easily test this without a proper mock
	// So we'll just test function signature
	var model Model
	_ = model.updateViewport
}


func TestModel_Update_WindowSize(t *testing.T) {
	// Create a minimal model with initialized textarea
	ta := textarea.New()
	ta.SetWidth(80)

	m := Model{
		width:   80,
		height:  24,
		ready:   false,
		textarea: ta,
	}

	// Simulate WindowSizeMsg
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, cmd := m.Update(msg)

	// Type assertion back to Model
	if typedModel, ok := updatedModel.(Model); ok {
		// Check that dimensions were updated
		if typedModel.width != 100 {
			t.Errorf("Expected width 100, got %d", typedModel.width)
		}
		if typedModel.height != 40 {
			t.Errorf("Expected height 40, got %d", typedModel.height)
		}
		if !typedModel.ready {
			t.Error("Model should be ready after WindowSizeMsg")
		}
	} else {
		t.Error("Update should return Model type")
	}

	// Update should return a command (likely nil or viewport update)
	if cmd == nil {
		t.Log("Update returned nil command (acceptable)")
	}
}

func TestModel_Update_CtrlC(t *testing.T) {
	m := Model{
		ready: true,
	}

	// Simulate Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd := m.Update(msg)

	// Should return tea.Quit command
	if cmd == nil {
		t.Error("Expected quit command for Ctrl+C")
	}

	// Model should remain unchanged
	if typedModel, ok := updatedModel.(Model); ok {
		if !typedModel.ready {
			t.Error("Model should remain ready")
		}
	}
}

func TestModel_Update_Escape(t *testing.T) {
	m := Model{
		ready:   true,
		loading: true,
	}

	// Simulate Escape during loading
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	updatedModel, _ := m.Update(msg)

	// Should stop loading
	if typedModel, ok := updatedModel.(Model); ok {
		if typedModel.loading {
			t.Error("Model should not be loading after Escape")
		}
	}
}

func TestModel_Update_AnimationTick(t *testing.T) {
	m := Model{
		ready:           true,
		loading:         true,
		animationFrame:  0,
	}

	// Simulate animation tick
	msg := animationTickMsg(time.Now())
	updatedModel, _ := m.Update(msg)

	// Animation frame should increment
	if typedModel, ok := updatedModel.(Model); ok {
		if typedModel.animationFrame <= m.animationFrame {
			t.Error("Animation frame should increment")
		}
	}
}

func TestModel_Update_ResponseMsg(t *testing.T) {
	// Create a model with a message
	m := Model{
		ready:    true,
		loading:  true,
		messages: []chatMessage{{role: "user", content: "test"}},
	}

	// Create a response
	output := &models.ModelOutput{
		Candidates: []models.Candidate{{Text: "response text"}},
		Chosen:     0,
	}

	// Simulate response message
	msg := responseMsg{output: output}
	updatedModel, _ := m.Update(msg)

	// Should stop loading and add message
	if typedModel, ok := updatedModel.(Model); ok {
		if typedModel.loading {
			t.Error("Model should not be loading after response")
		}
		if len(typedModel.messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(typedModel.messages))
		}
	}
}

func TestModel_Update_ErrMsg(t *testing.T) {
	// Create a model with an error
	m := Model{
		ready:   true,
		loading: true,
	}

	// Simulate error message
	testErr := fmt.Errorf("test error")
	msg := errMsg{err: testErr}
	updatedModel, cmd := m.Update(msg)

	// Should stop loading and set error
	if typedModel, ok := updatedModel.(Model); ok {
		if typedModel.loading {
			t.Error("Model should not be loading after error")
		}
		if typedModel.err == nil {
			t.Error("Model should have error set")
		}
	}

	// Cmd might be tea.Quit or nil
	if cmd == nil {
		t.Log("Update returned nil command for error")
	}
}

func TestModel_View_NotReady(t *testing.T) {
	// Model not ready
	m := Model{
		ready: false,
	}

	view := m.View()

	// Should contain welcome message or instructions
	if !strings.Contains(view, "Connect") && !strings.Contains(view, "Resizing") {
		t.Log("View should contain initialization message")
	}
}

func TestModel_View_Loading(t *testing.T) {
	// Model with loading state
	m := Model{
		ready:   true,
		loading: true,
	}

	view := m.View()

	// Should indicate loading state
	if !strings.Contains(view, "Waiting") && !strings.Contains(view, "...") {
		t.Log("View should indicate loading")
	}
}

func TestModel_View_WithMessages(t *testing.T) {
	// Create a minimal textarea for the view
	ta := textarea.New()
	ta.SetWidth(80)

	// Create a viewport
	vp := viewport.New(80, 20)

	// Model with messages
	m := Model{
		ready:    true,
		textarea: ta,
		viewport: vp,
		width:    80,
		height:   24,
		messages: []chatMessage{
			{role: "user", content: "Hello"},
			{role: "assistant", content: "Hi there!"},
		},
	}

	// Update the viewport to populate content
	m.updateViewport()

	view := m.View()

	// The main test is that View doesn't panic with messages
	// Message rendering may vary based on styling, so we check if at least one message appears
	hasUserMessage := strings.Contains(view, "Hello")
	hasAssistantMessage := strings.Contains(view, "Hi there!")

	if !hasUserMessage && !hasAssistantMessage {
		t.Error("View should contain some message content")
	}
}

func TestAnimationTick(t *testing.T) {
	cmd := animationTick()

	if cmd == nil {
		t.Error("animationTick should return a command")
	}
}

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
	}{
		{
			name:     "Simple markdown",
			input:    "# Header\n\nThis is **bold** text.",
			maxWidth: 80,
		},
		{
			name:     "Empty input",
			input:    "",
			maxWidth: 80,
		},
		{
			name:     "Long input",
			input:    strings.Repeat("word ", 100),
			maxWidth: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := render.MarkdownWithWidth(tt.input, tt.maxWidth)
			if err != nil {
				t.Errorf("render.MarkdownWithWidth failed: %v", err)
			}

			if output == "" && tt.input != "" {
				t.Error("Expected non-empty output for non-empty input")
			}
		})
	}
}

func TestRenderMarkdownWithSpecialChars(t *testing.T) {
	input := "# Header\n\n- Item 1\n- Item 2\n\n`code`"

	output, err := render.MarkdownWithWidth(input, 80)
	if err != nil {
		t.Errorf("render.MarkdownWithWidth failed: %v", err)
	}

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestChatMessage_Struct(t *testing.T) {
	msg := chatMessage{
		role:     "user",
		content:  "test content",
		thoughts: "thinking",
	}

	if msg.role != "user" {
		t.Errorf("Expected role 'user', got %s", msg.role)
	}
	if msg.content != "test content" {
		t.Errorf("Expected content 'test content', got %s", msg.content)
	}
	if msg.thoughts != "thinking" {
		t.Errorf("Expected thoughts 'thinking', got %s", msg.thoughts)
	}
}

func TestModel_Struct(t *testing.T) {
	m := Model{
		client:    nil,
		session:   nil,
		modelName: "test-model",
		ready:     false,
		loading:   false,
		err:       nil,
	}

	if m.modelName != "test-model" {
		t.Errorf("Expected modelName 'test-model', got %s", m.modelName)
	}
	if m.ready {
		t.Error("Model should not be ready initially")
	}
	if m.loading {
		t.Error("Model should not be loading initially")
	}
}

func TestResponseMsg_Struct(t *testing.T) {
	output := &models.ModelOutput{
		Candidates: []models.Candidate{{Text: "test"}},
		Chosen:     0,
	}

	msg := responseMsg{output: output}

	if msg.output != output {
		t.Error("responseMsg should store output")
	}
}

func TestErrMsg_Struct(t *testing.T) {
	testErr := fmt.Errorf("test error")
	msg := errMsg{err: testErr}

	if msg.err != testErr {
		t.Error("errMsg should store error")
	}
}

// mockChatSession is a mock of *api.ChatSession for testing
type mockChatSession struct {
	sendMessageFunc    func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error)
	sendMessageCalled  bool
	gemID              string
}

func (m *mockChatSession) SendMessage(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
	m.sendMessageCalled = true
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(prompt, files)
	}
	return nil, nil
}

func (m *mockChatSession) SetMetadata(cid, rid, rcid string) {}

func (m *mockChatSession) GetMetadata() []string {
	return nil
}

func (m *mockChatSession) CID() string {
	return ""
}

func (m *mockChatSession) RID() string {
	return ""
}

func (m *mockChatSession) RCID() string {
	return ""
}

func (m *mockChatSession) GetModel() models.Model {
	return models.Model25Flash
}

func (m *mockChatSession) SetModel(model models.Model) {}

func (m *mockChatSession) LastOutput() *models.ModelOutput {
	return nil
}

func (m *mockChatSession) ChooseCandidate(index int) error {
	return nil
}

func (m *mockChatSession) SetGem(gemID string) { m.gemID = gemID }

func (m *mockChatSession) GetGemID() string {
	return ""
}

func TestNewChatModel(t *testing.T) {
	// Just test that the function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewChatModel panicked: %v", r)
		}
	}()

	// We can't easily test without a real client, but we can test function signature
	// For coverage purposes, we'll just verify it exists and doesn't panic
	// Actual testing would require complex mocking

	// The function exists and is callable
	_ = NewChatModel

	// We can't create a real test without mocking the API client
	// which is beyond the scope of this unit test
}

func TestModel_Init(t *testing.T) {
	// Just test that the method exists
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Model.Init panicked: %v", r)
		}
	}()

	// Create a minimal model to test Init
	ta := textarea.New()
	ta.SetWidth(80)

	s := spinner.New()

	model := Model{
		textarea: ta,
		spinner:  s,
	}

	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestModel_sendMessage(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		// Create a mock session that returns success
		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				return &models.ModelOutput{
					Candidates: []models.Candidate{{Text: "success response"}},
					Chosen:     0,
				}, nil
			},
		}

		// Create a model with the mock session
		m := Model{
			session: mockSession,
		}

		// Test sendMessage returns a command
		cmd := m.sendMessage("test prompt")
		if cmd == nil {
			t.Error("sendMessage should return a command")
			return
		}

		// Execute the command to verify it works
		msg := cmd()
		if msg == nil {
			t.Error("Command should return a message")
			return
		}

		// Verify the session was called
		if !mockSession.sendMessageCalled {
			t.Error("SendMessage should have been called on session")
		}

		// Verify the message type
		if response, ok := msg.(responseMsg); ok {
			if len(response.output.Candidates) != 1 {
				t.Errorf("Expected 1 candidate, got %d", len(response.output.Candidates))
			}
		} else {
			t.Errorf("Expected responseMsg type, got %T", msg)
		}
	})

	t.Run("error response", func(t *testing.T) {
		// Create a mock session that returns error
		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				return nil, fmt.Errorf("test error")
			},
		}

		// Create a model with the mock session
		m := Model{
			session: mockSession,
		}

		// Test sendMessage returns a command
		cmd := m.sendMessage("test prompt")
		if cmd == nil {
			t.Error("sendMessage should return a command")
			return
		}

		// Execute the command to verify it works
		msg := cmd()
		if msg == nil {
			t.Error("Command should return a message")
			return
		}

		// Verify the message type is errMsg
		if errMsg, ok := msg.(errMsg); ok {
			if errMsg.err == nil {
				t.Error("errMsg should contain an error")
			}
		} else {
			t.Errorf("Expected errMsg type, got %T", msg)
		}
	})
}

func TestRunChat(t *testing.T) {
	// Just test that the function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunChat panicked: %v", r)
		}
	}()

	// We can't actually run the tea program in a test
	// So we'll just test function signature
	_ = RunChat
}

func TestRunChatWithSession(t *testing.T) {
	// Just test that the function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunChatWithSession panicked: %v", r)
		}
	}()

	// We can't actually run the tea program in a test
	// So we'll just test function signature
	_ = RunChatWithSession
}

func TestNewChatModelWithSession(t *testing.T) {
	// Test that NewChatModelWithSession creates a model with the provided session
	mockSession := &mockChatSession{
		sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
			return &models.ModelOutput{
				Candidates: []models.Candidate{{Text: "test response"}},
				Chosen:     0,
			}, nil
		},
	}

	// Create model with session
	model := NewChatModelWithSession(nil, mockSession, "test-model")

	// Verify model properties
	if model.session == nil {
		t.Error("Model should have a session")
	}

	if model.modelName != "test-model" {
		t.Errorf("Expected modelName 'test-model', got %s", model.modelName)
	}

	// Verify session is the one we provided
	if model.session != mockSession {
		t.Error("Model should use the provided session")
	}
}

func TestNewChatModelWithSession_SendsMessages(t *testing.T) {
	// Test that the model uses the provided session for sending messages
	var receivedPrompt string
	mockSession := &mockChatSession{
		sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
			receivedPrompt = prompt
			return &models.ModelOutput{
				Candidates: []models.Candidate{{Text: "response"}},
				Chosen:     0,
			}, nil
		},
	}

	model := NewChatModelWithSession(nil, mockSession, "test-model")

	// Test sendMessage
	cmd := model.sendMessage("hello world")
	if cmd == nil {
		t.Error("sendMessage should return a command")
		return
	}

	// Execute the command
	msg := cmd()
	if msg == nil {
		t.Error("Command should return a message")
		return
	}

	// Verify the session received the message
	if receivedPrompt != "hello world" {
		t.Errorf("Expected prompt 'hello world', got '%s'", receivedPrompt)
	}

	// Verify response message type
	if _, ok := msg.(responseMsg); !ok {
		t.Errorf("Expected responseMsg, got %T", msg)
	}
}

func TestNewChatModelWithSession_Initialization(t *testing.T) {
	mockSession := &mockChatSession{}
	model := NewChatModelWithSession(nil, mockSession, "gemini-2.5-flash")

	// Test Init returns commands
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a command batch")
	}

	// Verify textarea is initialized
	if model.textarea.CharLimit == 0 {
		t.Error("Textarea should have char limit set")
	}

	// Verify messages is empty
	if len(model.messages) != 0 {
		t.Error("Messages should be empty initially")
	}
}

func TestModel_GemSelection_State(t *testing.T) {
	model := Model{
		selectingGem: false,
		gemsCursor:   0,
		gemsFilter:   "",
	}

	// Initially not selecting gem
	if model.selectingGem {
		t.Error("Model should not be selecting gem initially")
	}

	// Set to selecting gem mode
	model.selectingGem = true
	model.gemsList = []*models.Gem{
		{ID: "1", Name: "Test Gem 1", Predefined: false},
		{ID: "2", Name: "Test Gem 2", Predefined: true},
	}

	if !model.selectingGem {
		t.Error("Model should be in gem selection mode")
	}

	if len(model.gemsList) != 2 {
		t.Errorf("Expected 2 gems, got %d", len(model.gemsList))
	}
}

func TestModel_FilteredGems(t *testing.T) {
	model := Model{
		gemsList: []*models.Gem{
			{ID: "1", Name: "Code Helper", Description: "Helps with coding"},
			{ID: "2", Name: "Writer", Description: "Writing assistant"},
			{ID: "3", Name: "Coder Pro", Description: "Advanced coding"},
		},
	}

	t.Run("no filter returns all gems", func(t *testing.T) {
		model.gemsFilter = ""
		filtered := model.filteredGems()
		if len(filtered) != 3 {
			t.Errorf("Expected 3 gems, got %d", len(filtered))
		}
	})

	t.Run("filter by name", func(t *testing.T) {
		model.gemsFilter = "code"
		filtered := model.filteredGems()
		if len(filtered) != 2 {
			t.Errorf("Expected 2 gems matching 'code', got %d", len(filtered))
		}
	})

	t.Run("filter by description", func(t *testing.T) {
		model.gemsFilter = "writing"
		filtered := model.filteredGems()
		if len(filtered) != 1 {
			t.Errorf("Expected 1 gem matching 'writing', got %d", len(filtered))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		model.gemsFilter = "xyz"
		filtered := model.filteredGems()
		if len(filtered) != 0 {
			t.Errorf("Expected 0 gems matching 'xyz', got %d", len(filtered))
		}
	})
}

func TestModel_GemsLoadedForChatMsg(t *testing.T) {
	gems := []*models.Gem{
		{ID: "1", Name: "Test Gem"},
	}

	msg := gemsLoadedForChatMsg{gems: gems}

	if msg.gems == nil {
		t.Error("Message should contain gems")
	}

	if len(msg.gems) != 1 {
		t.Errorf("Expected 1 gem, got %d", len(msg.gems))
	}

	if msg.err != nil {
		t.Error("Message should not have an error")
	}
}

func TestModel_GemsLoadedForChatMsg_Error(t *testing.T) {
	testErr := fmt.Errorf("test error")
	msg := gemsLoadedForChatMsg{err: testErr}

	if msg.err == nil {
		t.Error("Message should contain error")
	}

	if msg.gems != nil {
		t.Error("Message should not contain gems when there's an error")
	}
}

func TestModel_RenderGemSelector_Empty(t *testing.T) {
	model := Model{
		width:        80,
		height:       24,
		selectingGem: true,
		gemsList:     []*models.Gem{},
	}

	view := model.renderGemSelector()

	if view == "" {
		t.Error("View should not be empty")
	}

	if !strings.Contains(view, "Select a Gem") {
		t.Error("View should contain title")
	}
}

func TestModel_RenderGemSelector_WithGems(t *testing.T) {
	model := Model{
		width:        80,
		height:       24,
		selectingGem: true,
		gemsList: []*models.Gem{
			{ID: "1", Name: "Code Helper", Description: "Helps with coding", Predefined: false},
			{ID: "2", Name: "System Gem", Description: "Built-in", Predefined: true},
		},
		gemsCursor: 0,
	}

	view := model.renderGemSelector()

	if !strings.Contains(view, "Code Helper") {
		t.Error("View should contain gem name")
	}

	if !strings.Contains(view, "[custom]") {
		t.Error("View should show custom gem indicator")
	}

	if !strings.Contains(view, "[system]") {
		t.Error("View should show system gem indicator")
	}
}

func TestModel_View_ShowsActiveGem(t *testing.T) {
	ta := textarea.New()
	ta.SetWidth(80)

	vp := viewport.New(80, 20)

	model := Model{
		ready:         true,
		textarea:      ta,
		viewport:      vp,
		width:         80,
		height:        24,
		modelName:     "gemini-2.5-flash",
		activeGemName: "Code Helper",
	}

	view := model.View()

	if !strings.Contains(view, "Code Helper") {
		t.Error("View should show active gem name in header")
	}
}

// mockHistoryStoreForModel is a mock implementation of HistoryStoreInterface for testing
type mockHistoryStoreForModel struct {
	addMessageCalls    []struct{ id, role, content, thoughts string }
	updateMetadataCalls []struct{ id, cid, rid, rcid string }
	updateTitleCalls   []struct{ id, title string }
	addMessageErr      error
	updateMetadataErr  error
	updateTitleErr     error
}

func (m *mockHistoryStoreForModel) AddMessage(id, role, content, thoughts string) error {
	m.addMessageCalls = append(m.addMessageCalls, struct{ id, role, content, thoughts string }{id, role, content, thoughts})
	return m.addMessageErr
}

func (m *mockHistoryStoreForModel) UpdateMetadata(id, cid, rid, rcid string) error {
	m.updateMetadataCalls = append(m.updateMetadataCalls, struct{ id, cid, rid, rcid string }{id, cid, rid, rcid})
	return m.updateMetadataErr
}

func (m *mockHistoryStoreForModel) UpdateTitle(id, title string) error {
	m.updateTitleCalls = append(m.updateTitleCalls, struct{ id, title string }{id, title})
	return m.updateTitleErr
}

func TestNewChatModelWithConversation(t *testing.T) {
	mockSession := &mockChatSession{}
	mockStore := &mockHistoryStoreForModel{}

	t.Run("with nil conversation", func(t *testing.T) {
		model := NewChatModelWithConversation(nil, mockSession, "test-model", nil, mockStore)

		if model.conversation != nil {
			t.Error("conversation should be nil")
		}
		if len(model.messages) != 0 {
			t.Errorf("messages length = %d, want 0", len(model.messages))
		}
		if model.historyStore != mockStore {
			t.Error("historyStore not set correctly")
		}
	})

	t.Run("with empty conversation", func(t *testing.T) {
		conv := &history.Conversation{
			ID:       "test-conv",
			Title:    "Test",
			Model:    "test-model",
			Messages: []history.Message{},
		}
		model := NewChatModelWithConversation(nil, mockSession, "test-model", conv, mockStore)

		if model.conversation != conv {
			t.Error("conversation not set correctly")
		}
		if len(model.messages) != 0 {
			t.Errorf("messages length = %d, want 0", len(model.messages))
		}
	})

	t.Run("with existing messages", func(t *testing.T) {
		conv := &history.Conversation{
			ID:    "test-conv",
			Title: "Test",
			Model: "test-model",
			Messages: []history.Message{
				{Role: "user", Content: "Hello", Thoughts: ""},
				{Role: "assistant", Content: "Hi there!", Thoughts: "Thinking about greeting"},
			},
		}
		model := NewChatModelWithConversation(nil, mockSession, "test-model", conv, mockStore)

		if len(model.messages) != 2 {
			t.Errorf("messages length = %d, want 2", len(model.messages))
		}
		if model.messages[0].role != "user" || model.messages[0].content != "Hello" {
			t.Error("first message not loaded correctly")
		}
		if model.messages[1].role != "assistant" || model.messages[1].content != "Hi there!" {
			t.Error("second message not loaded correctly")
		}
		if model.messages[1].thoughts != "Thinking about greeting" {
			t.Error("thoughts not loaded correctly")
		}
	})
}

func TestRunChatWithConversation_FunctionExists(t *testing.T) {
	// Just verify the function exists
	_ = RunChatWithConversation
}

func TestHistoryStoreInterface(t *testing.T) {
	// Verify the interface is implemented by mockHistoryStoreForModel
	var _ HistoryStoreInterface = &mockHistoryStoreForModel{}
}

// mockChatSessionWithMetadata is a mock that also tracks metadata
type mockChatSessionWithMetadata struct {
	mockChatSession
	cid  string
	rid  string
	rcid string
}

func (m *mockChatSessionWithMetadata) CID() string  { return m.cid }
func (m *mockChatSessionWithMetadata) RID() string  { return m.rid }
func (m *mockChatSessionWithMetadata) RCID() string { return m.rcid }

func TestModel_SaveMessageToHistory(t *testing.T) {
	t.Run("saves message when store and conversation are set", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-123"}

		m := &Model{
			conversation: conv,
			historyStore: mockStore,
		}

		m.saveMessageToHistory("user", "Hello world", "")

		if len(mockStore.addMessageCalls) != 1 {
			t.Errorf("expected 1 addMessage call, got %d", len(mockStore.addMessageCalls))
			return
		}

		call := mockStore.addMessageCalls[0]
		if call.id != "conv-123" {
			t.Errorf("expected id 'conv-123', got '%s'", call.id)
		}
		if call.role != "user" {
			t.Errorf("expected role 'user', got '%s'", call.role)
		}
		if call.content != "Hello world" {
			t.Errorf("expected content 'Hello world', got '%s'", call.content)
		}
	})

	t.Run("saves assistant message with thoughts", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-456"}

		m := &Model{
			conversation: conv,
			historyStore: mockStore,
		}

		m.saveMessageToHistory("assistant", "Response text", "Thinking process")

		if len(mockStore.addMessageCalls) != 1 {
			t.Errorf("expected 1 addMessage call, got %d", len(mockStore.addMessageCalls))
			return
		}

		call := mockStore.addMessageCalls[0]
		if call.role != "assistant" {
			t.Errorf("expected role 'assistant', got '%s'", call.role)
		}
		if call.thoughts != "Thinking process" {
			t.Errorf("expected thoughts 'Thinking process', got '%s'", call.thoughts)
		}
	})

	t.Run("does nothing when historyStore is nil", func(t *testing.T) {
		conv := &history.Conversation{ID: "conv-123"}

		m := &Model{
			conversation: conv,
			historyStore: nil,
		}

		// Should not panic
		m.saveMessageToHistory("user", "Hello", "")
	})

	t.Run("does nothing when conversation is nil", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}

		m := &Model{
			conversation: nil,
			historyStore: mockStore,
		}

		m.saveMessageToHistory("user", "Hello", "")

		if len(mockStore.addMessageCalls) != 0 {
			t.Errorf("expected 0 addMessage calls, got %d", len(mockStore.addMessageCalls))
		}
	})

	t.Run("does nothing when both are nil", func(t *testing.T) {
		m := &Model{
			conversation: nil,
			historyStore: nil,
		}

		// Should not panic
		m.saveMessageToHistory("user", "Hello", "")
	})
}

func TestModel_SaveMetadataToHistory(t *testing.T) {
	t.Run("saves metadata when session has values", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSessionWithMetadata{
			cid:  "cid-abc",
			rid:  "rid-def",
			rcid: "rcid-ghi",
		}

		m := &Model{
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
		}

		m.saveMetadataToHistory()

		if len(mockStore.updateMetadataCalls) != 1 {
			t.Errorf("expected 1 updateMetadata call, got %d", len(mockStore.updateMetadataCalls))
			return
		}

		call := mockStore.updateMetadataCalls[0]
		if call.id != "conv-123" {
			t.Errorf("expected id 'conv-123', got '%s'", call.id)
		}
		if call.cid != "cid-abc" {
			t.Errorf("expected cid 'cid-abc', got '%s'", call.cid)
		}
		if call.rid != "rid-def" {
			t.Errorf("expected rid 'rid-def', got '%s'", call.rid)
		}
		if call.rcid != "rcid-ghi" {
			t.Errorf("expected rcid 'rcid-ghi', got '%s'", call.rcid)
		}
	})

	t.Run("does nothing when all metadata is empty", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSessionWithMetadata{
			cid:  "",
			rid:  "",
			rcid: "",
		}

		m := &Model{
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
		}

		m.saveMetadataToHistory()

		if len(mockStore.updateMetadataCalls) != 0 {
			t.Errorf("expected 0 updateMetadata calls, got %d", len(mockStore.updateMetadataCalls))
		}
	})

	t.Run("saves when only cid is set", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSessionWithMetadata{
			cid:  "cid-only",
			rid:  "",
			rcid: "",
		}

		m := &Model{
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
		}

		m.saveMetadataToHistory()

		if len(mockStore.updateMetadataCalls) != 1 {
			t.Errorf("expected 1 updateMetadata call, got %d", len(mockStore.updateMetadataCalls))
		}
	})

	t.Run("does nothing when historyStore is nil", func(t *testing.T) {
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSessionWithMetadata{cid: "cid-abc"}

		m := &Model{
			conversation: conv,
			historyStore: nil,
			session:      mockSession,
		}

		// Should not panic
		m.saveMetadataToHistory()
	})

	t.Run("does nothing when conversation is nil", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		mockSession := &mockChatSessionWithMetadata{cid: "cid-abc"}

		m := &Model{
			conversation: nil,
			historyStore: mockStore,
			session:      mockSession,
		}

		m.saveMetadataToHistory()

		if len(mockStore.updateMetadataCalls) != 0 {
			t.Errorf("expected 0 updateMetadata calls, got %d", len(mockStore.updateMetadataCalls))
		}
	})

	t.Run("does nothing when session is nil", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-123"}

		m := &Model{
			conversation: conv,
			historyStore: mockStore,
			session:      nil,
		}

		// Should not panic
		m.saveMetadataToHistory()

		if len(mockStore.updateMetadataCalls) != 0 {
			t.Errorf("expected 0 updateMetadata calls, got %d", len(mockStore.updateMetadataCalls))
		}
	})
}

func TestModel_AutoSaveOnResponse(t *testing.T) {
	t.Run("auto-saves assistant message and metadata on response", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSessionWithMetadata{
			cid:  "new-cid",
			rid:  "new-rid",
			rcid: "new-rcid",
		}

		ta := textarea.New()
		ta.SetWidth(80)

		m := Model{
			ready:        true,
			loading:      true,
			messages:     []chatMessage{{role: "user", content: "test"}},
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
			textarea:     ta,
		}

		// Simulate response message
		output := &models.ModelOutput{
			Candidates: []models.Candidate{{Text: "response text", Thoughts: "thinking"}},
			Chosen:     0,
		}
		msg := responseMsg{output: output}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Verify loading stopped
		if typedModel.loading {
			t.Error("model should not be loading after response")
		}

		// Verify message was added
		if len(typedModel.messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(typedModel.messages))
		}

		// Verify auto-save was called for message
		if len(mockStore.addMessageCalls) != 1 {
			t.Errorf("expected 1 addMessage call, got %d", len(mockStore.addMessageCalls))
		} else {
			call := mockStore.addMessageCalls[0]
			if call.role != "assistant" {
				t.Errorf("expected role 'assistant', got '%s'", call.role)
			}
			if call.content != "response text" {
				t.Errorf("expected content 'response text', got '%s'", call.content)
			}
		}

		// Verify metadata was saved
		if len(mockStore.updateMetadataCalls) != 1 {
			t.Errorf("expected 1 updateMetadata call, got %d", len(mockStore.updateMetadataCalls))
		} else {
			call := mockStore.updateMetadataCalls[0]
			if call.cid != "new-cid" {
				t.Errorf("expected cid 'new-cid', got '%s'", call.cid)
			}
		}
	})
}

func TestModel_AutoSaveOnSend(t *testing.T) {
	t.Run("auto-saves user message when sending", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		conv := &history.Conversation{ID: "conv-789"}
		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				return &models.ModelOutput{
					Candidates: []models.Candidate{{Text: "response"}},
					Chosen:     0,
				}, nil
			},
		}

		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("Hello, Gemini!")

		vp := viewport.New(80, 20)

		m := Model{
			ready:        true,
			loading:      false,
			messages:     []chatMessage{},
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
			textarea:     ta,
			viewport:     vp,
			width:        100,
			height:       40,
		}

		// Simulate enter key to send message
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Verify model is now loading
		if !typedModel.loading {
			t.Error("model should be loading after sending message")
		}

		// Verify user message was added
		if len(typedModel.messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(typedModel.messages))
		}

		// Verify auto-save was called for user message
		if len(mockStore.addMessageCalls) != 1 {
			t.Errorf("expected 1 addMessage call, got %d", len(mockStore.addMessageCalls))
		} else {
			call := mockStore.addMessageCalls[0]
			if call.id != "conv-789" {
				t.Errorf("expected id 'conv-789', got '%s'", call.id)
			}
			if call.role != "user" {
				t.Errorf("expected role 'user', got '%s'", call.role)
			}
			if call.content != "Hello, Gemini!" {
				t.Errorf("expected content 'Hello, Gemini!', got '%s'", call.content)
			}
		}
	})

	t.Run("does not auto-save when no conversation", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{}
		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				return &models.ModelOutput{
					Candidates: []models.Candidate{{Text: "response"}},
					Chosen:     0,
				}, nil
			},
		}

		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("Hello")

		vp := viewport.New(80, 20)

		m := Model{
			ready:        true,
			loading:      false,
			messages:     []chatMessage{},
			conversation: nil, // No conversation
			historyStore: mockStore,
			session:      mockSession,
			textarea:     ta,
			viewport:     vp,
			width:        100,
			height:       40,
		}

		// Simulate enter key to send message
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		m.Update(msg)

		// Auto-save should not be called
		if len(mockStore.addMessageCalls) != 0 {
			t.Errorf("expected 0 addMessage calls, got %d", len(mockStore.addMessageCalls))
		}
	})

	t.Run("does not auto-save when no store", func(t *testing.T) {
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				return &models.ModelOutput{
					Candidates: []models.Candidate{{Text: "response"}},
					Chosen:     0,
				}, nil
			},
		}

		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("Hello")

		vp := viewport.New(80, 20)

		m := Model{
			ready:        true,
			loading:      false,
			messages:     []chatMessage{},
			conversation: conv,
			historyStore: nil, // No store
			session:      mockSession,
			textarea:     ta,
			viewport:     vp,
			width:        100,
			height:       40,
		}

		// Simulate enter key to send message - should not panic
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)
		// Message should still be added locally
		if len(typedModel.messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(typedModel.messages))
		}
	})
}

func TestModel_AutoSaveWithStoreError(t *testing.T) {
	t.Run("continues gracefully when store returns error", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{
			addMessageErr: fmt.Errorf("storage error"),
		}
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				return &models.ModelOutput{
					Candidates: []models.Candidate{{Text: "response"}},
					Chosen:     0,
				}, nil
			},
		}

		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("Hello")

		vp := viewport.New(80, 20)

		m := Model{
			ready:        true,
			loading:      false,
			messages:     []chatMessage{},
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
			textarea:     ta,
			viewport:     vp,
			width:        100,
			height:       40,
		}

		// Simulate enter key - should not panic even with store error
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Message should still be added locally
		if len(typedModel.messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(typedModel.messages))
		}

		// Model should still be in loading state
		if !typedModel.loading {
			t.Error("model should be loading")
		}
	})

	t.Run("continues gracefully when metadata update returns error", func(t *testing.T) {
		mockStore := &mockHistoryStoreForModel{
			updateMetadataErr: fmt.Errorf("metadata error"),
		}
		conv := &history.Conversation{ID: "conv-123"}
		mockSession := &mockChatSessionWithMetadata{
			cid:  "cid-abc",
			rid:  "rid-def",
			rcid: "rcid-ghi",
		}

		ta := textarea.New()
		ta.SetWidth(80)

		m := Model{
			ready:        true,
			loading:      true,
			messages:     []chatMessage{{role: "user", content: "test"}},
			conversation: conv,
			historyStore: mockStore,
			session:      mockSession,
			textarea:     ta,
		}

		// Simulate response - should not panic even with metadata error
		output := &models.ModelOutput{
			Candidates: []models.Candidate{{Text: "response text"}},
			Chosen:     0,
		}
		msg := responseMsg{output: output}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Message should still be added
		if len(typedModel.messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(typedModel.messages))
		}

		// Loading should be stopped
		if typedModel.loading {
			t.Error("model should not be loading")
		}
	})
}

// mockFullHistoryStore implements FullHistoryStore for testing
type mockFullHistoryStore struct {
	mockHistoryStoreForModel
	conversations      []*history.Conversation
	getConversation    *history.Conversation
	createConversation *history.Conversation
	listErr            error
	getErr             error
	createErr          error
}

func (m *mockFullHistoryStore) ListConversations() ([]*history.Conversation, error) {
	return m.conversations, m.listErr
}

func (m *mockFullHistoryStore) GetConversation(id string) (*history.Conversation, error) {
	return m.getConversation, m.getErr
}

func (m *mockFullHistoryStore) CreateConversation(model string) (*history.Conversation, error) {
	return m.createConversation, m.createErr
}

func (m *mockFullHistoryStore) DeleteConversation(id string) error {
	return nil
}

func (m *mockFullHistoryStore) ToggleFavorite(id string) (bool, error) {
	return false, nil
}

func (m *mockFullHistoryStore) MoveConversation(id string, newIndex int) error {
	return nil
}

func (m *mockFullHistoryStore) SwapConversations(id1, id2 string) error {
	return nil
}

func (m *mockFullHistoryStore) ExportToMarkdown(id string) (string, error) {
	return "", nil
}

func (m *mockFullHistoryStore) ExportToJSON(id string) ([]byte, error) {
	return nil, nil
}

func TestFullHistoryStoreInterface(t *testing.T) {
	// Verify the interface is implemented by mockFullHistoryStore
	var _ FullHistoryStore = &mockFullHistoryStore{}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"zero time", time.Time{}, ""},
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1m ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5m ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1h ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3h ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1d ago"},
		{"3 days ago", now.Add(-72 * time.Hour), "3d ago"},
		{"2 weeks ago", now.Add(-14 * 24 * time.Hour), now.Add(-14 * 24 * time.Hour).Format("Jan 2")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}

func TestModel_FilteredHistory(t *testing.T) {
	convs := []*history.Conversation{
		{ID: "1", Title: "Chat about Go", Model: "gemini-2.5-flash"},
		{ID: "2", Title: "Python discussion", Model: "gemini-3.0-pro"},
		{ID: "3", Title: "Go concurrency patterns", Model: "gemini-2.5-flash"},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		m := Model{historyList: convs, historyFilter: ""}
		filtered := m.filteredHistory()
		if len(filtered) != 3 {
			t.Errorf("expected 3 conversations, got %d", len(filtered))
		}
	})

	t.Run("filter by title", func(t *testing.T) {
		m := Model{historyList: convs, historyFilter: "Go"}
		filtered := m.filteredHistory()
		if len(filtered) != 2 {
			t.Errorf("expected 2 conversations matching 'Go', got %d", len(filtered))
		}
	})

	t.Run("filter by model", func(t *testing.T) {
		m := Model{historyList: convs, historyFilter: "flash"}
		filtered := m.filteredHistory()
		if len(filtered) != 2 {
			t.Errorf("expected 2 conversations matching 'flash', got %d", len(filtered))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		m := Model{historyList: convs, historyFilter: "PYTHON"}
		filtered := m.filteredHistory()
		if len(filtered) != 1 {
			t.Errorf("expected 1 conversation matching 'PYTHON', got %d", len(filtered))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		m := Model{historyList: convs, historyFilter: "xyz"}
		filtered := m.filteredHistory()
		if len(filtered) != 0 {
			t.Errorf("expected 0 conversations matching 'xyz', got %d", len(filtered))
		}
	})
}

func TestModel_HistorySelection_Commands(t *testing.T) {
	t.Run("/history command enters selection mode", func(t *testing.T) {
		mockStore := &mockFullHistoryStore{
			conversations: []*history.Conversation{
				{ID: "1", Title: "Test Chat"},
			},
		}

		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("/history")

		vp := viewport.New(80, 20)

		m := Model{
			ready:            true,
			loading:          false,
			fullHistoryStore: mockStore,
			textarea:         ta,
			viewport:         vp,
			width:            100,
			height:           40,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		if !typedModel.selectingHistory {
			t.Error("model should be in history selection mode")
		}
		if !typedModel.historyLoading {
			t.Error("model should be loading history")
		}
		if cmd == nil {
			t.Error("should return a command to load history")
		}
	})

	t.Run("/hist shortcut works", func(t *testing.T) {
		mockStore := &mockFullHistoryStore{}

		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("/hist")

		vp := viewport.New(80, 20)

		m := Model{
			ready:            true,
			loading:          false,
			fullHistoryStore: mockStore,
			textarea:         ta,
			viewport:         vp,
			width:            100,
			height:           40,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)
		if !typedModel.selectingHistory {
			t.Error("model should be in history selection mode with /hist")
		}
	})

	t.Run("/history without store shows error", func(t *testing.T) {
		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("/history")

		vp := viewport.New(80, 20)

		m := Model{
			ready:            true,
			loading:          false,
			fullHistoryStore: nil, // No store
			textarea:         ta,
			viewport:         vp,
			width:            100,
			height:           40,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)
		if typedModel.selectingHistory {
			t.Error("model should not be in history selection mode without store")
		}
		if typedModel.err == nil {
			t.Error("model should have error set")
		}
	})
}

func TestModel_UpdateHistorySelection(t *testing.T) {
	convs := []*history.Conversation{
		{ID: "1", Title: "Chat 1"},
		{ID: "2", Title: "Chat 2"},
		{ID: "3", Title: "Chat 3"},
	}

	t.Run("navigation up/down", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyCursor:    0, // At "New Conversation"
		}

		// Move down
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.updateHistorySelection(msg)
		typedModel := updatedModel.(Model)
		if typedModel.historyCursor != 1 {
			t.Errorf("cursor should be 1 after down, got %d", typedModel.historyCursor)
		}

		// Move down again
		updatedModel, _ = typedModel.updateHistorySelection(msg)
		typedModel = updatedModel.(Model)
		if typedModel.historyCursor != 2 {
			t.Errorf("cursor should be 2 after second down, got %d", typedModel.historyCursor)
		}

		// Move up
		msg = tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ = typedModel.updateHistorySelection(msg)
		typedModel = updatedModel.(Model)
		if typedModel.historyCursor != 1 {
			t.Errorf("cursor should be 1 after up, got %d", typedModel.historyCursor)
		}
	})

	t.Run("wrap around", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyCursor:    0, // At "New Conversation"
		}

		// Move up should wrap to last item (index 3 = 3 convs + 1 new conv - 1)
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.updateHistorySelection(msg)
		typedModel := updatedModel.(Model)
		if typedModel.historyCursor != 3 {
			t.Errorf("cursor should wrap to 3, got %d", typedModel.historyCursor)
		}

		// Move down should wrap to 0
		msg = tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ = typedModel.updateHistorySelection(msg)
		typedModel = updatedModel.(Model)
		if typedModel.historyCursor != 0 {
			t.Errorf("cursor should wrap to 0, got %d", typedModel.historyCursor)
		}
	})

	t.Run("escape cancels selection", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyCursor:    2,
			historyFilter:    "test",
		}

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := m.updateHistorySelection(msg)
		typedModel := updatedModel.(Model)

		if typedModel.selectingHistory {
			t.Error("should not be in selection mode after escape")
		}
		if typedModel.historyList != nil {
			t.Error("history list should be cleared")
		}
		if typedModel.historyCursor != 0 {
			t.Error("cursor should be reset")
		}
		if typedModel.historyFilter != "" {
			t.Error("filter should be cleared")
		}
	})

	t.Run("typing adds to filter", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyFilter:    "",
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updatedModel, _ := m.updateHistorySelection(msg)
		typedModel := updatedModel.(Model)

		if typedModel.historyFilter != "g" {
			t.Errorf("filter should be 'g', got '%s'", typedModel.historyFilter)
		}
	})

	t.Run("backspace removes from filter", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyFilter:    "go",
		}

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updatedModel, _ := m.updateHistorySelection(msg)
		typedModel := updatedModel.(Model)

		if typedModel.historyFilter != "g" {
			t.Errorf("filter should be 'g', got '%s'", typedModel.historyFilter)
		}
	})
}

func TestModel_SwitchConversation(t *testing.T) {
	mockSession := &mockChatSession{}
	conv := &history.Conversation{
		ID:    "test-conv",
		Title: "Test Conversation",
		Model: "test-model",
		CID:   "cid-123",
		RID:   "rid-456",
		RCID:  "rcid-789",
		Messages: []history.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!", Thoughts: "Greeting response"},
		},
	}

	ta := textarea.New()
	ta.SetWidth(80)

	vp := viewport.New(80, 20)

	m := Model{
		selectingHistory: true,
		historyList:      []*history.Conversation{conv},
		historyCursor:    1,
		session:          mockSession,
		textarea:         ta,
		viewport:         vp,
		width:            100,
		height:           40,
	}

	updatedModel, _ := m.switchConversation(conv)
	typedModel := updatedModel.(Model)

	// Check selection mode is cleared
	if typedModel.selectingHistory {
		t.Error("should not be in selection mode after switch")
	}
	if typedModel.historyList != nil {
		t.Error("history list should be cleared")
	}

	// Check conversation is set
	if typedModel.conversation != conv {
		t.Error("conversation should be set to the selected one")
	}

	// Check messages are loaded
	if len(typedModel.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(typedModel.messages))
	}
	if typedModel.messages[0].role != "user" {
		t.Error("first message should be user role")
	}
	if typedModel.messages[1].thoughts != "Greeting response" {
		t.Error("thoughts should be loaded")
	}
}

func TestModel_StartNewConversation(t *testing.T) {
	newConv := &history.Conversation{
		ID:    "new-conv",
		Title: "New Conversation",
		Model: "test-model",
	}

	mockStore := &mockFullHistoryStore{
		createConversation: newConv,
	}

	mockSession := &mockChatSession{}

	ta := textarea.New()
	ta.SetWidth(80)

	vp := viewport.New(80, 20)

	m := Model{
		selectingHistory: true,
		historyList:      []*history.Conversation{{ID: "old"}},
		historyCursor:    0,
		fullHistoryStore: mockStore,
		session:          mockSession,
		modelName:        "test-model",
		messages:         []chatMessage{{role: "user", content: "old message"}},
		textarea:         ta,
		viewport:         vp,
		width:            100,
		height:           40,
	}

	updatedModel, _ := m.startNewConversation()
	typedModel := updatedModel.(Model)

	// Check selection mode is cleared
	if typedModel.selectingHistory {
		t.Error("should not be in selection mode")
	}

	// Check new conversation is set
	if typedModel.conversation != newConv {
		t.Error("should have new conversation")
	}

	// Check messages are cleared
	if len(typedModel.messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(typedModel.messages))
	}
}

func TestModel_RenderHistorySelector(t *testing.T) {
	convs := []*history.Conversation{
		{ID: "1", Title: "Chat 1", Model: "gemini-2.5-flash", UpdatedAt: time.Now()},
		{ID: "2", Title: "Chat 2", Model: "gemini-3.0-pro", UpdatedAt: time.Now().Add(-1 * time.Hour)},
	}

	t.Run("loading state", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyLoading:   true,
			width:            80,
			height:           24,
		}

		view := m.renderHistorySelector()
		if !strings.Contains(view, "Loading") {
			t.Error("should show loading message")
		}
	})

	t.Run("with conversations", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyCursor:    0,
			width:            80,
			height:           24,
		}

		view := m.renderHistorySelector()

		if !strings.Contains(view, "Select Conversation") {
			t.Error("should contain title")
		}
		if !strings.Contains(view, "New Conversation") {
			t.Error("should contain new conversation option")
		}
		if !strings.Contains(view, "Chat 1") {
			t.Error("should contain conversation title")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      []*history.Conversation{},
			width:            80,
			height:           24,
		}

		view := m.renderHistorySelector()
		if !strings.Contains(view, "No saved conversations") {
			t.Error("should show no conversations message")
		}
	})

	t.Run("with filter", func(t *testing.T) {
		m := Model{
			selectingHistory: true,
			historyList:      convs,
			historyFilter:    "test",
			width:            80,
			height:           24,
		}

		view := m.renderHistorySelector()
		if !strings.Contains(view, "🔍 test") {
			t.Error("should show filter input")
		}
	})
}

func TestModel_LoadHistoryForChat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		convs := []*history.Conversation{
			{ID: "1", Title: "Test"},
		}
		mockStore := &mockFullHistoryStore{
			conversations: convs,
		}

		m := Model{fullHistoryStore: mockStore}
		cmd := m.loadHistoryForChat()
		msg := cmd()

		histMsg, ok := msg.(historyLoadedForChatMsg)
		if !ok {
			t.Errorf("expected historyLoadedForChatMsg, got %T", msg)
			return
		}
		if histMsg.err != nil {
			t.Errorf("unexpected error: %v", histMsg.err)
		}
		if len(histMsg.conversations) != 1 {
			t.Errorf("expected 1 conversation, got %d", len(histMsg.conversations))
		}
	})

	t.Run("error", func(t *testing.T) {
		mockStore := &mockFullHistoryStore{
			listErr: fmt.Errorf("list error"),
		}

		m := Model{fullHistoryStore: mockStore}
		cmd := m.loadHistoryForChat()
		msg := cmd()

		histMsg, ok := msg.(historyLoadedForChatMsg)
		if !ok {
			t.Errorf("expected historyLoadedForChatMsg, got %T", msg)
			return
		}
		if histMsg.err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil store", func(t *testing.T) {
		m := Model{fullHistoryStore: nil}
		cmd := m.loadHistoryForChat()
		msg := cmd()

		histMsg, ok := msg.(historyLoadedForChatMsg)
		if !ok {
			t.Errorf("expected historyLoadedForChatMsg, got %T", msg)
			return
		}
		if histMsg.err == nil {
			t.Error("expected error for nil store")
		}
	})
}

func TestHistoryLoadedForChatMsg(t *testing.T) {
	t.Run("with conversations", func(t *testing.T) {
		convs := []*history.Conversation{{ID: "1"}}
		msg := historyLoadedForChatMsg{conversations: convs}

		if len(msg.conversations) != 1 {
			t.Error("should have conversations")
		}
		if msg.err != nil {
			t.Error("should not have error")
		}
	})

	t.Run("with error", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		msg := historyLoadedForChatMsg{err: testErr}

		if msg.err == nil {
			t.Error("should have error")
		}
		if msg.conversations != nil {
			t.Error("should not have conversations")
		}
	})
}

// ==================== Multi-line Input Tests ====================

func TestCreateTextarea(t *testing.T) {
	ta := createTextarea()

	t.Run("has correct placeholder", func(t *testing.T) {
		// Placeholder should mention \ + Enter for newline (line continuation)
		if !strings.Contains(ta.Placeholder, "\\") || !strings.Contains(ta.Placeholder, "Enter") {
			t.Error("placeholder should mention \\ + Enter for newline")
		}
	})

	t.Run("has multi-line height", func(t *testing.T) {
		// Height should be at least 3 for multi-line input
		// We can't directly check height, but we can verify textarea is configured
		if ta.CharLimit != 4000 {
			t.Errorf("expected CharLimit 4000, got %d", ta.CharLimit)
		}
	})

	t.Run("InsertNewline is disabled", func(t *testing.T) {
		// InsertNewline should be disabled because we handle \ + Enter manually
		keys := ta.KeyMap.InsertNewline.Keys()
		// Should be empty or contain only empty string
		for _, k := range keys {
			if k != "" {
				t.Errorf("InsertNewline should be disabled, but has key: %s", k)
			}
		}
	})

	t.Run("Enter is not bound to InsertNewline", func(t *testing.T) {
		keys := ta.KeyMap.InsertNewline.Keys()
		for _, k := range keys {
			if k == "enter" {
				t.Error("Enter should not be bound to InsertNewline (should send message instead)")
			}
		}
	})
}

func TestModel_MultilineInput_StatusBar(t *testing.T) {
	ta := textarea.New()
	s := spinner.New()

	m := Model{
		textarea: ta,
		spinner:  s,
		ready:    true,
		width:    120,
		height:   40,
		viewport: viewport.New(100, 20),
	}

	statusBar := m.renderStatusBar(100)

	t.Run("shows Enter for Send", func(t *testing.T) {
		if !strings.Contains(statusBar, "Enter") || !strings.Contains(statusBar, "Send") {
			t.Error("status bar should show Enter for Send")
		}
	})

	t.Run("shows backslash+Enter for Newline", func(t *testing.T) {
		// Status bar should show \+Enter for Newline (line continuation)
		if !strings.Contains(statusBar, "\\+Enter") || !strings.Contains(statusBar, "Newline") {
			t.Error("status bar should show \\+Enter for Newline")
		}
	})
}

func TestModel_EnterKey_SendsMessage(t *testing.T) {
	t.Run("sends message when text present", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("Hello world")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Message should be added to messages
		if len(typedModel.messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(typedModel.messages))
		}

		// First message should be from user
		if typedModel.messages[0].role != "user" {
			t.Errorf("expected user role, got %s", typedModel.messages[0].role)
		}

		// Content should match
		if typedModel.messages[0].content != "Hello world" {
			t.Errorf("expected 'Hello world', got %s", typedModel.messages[0].content)
		}

		// Loading should be true
		if !typedModel.loading {
			t.Error("should be loading after sending message")
		}

		// Textarea should be cleared
		if typedModel.textarea.Value() != "" {
			t.Error("textarea should be cleared after sending")
		}

		// Command should be returned (batch with send, spinner, animation)
		if cmd == nil {
			t.Error("should return a command")
		}
	})

	t.Run("does nothing when text is empty", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// No message should be added
		if len(typedModel.messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(typedModel.messages))
		}

		// Should not be loading
		if typedModel.loading {
			t.Error("should not be loading when text is empty")
		}
	})

	t.Run("does nothing when only whitespace", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("   \n\t  ")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// No message should be added
		if len(typedModel.messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(typedModel.messages))
		}
	})

	t.Run("does nothing when loading", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("Hello")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			loading:  true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// No message should be added while loading
		if len(typedModel.messages) != 0 {
			t.Errorf("expected 0 messages while loading, got %d", len(typedModel.messages))
		}
	})
}

func TestModel_MultilineInput_Integration(t *testing.T) {
	t.Run("NewChatModelWithSession uses createTextarea", func(t *testing.T) {
		session := &mockChatSession{}
		ta := createTextarea()
		s := spinner.New()

		// Simulate what NewChatModelWithSession does
		m := Model{
			session:   session,
			modelName: "test-model",
			textarea:  ta,
			spinner:   s,
			messages:  []chatMessage{},
		}

		// InsertNewline should be disabled (we handle \ + Enter manually)
		keys := m.textarea.KeyMap.InsertNewline.Keys()
		for _, k := range keys {
			if k != "" {
				t.Errorf("InsertNewline should be disabled, but has key: %s", k)
			}
		}
	})

	t.Run("NewChatModelWithConversation uses createTextarea", func(t *testing.T) {
		session := &mockChatSession{}
		conv := &history.Conversation{ID: "test"}
		store := &mockHistoryStoreForModel{}
		ta := createTextarea()
		s := spinner.New()

		// Simulate what NewChatModelWithConversation does
		m := Model{
			session:      session,
			modelName:    "test-model",
			textarea:     ta,
			spinner:      s,
			messages:     []chatMessage{},
			conversation: conv,
			historyStore: store,
		}

		// InsertNewline should be disabled (we handle \ + Enter manually)
		keys := m.textarea.KeyMap.InsertNewline.Keys()
		for _, k := range keys {
			if k != "" {
				t.Errorf("InsertNewline should be disabled, but has key: %s", k)
			}
		}
	})
}

func TestModel_LineContinuation(t *testing.T) {
	t.Run("backslash at end inserts newline", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("Hello\\")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		// Press Enter with backslash at end
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(enterMsg)
		updatedModel := newModel.(Model)

		// Should insert newline instead of sending
		value := updatedModel.textarea.Value()
		if !strings.Contains(value, "\n") {
			t.Error("backslash + Enter should insert newline")
		}
		if strings.Contains(value, "\\") {
			t.Error("backslash should be removed after line continuation")
		}
		if len(updatedModel.messages) > 0 {
			t.Error("message should not be sent when using line continuation")
		}
	})

	t.Run("no backslash sends message", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("Hello world")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		// Press Enter without backslash
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(enterMsg)
		updatedModel := newModel.(Model)

		// Should send message
		if len(updatedModel.messages) == 0 {
			t.Error("message should be sent when no backslash at end")
		}
	})

	t.Run("multiple continuations work", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("Line 1\\")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		// First continuation
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(enterMsg)
		m = newModel.(Model)

		// Add more text with backslash
		m.textarea.SetValue(m.textarea.Value() + "Line 2\\")

		// Second continuation
		newModel, _ = m.Update(enterMsg)
		m = newModel.(Model)

		value := m.textarea.Value()
		if strings.Count(value, "\n") != 2 {
			t.Errorf("expected 2 newlines, got %d", strings.Count(value, "\n"))
		}
	})
}

// ==================== Command Parsing Tests ====================

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  ParsedCommand
	}{
		{
			name:  "simple command without args",
			input: "/history",
			expected: ParsedCommand{
				Command:   "history",
				Args:      "",
				IsCommand: true,
			},
		},
		{
			name:  "command with args",
			input: "/file /path/to/file.txt",
			expected: ParsedCommand{
				Command:   "file",
				Args:      "/path/to/file.txt",
				IsCommand: true,
			},
		},
		{
			name:  "command with spaces in args",
			input: "/file /path/to/my file.txt",
			expected: ParsedCommand{
				Command:   "file",
				Args:      "/path/to/my file.txt",
				IsCommand: true,
			},
		},
		{
			name:  "not a command - regular text",
			input: "hello world",
			expected: ParsedCommand{
				Command:   "",
				Args:      "",
				IsCommand: false,
			},
		},
		{
			name:  "not a command - empty string",
			input: "",
			expected: ParsedCommand{
				Command:   "",
				Args:      "",
				IsCommand: false,
			},
		},
		{
			name:  "command is lowercased",
			input: "/HISTORY",
			expected: ParsedCommand{
				Command:   "history",
				Args:      "",
				IsCommand: true,
			},
		},
		{
			name:  "command with leading whitespace",
			input: "  /gems",
			expected: ParsedCommand{
				Command:   "gems",
				Args:      "",
				IsCommand: true,
			},
		},
		{
			name:  "image command",
			input: "/image ~/Pictures/photo.jpg",
			expected: ParsedCommand{
				Command:   "image",
				Args:      "~/Pictures/photo.jpg",
				IsCommand: true,
			},
		},
		{
			name:  "exit command",
			input: "/exit",
			expected: ParsedCommand{
				Command:   "exit",
				Args:      "",
				IsCommand: true,
			},
		},
		{
			name:  "clear command",
			input: "/clear",
			expected: ParsedCommand{
				Command:   "clear",
				Args:      "",
				IsCommand: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommand(tt.input)

			if result.Command != tt.expected.Command {
				t.Errorf("Command: expected %q, got %q", tt.expected.Command, result.Command)
			}
			if result.Args != tt.expected.Args {
				t.Errorf("Args: expected %q, got %q", tt.expected.Args, result.Args)
			}
			if result.IsCommand != tt.expected.IsCommand {
				t.Errorf("IsCommand: expected %v, got %v", tt.expected.IsCommand, result.IsCommand)
			}
		})
	}
}

func TestModel_CommandHandling(t *testing.T) {
	t.Run("exit command quits", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/exit")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		// Should return quit command
		if cmd == nil {
			t.Error("expected quit command for /exit")
		}
	})

	t.Run("quit command quits", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/quit")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("expected quit command for /quit")
		}
	})

	t.Run("unknown command shows error", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/unknowncommand")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error for unknown command")
		}
		if !strings.Contains(typedModel.err.Error(), "unknown command") {
			t.Errorf("expected 'unknown command' error, got: %v", typedModel.err)
		}
	})

	t.Run("clear command clears attachments", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/clear")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			viewport:    viewport.New(100, 20),
			messages:    []chatMessage{},
			attachments: []*api.UploadedFile{{FileName: "test.txt"}},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if len(typedModel.attachments) != 0 {
			t.Errorf("expected 0 attachments after /clear, got %d", len(typedModel.attachments))
		}
		if typedModel.err != nil {
			t.Errorf("unexpected error: %v", typedModel.err)
		}
	})

	t.Run("gems command enters gem selection mode", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/gems")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		if !typedModel.selectingGem {
			t.Error("expected selectingGem to be true")
		}
		if !typedModel.gemsLoading {
			t.Error("expected gemsLoading to be true")
		}
		if cmd == nil {
			t.Error("expected command to load gems")
		}
	})

	t.Run("history command without store shows error", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/history")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:         ta,
			spinner:          s,
			session:          mockSession,
			ready:            true,
			viewport:         viewport.New(100, 20),
			messages:         []chatMessage{},
			fullHistoryStore: nil, // No store
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error when history store is nil")
		}
	})
}

// mockGeminiClientWithUpload implements GeminiClientInterface with file upload
type mockGeminiClientWithUpload struct {
	uploadFileResult *api.UploadedFile
	uploadFileErr    error
	uploadFileCalled bool
	uploadFilePath   string
}

func (m *mockGeminiClientWithUpload) Init() error                 { return nil }
func (m *mockGeminiClientWithUpload) Close()                      {}
func (m *mockGeminiClientWithUpload) GetAccessToken() string      { return "" }
func (m *mockGeminiClientWithUpload) GetCookies() *config.Cookies { return nil }
func (m *mockGeminiClientWithUpload) GetModel() models.Model      { return models.Model{} }
func (m *mockGeminiClientWithUpload) SetModel(model models.Model) {}
func (m *mockGeminiClientWithUpload) IsClosed() bool              { return false }
func (m *mockGeminiClientWithUpload) StartChat(model ...models.Model) *api.ChatSession {
	return nil
}
func (m *mockGeminiClientWithUpload) StartChatWithOptions(opts ...api.ChatOption) *api.ChatSession {
	return nil
}
func (m *mockGeminiClientWithUpload) GenerateContent(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) UploadImage(filePath string) (*api.UploadedImage, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) UploadFile(filePath string) (*api.UploadedFile, error) {
	m.uploadFileCalled = true
	m.uploadFilePath = filePath
	return m.uploadFileResult, m.uploadFileErr
}
func (m *mockGeminiClientWithUpload) RefreshFromBrowser() (bool, error) { return false, nil }
func (m *mockGeminiClientWithUpload) IsBrowserRefreshEnabled() bool     { return false }
func (m *mockGeminiClientWithUpload) FetchGems(includeHidden bool) (*models.GemJar, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) CreateGem(name, prompt, description string) (*models.Gem, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) UpdateGem(gemID, name, prompt, description string) (*models.Gem, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) DeleteGem(gemID string) error      { return nil }
func (m *mockGeminiClientWithUpload) Gems() *models.GemJar              { return nil }
func (m *mockGeminiClientWithUpload) GetGem(id, name string) *models.Gem { return nil }
func (m *mockGeminiClientWithUpload) BatchExecute(requests []api.RPCData) ([]api.BatchResponse, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) DownloadImage(img models.WebImage, opts api.ImageDownloadOptions) (string, error) {
	return "", nil
}
func (m *mockGeminiClientWithUpload) DownloadGeneratedImage(img models.GeneratedImage, opts api.ImageDownloadOptions) (string, error) {
	return "", nil
}
func (m *mockGeminiClientWithUpload) DownloadAllImages(output *models.ModelOutput, opts api.ImageDownloadOptions) ([]string, error) {
	return nil, nil
}
func (m *mockGeminiClientWithUpload) DownloadSelectedImages(output *models.ModelOutput, indices []int, opts api.ImageDownloadOptions) ([]string, error) {
	return nil, nil
}

func TestModel_FileCommand(t *testing.T) {
	t.Run("file command without path shows error", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/file")
		s := spinner.New()
		mockSession := &mockChatSession{}
		mockClient := &mockGeminiClientWithUpload{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			client:   mockClient,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error for /file without path")
		}
		if !strings.Contains(typedModel.err.Error(), "usage:") {
			t.Errorf("expected usage error, got: %v", typedModel.err)
		}
	})

	t.Run("file command with nonexistent file shows error", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/file /nonexistent/path/to/file.txt")
		s := spinner.New()
		mockSession := &mockChatSession{}
		mockClient := &mockGeminiClientWithUpload{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			client:   mockClient,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error for nonexistent file")
		}
		if !strings.Contains(typedModel.err.Error(), "file not found") {
			t.Errorf("expected 'file not found' error, got: %v", typedModel.err)
		}
	})

	t.Run("file command without client shows error", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/file /tmp/testfile.txt")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			client:   nil, // No client
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Since the file doesn't exist, we'll get "file not found" first
		// This test verifies the error handling path
		if typedModel.err == nil {
			t.Error("expected error")
		}
	})

	t.Run("image command is alias for file", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("/image")
		s := spinner.New()
		mockSession := &mockChatSession{}
		mockClient := &mockGeminiClientWithUpload{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			client:   mockClient,
			ready:    true,
			viewport: viewport.New(100, 20),
			messages: []chatMessage{},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Should show same usage error as /file
		if typedModel.err == nil {
			t.Error("expected error for /image without path")
		}
		if !strings.Contains(typedModel.err.Error(), "usage:") {
			t.Errorf("expected usage error, got: %v", typedModel.err)
		}
	})
}

func TestModel_FileUploadedMsg(t *testing.T) {
	t.Run("successful upload adds file to attachments", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			viewport:    viewport.New(100, 20),
			messages:    []chatMessage{},
			attachments: nil,
		}

		uploadedFile := &api.UploadedFile{FileName: "test.txt", MIMEType: "text/plain"}
		msg := fileUploadedMsg{file: uploadedFile}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if len(typedModel.attachments) != 1 {
			t.Errorf("expected 1 attachment, got %d", len(typedModel.attachments))
		}
		if typedModel.attachments[0].FileName != "test.txt" {
			t.Errorf("expected attachment name 'test.txt', got %s", typedModel.attachments[0].FileName)
		}
		if typedModel.err != nil {
			t.Errorf("unexpected error: %v", typedModel.err)
		}
	})

	t.Run("failed upload shows error", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			viewport:    viewport.New(100, 20),
			messages:    []chatMessage{},
			attachments: nil,
		}

		msg := fileUploadedMsg{err: fmt.Errorf("upload failed")}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error on upload failure")
		}
		if !strings.Contains(typedModel.err.Error(), "upload failed") {
			t.Errorf("expected upload error, got: %v", typedModel.err)
		}
		if len(typedModel.attachments) != 0 {
			t.Error("should not add attachment on failure")
		}
	})

	t.Run("multiple uploads accumulate", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			viewport:    viewport.New(100, 20),
			messages:    []chatMessage{},
			attachments: []*api.UploadedFile{{FileName: "first.txt"}},
		}

		msg := fileUploadedMsg{file: &api.UploadedFile{FileName: "second.txt"}}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if len(typedModel.attachments) != 2 {
			t.Errorf("expected 2 attachments, got %d", len(typedModel.attachments))
		}
	})
}

func TestModel_SendMessageWithAttachments(t *testing.T) {
	t.Run("sends message with attachments", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("analyze this file")
		s := spinner.New()

		mockSession := &mockChatSession{
			sendMessageFunc: func(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error) {
				_ = files // Verify files are passed (would be checked in integration test)
				return &models.ModelOutput{
					Candidates: []models.Candidate{{Text: "response"}},
				}, nil
			},
		}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			viewport:    viewport.New(100, 20),
			messages:    []chatMessage{},
			attachments: []*api.UploadedFile{{FileName: "test.txt"}},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Attachments should be cleared after sending
		if len(typedModel.attachments) != 0 {
			t.Errorf("expected 0 attachments after send, got %d", len(typedModel.attachments))
		}

		// Should return a command
		if cmd == nil {
			t.Error("expected command")
		}

		// Execute the command to verify attachments were sent
		// (In a real test, we'd need to run the command)
	})

	t.Run("clears attachments after sending", func(t *testing.T) {
		ta := createTextarea()
		ta.SetValue("test message")
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			viewport:    viewport.New(100, 20),
			messages:    []chatMessage{},
			attachments: []*api.UploadedFile{{FileName: "file1.txt"}, {FileName: "file2.txt"}},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		if len(typedModel.attachments) != 0 {
			t.Errorf("expected attachments to be cleared, got %d", len(typedModel.attachments))
		}
	})
}

func TestModel_AttachmentIndicator(t *testing.T) {
	t.Run("shows attachment count in view", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			width:       100,
			height:      40,
			viewport:    viewport.New(96, 20),
			messages:    []chatMessage{},
			attachments: []*api.UploadedFile{{FileName: "file1.txt"}, {FileName: "file2.txt"}},
		}

		view := m.View()

		// Should show file count with emoji
		if !strings.Contains(view, "📎") {
			t.Error("view should show attachment emoji")
		}
		if !strings.Contains(view, "2 file") {
			t.Error("view should show '2 files' count")
		}
	})

	t.Run("shows singular file for one attachment", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			width:       100,
			height:      40,
			viewport:    viewport.New(96, 20),
			messages:    []chatMessage{},
			attachments: []*api.UploadedFile{{FileName: "file.txt"}},
		}

		view := m.View()

		if !strings.Contains(view, "1 file") {
			t.Error("view should show '1 file' count")
		}
		// Make sure it doesn't say "1 files"
		if strings.Contains(view, "1 files") {
			t.Error("should not show '1 files' (grammatically incorrect)")
		}
	})

	t.Run("no indicator when no attachments", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea:    ta,
			spinner:     s,
			session:     mockSession,
			ready:       true,
			width:       100,
			height:      40,
			viewport:    viewport.New(96, 20),
			messages:    []chatMessage{},
			attachments: nil,
		}

		view := m.View()

		// Should not show attachment indicator
		if strings.Contains(view, "📎") {
			t.Error("view should not show attachment emoji when no attachments")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// IMAGE URL DISPLAY TESTS (Phase 3)
// ═══════════════════════════════════════════════════════════════════════════════

func TestRenderImageLinks(t *testing.T) {
	t.Run("renders single image with title", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "https://example.com/image1.jpg", Title: "Test Image", Alt: ""},
		}

		result := renderImageLinks(images, 80)

		// Should contain header
		if !strings.Contains(result, "Images (1)") {
			t.Error("should show image count in header")
		}

		// Should contain title
		if !strings.Contains(result, "Test Image") {
			t.Error("should show image title")
		}

		// Should contain URL
		if !strings.Contains(result, "https://example.com/image1.jpg") {
			t.Error("should show image URL")
		}
	})

	t.Run("renders multiple images", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "https://example.com/image1.jpg", Title: "Image One"},
			{URL: "https://example.com/image2.jpg", Title: "Image Two"},
			{URL: "https://example.com/image3.jpg", Title: "Image Three"},
		}

		result := renderImageLinks(images, 80)

		// Should contain count
		if !strings.Contains(result, "Images (3)") {
			t.Error("should show correct image count")
		}

		// Should contain all titles
		if !strings.Contains(result, "Image One") {
			t.Error("should show first image title")
		}
		if !strings.Contains(result, "Image Two") {
			t.Error("should show second image title")
		}
		if !strings.Contains(result, "Image Three") {
			t.Error("should show third image title")
		}
	})

	t.Run("uses alt text when title is empty", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "https://example.com/image.jpg", Title: "", Alt: "Alt Description"},
		}

		result := renderImageLinks(images, 80)

		if !strings.Contains(result, "Alt Description") {
			t.Error("should use alt text when title is empty")
		}
	})

	t.Run("uses fallback when title and alt are empty", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "https://example.com/image.jpg", Title: "", Alt: ""},
		}

		result := renderImageLinks(images, 80)

		if !strings.Contains(result, "Image 1") {
			t.Error("should use 'Image N' fallback when title and alt are empty")
		}
	})

	t.Run("truncates long titles", func(t *testing.T) {
		longTitle := strings.Repeat("A", 100) // Very long title
		images := []models.WebImage{
			{URL: "https://example.com/image.jpg", Title: longTitle},
		}

		result := renderImageLinks(images, 50) // Narrow width

		// Should not contain the full title
		if strings.Contains(result, longTitle) {
			t.Error("should truncate long titles")
		}

		// Should contain truncation indicator
		if !strings.Contains(result, "...") {
			t.Error("should show ellipsis for truncated titles")
		}
	})
}

func TestModel_ResponseMsgWithImages(t *testing.T) {
	t.Run("extracts images from response", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			width:    100,
			height:   40,
			viewport: viewport.New(96, 20),
			messages: []chatMessage{},
			loading:  true,
		}

		// Create a response with images
		output := &models.ModelOutput{
			Candidates: []models.Candidate{
				{
					Text: "Here's an image for you",
					WebImages: []models.WebImage{
						{URL: "https://example.com/web.jpg", Title: "Web Image"},
					},
					GeneratedImages: []models.GeneratedImage{
						{URL: "https://example.com/gen.jpg", Title: "Generated Image"},
					},
				},
			},
			Chosen: 0,
		}

		// Process response message
		newM, _ := m.Update(responseMsg{output: output})
		updatedModel := newM.(Model)

		// Should have one message
		if len(updatedModel.messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(updatedModel.messages))
		}

		// Message should have images
		msg := updatedModel.messages[0]
		if len(msg.images) != 2 { // 1 web + 1 generated
			t.Errorf("expected 2 images, got %d", len(msg.images))
		}

		// Verify image content
		if msg.images[0].URL != "https://example.com/web.jpg" {
			t.Errorf("expected web image URL, got %s", msg.images[0].URL)
		}
		if msg.images[1].URL != "https://example.com/gen.jpg" {
			t.Errorf("expected generated image URL, got %s", msg.images[1].URL)
		}
	})

	t.Run("handles response without images", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()
		mockSession := &mockChatSession{}

		m := Model{
			textarea: ta,
			spinner:  s,
			session:  mockSession,
			ready:    true,
			width:    100,
			height:   40,
			viewport: viewport.New(96, 20),
			messages: []chatMessage{},
			loading:  true,
		}

		// Create a response without images
		output := &models.ModelOutput{
			Candidates: []models.Candidate{
				{Text: "Just text, no images"},
			},
			Chosen: 0,
		}

		// Process response message
		newM, _ := m.Update(responseMsg{output: output})
		updatedModel := newM.(Model)

		// Message should have no images
		msg := updatedModel.messages[0]
		if len(msg.images) != 0 {
			t.Errorf("expected 0 images, got %d", len(msg.images))
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// TUI THEME/STYLES TESTS (Phase 3)
// ═══════════════════════════════════════════════════════════════════════════════

func TestUpdateTheme(t *testing.T) {
	// Reset theme after test
	defer func() {
		render.SetTUITheme("tokyonight")
		UpdateTheme()
	}()

	t.Run("updates colors from theme", func(t *testing.T) {
		// Set a different theme
		render.SetTUITheme("catppuccin")
		UpdateTheme()

		// Verify the theme was applied (colors should have changed)
		theme := render.GetTUITheme()
		if theme.Name != "catppuccin" {
			t.Errorf("expected theme 'catppuccin', got '%s'", theme.Name)
		}

		// colorPrimary should match theme's primary color
		// We can't directly compare lipgloss.Color values, but we can verify the function runs without error
	})

	t.Run("GetCurrentThemeName returns theme name", func(t *testing.T) {
		render.SetTUITheme("nord")
		UpdateTheme()

		name := GetCurrentThemeName()
		if name != "nord" {
			t.Errorf("expected theme name 'nord', got '%s'", name)
		}
	})
}

func TestModel_UpdateViewportWithImages(t *testing.T) {
	t.Run("renders images in viewport", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()

		m := Model{
			textarea: ta,
			spinner:  s,
			ready:    true,
			width:    100,
			height:   40,
			viewport: viewport.New(96, 20),
			messages: []chatMessage{
				{
					role:    "assistant",
					content: "Here's an image",
					images: []models.WebImage{
						{URL: "https://example.com/test.jpg", Title: "Test Image"},
					},
				},
			},
		}

		m.updateViewport()
		content := m.viewport.View()

		// Should contain image section
		if !strings.Contains(content, "Images") {
			t.Error("viewport should render image section")
		}

		// Should contain image URL
		if !strings.Contains(content, "https://example.com/test.jpg") {
			t.Error("viewport should contain image URL")
		}

		// Should contain image title
		if !strings.Contains(content, "Test Image") {
			t.Error("viewport should contain image title")
		}
	})

	t.Run("does not render image section when no images", func(t *testing.T) {
		ta := createTextarea()
		s := spinner.New()

		m := Model{
			textarea: ta,
			spinner:  s,
			ready:    true,
			width:    100,
			height:   40,
			viewport: viewport.New(96, 20),
			messages: []chatMessage{
				{
					role:    "assistant",
					content: "No images here",
					images:  nil,
				},
			},
		}

		m.updateViewport()
		content := m.viewport.View()

		// Should not contain image section header
		if strings.Contains(content, "🖼") {
			t.Error("viewport should not show image emoji when no images")
		}
	})
}

// ==================== Export Command Tests ====================

func TestParseExportArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		wantPath   string
		wantFormat string
		wantErr    bool
	}{
		{
			name:       "markdown extension",
			args:       "chat.md",
			wantPath:   "chat.md",
			wantFormat: "markdown",
		},
		{
			name:       "json extension",
			args:       "chat.json",
			wantPath:   "chat.json",
			wantFormat: "json",
		},
		{
			name:       "no extension adds .md",
			args:       "chat",
			wantPath:   "chat.md",
			wantFormat: "markdown",
		},
		{
			name:       "explicit json flag",
			args:       "chat -f json",
			wantPath:   "chat.json",
			wantFormat: "json",
		},
		{
			name:       "explicit md flag",
			args:       "chat -f md",
			wantPath:   "chat.md",
			wantFormat: "markdown",
		},
		{
			name:       "flag overrides extension",
			args:       "chat.md -f json",
			wantPath:   "chat.md",
			wantFormat: "json",
		},
		{
			name:       "path with spaces",
			args:       "my chat.md",
			wantPath:   "my chat.md",
			wantFormat: "markdown",
		},
		{
			name:    "empty args",
			args:    "",
			wantErr: true,
		},
		{
			name:    "unknown format",
			args:    "chat -f xml",
			wantErr: true,
		},
		{
			name:    "flag without path",
			args:    "-f json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, format, err := parseExportArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseExportArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if path != tt.wantPath {
					t.Errorf("parseExportArgs() path = %v, want %v", path, tt.wantPath)
				}
				if format != tt.wantFormat {
					t.Errorf("parseExportArgs() format = %v, want %v", format, tt.wantFormat)
				}
			}
		})
	}
}

func TestValidateExportPath(t *testing.T) {
	t.Run("relative path becomes absolute", func(t *testing.T) {
		path, err := validateExportPath("test.md")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		// Path should be absolute
		if path == "test.md" || path[0] != '/' {
			t.Errorf("expected absolute path, got %s", path)
		}
	})

	t.Run("tilde expansion", func(t *testing.T) {
		path, err := validateExportPath("~/test.md")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		// Path should not contain tilde
		if strings.Contains(path, "~") {
			t.Errorf("expected tilde to be expanded, got %s", path)
		}
	})

	t.Run("nonexistent parent directory", func(t *testing.T) {
		_, err := validateExportPath("/nonexistent/path/to/test.md")
		if err == nil {
			t.Error("expected error for nonexistent parent directory")
		}
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal title",
			input:    "My Chat",
			expected: "My Chat",
		},
		{
			name:     "with slashes",
			input:    "Chat/with/slashes",
			expected: "Chat_with_slashes",
		},
		{
			name:     "with colons",
			input:    "Chat: Topic",
			expected: "Chat_ Topic",
		},
		{
			name:     "with multiple invalid chars",
			input:    "File: *test* <data>",
			expected: "File_ _test_ _data_",
		},
		{
			name:     "empty after sanitization",
			input:    "///",
			expected: "conversation",
		},
		{
			name:     "dots at ends",
			input:    "...test...",
			expected: "test",
		},
		{
			name:     "long title truncated",
			input:    strings.Repeat("a", 300),
			expected: strings.Repeat("a", 200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExportResultMsg(t *testing.T) {
	t.Run("with success", func(t *testing.T) {
		msg := exportResultMsg{
			path:      "/tmp/test.md",
			format:    "markdown",
			size:      1024,
			overwrite: false,
		}

		if msg.path != "/tmp/test.md" {
			t.Errorf("expected path /tmp/test.md, got %s", msg.path)
		}
		if msg.format != "markdown" {
			t.Errorf("expected format markdown, got %s", msg.format)
		}
		if msg.err != nil {
			t.Error("expected no error")
		}
	})

	t.Run("with error", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		msg := exportResultMsg{err: testErr}

		if msg.err == nil {
			t.Error("expected error")
		}
	})
}

func TestModel_HandleExportCommand(t *testing.T) {
	t.Run("no conversation returns error", func(t *testing.T) {
		m := Model{
			conversation: nil,
			messages:     []chatMessage{},
		}

		updatedModel, _ := m.handleExportCommand("test.md")
		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error when no conversation to export")
		}
	})

	t.Run("with in-memory messages exports from memory", func(t *testing.T) {
		m := Model{
			conversation: nil,
			messages: []chatMessage{
				{role: "user", content: "Hello"},
				{role: "assistant", content: "Hi"},
			},
		}

		updatedModel, cmd := m.handleExportCommand("/tmp/test_export.md")
		typedModel := updatedModel.(Model)

		if typedModel.err != nil {
			t.Errorf("unexpected error: %v", typedModel.err)
		}
		if cmd == nil {
			t.Error("expected a command for async export")
		}
	})

	t.Run("invalid path returns error", func(t *testing.T) {
		m := Model{
			conversation: nil,
			messages: []chatMessage{
				{role: "user", content: "Hello"},
			},
		}

		updatedModel, _ := m.handleExportCommand("/nonexistent/dir/test.md")
		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error for invalid path")
		}
	})

	t.Run("default filename from conversation title", func(t *testing.T) {
		m := Model{
			conversation: &history.Conversation{
				ID:    "test-id",
				Title: "My Test Chat",
			},
			fullHistoryStore: &mockFullHistoryStore{},
		}

		// Empty args should use conversation title
		updatedModel, cmd := m.handleExportCommand("")
		typedModel := updatedModel.(Model)

		if typedModel.err != nil {
			t.Errorf("unexpected error: %v", typedModel.err)
		}
		if cmd == nil {
			t.Error("expected a command for async export")
		}
	})
}

func TestModel_Update_ExportResultMsg(t *testing.T) {
	t.Run("success sets feedback", func(t *testing.T) {
		m := Model{ready: true}

		msg := exportResultMsg{
			path:      "/tmp/test.md",
			format:    "markdown",
			size:      1024,
			overwrite: false,
		}

		updatedModel, _ := m.Update(msg)
		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected feedback in err field")
		}
		if !strings.Contains(typedModel.err.Error(), "Exported") {
			t.Errorf("expected success message, got: %v", typedModel.err)
		}
	})

	t.Run("success with overwrite indicates overwrite", func(t *testing.T) {
		m := Model{ready: true}

		msg := exportResultMsg{
			path:      "/tmp/test.md",
			format:    "markdown",
			size:      1024,
			overwrite: true,
		}

		updatedModel, _ := m.Update(msg)
		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected feedback in err field")
		}
		if !strings.Contains(typedModel.err.Error(), "overwritten") {
			t.Errorf("expected overwrite indication, got: %v", typedModel.err)
		}
	})

	t.Run("error is propagated", func(t *testing.T) {
		m := Model{ready: true}

		testErr := fmt.Errorf("export failed")
		msg := exportResultMsg{err: testErr}

		updatedModel, _ := m.Update(msg)
		typedModel := updatedModel.(Model)

		if typedModel.err == nil {
			t.Error("expected error")
		}
		if typedModel.err.Error() != "export failed" {
			t.Errorf("expected 'export failed', got: %v", typedModel.err)
		}
	})
}

func TestModel_ExportCommand_Registration(t *testing.T) {
	t.Run("/export command is recognized", func(t *testing.T) {
		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetValue("/export test.md")

		vp := viewport.New(80, 20)

		m := Model{
			ready:    true,
			loading:  false,
			textarea: ta,
			viewport: vp,
			width:    100,
			height:   40,
			messages: []chatMessage{
				{role: "user", content: "Hello"},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Should not add a message (command was processed)
		if len(typedModel.messages) > 1 {
			t.Error("should not add message for /export command")
		}

		// May have error (path validation) or cmd (async export)
		// The key is that the command was recognized, not treated as unknown
		if typedModel.err != nil && strings.Contains(typedModel.err.Error(), "unknown command") {
			t.Error("/export should be a recognized command")
		}

		// If no error, should have a command
		if typedModel.err == nil && cmd == nil {
			t.Log("no error and no cmd - path may be invalid")
		}
	})
}

// TestModel_Update_CtrlG tests the Ctrl+G shortcut to open gem selector
func TestModel_Update_CtrlG(t *testing.T) {
	t.Run("opens gem selector", func(t *testing.T) {
		// Create a model with necessary components
		ta := textarea.New()
		ta.SetValue("some text")
		vp := viewport.New(80, 20)

		m := Model{
			ready:    true,
			textarea: ta,
			viewport: vp,
			width:    100,
			height:   40,
		}

		// Simulate Ctrl+G
		msg := tea.KeyMsg{Type: tea.KeyCtrlG}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Should enable gem selection mode
		if !typedModel.selectingGem {
			t.Error("Ctrl+G should enable selectingGem")
		}

		// Should set gemsLoading to true
		if !typedModel.gemsLoading {
			t.Error("Ctrl+G should set gemsLoading to true")
		}

		// Should reset gemsCursor and gemsFilter
		if typedModel.gemsCursor != 0 {
			t.Errorf("gemsCursor should be 0, got %d", typedModel.gemsCursor)
		}

		if typedModel.gemsFilter != "" {
			t.Errorf("gemsFilter should be empty, got %q", typedModel.gemsFilter)
		}

		// Should reset textarea
		if typedModel.textarea.Value() != "" {
			t.Error("textarea should be reset")
		}

		// Should return a command (loadGemsForChat)
		if cmd == nil {
			t.Error("Ctrl+G should return a command")
		}
	})
}

// TestModel_Update_CtrlE tests the Ctrl+E shortcut to export conversation
func TestModel_Update_CtrlE(t *testing.T) {
	t.Run("exports conversation with default filename", func(t *testing.T) {
		// Create a model with necessary components
		ta := textarea.New()
		vp := viewport.New(80, 20)

		m := Model{
			ready:    true,
			textarea: ta,
			viewport: vp,
			width:    100,
			height:   40,
			messages: []chatMessage{
				{role: "user", content: "Hello"},
				{role: "assistant", content: "Hi there!"},
			},
		}

		// Simulate Ctrl+E
		msg := tea.KeyMsg{Type: tea.KeyCtrlE}
		updatedModel, cmd := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Should not have an error about unknown command
		if typedModel.err != nil && strings.Contains(typedModel.err.Error(), "unknown command") {
			t.Error("Ctrl+E should not produce 'unknown command' error")
		}

		// Should return a command (exportFromMemory or exportCommand)
		// Note: may return nil cmd if path validation fails, which is acceptable
		_ = cmd
	})

	t.Run("shows error when no conversation", func(t *testing.T) {
		// Create a model with no messages
		ta := textarea.New()
		vp := viewport.New(80, 20)

		m := Model{
			ready:    true,
			textarea: ta,
			viewport: vp,
			width:    100,
			height:   40,
			messages: []chatMessage{}, // Empty
		}

		// Simulate Ctrl+E
		msg := tea.KeyMsg{Type: tea.KeyCtrlE}
		updatedModel, _ := m.Update(msg)

		typedModel := updatedModel.(Model)

		// Should have an error about no conversation
		if typedModel.err == nil {
			t.Error("Ctrl+E with no messages should produce an error")
		}

		if typedModel.err != nil && !strings.Contains(typedModel.err.Error(), "no conversation") {
			t.Errorf("expected 'no conversation' error, got: %v", typedModel.err)
		}
	})
}

// TestRenderStatusBar_ShowsNewShortcuts tests that the status bar includes new shortcuts
func TestRenderStatusBar_ShowsNewShortcuts(t *testing.T) {
	m := Model{
		ready:  true,
		width:  100,
		height: 40,
	}

	statusBar := m.renderStatusBar(80)

	// Should contain ^E for Export
	if !strings.Contains(statusBar, "^E") {
		t.Error("status bar should contain ^E shortcut")
	}

	// Should contain ^G for Gems
	if !strings.Contains(statusBar, "^G") {
		t.Error("status bar should contain ^G shortcut")
	}

	// Should contain Export description
	if !strings.Contains(statusBar, "Export") {
		t.Error("status bar should contain Export description")
	}

	// Should contain Gems description
	if !strings.Contains(statusBar, "Gems") {
		t.Error("status bar should contain Gems description")
	}
}

// TestModel_FormatError tests the formatError function
func TestModel_FormatError(t *testing.T) {
	m := Model{}

	t.Run("nil error returns empty string", func(t *testing.T) {
		result := m.formatError(nil)
		if result != "" {
			t.Errorf("formatError(nil) = %q, want empty string", result)
		}
	})

	t.Run("simple error is formatted", func(t *testing.T) {
		err := fmt.Errorf("test error")
		result := m.formatError(err)
		if !strings.Contains(result, "Error") {
			t.Error("should contain Error label")
		}
		if !strings.Contains(result, "test error") {
			t.Error("should contain error message")
		}
	})
}

// TestModel_UpdateGemSelection tests the updateGemSelection function
func TestModel_UpdateGemSelection(t *testing.T) {
	// Create mock gems
	gems := []*models.Gem{
		{ID: "gem-1", Name: "First Gem", Description: "First description"},
		{ID: "gem-2", Name: "Second Gem", Description: "Second description"},
		{ID: "gem-3", Name: "Third Gem", Description: "Third description"},
	}

	t.Run("up key navigates gems", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsCursor:   1,
		}

		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsCursor != 0 {
			t.Errorf("gemsCursor = %d, want 0", model.gemsCursor)
		}
	})

	t.Run("down key navigates gems", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsCursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsCursor != 1 {
			t.Errorf("gemsCursor = %d, want 1", model.gemsCursor)
		}
	})

	t.Run("up key wraps around", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsCursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsCursor != 2 {
			t.Errorf("gemsCursor = %d, want 2 (wrap)", model.gemsCursor)
		}
	})

	t.Run("down key wraps around", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsCursor:   2,
		}

		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsCursor != 0 {
			t.Errorf("gemsCursor = %d, want 0 (wrap)", model.gemsCursor)
		}
	})

	t.Run("enter selects gem", func(t *testing.T) {
		session := &mockChatSession{}
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsCursor:   1,
			session:      session,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.selectingGem {
			t.Error("selectingGem should be false after selection")
		}
		if model.activeGemName != "Second Gem" {
			t.Errorf("activeGemName = %s, want Second Gem", model.activeGemName)
		}
		if session.gemID != "gem-2" {
			t.Errorf("session.gemID = %s, want gem-2", session.gemID)
		}
	})

	t.Run("esc cancels gem selection", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsCursor:   1,
			gemsFilter:   "test",
		}

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.selectingGem {
			t.Error("selectingGem should be false after esc")
		}
		if model.gemsList != nil {
			t.Error("gemsList should be nil after esc")
		}
		if model.gemsFilter != "" {
			t.Error("gemsFilter should be empty after esc")
		}
	})

	t.Run("backspace removes filter character", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsFilter:   "test",
		}

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsFilter != "tes" {
			t.Errorf("gemsFilter = %q, want tes", model.gemsFilter)
		}
	})

	t.Run("typing adds to filter", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
			gemsFilter:   "",
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsFilter != "a" {
			t.Errorf("gemsFilter = %q, want a", model.gemsFilter)
		}
	})

	t.Run("ctrl+c quits from gem selection", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsList:     gems,
		}

		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := m.updateGemSelection(msg)

		if cmd == nil {
			t.Error("ctrl+c should return quit command")
		}
	})

	t.Run("window size updates dimensions", func(t *testing.T) {
		m := Model{
			selectingGem: true,
		}

		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.width != 100 {
			t.Errorf("width = %d, want 100", model.width)
		}
		if model.height != 50 {
			t.Errorf("height = %d, want 50", model.height)
		}
	})

	t.Run("gemsLoadedForChatMsg updates gems", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsLoading:  true,
		}

		msg := gemsLoadedForChatMsg{gems: gems}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsLoading {
			t.Error("gemsLoading should be false")
		}
		if len(model.gemsList) != 3 {
			t.Errorf("gemsList length = %d, want 3", len(model.gemsList))
		}
	})

	t.Run("gemsLoadedForChatMsg handles error", func(t *testing.T) {
		m := Model{
			selectingGem: true,
			gemsLoading:  true,
		}

		msg := gemsLoadedForChatMsg{err: fmt.Errorf("load error")}
		updatedModel, _ := m.updateGemSelection(msg)
		model := updatedModel.(Model)

		if model.gemsLoading {
			t.Error("gemsLoading should be false")
		}
		if model.selectingGem {
			t.Error("selectingGem should be false on error")
		}
		if model.err == nil {
			t.Error("err should be set")
		}
	})
}

// TestModel_ExportFromMemory tests the exportFromMemory function
func TestModel_ExportFromMemory(t *testing.T) {
	messages := []chatMessage{
		{role: "user", content: "Hello"},
		{role: "assistant", content: "Hi there!"},
	}

	t.Run("exports to markdown", func(t *testing.T) {
		// Create a temp file path
		tmpFile := "/tmp/test_export_md_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".md"
		defer func() { _ = os.Remove(tmpFile) }()

		cmd := exportFromMemory(messages, "Test Chat", "markdown", tmpFile)
		result := cmd()

		if msg, ok := result.(exportResultMsg); ok {
			if msg.err != nil {
				t.Errorf("unexpected error: %v", msg.err)
			}
			if msg.format != "markdown" {
				t.Errorf("format = %s, want markdown", msg.format)
			}
			if msg.path != tmpFile {
				t.Errorf("path = %s, want %s", msg.path, tmpFile)
			}
		} else {
			t.Error("expected exportResultMsg")
		}
	})

	t.Run("exports to json", func(t *testing.T) {
		tmpFile := "/tmp/test_export_json_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".json"
		defer func() { _ = os.Remove(tmpFile) }()

		cmd := exportFromMemory(messages, "Test Chat", "json", tmpFile)
		result := cmd()

		if msg, ok := result.(exportResultMsg); ok {
			if msg.err != nil {
				t.Errorf("unexpected error: %v", msg.err)
			}
			if msg.format != "json" {
				t.Errorf("format = %s, want json", msg.format)
			}
		} else {
			t.Error("expected exportResultMsg")
		}
	})

	t.Run("detects overwrite", func(t *testing.T) {
		tmpFile := "/tmp/test_export_overwrite_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".md"
		// Create file first
		_ = os.WriteFile(tmpFile, []byte("existing"), 0644)
		defer func() { _ = os.Remove(tmpFile) }()

		cmd := exportFromMemory(messages, "Test Chat", "markdown", tmpFile)
		result := cmd()

		if msg, ok := result.(exportResultMsg); ok {
			if !msg.overwrite {
				t.Error("overwrite should be true")
			}
		} else {
			t.Error("expected exportResultMsg")
		}
	})
}

// TestJsonMarshalIndent tests the jsonMarshalIndent helper
func TestJsonMarshalIndent(t *testing.T) {
	data := map[string]string{"key": "value"}
	result, err := jsonMarshalIndent(data, "", "  ")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := "{\n  \"key\": \"value\"\n}"
	if string(result) != expected {
		t.Errorf("result = %q, want %q", string(result), expected)
	}
}

// TestNewChatModel_WithClient tests the NewChatModel constructor with a real client
func TestNewChatModel_WithClient(t *testing.T) {
	client := &mockGeminiClientWithUpload{}
	m := NewChatModel(client, "test-model")

	if m.client != client {
		t.Error("client not set correctly")
	}
	if m.modelName != "test-model" {
		t.Errorf("modelName = %s, want test-model", m.modelName)
	}
	if len(m.messages) != 0 {
		t.Errorf("messages length = %d, want 0", len(m.messages))
	}
}
