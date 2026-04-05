package db

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	bstorage "github.com/raghav-anand/briefcase-internal/storage"
	"github.com/raghav-anand/briefcase-internal/models"
)

// maxInlineBytes is the content size threshold for inline vs. Cloud Storage.
// Docs smaller than this are stored directly in Firestore; larger docs go to GCS.
const maxInlineBytes = 500 * 1024

// UpsertDoc creates or updates a repo doc. If doc.ID is set, the existing doc is updated;
// otherwise a new doc is created. Content larger than maxInlineBytes is stored in Cloud Storage.
// Returns (docID, storageType, error) where storageType is "inline" or "cloud_storage".
func (c *Client) UpsertDoc(ctx context.Context, uid, pid string, doc *models.DocInput, gcsClient *bstorage.GCSClient) (string, string, error) {
	now := time.Now()
	content := []byte(doc.Content)
	storageType := "inline"

	var gcsPath string
	var inlineContent string

	if len(content) > maxInlineBytes {
		if gcsClient == nil {
			return "", "", fmt.Errorf("UpsertDoc: content size %d exceeds inline limit (%d bytes); a GCS client is required", len(content), maxInlineBytes)
		}
		storageType = "cloud_storage"
		docID := doc.ID
		if docID == "" {
			docID = c.repoDocsCol(uid, pid).NewDoc().ID
		}
		gcsPath = fmt.Sprintf("docs/%s/%s/%s.md", uid, pid, docID)
		if err := gcsClient.Upload(ctx, gcsPath, content); err != nil {
			return "", "", fmt.Errorf("UpsertDoc upload to GCS: %w", err)
		}
		doc.ID = docID
	} else {
		inlineContent = doc.Content
	}

	var ref *firestore.DocumentRef
	if doc.ID != "" {
		ref = c.repoDocsCol(uid, pid).Doc(doc.ID)
	} else {
		ref = c.repoDocsCol(uid, pid).NewDoc()
	}

	existing, _ := ref.Get(ctx)
	version := 1
	if existing != nil && existing.Exists() {
		if v, ok := existing.Data()["version"].(int64); ok {
			version = int(v) + 1
		}
	}

	data := map[string]interface{}{
		"title":      doc.Title,
		"doc_type":   doc.DocType,
		"format":     doc.Format,
		"version":    version,
		"updated_by": doc.UpdatedBy,
		"session_id": doc.SessionID,
		"updated_at": now,
	}

	if storageType == "cloud_storage" {
		data["gcs_path"] = gcsPath
		data["content"] = ""
	} else {
		data["content"] = inlineContent
		data["gcs_path"] = ""
	}

	if existing == nil || !existing.Exists() {
		data["created_at"] = now
		if _, err := ref.Set(ctx, data); err != nil {
			return "", "", fmt.Errorf("UpsertDoc set: %w", err)
		}
	} else {
		firestoreUpdates := make([]firestore.Update, 0, len(data))
		for k, v := range data {
			firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: k, Value: v})
		}
		if _, err := ref.Update(ctx, firestoreUpdates); err != nil {
			return "", "", fmt.Errorf("UpsertDoc update: %w", err)
		}
	}

	return ref.ID, storageType, nil
}

// GetDoc returns a doc with its content. If the doc is stored in GCS, the content is fetched
// from Cloud Storage and populated into the returned RepoDoc.
func (c *Client) GetDoc(ctx context.Context, uid, pid, did string, gcsClient *bstorage.GCSClient) (*models.RepoDoc, error) {
	docSnap, err := c.repoDocsCol(uid, pid).Doc(did).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetDoc %s: %w", did, err)
	}

	var d models.RepoDoc
	if err := docSnap.DataTo(&d); err != nil {
		return nil, fmt.Errorf("GetDoc decode %s: %w", did, err)
	}
	d.ID = docSnap.Ref.ID

	if d.GCSPath != "" && gcsClient != nil {
		content, err := gcsClient.Download(ctx, d.GCSPath)
		if err != nil {
			return nil, fmt.Errorf("GetDoc fetch GCS content: %w", err)
		}
		d.Content = string(content)
	}

	return &d, nil
}

// ListDocs returns doc metadata (without content) for a project, ordered by most recently updated.
// Pass a non-nil docType to filter by type.
func (c *Client) ListDocs(ctx context.Context, uid, pid string, docType *string) ([]models.RepoDocMeta, error) {
	q := c.repoDocsCol(uid, pid).Query
	if docType != nil {
		q = q.Where("doc_type", "==", *docType)
	}
	q = q.OrderBy("updated_at", firestore.Desc)

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("ListDocs: %w", err)
	}

	metas := make([]models.RepoDocMeta, 0, len(docs))
	for _, doc := range docs {
		var m models.RepoDocMeta
		if err := doc.DataTo(&m); err != nil {
			return nil, fmt.Errorf("ListDocs decode %s: %w", doc.Ref.ID, err)
		}
		m.ID = doc.Ref.ID
		metas = append(metas, m)
	}
	return metas, nil
}
