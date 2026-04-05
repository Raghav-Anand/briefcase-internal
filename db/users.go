package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/raghav-anand/briefcase-internal/auth"
)

// UpsertUser creates a user document if one doesn't exist, or updates last_login on subsequent calls.
// Called on first API request or MCP connection after auth.
func (c *Client) UpsertUser(ctx context.Context, claims *auth.Claims) error {
	ref := c.userDoc(claims.UID)
	now := time.Now()

	doc, err := ref.Get(ctx)
	if err != nil || !doc.Exists() {
		// First sign-in: create the user document.
		if _, err := ref.Set(ctx, map[string]interface{}{
			"email":        claims.Email,
			"display_name": claims.Name,
			"photo_url":    claims.Picture,
			"created_at":   now,
			"updated_at":   now,
		}); err != nil {
			return fmt.Errorf("UpsertUser create %s: %w", claims.UID, err)
		}
		return nil
	}

	// Subsequent sign-ins: update mutable profile fields and last-seen timestamp.
	if _, err := ref.Update(ctx, []firestore.Update{
		{Path: "email", Value: claims.Email},
		{Path: "display_name", Value: claims.Name},
		{Path: "photo_url", Value: claims.Picture},
		{Path: "updated_at", Value: now},
	}); err != nil {
		return fmt.Errorf("UpsertUser update %s: %w", claims.UID, err)
	}
	return nil
}
