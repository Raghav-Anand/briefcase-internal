package validation_test

import (
	"testing"

	"github.com/raghav-anand/briefcase-internal/validation"
)

func TestValidateProjectStatus(t *testing.T) {
	valid := []string{"active", "paused", "completed", "archived"}
	for _, s := range valid {
		if err := validation.ValidateProjectStatus(s); err != nil {
			t.Errorf("expected %q to be valid: %v", s, err)
		}
	}

	invalid := []string{"", "draft", "ACTIVE", "Active", "deleted"}
	for _, s := range invalid {
		if err := validation.ValidateProjectStatus(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestValidateNoteType(t *testing.T) {
	valid := []string{"general", "bug", "idea", "todo"}
	for _, s := range valid {
		if err := validation.ValidateNoteType(s); err != nil {
			t.Errorf("expected %q to be valid: %v", s, err)
		}
	}

	invalid := []string{"", "note", "BUG", "feature"}
	for _, s := range invalid {
		if err := validation.ValidateNoteType(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestValidateDocType(t *testing.T) {
	valid := []string{"api_docs", "architecture", "readme", "custom"}
	for _, s := range valid {
		if err := validation.ValidateDocType(s); err != nil {
			t.Errorf("expected %q to be valid: %v", s, err)
		}
	}

	invalid := []string{"", "docs", "API_DOCS", "diagram"}
	for _, s := range invalid {
		if err := validation.ValidateDocType(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestValidateDocFormat(t *testing.T) {
	valid := []string{"markdown", "mermaid"}
	for _, s := range valid {
		if err := validation.ValidateDocFormat(s); err != nil {
			t.Errorf("expected %q to be valid: %v", s, err)
		}
	}

	invalid := []string{"", "html", "MARKDOWN", "text"}
	for _, s := range invalid {
		if err := validation.ValidateDocFormat(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}
