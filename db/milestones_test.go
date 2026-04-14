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

	// Add a repo so we can validate repo name references.
	_, _ = c.AddRepo(ctx, uid, pid, &models.CreateRepoInput{
		Name: "briefcase-api",
		URL:  "https://github.com/raghav-anand/briefcase-api",
	})

	// --- CreateMilestone (with pre-populated tasks, one with a repo) ---
	mid, err := c.CreateMilestone(ctx, uid, pid, &models.CreateMilestoneInput{
		Title:       "Launch v1",
		Description: "Ship the first version",
		SessionID:   "sess-1",
		Tasks: []models.MilestoneTaskInput{
			{Title: "Write tests", RepoName: "briefcase-api"},
			{Title: "Update docs"},
		},
	})
	if err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}

	// CreateMilestone should increment open_milestone_count and milestone_seq on the project.
	proj, _ := c.GetProject(ctx, uid, pid)
	if proj.OpenMilestoneCount != 1 {
		t.Errorf("open_milestone_count after create: got %d, want 1", proj.OpenMilestoneCount)
	}
	if proj.MilestoneSeq != 1 {
		t.Errorf("milestone_seq after first create: got %d, want 1", proj.MilestoneSeq)
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
	if m.Seq != 1 {
		t.Errorf("Seq: got %d, want 1", m.Seq)
	}
	if len(m.Tasks) != 2 {
		t.Fatalf("Tasks: got %d, want 2", len(m.Tasks))
	}
	if m.Tasks[0].Title != "Write tests" {
		t.Errorf("Tasks[0].Title: got %q, want \"Write tests\"", m.Tasks[0].Title)
	}
	if m.Tasks[0].ID == "" {
		t.Error("Tasks[0].ID should be non-empty")
	}
	if m.Tasks[0].Completed {
		t.Error("Tasks[0].Completed should be false")
	}
	if m.Tasks[0].RepoName != "briefcase-api" {
		t.Errorf("Tasks[0].RepoName: got %q, want \"briefcase-api\"", m.Tasks[0].RepoName)
	}
	if m.Tasks[1].RepoName != "" {
		t.Errorf("Tasks[1].RepoName: got %q, want empty", m.Tasks[1].RepoName)
	}

	// --- Second milestone gets seq=2 ---
	mid2, err := c.CreateMilestone(ctx, uid, pid, &models.CreateMilestoneInput{
		Title:     "Launch v2",
		SessionID: "sess-1",
	})
	if err != nil {
		t.Fatalf("CreateMilestone v2: %v", err)
	}
	m2, _ := c.GetMilestone(ctx, uid, pid, mid2)
	if m2.Seq != 2 {
		t.Errorf("second milestone Seq: got %d, want 2", m2.Seq)
	}
	proj, _ = c.GetProject(ctx, uid, pid)
	if proj.MilestoneSeq != 2 {
		t.Errorf("milestone_seq after second create: got %d, want 2", proj.MilestoneSeq)
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

	// --- AddMilestoneTask (invalid repo name should fail) ---
	_, err = c.AddMilestoneTask(ctx, uid, pid, mid, "Bad task", "nonexistent-repo")
	if err == nil {
		t.Error("AddMilestoneTask with unknown repo name should return error")
	}

	// --- AddMilestoneTask (valid repo name) ---
	taskID, err := c.AddMilestoneTask(ctx, uid, pid, mid, "Deploy to staging", "briefcase-api")
	if err != nil {
		t.Fatalf("AddMilestoneTask: %v", err)
	}
	if taskID == "" {
		t.Error("AddMilestoneTask returned empty ID")
	}
	m, _ = c.GetMilestone(ctx, uid, pid, mid)
	if len(m.Tasks) != 3 {
		t.Fatalf("Tasks after add: got %d, want 3", len(m.Tasks))
	}

	// --- CheckMilestoneTask (complete) ---
	if err := c.CheckMilestoneTask(ctx, uid, pid, mid, taskID, true); err != nil {
		t.Fatalf("CheckMilestoneTask complete: %v", err)
	}
	m, _ = c.GetMilestone(ctx, uid, pid, mid)
	var checkedTask *models.MilestoneTask
	for i := range m.Tasks {
		if m.Tasks[i].ID == taskID {
			checkedTask = &m.Tasks[i]
			break
		}
	}
	if checkedTask == nil {
		t.Fatal("task not found after check")
	}
	if !checkedTask.Completed {
		t.Error("task should be completed")
	}
	if checkedTask.CompletedAt == nil {
		t.Error("completed_at should be set")
	}

	// --- CheckMilestoneTask (uncheck) ---
	if err := c.CheckMilestoneTask(ctx, uid, pid, mid, taskID, false); err != nil {
		t.Fatalf("CheckMilestoneTask uncheck: %v", err)
	}
	m, _ = c.GetMilestone(ctx, uid, pid, mid)
	for i := range m.Tasks {
		if m.Tasks[i].ID == taskID {
			if m.Tasks[i].Completed {
				t.Error("task should be unchecked")
			}
			if m.Tasks[i].CompletedAt != nil {
				t.Error("completed_at should be nil after uncheck")
			}
			break
		}
	}

	// --- RemoveMilestoneTask ---
	if err := c.RemoveMilestoneTask(ctx, uid, pid, mid, taskID); err != nil {
		t.Fatalf("RemoveMilestoneTask: %v", err)
	}
	m, _ = c.GetMilestone(ctx, uid, pid, mid)
	if len(m.Tasks) != 2 {
		t.Errorf("Tasks after remove: got %d, want 2", len(m.Tasks))
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
	if proj.OpenMilestoneCount != 1 {
		t.Errorf("open_milestone_count after complete: got %d, want 1", proj.OpenMilestoneCount)
	}

	// --- UncompleteMilestone ---
	if err := c.UncompleteMilestone(ctx, uid, pid, mid); err != nil {
		t.Fatalf("UncompleteMilestone: %v", err)
	}
	reopened, _ := c.GetMilestone(ctx, uid, pid, mid)
	if reopened.Status != "open" {
		t.Errorf("status after uncomplete: got %q, want open", reopened.Status)
	}
	if reopened.CompletedAt != nil {
		t.Error("completed_at should be cleared after uncomplete")
	}
	proj, _ = c.GetProject(ctx, uid, pid)
	if proj.OpenMilestoneCount != 2 {
		t.Errorf("open_milestone_count after uncomplete: got %d, want 2", proj.OpenMilestoneCount)
	}
}
