# briefcase-internal

Shared Go library for the briefcase project. Imported as a module dependency by the MCP server and API server. Not deployed — it compiles into each backend binary.

## Packages

| Package | Description |
|---------|-------------|
| `auth` | Firebase ID token verification. Returns a `Claims` struct with `UID`, `Email`, `Name`, `Picture`. |
| `db` | Firestore client + CRUD for all entities: projects, sessions, milestones, decisions, notes, repo docs, tool call log, users. |
| `models` | Go struct definitions for every Firestore entity and their input types. |
| `storage` | Cloud Storage client for uploading and downloading large repo docs. |
| `validation` | Enum validation helpers for status values, note types, doc types, and doc formats. |

## Prerequisites

- Go 1.22+
- A GCP project with Firestore (Native mode), Cloud Storage, and Firebase Auth enabled
- For local development: [Firebase Local Emulator Suite](https://firebase.google.com/docs/emulator-suite) and optionally [fake-gcs-server](https://github.com/fsouza/fake-gcs-server)

## Usage

```bash
go get github.com/raghav-anand/briefcase-internal@latest
```

```go
import (
    "github.com/raghav-anand/briefcase-internal/auth"
    "github.com/raghav-anand/briefcase-internal/db"
    "github.com/raghav-anand/briefcase-internal/models"
    "github.com/raghav-anand/briefcase-internal/storage"
    "github.com/raghav-anand/briefcase-internal/validation"
)

// Initialise clients once at startup.
dbClient, err := db.NewClient(ctx, os.Getenv("GCP_PROJECT_ID"))
authClient, err := auth.InitAuth(ctx, os.Getenv("GCP_PROJECT_ID"))
gcsClient, err := storage.NewGCSClient(ctx, os.Getenv("GCS_BUCKET"))
```

On Cloud Run, all clients authenticate automatically via the service account (Application Default Credentials). Locally, set `GOOGLE_APPLICATION_CREDENTIALS` to a service account key file.

## Running Tests

Tests are split into two groups:

### Unit tests (no infrastructure needed)

```bash
go test ./validation/...
```

### Integration tests (Firestore emulator required)

**1. Install the Firebase CLI and start the emulator:**

```bash
npm install -g firebase-tools
firebase emulators:start --only firestore
```

The emulator starts on `localhost:8080` by default.

**2. Run the db integration tests:**

```bash
export FIRESTORE_EMULATOR_HOST=localhost:8080
go test ./db/...
```

**3. Run all tests (unit + integration):**

```bash
export FIRESTORE_EMULATOR_HOST=localhost:8080
go test ./...
```

Tests that require `FIRESTORE_EMULATOR_HOST` are automatically skipped when the variable is not set.

### GCS integration tests (fake-gcs-server required)

```bash
# Start fake-gcs-server
docker run -p 4443:4443 fsouza/fake-gcs-server -scheme http -port 4443

export STORAGE_EMULATOR_HOST=localhost:4443
go test ./storage/...
```

These tests are skipped when `STORAGE_EMULATOR_HOST` is not set.

## Versioning

Services pin to a specific version tag:

```bash
# In the consuming service's module
go get github.com/raghav-anand/briefcase-internal@v0.1.0
```

After making changes to this library:

```bash
git tag v0.x.x
git push origin v0.x.x
```

Then run `go get github.com/raghav-anand/briefcase-internal@v0.x.x` in the consuming service.

## Design Notes

- **User scoping is mandatory.** Every `db` function takes a `uid` parameter. There are no cross-user queries except `FindStaleSessions`, which returns path info so callers can scope follow-up operations.
- **Timestamps are server-side.** The `db` package always sets `created_at` / `updated_at` / `ended_at`; callers never pass timestamps.
- **Inline vs. Cloud Storage.** `UpsertDoc` stores content inline in Firestore for docs under ~500 KB. Larger content goes to Cloud Storage automatically.
- **No orchestration logic here.** Crash recovery, session cleanup scheduling, and context assembly belong in the MCP server, not in this library.
