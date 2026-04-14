package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/raghav-anand/briefcase-internal/models"
)

// validRepoName returns an error if repoName is non-empty and does not match any repo on the project.
func (c *Client) validRepoName(ctx context.Context, uid, pid, repoName string) error {
	if repoName == "" {
		return nil
	}
	repos, err := c.ListRepos(ctx, uid, pid)
	if err != nil {
		return fmt.Errorf("validRepoName: %w", err)
	}
	for _, r := range repos {
		if r.Name == repoName {
			return nil
		}
	}
	return fmt.Errorf("repo %q not found on project", repoName)
}

// CreateMilestone creates a new milestone for a project.
// It atomically increments the project's milestone_seq counter (used as the human-readable
// sequential identifier, e.g. M-1, M-2) and the open_milestone_count.
// Returns the new milestone ID.
func (c *Client) CreateMilestone(ctx context.Context, uid, pid string, m *models.CreateMilestoneInput) (string, error) {
	now := time.Now()

	// Validate repo names and build initial tasks.
	tasks := make([]models.MilestoneTask, 0, len(m.Tasks))
	for _, t := range m.Tasks {
		if err := c.validRepoName(ctx, uid, pid, t.RepoName); err != nil {
			return "", fmt.Errorf("CreateMilestone task %q: %w", t.Title, err)
		}
		tasks = append(tasks, models.MilestoneTask{
			ID:        uuid.New().String(),
			Title:     t.Title,
			RepoName:  t.RepoName,
			Completed: false,
		})
	}

	var newID string
	err := c.firestore.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		projSnap, err := tx.Get(c.projectDoc(uid, pid))
		if err != nil {
			return err
		}

		// Read current seq counter; default to 0 if field is missing on older projects.
		var seq int64
		if v, err := projSnap.DataAt("milestone_seq"); err == nil {
			if n, ok := v.(int64); ok {
				seq = n
			}
		}
		seq++

		milestoneRef := c.milestonesCol(uid, pid).NewDoc()
		newID = milestoneRef.ID

		milestoneData := map[string]interface{}{
			"seq":         seq,
			"title":       m.Title,
			"description": m.Description,
			"tasks":       tasks,
			"status":      "open",
			"session_id":  m.SessionID,
			"created_at":  now,
		}
		if m.DueDate != nil {
			milestoneData["due_date"] = m.DueDate
		}

		if err := tx.Create(milestoneRef, milestoneData); err != nil {
			return err
		}

		return tx.Update(c.projectDoc(uid, pid), []firestore.Update{
			{Path: "milestone_seq", Value: seq},
			{Path: "open_milestone_count", Value: firestore.Increment(1)},
			{Path: "updated_at", Value: now},
		})
	})
	if err != nil {
		return "", fmt.Errorf("CreateMilestone: %w", err)
	}
	return newID, nil
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

// UncompleteMilestone re-opens a completed milestone and increments the project's open_milestone_count.
func (c *Client) UncompleteMilestone(ctx context.Context, uid, pid, mid string) error {
	now := time.Now()

	batch := c.firestore.Batch()

	batch.Update(c.milestonesCol(uid, pid).Doc(mid), []firestore.Update{
		{Path: "status", Value: "open"},
		{Path: "completed_at", Value: firestore.Delete},
	})

	batch.Update(c.projectDoc(uid, pid), []firestore.Update{
		{Path: "open_milestone_count", Value: firestore.Increment(1)},
		{Path: "updated_at", Value: now},
	})

	if _, err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("UncompleteMilestone %s: %w", mid, err)
	}
	return nil
}

// AddMilestoneTask appends a new task to a milestone's task list.
// repoName is optional; if non-empty it must match an existing repo name on the project.
// Returns the new task's ID.
func (c *Client) AddMilestoneTask(ctx context.Context, uid, pid, mid, title, repoName string) (string, error) {
	if err := c.validRepoName(ctx, uid, pid, repoName); err != nil {
		return "", fmt.Errorf("AddMilestoneTask: %w", err)
	}

	taskID := uuid.New().String()

	var newTasks []models.MilestoneTask
	err := c.firestore.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(c.milestonesCol(uid, pid).Doc(mid))
		if err != nil {
			return err
		}
		var m models.Milestone
		if err := snap.DataTo(&m); err != nil {
			return err
		}
		newTasks = append(m.Tasks, models.MilestoneTask{
			ID:        taskID,
			Title:     title,
			RepoName:  repoName,
			Completed: false,
		})
		return tx.Update(c.milestonesCol(uid, pid).Doc(mid), []firestore.Update{
			{Path: "tasks", Value: newTasks},
		})
	})
	if err != nil {
		return "", fmt.Errorf("AddMilestoneTask %s: %w", mid, err)
	}
	return taskID, nil
}

// CheckMilestoneTask marks a task as completed or incomplete.
// completed=true sets completed_at to now; completed=false clears it.
func (c *Client) CheckMilestoneTask(ctx context.Context, uid, pid, mid, taskID string, completed bool) error {
	now := time.Now()
	err := c.firestore.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(c.milestonesCol(uid, pid).Doc(mid))
		if err != nil {
			return err
		}
		var m models.Milestone
		if err := snap.DataTo(&m); err != nil {
			return err
		}
		found := false
		for i := range m.Tasks {
			if m.Tasks[i].ID == taskID {
				m.Tasks[i].Completed = completed
				if completed {
					m.Tasks[i].CompletedAt = &now
				} else {
					m.Tasks[i].CompletedAt = nil
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("task %s not found in milestone %s", taskID, mid)
		}
		return tx.Update(c.milestonesCol(uid, pid).Doc(mid), []firestore.Update{
			{Path: "tasks", Value: m.Tasks},
		})
	})
	if err != nil {
		return fmt.Errorf("CheckMilestoneTask %s/%s: %w", mid, taskID, err)
	}
	return nil
}

// RemoveMilestoneTask removes a task from a milestone by task ID.
func (c *Client) RemoveMilestoneTask(ctx context.Context, uid, pid, mid, taskID string) error {
	err := c.firestore.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(c.milestonesCol(uid, pid).Doc(mid))
		if err != nil {
			return err
		}
		var m models.Milestone
		if err := snap.DataTo(&m); err != nil {
			return err
		}
		filtered := make([]models.MilestoneTask, 0, len(m.Tasks))
		for _, task := range m.Tasks {
			if task.ID != taskID {
				filtered = append(filtered, task)
			}
		}
		return tx.Update(c.milestonesCol(uid, pid).Doc(mid), []firestore.Update{
			{Path: "tasks", Value: filtered},
		})
	})
	if err != nil {
		return fmt.Errorf("RemoveMilestoneTask %s/%s: %w", mid, taskID, err)
	}
	return nil
}
