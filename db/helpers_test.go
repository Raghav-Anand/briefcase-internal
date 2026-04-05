package db_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/raghav-anand/briefcase-internal/db"
)

// requireEmulator skips the test if FIRESTORE_EMULATOR_HOST is not set.
// Run: export FIRESTORE_EMULATOR_HOST=localhost:8080 before testing.
func requireEmulator(t *testing.T) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set; skipping integration test (see README for setup)")
	}
}

// newTestClient creates a db.Client connected to the Firestore emulator.
func newTestClient(t *testing.T) *db.Client {
	t.Helper()
	ctx := context.Background()
	c, err := db.NewClient(ctx, "test-project")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// testUID returns a stable, test-scoped user ID derived from the test name.
// Using t.Name() ensures tests don't collide with each other within a run.
func testUID(t *testing.T) string {
	t.Helper()
	// Sanitise slashes from subtests.
	return fmt.Sprintf("uid-%s", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()))
}
