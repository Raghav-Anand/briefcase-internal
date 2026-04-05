package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/raghav-anand/briefcase-internal/models"
)

// CreateDecision records a decision made during a session. Returns the new decision ID.
func (c *Client) CreateDecision(ctx context.Context, uid, pid string, d *models.CreateDecisionInput) (string, error) {
	ref, _, err := c.decisionsCol(uid, pid).Add(ctx, map[string]interface{}{
		"decision":   d.Decision,
		"rationale":  d.Rationale,
		"session_id": d.SessionID,
		"tags":       d.Tags,
		"created_at": time.Now(),
	})
	if err != nil {
		return "", fmt.Errorf("CreateDecision: %w", err)
	}
	return ref.ID, nil
}

// ListDecisions returns decisions for a project in reverse chronological order.
func (c *Client) ListDecisions(ctx context.Context, uid, pid string, limit int) ([]models.Decision, error) {
	q := c.decisionsCol(uid, pid).OrderBy("created_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListDecisions: %w", err)
	}

	decisions := make([]models.Decision, 0, len(docs))
	for _, doc := range docs {
		var d models.Decision
		if err := doc.DataTo(&d); err != nil {
			return nil, fmt.Errorf("ListDecisions decode %s: %w", doc.Ref.ID, err)
		}
		d.ID = doc.Ref.ID
		decisions = append(decisions, d)
	}
	return decisions, nil
}

// GetDecision returns a single decision by ID.
func (c *Client) GetDecision(ctx context.Context, uid, pid, did string) (*models.Decision, error) {
	doc, err := c.decisionsCol(uid, pid).Doc(did).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetDecision %s: %w", did, err)
	}
	var d models.Decision
	if err := doc.DataTo(&d); err != nil {
		return nil, fmt.Errorf("GetDecision decode %s: %w", did, err)
	}
	d.ID = doc.Ref.ID
	return &d, nil
}
