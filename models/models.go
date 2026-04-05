package models

import "time"

// User represents a user document at users/{userId}.
type User struct {
	Email       string    `firestore:"email" json:"email"`
	DisplayName string    `firestore:"display_name" json:"display_name"`
	PhotoURL    string    `firestore:"photo_url,omitempty" json:"photo_url,omitempty"`
	CreatedAt   time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at" json:"updated_at"`
}

// Project represents a project document at users/{userId}/projects/{projectId}.
type Project struct {
	ID                 string    `firestore:"-" json:"id"`
	Name               string    `firestore:"name" json:"name"`
	Description        string    `firestore:"description" json:"description"`
	Status             string    `firestore:"status" json:"status"` // "active" | "paused" | "completed" | "archived"
	RepoURL            string    `firestore:"repo_url,omitempty" json:"repo_url,omitempty"`
	TechStack          []string  `firestore:"tech_stack,omitempty" json:"tech_stack,omitempty"`
	LastSessionID      string    `firestore:"last_session_id,omitempty" json:"last_session_id,omitempty"`
	LastSessionSummary string    `firestore:"last_session_summary,omitempty" json:"last_session_summary,omitempty"`
	OpenMilestoneCount int       `firestore:"open_milestone_count" json:"open_milestone_count"`
	CreatedAt          time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt          time.Time `firestore:"updated_at" json:"updated_at"`
}

// Session represents a session document at users/{userId}/projects/{projectId}/sessions/{sessionId}.
type Session struct {
	ID            string     `firestore:"-" json:"id"`
	Status        string     `firestore:"status" json:"status"` // "active" | "completed" | "auto_closed"
	StartedAt     time.Time  `firestore:"started_at" json:"started_at"`
	EndedAt       *time.Time `firestore:"ended_at,omitempty" json:"ended_at,omitempty"`
	Summary       string     `firestore:"summary,omitempty" json:"summary,omitempty"`
	NextSteps     []string   `firestore:"next_steps,omitempty" json:"next_steps,omitempty"`
	Decisions     []string   `firestore:"decisions,omitempty" json:"decisions,omitempty"` // decision IDs made in this session
	ToolCallCount int        `firestore:"tool_call_count" json:"tool_call_count"`
	ClientType    string     `firestore:"client_type" json:"client_type"` // "claude_desktop" | "claude_code" | "claude_web" | "unknown"
	AutoClosed    bool       `firestore:"auto_closed" json:"auto_closed"`
	CreatedAt     time.Time  `firestore:"created_at" json:"created_at"`
}

// Milestone represents a milestone document at users/{userId}/projects/{projectId}/milestones/{milestoneId}.
type Milestone struct {
	ID          string     `firestore:"-" json:"id"`
	Title       string     `firestore:"title" json:"title"`
	Description string     `firestore:"description,omitempty" json:"description,omitempty"`
	DueDate     *time.Time `firestore:"due_date,omitempty" json:"due_date,omitempty"`
	CompletedAt *time.Time `firestore:"completed_at,omitempty" json:"completed_at,omitempty"`
	Status      string     `firestore:"status" json:"status"` // "open" | "completed"
	SessionID   string     `firestore:"session_id" json:"session_id"`
	CreatedAt   time.Time  `firestore:"created_at" json:"created_at"`
}

// Decision represents a decision document at users/{userId}/projects/{projectId}/decisions/{decisionId}.
type Decision struct {
	ID        string    `firestore:"-" json:"id"`
	Decision  string    `firestore:"decision" json:"decision"`
	Rationale string    `firestore:"rationale,omitempty" json:"rationale,omitempty"`
	SessionID string    `firestore:"session_id" json:"session_id"`
	Tags      []string  `firestore:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
}

// Note represents a note document at users/{userId}/projects/{projectId}/notes/{noteId}.
type Note struct {
	ID        string    `firestore:"-" json:"id"`
	Content   string    `firestore:"content" json:"content"`
	SessionID string    `firestore:"session_id" json:"session_id"`
	NoteType  string    `firestore:"note_type" json:"note_type"` // "general" | "bug" | "idea" | "todo"
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
}

// RepoDoc represents a repo_doc document at users/{userId}/projects/{projectId}/repo_docs/{docId}.
type RepoDoc struct {
	ID        string    `firestore:"-" json:"id"`
	Title     string    `firestore:"title" json:"title"`
	DocType   string    `firestore:"doc_type" json:"doc_type"` // "api_docs" | "architecture" | "readme" | "custom"
	Format    string    `firestore:"format" json:"format"`     // "markdown" | "mermaid"
	Content   string    `firestore:"content,omitempty" json:"content,omitempty"`
	GCSPath   string    `firestore:"gcs_path,omitempty" json:"gcs_path,omitempty"`
	Version   int       `firestore:"version" json:"version"`
	UpdatedBy string    `firestore:"updated_by" json:"updated_by"` // "claude" | "user"
	SessionID string    `firestore:"session_id" json:"session_id"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at" json:"updated_at"`
}

// RepoDocMeta is RepoDoc without the content field, used for list operations.
type RepoDocMeta struct {
	ID        string    `firestore:"-" json:"id"`
	Title     string    `firestore:"title" json:"title"`
	DocType   string    `firestore:"doc_type" json:"doc_type"`
	Format    string    `firestore:"format" json:"format"`
	GCSPath   string    `firestore:"gcs_path,omitempty" json:"gcs_path,omitempty"`
	Version   int       `firestore:"version" json:"version"`
	UpdatedBy string    `firestore:"updated_by" json:"updated_by"`
	SessionID string    `firestore:"session_id" json:"session_id"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at" json:"updated_at"`
}

// ToolCallEntry represents a tool_call_log document at .../sessions/{sessionId}/tool_call_log/{logId}.
type ToolCallEntry struct {
	ID       string                 `firestore:"-" json:"id"`
	ToolName string                 `firestore:"tool_name" json:"tool_name"`
	Args     map[string]interface{} `firestore:"args" json:"args"`
	Result   string                 `firestore:"result,omitempty" json:"result,omitempty"`
	CalledAt time.Time              `firestore:"called_at" json:"called_at"`
}

// StaleSession includes path info so the cleanup routine can scope operations correctly.
type StaleSession struct {
	Session
	UserID    string
	ProjectID string
}

// --- Input types ---

type CreateProjectInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RepoURL     string   `json:"repo_url,omitempty"`
	TechStack   []string `json:"tech_stack,omitempty"`
}

type CreateMilestoneInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	SessionID   string     `json:"session_id"`
}

type CreateDecisionInput struct {
	Decision  string   `json:"decision"`
	Rationale string   `json:"rationale,omitempty"`
	SessionID string   `json:"session_id"`
	Tags      []string `json:"tags,omitempty"`
}

type CreateNoteInput struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id"`
	NoteType  string `json:"note_type"`
}

// DocInput is used for creating or updating a repo doc.
// If ID is set, the existing doc is updated; if empty, a new doc is created.
type DocInput struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title"`
	DocType   string `json:"doc_type"`
	Format    string `json:"format"`
	Content   string `json:"content"`
	UpdatedBy string `json:"updated_by"`
	SessionID string `json:"session_id"`
}
