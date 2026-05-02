package db_test

import (
	"context"
	"strings"
	"testing"

	"github.com/raghav-anand/briefcase-internal/models"
)

func TestDocs_Inline(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Docs Test"})

	// --- UpsertDoc (create, inline) ---
	input := &models.DocInput{
		Title:     "Architecture Overview",
		DocType:   "architecture",
		Format:    "mermaid",
		Content:   "graph TD; A-->B",
		UpdatedBy: "claude",
		SessionID: "sess-1",
	}
	docID, storageType, err := c.UpsertDoc(ctx, uid, pid, input, nil)
	if err != nil {
		t.Fatalf("UpsertDoc (create): %v", err)
	}
	if docID == "" {
		t.Fatal("UpsertDoc returned empty docID")
	}
	if storageType != "inline" {
		t.Errorf("storageType: got %q, want inline", storageType)
	}

	// --- GetDoc ---
	doc, err := c.GetDoc(ctx, uid, pid, docID, nil)
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if doc.ID != docID {
		t.Errorf("ID: got %q, want %q", doc.ID, docID)
	}
	if doc.Title != "Architecture Overview" {
		t.Errorf("Title: got %q", doc.Title)
	}
	if doc.Content != "graph TD; A-->B" {
		t.Errorf("Content: got %q", doc.Content)
	}
	if doc.Format != "mermaid" {
		t.Errorf("Format: got %q, want mermaid", doc.Format)
	}
	if doc.Version != 1 {
		t.Errorf("Version on create: got %d, want 1", doc.Version)
	}

	// --- UpsertDoc (update same doc) ---
	input.ID = docID
	input.Content = "graph TD; A-->B-->C"
	input.SessionID = "sess-2"
	_, _, err = c.UpsertDoc(ctx, uid, pid, input, nil)
	if err != nil {
		t.Fatalf("UpsertDoc (update): %v", err)
	}

	updated, _ := c.GetDoc(ctx, uid, pid, docID, nil)
	if updated.Content != "graph TD; A-->B-->C" {
		t.Errorf("Content after update: got %q", updated.Content)
	}
	if updated.Version != 2 {
		t.Errorf("Version after update: got %d, want 2", updated.Version)
	}

	// --- ListDocs ---
	// Add a second doc of a different type.
	input2 := &models.DocInput{
		Title:     "API Reference",
		DocType:   "api_docs",
		Format:    "markdown",
		Content:   "# API\n\n...",
		UpdatedBy: "claude",
		SessionID: "sess-1",
	}
	_, _, _ = c.UpsertDoc(ctx, uid, pid, input2, nil)

	all, err := c.ListDocs(ctx, uid, pid, nil)
	if err != nil {
		t.Fatalf("ListDocs (all): %v", err)
	}
	if len(all) < 2 {
		t.Errorf("ListDocs: got %d docs, want >=2", len(all))
	}
	// Verify metadata fields are populated (title and doc_type are enough).
	for _, m := range all {
		if m.Title == "" {
			t.Errorf("ListDocs returned doc with empty title for doc %q", m.ID)
		}
	}

	// Filter by doc_type.
	archType := "architecture"
	filtered, _ := c.ListDocs(ctx, uid, pid, &archType)
	for _, m := range filtered {
		if m.DocType != "architecture" {
			t.Errorf("type filter returned wrong type: %q", m.DocType)
		}
	}

	// --- DeleteDoc ---
	if err := c.DeleteDoc(ctx, uid, pid, docID, nil); err != nil {
		t.Fatalf("DeleteDoc: %v", err)
	}

	// Verify the doc is gone.
	_, err = c.GetDoc(ctx, uid, pid, docID, nil)
	if err == nil {
		t.Error("expected error getting deleted doc, got nil")
	}

	// Verify it no longer appears in list.
	remaining, err := c.ListDocs(ctx, uid, pid, nil)
	if err != nil {
		t.Fatalf("ListDocs after delete: %v", err)
	}
	for _, m := range remaining {
		if m.ID == docID {
			t.Errorf("deleted doc %q still appears in ListDocs", docID)
		}
	}
}

func TestDocs_LargeContent(t *testing.T) {
	// This test verifies the size-routing logic without a real GCS client.
	// With a nil GCS client, UpsertDoc should return an error when content exceeds
	// the inline threshold, because it will try to upload and fail.
	// We just verify the threshold check is exercised — not the GCS path itself.
	// Full GCS integration is covered by storage/gcs_test.go with the emulator.
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Large Doc Test"})

	// Build content larger than 500 KB.
	large := strings.Repeat("x", 501*1024)
	input := &models.DocInput{
		Title:     "Huge Doc",
		DocType:   "readme",
		Format:    "markdown",
		Content:   large,
		UpdatedBy: "claude",
		SessionID: "sess-1",
	}

	// Should fail because gcsClient is nil and content exceeds threshold.
	_, _, err := c.UpsertDoc(ctx, uid, pid, input, nil)
	if err == nil {
		t.Error("expected error for large content with nil GCS client, got nil")
	}
}

func TestDocs_SummaryAndHeadings(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Summary Test"})

	mdContent := "# Overview\n\nSome text.\n\n## Installation\n\nMore text.\n\n### Advanced\n\nEven more."
	input := &models.DocInput{
		Title:     "README",
		DocType:   "readme",
		Format:    "markdown",
		Summary:   "A short summary of the README.",
		Content:   mdContent,
		UpdatedBy: "claude",
		SessionID: "sess-1",
	}

	docID, _, err := c.UpsertDoc(ctx, uid, pid, input, nil)
	if err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}

	doc, err := c.GetDoc(ctx, uid, pid, docID, nil)
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if doc.Summary != "A short summary of the README." {
		t.Errorf("Summary: got %q", doc.Summary)
	}
	wantHeadings := []string{"Overview", "Installation", "Advanced"}
	if len(doc.Headings) != len(wantHeadings) {
		t.Fatalf("Headings len: got %d, want %d: %v", len(doc.Headings), len(wantHeadings), doc.Headings)
	}
	for i, h := range wantHeadings {
		if doc.Headings[i] != h {
			t.Errorf("Headings[%d]: got %q, want %q", i, doc.Headings[i], h)
		}
	}

	// Headings and summary appear in ListDocs metadata too.
	metas, err := c.ListDocs(ctx, uid, pid, nil)
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	var meta *models.RepoDocMeta
	for i := range metas {
		if metas[i].ID == docID {
			meta = &metas[i]
			break
		}
	}
	if meta == nil {
		t.Fatal("doc not found in ListDocs")
	}
	if meta.Summary != "A short summary of the README." {
		t.Errorf("meta.Summary: got %q", meta.Summary)
	}
	if len(meta.Headings) != len(wantHeadings) {
		t.Errorf("meta.Headings len: got %d, want %d", len(meta.Headings), len(wantHeadings))
	}
}

func TestDocs_HeadingsNotExtractedForMermaid(t *testing.T) {
	requireEmulator(t)
	c := newTestClient(t)
	ctx := context.Background()
	uid := testUID(t)

	pid, _ := c.CreateProject(ctx, uid, &models.CreateProjectInput{Name: "Mermaid Test"})

	input := &models.DocInput{
		Title:     "Diagram",
		DocType:   "architecture",
		Format:    "mermaid",
		Content:   "graph TD\n  # not a heading\n  A-->B",
		UpdatedBy: "claude",
		SessionID: "sess-1",
	}

	docID, _, err := c.UpsertDoc(ctx, uid, pid, input, nil)
	if err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}

	doc, err := c.GetDoc(ctx, uid, pid, docID, nil)
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if len(doc.Headings) != 0 {
		t.Errorf("expected no headings for mermaid format, got %v", doc.Headings)
	}
}
