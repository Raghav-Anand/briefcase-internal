package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/raghav-anand/briefcase-internal/models"
)

// CreateMilestone creates a new milestone for a project and increments the project's open_milestone_count.
// Returns the new milestone ID.
func (c *Client) CreateMilestone(ctx context.Context, uid, pid string, m *models.CreateMilestoneInput) (string, error) {
	now := time.Now()

	ref, _, err := c.milestonesCol(uid, pid).Add(ctx, map[string]interface{}{
		"title":       m.Title,
		"description": m.Description,
		"due_date":    m.DueDate,
		"status":      "open",
		"session_id":  m.SessionID,
		"created_at":  now,
	})
	if err != nil {
		return "", fmt.Errorf("CreateMilestone: %w", err)
	}

	// Increment denormalized counter on the project.
	if _, err := c.projectDoc(uid, pid).Update(ctx, []firestore.Update{
		{Path: "open_milestone_count", Value: firestore.Increment(1)},
		{Path: "updated_at", Value: now},
	}); err != nil {
		return ref.ID, fmt.Errorf("CreateMilestone increment counter: %w", err)
	}

	return ref.ID, nil
}

// ListMilestones returns milestones for a project. Pass status="" to return all;
// pass "open" or "completed" to filter.
func (c *Client) ListMilestones(ctx context.Context, uid, pid string, status string) ([]models.Milestone, error) {
	q := c.milestonesCol(uid, pid).Query
	if status != "" {
		q = q.Where("status", "==", status)
	}
	q = q.OrderBy("created_at", firestore.Asc)

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListMilestones: %w", err)
	}

	milestones := make([]models.Milestone, 0, len(docs))
	for _, doc := range docs {
		var m models.Milestone
		if err := doc.DataTo(&m); err != nil {
			return nil, fmt.Errorf("ListMilestones decode %s: %w", doc.Ref.ID, err)
		}
		m.ID = doc.Ref.ID
		milestones = append(milestones, m)
	}
	return milestones, nil
}

// GetMilestone returns a single milestone by ID.
func (c *Client) GetMilestone(ctx context.Context, uid, pid, mid string) (*models.Milestone, error) {
	doc, err := c.milestonesCol(uid, pid).Doc(mid).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetMilestone %s: %w", mid, err)
	}
	var m models.Milestone
	if err := doc.DataTo(&m); err != nil {
		return nil, fmt.Errorf("GetMilestone decode %s: %w", mid, err)
	}
	m.ID = doc.Ref.ID
	return &m, nil
}

// UpdateMilestone updates editable fields (title, description, due_date) on a milestone.
// Only keys present in updates are modified; status and completion state are unchanged.
func (c *Client) UpdateMilestone(ctx context.Context, uid, pid, mid string, updates map[string]interface{}) error {
	firestoreUpdates := make([]firestore.Update, 0, len(updates))
	for k, v := range updates {
		firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: k, Value: v})
	}
	if len(firestoreUpdates) == 0 {
		return nil
	}
	if _, err := c.milestonesCol(uid, pid).Doc(mid).Update(ctx, firestoreUpdates); err != nil {
		return fmt.Errorf("UpdateMilestone %s: %w", mid, err)
	}
	return nil
}

// CompleteMilestone marks a milestone as completed and decrements the project's open_milestone_count.
func (c *Client) CompleteMilestone(ctx context.Context, uid, pid, mid string) error {
	now := time.Now()

	batch := c.firestore.Batch()

	batch.Update(c.milestonesCol(uid, pid).Doc(mid), []firestore.Update{
		{Path: "status", Value: "completed"},
		{Path: "completed_at", Value: now},
	})

	batch.Update(c.projectDoc(uid, pid), []firestore.Update{
		{Path: "open_milestone_count", Value: firestore.Increment(-1)},
		{Path: "updated_at", Value: now},
	})

	if _, err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("CompleteMilestone %s: %w", mid, err)
	}
	return nil
}
