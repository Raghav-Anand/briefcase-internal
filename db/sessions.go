package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/raghav-anand/briefcase-internal/models"
)

// CreateSession creates a new session for a project. Auto-closes any currently active session first.
// Returns the new session ID.
func (c *Client) CreateSession(ctx context.Context, uid, pid string, clientType string) (string, error) {
	// Auto-close any stale active session before creating a new one.
	active, err := c.GetActiveSession(ctx, uid, pid)
	if err == nil && active != nil {
		_ = c.AutoCloseSession(ctx, uid, pid, active.ID, "Session auto-closed: new session started.")
	}

	now := time.Now()
	ref, _, err := c.sessionsCol(uid, pid).Add(ctx, map[string]interface{}{
		"status":          "active",
		"started_at":      now,
		"tool_call_count": 0,
		"client_type":     clientType,
		"auto_closed":     false,
		"created_at":      now,
	})
	if err != nil {
		return "", fmt.Errorf("CreateSession: %w", err)
	}
	return ref.ID, nil
}

// GetActiveSession returns the currently active session for a project, or nil if none.
func (c *Client) GetActiveSession(ctx context.Context, uid, pid string) (*models.Session, error) {
	docs, err := c.sessionsCol(uid, pid).
		Where("status", "==", "active").
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("GetActiveSession: %w", err)
	}
	if len(docs) == 0 {
		return nil, nil
	}
	var s models.Session
	if err := docs[0].DataTo(&s); err != nil {
		return nil, fmt.Errorf("GetActiveSession decode: %w", err)
	}
	s.ID = docs[0].Ref.ID
	return &s, nil
}

// EndSession marks a session as completed. Decisions are assembled automatically from
// the decisions subcollection for this session.
func (c *Client) EndSession(ctx context.Context, uid, pid, sid string, summary string, nextSteps []string) error {
	// Collect decision IDs created during this session.
	decisionDocs, err := c.decisionsCol(uid, pid).
		Where("session_id", "==", sid).
		Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("EndSession query decisions: %w", err)
	}
	decisionIDs := make([]string, 0, len(decisionDocs))
	for _, d := range decisionDocs {
		decisionIDs = append(decisionIDs, d.Ref.ID)
	}

	now := time.Now()
	batch := c.firestore.Batch()

	batch.Update(c.sessionDoc(uid, pid, sid), []firestore.Update{
		{Path: "status", Value: "completed"},
		{Path: "ended_at", Value: now},
		{Path: "summary", Value: summary},
		{Path: "next_steps", Value: nextSteps},
		{Path: "decisions", Value: decisionIDs},
	})

	batch.Update(c.projectDoc(uid, pid), []firestore.Update{
		{Path: "last_session_id", Value: sid},
		{Path: "last_session_summary", Value: summary},
		{Path: "updated_at", Value: now},
	})

	if _, err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("EndSession commit: %w", err)
	}
	return nil
}

// AutoCloseSession marks a session as auto-closed with a server-generated summary.
// Used by the crash recovery routine in the MCP server.
func (c *Client) AutoCloseSession(ctx context.Context, uid, pid, sid string, summary string) error {
	now := time.Now()
	if _, err := c.sessionDoc(uid, pid, sid).Update(ctx, []firestore.Update{
		{Path: "status", Value: "auto_closed"},
		{Path: "ended_at", Value: now},
		{Path: "summary", Value: summary},
		{Path: "auto_closed", Value: true},
	}); err != nil {
		return fmt.Errorf("AutoCloseSession %s: %w", sid, err)
	}
	return nil
}

// ListSessions returns sessions for a project in reverse chronological order.
// Pass a non-nil before to paginate (returns sessions started before that time).
func (c *Client) ListSessions(ctx context.Context, uid, pid string, limit int, before *time.Time) ([]models.Session, error) {
	q := c.sessionsCol(uid, pid).OrderBy("started_at", firestore.Desc).Limit(limit)
	if before != nil {
		q = q.StartAfter(*before)
	}

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListSessions: %w", err)
	}

	sessions := make([]models.Session, 0, len(docs))
	for _, doc := range docs {
		var s models.Session
		if err := doc.DataTo(&s); err != nil {
			return nil, fmt.Errorf("ListSessions decode %s: %w", doc.Ref.ID, err)
		}
		s.ID = doc.Ref.ID
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// GetSession returns a single session with full details.
func (c *Client) GetSession(ctx context.Context, uid, pid, sid string) (*models.Session, error) {
	doc, err := c.sessionDoc(uid, pid, sid).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetSession %s: %w", sid, err)
	}
	var s models.Session
	if err := doc.DataTo(&s); err != nil {
		return nil, fmt.Errorf("GetSession decode %s: %w", sid, err)
	}
	s.ID = doc.Ref.ID
	return &s, nil
}

// FindStaleSessions returns active sessions that have been open longer than maxAge.
// Uses a collection group query across all users. The returned StaleSession includes
// the UserID and ProjectID extracted from the document path.
// Requires a composite index on the "sessions" collection group: status ASC, started_at ASC.
func (c *Client) FindStaleSessions(ctx context.Context, maxAge time.Duration) ([]models.StaleSession, error) {
	cutoff := time.Now().Add(-maxAge)

	iter := c.firestore.CollectionGroup("sessions").
		Where("status", "==", "active").
		Where("started_at", "<", cutoff).
		Documents(ctx)
	defer iter.Stop()

	var results []models.StaleSession
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("FindStaleSessions: %w", err)
		}

		var s models.Session
		if err := doc.DataTo(&s); err != nil {
			continue
		}
		s.ID = doc.Ref.ID

		// Path structure: users/{uid}/projects/{pid}/sessions/{sid}
		// Traverse up: session → sessions col → project → projects col → user
		projectRef := doc.Ref.Parent.Parent
		userRef := projectRef.Parent.Parent

		results = append(results, models.StaleSession{
			Session:   s,
			ProjectID: projectRef.ID,
			UserID:    userRef.ID,
		})
	}
	return results, nil
}
