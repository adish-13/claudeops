// Package store is the SQLite persistence layer.
// store.go owns the schema and connection lifecycle; per-resource files own queries.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db   *sql.DB
	path string
}

const schema = `
CREATE TABLE IF NOT EXISTS epics (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT UNIQUE NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    repo_path    TEXT NOT NULL,
    base_branch  TEXT NOT NULL DEFAULT 'master',
    context_md   TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    archived_at  INTEGER
);

CREATE TABLE IF NOT EXISTS workspaces (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id       INTEGER NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
    slug          TEXT NOT NULL,
    name          TEXT NOT NULL,
    branch_name   TEXT NOT NULL,
    worktree_path TEXT NOT NULL UNIQUE,
    pr_url        TEXT NOT NULL DEFAULT '',
    created_at    INTEGER NOT NULL,
    archived_at   INTEGER,
    UNIQUE(epic_id, slug)
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id          TEXT PRIMARY KEY,
    project_dir         TEXT NOT NULL,
    cwd                 TEXT,
    git_branch          TEXT,
    model               TEXT,
    version             TEXT,
    last_activity       INTEGER NOT NULL,
    last_user_preview   TEXT,
    last_assistant_text TEXT,
    file_path           TEXT NOT NULL,
    file_size_bytes     INTEGER NOT NULL,
    num_events          INTEGER NOT NULL DEFAULT 0,
    workspace_id        INTEGER REFERENCES workspaces(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_activity ON sessions(last_activity DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_sessions_cwd ON sessions(cwd);
`

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	// Lazy ALTER for upgrades from older schemas — safe to ignore "duplicate column" errors.
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN workspace_id INTEGER REFERENCES workspaces(id) ON DELETE SET NULL`)
	return &Store{db: db, path: path}, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Path() string { return s.path }

// DB returns the underlying *sql.DB. Used by the debug page only.
func (s *Store) DB() *sql.DB { return s.db }
