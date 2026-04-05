package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/raghav-anand/briefcase-internal/models"
)


func TestSessions(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	// Need a project to attach sessions to.
	pid, err := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Session Test Project"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// --- No active session initially ---
	active, err := c.GetActiveSession(ctx, uid, pid)
	if err != nil {
		t.Fatalf("GetActiveSession (empty): %v", err)
	}
	if active != nil {
		t.Error("expected no active session, got one")
	}

	// --- CreateSession ---
	sid, err := c.CreateSession(ctx, uid, pid, "claude_code")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sid == "" {
		t.Fatal("CreateSession returned empty ID")
	}

	// --- GetActiveSession ---
	active, err = c.GetActiveSession(ctx, uid, pid)
	if err != nil {
		t.Fatalf("GetActiveSession: %v", err)
	}
	if active == nil {
		t.Fatal("expected active session, got nil")
	}
	if active.ID != sid {
		t.Errorf("active session ID: got %q, want %q", active.ID, sid)
	}
	if active.Status != "active" {
		t.Errorf("active session status: got %q, want active", active.Status)
	}
	if active.ClientType != "claude_code" {
		t.Errorf("client_type: got %q, want claude_code", active.ClientType)
	}

	// Add a decision so EndSession can assemble it.
	did, err := c.CreateDecision(ctx, uid, pid, &models.CreateDecisionInput{
		Decision:  "Use Firestore",
		Rationale: "Free tier",
		SessionID: sid,
	})
	if err != nil {
		t.Fatalf("CreateDecision (for EndSession test): %v", err)
	}

	// --- EndSession ---
	nextSteps := []string{"write tests", "deploy"}
	if err := c.EndSession(ctx, uid, pid, sid, "Implemented the thing", nextSteps); err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	// Session should now be completed.
	completed, err := c.GetSession(ctx, uid, pid, sid)
	if err != nil {
		t.Fatalf("GetSession after end: %v", err)
	}
	if completed.Status != "completed" {
		t.Errorf("status after end: got %q, want completed", completed.Status)
	}
	if completed.Summary != "Implemented the thing" {
		t.Errorf("summary: got %q", completed.Summary)
	}
	if len(completed.NextSteps) != 2 {
		t.Errorf("next_steps len: got %d, want 2", len(completed.NextSteps))
	}
	if len(completed.Decisions) != 1 || completed.Decisions[0] != did {
		t.Errorf("decisions: got %v, want [%s]", completed.Decisions, did)
	}
	if completed.EndedAt == nil {
		t.Error("ended_at should be set")
	}

	// Project should have last_session_id / last_session_summary denormalized.
	proj, _ := c.GetProject(ctx, uid, pid)
	if proj.LastSessionID != sid {
		t.Errorf("project.last_session_id: got %q, want %q", proj.LastSessionID, sid)
	}
	if proj.LastSessionSummary != "Implemented the thing" {
		t.Errorf("project.last_session_summary: got %q", proj.LastSessionSummary)
	}

	// No active session after end.
	active, _ = c.GetActiveSession(ctx, uid, pid)
	if active != nil {
		t.Error("expected no active session after EndSession")
	}

	// --- ListSessions ---
	sessions, err := c.ListSessions(ctx, uid, pid, 10, nil)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("ListSessions returned empty")
	}
	if sessions[0].ID != sid {
		t.Errorf("ListSessions first item: got %q, want %q", sessions[0].ID, sid)
	}
}

func TestSession_AutoClose(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "AutoClose Test"})

	sid, err := c.CreateSession(ctx, uid, pid, "claude_desktop")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := c.AutoCloseSession(ctx, uid, pid, sid, "Timed out"); err != nil {
		t.Fatalf("AutoCloseSession: %v", err)
	}

	s, _ := c.GetSession(ctx, uid, pid, sid)
	if s.Status != "auto_closed" {
		t.Errorf("status: got %q, want auto_closed", s.Status)
	}
	if !s.AutoClosed {
		t.Error("auto_closed flag should be true")
	}
	if s.Summary != "Timed out" {
		t.Errorf("summary: got %q", s.Summary)
	}
}

func TestSession_NewSessionAutoClosesStale(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Stale Test"})

	// Create first session.
	sid1, _ := c.CreateSession(ctx, uid, pid, "claude_code")

	// Creating a second session should auto-close the first.
	sid2, err := c.CreateSession(ctx, uid, pid, "claude_code")
	if err != nil {
		t.Fatalf("CreateSession (second): %v", err)
	}
	if sid1 == sid2 {
		t.Fatal("expected distinct session IDs")
	}

	stale, _ := c.GetSession(ctx, uid, pid, sid1)
	if stale.Status != "auto_closed" {
		t.Errorf("first session status: got %q, want auto_closed", stale.Status)
	}

	active, _ := c.GetActiveSession(ctx, uid, pid)
	if active == nil || active.ID != sid2 {
		t.Error("second session should be the active one")
	}
}

func TestFindStaleSessions_NoResults(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Stale Query Test"})
	// Create a fresh session.
	_, _ = c.CreateSession(ctx, uid, pid, "claude_code")

	// maxAge of 0 would match everything, but use a short positive duration
	// to test the query path. A session created just now should not be stale
	// relative to a 1-hour threshold.
	stale, err := c.FindStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("FindStaleSessions: %v", err)
	}
	for _, s := range stale {
		if s.UserID == uid {
			t.Errorf("fresh session from uid %q should not be stale", uid)
		}
	}
}
