package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/raghav-anand/briefcase-internal/models"
)

// LogToolCall records a tool invocation for crash recovery.
// Also increments the session's tool_call_count.
func (c *Client) LogToolCall(ctx context.Context, uid, pid, sid string, entry *models.ToolCallEntry) error {
	now := time.Now()

	batch := c.firestore.Batch()

	logRef := c.toolCallLogCol(uid, pid, sid).NewDoc()
	batch.Set(logRef, map[string]interface{}{
		"tool_name": entry.ToolName,
		"args":      entry.Args,
		"result":    entry.Result,
		"called_at": now,
	})

	batch.Update(c.sessionDoc(uid, pid, sid), []firestore.Update{
		{Path: "tool_call_count", Value: firestore.Increment(1)},
	})

	if _, err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("LogToolCall: %w", err)
	}
	return nil
}

// GetToolCallLog returns all tool calls for a session, ordered by time ascending.
func (c *Client) GetToolCallLog(ctx context.Context, uid, pid, sid string) ([]models.ToolCallEntry, error) {
	docs, err := c.toolCallLogCol(uid, pid, sid).
		OrderBy("called_at", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("GetToolCallLog: %w", err)
	}

	entries := make([]models.ToolCallEntry, 0, len(docs))
	for _, doc := range docs {
		var e models.ToolCallEntry
		if err := doc.DataTo(&e); err != nil {
			return nil, fmt.Errorf("GetToolCallLog decode %s: %w", doc.Ref.ID, err)
		}
		e.ID = doc.Ref.ID
		entries = append(entries, e)
	}
	return entries, nil
}
