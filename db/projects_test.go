package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestProjects(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	// --- CreateProject ---
	pid, err := c.CreateProject(ctx, uid, &models.CreateProjectInput{
		Name:        "Test Project",
		Description: "A project for testing",
		TechStack:   []string{"Go", "Firestore"},
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if pid == "" {
		t.Fatal("CreateProject returned empty ID")
	}

	// --- GetProject ---
	proj, err := c.GetProject(ctx, uid, pid)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if proj.ID != pid {
		t.Errorf("ID: got %q, want %q", proj.ID, pid)
	}
	if proj.Name != "Test Project" {
		t.Errorf("Name: got %q, want %q", proj.Name, "Test Project")
	}
	if proj.Status != "active" {
		t.Errorf("Status: got %q, want %q", proj.Status, "active")
	}
	if proj.OpenMilestoneCount != 0 {
		t.Errorf("OpenMilestoneCount: got %d, want 0", proj.OpenMilestoneCount)
	}
	if proj.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	// --- ListProjects (active only) ---
	projects, err := c.ListProjects(ctx, uid, false)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) == 0 {
		t.Fatal("ListProjects returned empty slice")
	}
	found := false
	for _, p := range projects {
		if p.ID == pid {
			found = true
		}
	}
	if !found {
		t.Errorf("ListProjects did not include project %q", pid)
	}

	// --- UpdateProject ---
	if err := c.UpdateProject(ctx, uid, pid, map[string]interface{}{
		"description": "Updated description",
		"status":      "paused",
	}); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	updated, _ := c.GetProject(ctx, uid, pid)
	if updated.Description != "Updated description" {
		t.Errorf("Description after update: got %q", updated.Description)
	}
	if updated.Status != "paused" {
		t.Errorf("Status after update: got %q", updated.Status)
	}

	// --- ArchiveProject ---
	if err := c.ArchiveProject(ctx, uid, pid); err != nil {
		t.Fatalf("ArchiveProject: %v", err)
	}
	archived, _ := c.GetProject(ctx, uid, pid)
	if archived.Status != "archived" {
		t.Errorf("Status after archive: got %q, want archived", archived.Status)
	}

	// Archived project should not appear when includeArchived=false.
	active, _ := c.ListProjects(ctx, uid, false)
	for _, p := range active {
		if p.ID == pid {
			t.Error("archived project appeared in non-archived list")
		}
	}

	// Should appear when includeArchived=true.
	all, _ := c.ListProjects(ctx, uid, true)
	found = false
	for _, p := range all {
		if p.ID == pid {
			found = true
		}
	}
	if !found {
		t.Error("archived project missing from includeArchived=true list")
	}
}

func TestGetProject_NotFound(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()

	_, err := c.GetProject(ctx, "nonexistent-uid", "nonexistent-pid")
	if err == nil {
		t.Error("expected error for missing project, got nil")
	}
}
