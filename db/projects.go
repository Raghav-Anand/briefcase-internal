package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/raghav-anand/briefcase-internal/models"
)

// CreateProject creates a new project under the user's collection. Returns the new project ID.
func (c *Client) CreateProject(ctx context.Context, uid string, p *models.CreateProjectInput) (string, error) {
	now := time.Now()
	ref, _, err := c.projectsCol(uid).Add(ctx, map[string]interface{}{
		"name":                  p.Name,
		"description":           p.Description,
		"status":                "active",
		"repo_url":              p.RepoURL,
		"tech_stack":            p.TechStack,
		"open_milestone_count":  0,
		"milestone_seq":         0,
		"created_at":            now,
		"updated_at":            now,
	})
	if err != nil {
		return "", fmt.Errorf("CreateProject: %w", err)
	}
	return ref.ID, nil
}

// ListProjects returns all projects for a user. Pass includeArchived=false to exclude archived projects.
func (c *Client) ListProjects(ctx context.Context, uid string, includeArchived bool) ([]models.Project, error) {
	col := c.projectsCol(uid)

	var q firestore.Query
	if includeArchived {
		q = col.OrderBy("updated_at", firestore.Desc)
	} else {
		q = col.Where("status", "in", []string{"active", "paused", "completed"}).
			OrderBy("updated_at", firestore.Desc)
	}

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListProjects: %w", err)
	}

	projects := make([]models.Project, 0, len(docs))
	for _, doc := range docs {
		var p models.Project
		if err := doc.DataTo(&p); err != nil {
			return nil, fmt.Errorf("ListProjects decode %s: %w", doc.Ref.ID, err)
		}
		p.ID = doc.Ref.ID
		projects = append(projects, p)
	}
	return projects, nil
}

// GetProject returns a single project by ID.
func (c *Client) GetProject(ctx context.Context, uid, pid string) (*models.Project, error) {
	doc, err := c.projectDoc(uid, pid).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetProject %s: %w", pid, err)
	}
	var p models.Project
	if err := doc.DataTo(&p); err != nil {
		return nil, fmt.Errorf("GetProject decode %s: %w", pid, err)
	}
	p.ID = doc.Ref.ID
	return &p, nil
}

// UpdateProject applies a partial update to a project. The updates map uses Firestore field paths as keys.
func (c *Client) UpdateProject(ctx context.Context, uid, pid string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	firestoreUpdates := make([]firestore.Update, 0, len(updates))
	for k, v := range updates {
		firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: k, Value: v})
	}

	if _, err := c.projectDoc(uid, pid).Update(ctx, firestoreUpdates); err != nil {
		return fmt.Errorf("UpdateProject %s: %w", pid, err)
	}
	return nil
}

// ArchiveProject sets a project's status to "archived".
func (c *Client) ArchiveProject(ctx context.Context, uid, pid string) error {
	return c.UpdateProject(ctx, uid, pid, map[string]interface{}{"status": "archived"})
}
