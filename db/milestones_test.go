package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestMilestones(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Milestone Test"})

	// --- CreateMilestone ---
	mid, err := c.CreateMilestone(ctx, uid, pid, &models.CreateMilestoneInput{
		Title:       "Launch v1",
		Description: "Ship the first version",
		SessionID:   "sess-1",
	})
	if err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}

	// CreateMilestone should increment open_milestone_count on the project.
	proj, _ := c.GetProject(ctx, uid, pid)
	if proj.OpenMilestoneCount != 1 {
		t.Errorf("open_milestone_count after create: got %d, want 1", proj.OpenMilestoneCount)
	}

	// --- GetMilestone ---
	m, err := c.GetMilestone(ctx, uid, pid, mid)
	if err != nil {
		t.Fatalf("GetMilestone: %v", err)
	}
	if m.ID != mid {
		t.Errorf("ID: got %q, want %q", m.ID, mid)
	}
	if m.Title != "Launch v1" {
		t.Errorf("Title: got %q", m.Title)
	}
	if m.Status != "open" {
		t.Errorf("Status: got %q, want open", m.Status)
	}
	if m.CompletedAt != nil {
		t.Error("CompletedAt should be nil for open milestone")
	}

	// --- ListMilestones (all) ---
	all, err := c.ListMilestones(ctx, uid, pid, "")
	if err != nil {
		t.Fatalf("ListMilestones: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("ListMilestones returned empty")
	}

	// Filter by status.
	open, _ := c.ListMilestones(ctx, uid, pid, "open")
	if len(open) == 0 {
		t.Error("open milestones should not be empty")
	}
	completed, _ := c.ListMilestones(ctx, uid, pid, "completed")
	if len(completed) != 0 {
		t.Error("no completed milestones expected yet")
	}

	// --- CompleteMilestone ---
	if err := c.CompleteMilestone(ctx, uid, pid, mid); err != nil {
		t.Fatalf("CompleteMilestone: %v", err)
	}

	done, _ := c.GetMilestone(ctx, uid, pid, mid)
	if done.Status != "completed" {
		t.Errorf("status after complete: got %q, want completed", done.Status)
	}
	if done.CompletedAt == nil {
		t.Error("completed_at should be set")
	}

	// open_milestone_count should be decremented.
	proj, _ = c.GetProject(ctx, uid, pid)
	if proj.OpenMilestoneCount != 0 {
		t.Errorf("open_milestone_count after complete: got %d, want 0", proj.OpenMilestoneCount)
	}
}
