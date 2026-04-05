package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/raghav-anand/briefcase-internal/models"
)

// CreateNote adds a note to a project. Returns the new note ID.
func (c *Client) CreateNote(ctx context.Context, uid, pid string, n *models.CreateNoteInput) (string, error) {
	ref, _, err := c.notesCol(uid, pid).Add(ctx, map[string]interface{}{
		"content":    n.Content,
		"session_id": n.SessionID,
		"note_type":  n.NoteType,
		"created_at": time.Now(),
	})
	if err != nil {
		return "", fmt.Errorf("CreateNote: %w", err)
	}
	return ref.ID, nil
}

// ListNotes returns notes for a project in reverse chronological order.
// Pass noteType="" to return all types; pass "bug", "idea", etc. to filter.
func (c *Client) ListNotes(ctx context.Context, uid, pid string, noteType string, limit int) ([]models.Note, error) {
	q := c.notesCol(uid, pid).Query
	if noteType != "" {
		q = q.Where("note_type", "==", noteType)
	}
	q = q.OrderBy("created_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListNotes: %w", err)
	}

	notes := make([]models.Note, 0, len(docs))
	for _, doc := range docs {
		var n models.Note
		if err := doc.DataTo(&n); err != nil {
			return nil, fmt.Errorf("ListNotes decode %s: %w", doc.Ref.ID, err)
		}
		n.ID = doc.Ref.ID
		notes = append(notes, n)
	}
	return notes, nil
}

// GetNote returns a single note by ID.
func (c *Client) GetNote(ctx context.Context, uid, pid, nid string) (*models.Note, error) {
	doc, err := c.notesCol(uid, pid).Doc(nid).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetNote %s: %w", nid, err)
	}
	var n models.Note
	if err := doc.DataTo(&n); err != nil {
		return nil, fmt.Errorf("GetNote decode %s: %w", nid, err)
	}
	n.ID = doc.Ref.ID
	return &n, nil
}
