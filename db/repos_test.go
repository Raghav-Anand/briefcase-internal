package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestRepos(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Repos Test"})

	// --- AddRepo ---
	rid, err := c.AddRepo(ctx, uid, pid, &models.CreateRepoInput{
		Name:        "briefcase-mcp",
		URL:         "https://github.com/raghav-anand/briefcase-mcp",
		Description: "MCP protocol server",
		Language:    "Go",
	})
	if err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	if rid == "" {
		t.Fatal("AddRepo returned empty ID")
	}

	// --- GetRepo ---
	r, err := c.GetRepo(ctx, uid, pid, rid)
	if err != nil {
		t.Fatalf("GetRepo: %v", err)
	}
	if r.ID != rid {
		t.Errorf("ID: got %q, want %q", r.ID, rid)
	}
	if r.Name != "briefcase-mcp" {
		t.Errorf("Name: got %q, want briefcase-mcp", r.Name)
	}
	if r.Language != "Go" {
		t.Errorf("Language: got %q, want Go", r.Language)
	}

	// --- AddRepo (second) ---
	_, _ = c.AddRepo(ctx, uid, pid, &models.CreateRepoInput{
		Name:     "briefcase-web",
		URL:      "https://github.com/raghav-anand/briefcase-web",
		Language: "TypeScript",
	})

	// --- ListRepos ---
	repos, err := c.ListRepos(ctx, uid, pid)
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("ListRepos: got %d repos, want 2", len(repos))
	}
	// Ordered by created_at asc — first added should be first.
	if repos[0].Name != "briefcase-mcp" {
		t.Errorf("first repo: got %q, want briefcase-mcp", repos[0].Name)
	}

	// --- UpdateRepo ---
	if err := c.UpdateRepo(ctx, uid, pid, rid, map[string]interface{}{
		"description": "MCP protocol server (Go) — deployed to Cloud Run",
	}); err != nil {
		t.Fatalf("UpdateRepo: %v", err)
	}
	updated, _ := c.GetRepo(ctx, uid, pid, rid)
	if updated.Description != "MCP protocol server (Go) — deployed to Cloud Run" {
		t.Errorf("UpdateRepo description: got %q", updated.Description)
	}
	if updated.UpdatedAt.Before(r.UpdatedAt) {
		t.Error("UpdateRepo: updated_at was not advanced")
	}

	// --- RemoveRepo ---
	if err := c.RemoveRepo(ctx, uid, pid, rid); err != nil {
		t.Fatalf("RemoveRepo: %v", err)
	}
	remaining, _ := c.ListRepos(ctx, uid, pid)
	if len(remaining) != 1 {
		t.Errorf("after RemoveRepo: got %d repos, want 1", len(remaining))
	}
	if remaining[0].Name != "briefcase-web" {
		t.Errorf("remaining repo: got %q, want briefcase-web", remaining[0].Name)
	}
}
