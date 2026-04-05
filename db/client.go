package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

// Client wraps the Firestore client with scoped helper methods.
type Client struct {
	firestore *firestore.Client
	projectID string
}

// NewClient creates a Firestore client using Application Default Credentials.
// On Cloud Run, ADC resolves automatically. Locally, set GOOGLE_APPLICATION_CREDENTIALS
// or FIRESTORE_EMULATOR_HOST to use the local emulator.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}
	return &Client{firestore: fs, projectID: projectID}, nil
}

// Close releases the underlying Firestore connection.
func (c *Client) Close() error {
	return c.firestore.Close()
}

// Collection reference helpers — all Firestore paths are built here, not in callers.

func (c *Client) usersCol() *firestore.CollectionRef {
	return c.firestore.Collection("users")
}

func (c *Client) userDoc(uid string) *firestore.DocumentRef {
	return c.usersCol().Doc(uid)
}

func (c *Client) projectsCol(uid string) *firestore.CollectionRef {
	return c.userDoc(uid).Collection("projects")
}

func (c *Client) projectDoc(uid, pid string) *firestore.DocumentRef {
	return c.projectsCol(uid).Doc(pid)
}

func (c *Client) sessionsCol(uid, pid string) *firestore.CollectionRef {
	return c.projectDoc(uid, pid).Collection("sessions")
}

func (c *Client) sessionDoc(uid, pid, sid string) *firestore.DocumentRef {
	return c.sessionsCol(uid, pid).Doc(sid)
}

func (c *Client) milestonesCol(uid, pid string) *firestore.CollectionRef {
	return c.projectDoc(uid, pid).Collection("milestones")
}

func (c *Client) decisionsCol(uid, pid string) *firestore.CollectionRef {
	return c.projectDoc(uid, pid).Collection("decisions")
}

func (c *Client) notesCol(uid, pid string) *firestore.CollectionRef {
	return c.projectDoc(uid, pid).Collection("notes")
}

func (c *Client) repoDocsCol(uid, pid string) *firestore.CollectionRef {
	return c.projectDoc(uid, pid).Collection("repo_docs")
}

func (c *Client) toolCallLogCol(uid, pid, sid string) *firestore.CollectionRef {
	return c.sessionDoc(uid, pid, sid).Collection("tool_call_log")
}
