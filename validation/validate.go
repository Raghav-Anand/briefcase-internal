package validation

import "fmt"

var validProjectStatuses = map[string]bool{
	"active":    true,
	"paused":    true,
	"completed": true,
	"archived":  true,
}

var validNoteTypes = map[string]bool{
	"general": true,
	"bug":     true,
	"idea":    true,
	"todo":    true,
}

var validDocTypes = map[string]bool{
	"api_docs":     true,
	"architecture": true,
	"readme":       true,
	"custom":       true,
}

var validDocFormats = map[string]bool{
	"markdown": true,
	"mermaid":  true,
}

// ValidateProjectStatus checks if a status string is valid.
func ValidateProjectStatus(s string) error {
	if !validProjectStatuses[s] {
		return fmt.Errorf("invalid project status %q: must be one of active, paused, completed, archived", s)
	}
	return nil
}

// ValidateNoteType checks if a note type is valid.
func ValidateNoteType(t string) error {
	if !validNoteTypes[t] {
		return fmt.Errorf("invalid note type %q: must be one of general, bug, idea, todo", t)
	}
	return nil
}

// ValidateDocType checks if a doc type is valid.
func ValidateDocType(t string) error {
	if !validDocTypes[t] {
		return fmt.Errorf("invalid doc type %q: must be one of api_docs, architecture, readme, custom", t)
	}
	return nil
}

// ValidateDocFormat checks if a doc format is valid.
func ValidateDocFormat(f string) error {
	if !validDocFormats[f] {
		return fmt.Errorf("invalid doc format %q: must be one of markdown, mermaid", f)
	}
	return nil
}
