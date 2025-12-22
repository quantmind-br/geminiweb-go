package tui

import (
	"testing"

	"github.com/diogo/geminiweb/internal/models"
)

func TestGemsModel_FormOps(t *testing.T) {
	m := NewGemsModel(nil, false)
	gem := &models.Gem{ID: "id1", Name: "G", Description: "D", Prompt: "line1\nline2"}
	m.populateForm(gem)
	if m.formInputs[formFieldName].Value() != "G" {
		t.Fatalf("expected name populated")
	}
	if !m.useTextarea {
		t.Fatalf("expected useTextarea to be true for multi-line prompt")
	}
	m.formFocus = formFieldPrompt
	m.blurCurrentField()
	m.focusCurrentField()
}

func TestGemsModel_SubmitFormValidation(t *testing.T) {
	m := NewGemsModel(nil, false)
	// Name required
	m.formInputs[formFieldName].SetValue("")
	m.formInputs[formFieldPrompt].SetValue("prompt")
	mRet, _ := m.submitForm()
	mm := mRet.(GemsModel)
	if mm.feedback != "Name is required" {
		t.Fatalf("expected name required feedback, got: %s", mm.feedback)
	}

	// Prompt required
	m.formInputs[formFieldName].SetValue("Name")
	m.useTextarea = false
	m.formInputs[formFieldPrompt].SetValue("")
	mRet, _ = m.submitForm()
	mm = mRet.(GemsModel)
	if mm.feedback != "Prompt is required" {
		t.Fatalf("expected prompt required feedback, got: %s", mm.feedback)
	}

	// Create mode returns a cmd and sets submitting
	m.view = gemsViewCreate
	m.formInputs[formFieldPrompt].SetValue("p")
	m.formInputs[formFieldDescription].SetValue("desc")
	m.formInputs[formFieldName].SetValue("Name")
	m.useTextarea = false
	mRet, cmd := m.submitForm()
	mm = mRet.(GemsModel)
	if !mm.submitting {
		t.Fatalf("expected submitting to be true")
	}
	if cmd == nil {
		t.Fatalf("expected a non-nil command for create")
	}
}

func TestGemsModel_RenderViews(t *testing.T) {
	m := NewGemsModel(nil, false)
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 40

	// List view (no gems)
	m.view = gemsViewList
	v := m.View()
	if v == "" {
		t.Fatalf("expected non-empty view for list")
	}

	// Details view (no selection)
	m.view = gemsViewDetails
	v = m.View()
	if v == "" {
		t.Fatalf("expected non-empty view for details")
	}

	// Create view
	m.view = gemsViewCreate
	v = m.View()
	if v == "" {
		t.Fatalf("expected non-empty view for create")
	}

	// Edit view with selected gem
	g := &models.Gem{ID: "id2", Name: "Name", Description: "desc", Prompt: "p"}
	m.selectedGem = g
	m.view = gemsViewEdit
	v = m.View()
	if v == "" {
		t.Fatalf("expected non-empty view for edit")
	}

	// Delete view
	m.view = gemsViewDelete
	v = m.View()
	if v == "" {
		t.Fatalf("expected non-empty view for delete")
	}
}
