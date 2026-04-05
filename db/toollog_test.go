package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestToolLog(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "ToolLog Test"})
	sid, _ := c.CreateSession(ctx, uid, pid, "claude_code")

	// --- LogToolCall ---
	entry := &models.ToolCallEntry{
		ToolName: "add_note",
		Args:     map[string]interface{}{"content": "hello", "note_type": "general"},
		Result:   "ok",
	}
	if err := c.LogToolCall(ctx, uid, pid, sid, entry); err != nil {
		t.Fatalf("LogToolCall: %v", err)
	}

	// tool_call_count on the session should have been incremented.
	s, _ := c.GetSession(ctx, uid, pid, sid)
	if s.ToolCallCount != 1 {
		t.Errorf("tool_call_count after first log: got %d, want 1", s.ToolCallCount)
	}

	// Log a second call.
	if err := c.LogToolCall(ctx, uid, pid, sid, &models.ToolCallEntry{
		ToolName: "log_decision",
		Args:     map[string]interface{}{"decision": "use Go"},
	}); err != nil {
		t.Fatalf("LogToolCall (second): %v", err)
	}

	s, _ = c.GetSession(ctx, uid, pid, sid)
	if s.ToolCallCount != 2 {
		t.Errorf("tool_call_count after second log: got %d, want 2", s.ToolCallCount)
	}

	// --- GetToolCallLog ---
	log, err := c.GetToolCallLog(ctx, uid, pid, sid)
	if err != nil {
		t.Fatalf("GetToolCallLog: %v", err)
	}
	if len(log) != 2 {
		t.Fatalf("GetToolCallLog: got %d entries, want 2", len(log))
	}
	if log[0].ToolName != "add_note" {
		t.Errorf("first entry tool_name: got %q, want add_note", log[0].ToolName)
	}
	if log[1].ToolName != "log_decision" {
		t.Errorf("second entry tool_name: got %q, want log_decision", log[1].ToolName)
	}
	// Verify ordering: first call should appear before second.
	if !log[0].CalledAt.Before(log[1].CalledAt) && log[0].CalledAt != log[1].CalledAt {
		t.Error("log entries are not in ascending order by called_at")
	}
}
