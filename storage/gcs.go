package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSClient wraps the Cloud Storage client for doc storage.
type GCSClient struct {
	bucket     *gcs.BucketHandle
	bucketName string
	// jsonEndpoint is set when STORAGE_EMULATOR_HOST is in use. Download uses
	// the JSON API ?alt=media path directly instead of the SDK's NewReader,
	// which sends a GET /{bucket}/{object} (XML API) that fake-gcs-server
	// cannot route when the object name contains encoded slashes (%2F).
	jsonEndpoint string
}

// NewGCSClient initializes a Cloud Storage client for the given bucket.
// Uses Application Default Credentials on Cloud Run.
// When STORAGE_EMULATOR_HOST is set (e.g. "localhost:4443"), requests are
// routed to the local fake-gcs-server.
func NewGCSClient(ctx context.Context, bucketName string) (*GCSClient, error) {
	c := &GCSClient{bucketName: bucketName}

	var opts []option.ClientOption
	if host := os.Getenv("STORAGE_EMULATOR_HOST"); host != "" {
		base := "http://" + host
		opts = append(opts,
			option.WithEndpoint(base+"/storage/v1/"),
			option.WithoutAuthentication(),
		)
		// Store the JSON API base so Download can bypass NewReader's XML path.
		c.jsonEndpoint = base
	}

	client, err := gcs.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	c.bucket = client.Bucket(bucketName)
	return c, nil
}

// Upload writes content to Cloud Storage at the given object path.
// Path format: "docs/{userId}/{projectId}/{docId}.md"
func (g *GCSClient) Upload(ctx context.Context, path string, content []byte) error {
	w := g.bucket.Object(path).NewWriter(ctx)
	w.ContentType = "text/plain; charset=utf-8"
	// ChunkSize = 0 forces a single-request upload (no resumable protocol).
	// Required for fake-gcs-server; fine in production since doc content is small.
	w.ChunkSize = 0
	if _, err := w.Write(content); err != nil {
		_ = w.Close()
		return fmt.Errorf("gcs write %q: %w", path, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("gcs close %q: %w", path, err)
	}
	return nil
}

// Download reads content from Cloud Storage at the given object path.
//
// When running against the real GCS service, it uses the SDK's NewReader.
// When running against fake-gcs-server (STORAGE_EMULATOR_HOST is set), it
// fetches via the JSON API (/storage/v1/b/{bucket}/o/{object}?alt=media)
// because the SDK's NewReader sends a GET /{bucket}/{object} (XML API) whose
// %2F-encoded object name is not resolved correctly by fake-gcs-server.
func (g *GCSClient) Download(ctx context.Context, path string) ([]byte, error) {
	if g.jsonEndpoint != "" {
		return g.downloadViaJSONAPI(ctx, path)
	}
	r, err := g.bucket.Object(path).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs open %q: %w", path, err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gcs read %q: %w", path, err)
	}
	return data, nil
}

// downloadViaJSONAPI fetches object content via the GCS JSON API ?alt=media
// endpoint. This avoids the SDK's XML-style GET /{bucket}/{object} which
// fake-gcs-server cannot handle for object names containing encoded slashes.
func (g *GCSClient) downloadViaJSONAPI(ctx context.Context, path string) ([]byte, error) {
	// /storage/v1/b/{bucket}/o/{encodedObject}?alt=media
	apiURL := fmt.Sprintf("%s/storage/v1/b/%s/o/%s?alt=media",
		g.jsonEndpoint,
		url.PathEscape(g.bucketName),
		url.PathEscape(path),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gcs download build request %q: %w", path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gcs download %q: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("gcs open %q: storage: object doesn't exist", path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gcs download %q: unexpected status %d", path, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gcs read %q: %w", path, err)
	}
	return data, nil
}

// Delete removes an object from Cloud Storage.
func (g *GCSClient) Delete(ctx context.Context, path string) error {
	if err := g.bucket.Object(path).Delete(ctx); err != nil {
		return fmt.Errorf("gcs delete %q: %w", path, err)
	}
	return nil
}
