package db_test

import (
	"context"
	"testing"

	"github.com/raghav-anand/briefcase-internal/auth"
)

func TestUpsertUser(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()

	claims := &auth.Claims{
		UID:     testUID(t),
		Email:   "test@example.com",
		Name:    "Test User",
		Picture: "https://example.com/photo.jpg",
	}

	// --- First call: creates the user document ---
	if err := c.UpsertUser(ctx, claims); err != nil {
		t.Fatalf("UpsertUser (create): %v", err)
	}

	// --- Second call: updates the user document ---
	claims.Name = "Updated Name"
	if err := c.UpsertUser(ctx, claims); err != nil {
		t.Fatalf("UpsertUser (update): %v", err)
	}

	// Both calls should succeed without error.
	// The document itself is not directly readable via this library's public API
	// (users are internal to the db layer), but no error is sufficient verification here.
}
