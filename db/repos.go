package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/raghav-anand/briefcase-internal/models"
)

// AddRepo adds a linked repository to a project. Returns the new repo ID.
func (c *Client) AddRepo(ctx context.Context, uid, pid string, r *models.CreateRepoInput) (string, error) {
	now := time.Now()
	ref, _, err := c.reposCol(uid, pid).Add(ctx, map[string]interface{}{
		"name":        r.Name,
		"url":         r.URL,
		"description": r.Description,
		"language":    r.Language,
		"created_at":  now,
		"updated_at":  now,
	})
	if err != nil {
		return "", fmt.Errorf("AddRepo: %w", err)
	}
	return ref.ID, nil
}

// ListRepos returns all linked repositories for a project, ordered by creation time.
func (c *Client) ListRepos(ctx context.Context, uid, pid string) ([]models.Repo, error) {
	docs, err := c.reposCol(uid, pid).OrderBy("created_at", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListRepos: %w", err)
	}

	repos := make([]models.Repo, 0, len(docs))
	for _, doc := range docs {
		var r models.Repo
		if err := doc.DataTo(&r); err != nil {
			return nil, fmt.Errorf("ListRepos decode %s: %w", doc.Ref.ID, err)
		}
		r.ID = doc.Ref.ID
		repos = append(repos, r)
	}
	return repos, nil
}

// GetRepo returns a single linked repository by ID.
func (c *Client) GetRepo(ctx context.Context, uid, pid, rid string) (*models.Repo, error) {
	doc, err := c.reposCol(uid, pid).Doc(rid).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetRepo %s: %w", rid, err)
	}
	var r models.Repo
	if err := doc.DataTo(&r); err != nil {
		return nil, fmt.Errorf("GetRepo decode %s: %w", rid, err)
	}
	r.ID = doc.Ref.ID
	return &r, nil
}

// UpdateRepo updates editable fields (name, url, description, language) on a repo.
// Only keys present in updates are modified.
func (c *Client) UpdateRepo(ctx context.Context, uid, pid, rid string, updates map[string]interface{}) error {
	firestoreUpdates := make([]firestore.Update, 0, len(updates)+1)
	for k, v := range updates {
		firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: k, Value: v})
	}
	firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: "updated_at", Value: time.Now()})

	if _, err := c.reposCol(uid, pid).Doc(rid).Update(ctx, firestoreUpdates); err != nil {
		return fmt.Errorf("UpdateRepo %s: %w", rid, err)
	}
	return nil
}

// RemoveRepo deletes a linked repository from a project.
func (c *Client) RemoveRepo(ctx context.Context, uid, pid, rid string) error {
	if _, err := c.reposCol(uid, pid).Doc(rid).Delete(ctx); err != nil {
		return fmt.Errorf("RemoveRepo %s: %w", rid, err)
	}
	return nil
}
