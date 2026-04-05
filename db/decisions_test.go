package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestDecisions(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Decision Test"})

	// --- CreateDecision ---
	did, err := c.CreateDecision(ctx, uid, pid, &models.CreateDecisionInput{
		Decision:  "Use Firestore over Cloud SQL",
		Rationale: "More generous free tier, no fixed costs",
		SessionID: "sess-1",
		Tags:      []string{"database", "infrastructure"},
	})
	if err != nil {
		t.Fatalf("CreateDecision: %v", err)
	}
	if did == "" {
		t.Fatal("CreateDecision returned empty ID")
	}

	// --- GetDecision ---
	d, err := c.GetDecision(ctx, uid, pid, did)
	if err != nil {
		t.Fatalf("GetDecision: %v", err)
	}
	if d.ID != did {
		t.Errorf("ID: got %q, want %q", d.ID, did)
	}
	if d.Decision != "Use Firestore over Cloud SQL" {
		t.Errorf("Decision: got %q", d.Decision)
	}
	if d.Rationale != "More generous free tier, no fixed costs" {
		t.Errorf("Rationale: got %q", d.Rationale)
	}
	if len(d.Tags) != 2 {
		t.Errorf("Tags len: got %d, want 2", len(d.Tags))
	}
	if d.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	// --- ListDecisions ---
	// Add a second decision.
	_, _ = c.CreateDecision(ctx, uid, pid, &models.CreateDecisionInput{
		Decision:  "Use Cloud Run over GKE",
		SessionID: "sess-1",
	})

	decisions, err := c.ListDecisions(ctx, uid, pid, 0)
	if err != nil {
		t.Fatalf("ListDecisions: %v", err)
	}
	if len(decisions) < 2 {
		t.Errorf("ListDecisions: got %d, want >=2", len(decisions))
	}

	// limit=1 should return only one.
	limited, _ := c.ListDecisions(ctx, uid, pid, 1)
	if len(limited) != 1 {
		t.Errorf("ListDecisions limit=1: got %d, want 1", len(limited))
	}
}
