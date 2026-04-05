package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestNotes(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Notes Test"})

	// --- CreateNote ---
	nid, err := c.CreateNote(ctx, uid, pid, &models.CreateNoteInput{
		Content:   "Found a bug in the auth middleware",
		SessionID: "sess-1",
		NoteType:  "bug",
	})
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if nid == "" {
		t.Fatal("CreateNote returned empty ID")
	}

	// --- GetNote ---
	n, err := c.GetNote(ctx, uid, pid, nid)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if n.ID != nid {
		t.Errorf("ID: got %q, want %q", n.ID, nid)
	}
	if n.NoteType != "bug" {
		t.Errorf("NoteType: got %q, want bug", n.NoteType)
	}
	if n.Content != "Found a bug in the auth middleware" {
		t.Errorf("Content: got %q", n.Content)
	}

	// --- ListNotes (all types) ---
	_, _ = c.CreateNote(ctx, uid, pid, &models.CreateNoteInput{
		Content:   "Idea: add export feature",
		SessionID: "sess-1",
		NoteType:  "idea",
	})

	all, err := c.ListNotes(ctx, uid, pid, "", 0)
	if err != nil {
		t.Fatalf("ListNotes (all): %v", err)
	}
	if len(all) < 2 {
		t.Errorf("ListNotes all: got %d, want >=2", len(all))
	}

	// Filter by type.
	bugs, _ := c.ListNotes(ctx, uid, pid, "bug", 0)
	for _, note := range bugs {
		if note.NoteType != "bug" {
			t.Errorf("type filter returned wrong type: %q", note.NoteType)
		}
	}

	// limit.
	one, _ := c.ListNotes(ctx, uid, pid, "", 1)
	if len(one) != 1 {
		t.Errorf("ListNotes limit=1: got %d", len(one))
	}
}
