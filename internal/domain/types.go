// Package domain holds the pure types that flow between store, services, and handlers.
// No external dependencies — these types are safe to use anywhere.
package domain

import "time"

type Epic struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	RepoPath    string
	BaseBranch  string
	ContextMD   string
	CreatedAt   time.Time
	ArchivedAt  *time.Time
}

type Workspace struct {
	ID           int64
	EpicID       int64
	Slug         string
	Name         string
	BranchName   string
	WorktreePath string
	PRURL        string
	NotesMD      string
	CreatedAt    time.Time
	ArchivedAt   *time.Time
}

type Session struct {
	SessionID         string
	ProjectDir        string
	Cwd               string
	GitBranch         string
	Model             string
	Version           string
	LastActivity      time.Time
	LastUserPreview   string
	LastAssistantText string
	FilePath          string
	FileSizeBytes     int64
	NumEvents         int64
	WorkspaceID       *int64
}

// Message is one rendered turn in a transcript.
type Message struct {
	Role      string // "user" | "assistant" | "tool" | "system"
	Text      string
	Model     string
	ToolName  string // for "tool" role
	Timestamp time.Time
}

// DiffStat is a per-file or aggregated diff summary.
//
// JSON tags are required: this type is returned directly by the HTTP API
// (see workspaceDetailJSON.Files) and the React client expects snake_case.
type DiffStat struct {
	Path    string `json:"path"`
	Status  string `json:"status"` // "M", "A", "D", "R", "??", etc.
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}
