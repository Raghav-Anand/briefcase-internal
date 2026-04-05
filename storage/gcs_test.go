package storage_test

import (
	"context"
	"os"
	"testing"

	gcs "cloud.google.com/go/storage"

	"github.com/raghav-anand/briefcase-internal/storage"
)

func testBucket() string {
	if b := os.Getenv("GCS_TEST_BUCKET"); b != "" {
		return b
	}
	return "test-bucket"
}

// TestMain creates the test bucket on the emulator before any tests run.
// fake-gcs-server starts empty, so the bucket must be created explicitly.
// The GCS SDK picks up STORAGE_EMULATOR_HOST natively.
func TestMain(m *testing.M) {
	if os.Getenv("STORAGE_EMULATOR_HOST") != "" {
		ctx := context.Background()
		client, err := gcs.NewClient(ctx)
		if err == nil {
			// Ignore error — bucket may already exist from a previous run.
			_ = client.Bucket(testBucket()).Create(ctx, "test-project", nil)
			_ = client.Close()
		}
	}
	os.Exit(m.Run())
}

// requireGCSEmulator skips the test if STORAGE_EMULATOR_HOST is not set.
// Use fake-gcs-server: https://github.com/fsouza/fake-gcs-server
// Run: docker run -p 4443:4443 fsouza/fake-gcs-server -scheme http -port 4443
// Then: export STORAGE_EMULATOR_HOST=localhost:4443
func requireGCSEmulator(t *testing.T) {
	t.Helper()
	if os.Getenv("STORAGE_EMULATOR_HOST") == "" {
		t.Skip("STORAGE_EMULATOR_HOST not set; skipping GCS integration test (see README for setup)")
	}
}

func TestGCSClient(t *testing.T) {
	requireGCSEmulator(t)
	ctx := context.Background()

	client, err := storage.NewGCSClient(ctx, testBucket())
	if err != nil {
		t.Fatalf("NewGCSClient: %v", err)
	}

	path := "docs/test-user/test-project/test-doc.md"
	content := []byte("# Test Document\n\nHello, world.")

	// --- Upload ---
	if err := client.Upload(ctx, path, content); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	// --- Download ---
	got, err := client.Download(ctx, path)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("Download content mismatch:\n got: %q\nwant: %q", got, content)
	}

	// --- Overwrite ---
	newContent := []byte("# Updated\n\nOverwritten.")
	if err := client.Upload(ctx, path, newContent); err != nil {
		t.Fatalf("Upload (overwrite): %v", err)
	}
	got, _ = client.Download(ctx, path)
	if string(got) != string(newContent) {
		t.Errorf("after overwrite, got %q, want %q", got, newContent)
	}

	// --- Delete ---
	if err := client.Delete(ctx, path); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Download after delete should fail.
	_, err = client.Download(ctx, path)
	if err == nil {
		t.Error("expected error downloading deleted object, got nil")
	}
}

func TestGCSClient_DownloadMissing(t *testing.T) {
	requireGCSEmulator(t)
	ctx := context.Background()

	client, err := storage.NewGCSClient(ctx, testBucket())
	if err != nil {
		t.Fatalf("NewGCSClient: %v", err)
	}

	_, err = client.Download(ctx, "does/not/exist.md")
	if err == nil {
		t.Error("expected error for missing object, got nil")
	}
}
