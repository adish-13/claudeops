package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

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

type Store struct {
	db *sql.DB
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
	// Lazy column add for upgrades from v1 schema.
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN workspace_id INTEGER REFERENCES workspaces(id) ON DELETE SET NULL`)
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ---------- Epics ----------

func (s *Store) CreateEpic(ctx context.Context, e Epic) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
        INSERT INTO epics (slug, name, description, repo_path, base_branch, context_md, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.Slug, e.Name, e.Description, e.RepoPath, e.BaseBranch, e.ContextMD, time.Now().UnixMilli())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateEpicContext(ctx context.Context, slug, ctxMD string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE epics SET context_md = ? WHERE slug = ?`, ctxMD, slug)
	return err
}

func (s *Store) ListEpics(ctx context.Context) ([]Epic, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, slug, name, description, repo_path, base_branch, context_md, created_at, archived_at
        FROM epics WHERE archived_at IS NULL
        ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Epic
	for rows.Next() {
		var e Epic
		var createdMs int64
		var archivedMs sql.NullInt64
		if err := rows.Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.RepoPath, &e.BaseBranch, &e.ContextMD, &createdMs, &archivedMs); err != nil {
			return nil, err
		}
		e.CreatedAt = time.UnixMilli(createdMs)
		if archivedMs.Valid {
			t := time.UnixMilli(archivedMs.Int64)
			e.ArchivedAt = &t
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetEpicBySlug(ctx context.Context, slug string) (*Epic, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT id, slug, name, description, repo_path, base_branch, context_md, created_at, archived_at
        FROM epics WHERE slug = ?`, slug)
	var e Epic
	var createdMs int64
	var archivedMs sql.NullInt64
	if err := row.Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.RepoPath, &e.BaseBranch, &e.ContextMD, &createdMs, &archivedMs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	e.CreatedAt = time.UnixMilli(createdMs)
	if archivedMs.Valid {
		t := time.UnixMilli(archivedMs.Int64)
		e.ArchivedAt = &t
	}
	return &e, nil
}

// ---------- Workspaces ----------

func (s *Store) CreateWorkspace(ctx context.Context, w Workspace) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
        INSERT INTO workspaces (epic_id, slug, name, branch_name, worktree_path, pr_url, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.EpicID, w.Slug, w.Name, w.BranchName, w.WorktreePath, w.PRURL, time.Now().UnixMilli())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateWorkspacePR(ctx context.Context, id int64, prURL string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workspaces SET pr_url = ? WHERE id = ?`, prURL, id)
	return err
}

func (s *Store) ListWorkspacesByEpic(ctx context.Context, epicID int64) ([]Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, epic_id, slug, name, branch_name, worktree_path, pr_url, created_at, archived_at
        FROM workspaces WHERE epic_id = ? AND archived_at IS NULL
        ORDER BY created_at DESC`, epicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var w Workspace
		var createdMs int64
		var archivedMs sql.NullInt64
		if err := rows.Scan(&w.ID, &w.EpicID, &w.Slug, &w.Name, &w.BranchName, &w.WorktreePath, &w.PRURL, &createdMs, &archivedMs); err != nil {
			return nil, err
		}
		w.CreatedAt = time.UnixMilli(createdMs)
		if archivedMs.Valid {
			t := time.UnixMilli(archivedMs.Int64)
			w.ArchivedAt = &t
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspaceBySlug(ctx context.Context, epicSlug, wsSlug string) (*Workspace, *Epic, error) {
	epic, err := s.GetEpicBySlug(ctx, epicSlug)
	if err != nil {
		return nil, nil, err
	}
	row := s.db.QueryRowContext(ctx, `
        SELECT id, epic_id, slug, name, branch_name, worktree_path, pr_url, created_at, archived_at
        FROM workspaces WHERE epic_id = ? AND slug = ?`, epic.ID, wsSlug)
	var w Workspace
	var createdMs int64
	var archivedMs sql.NullInt64
	if err := row.Scan(&w.ID, &w.EpicID, &w.Slug, &w.Name, &w.BranchName, &w.WorktreePath, &w.PRURL, &createdMs, &archivedMs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, epic, ErrNotFound
		}
		return nil, epic, err
	}
	w.CreatedAt = time.UnixMilli(createdMs)
	if archivedMs.Valid {
		t := time.UnixMilli(archivedMs.Int64)
		w.ArchivedAt = &t
	}
	return &w, epic, nil
}

// AllWorkspaces returns every active workspace (used by the indexer for cwd binding).
func (s *Store) AllWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, epic_id, slug, name, branch_name, worktree_path, pr_url, created_at, archived_at
        FROM workspaces WHERE archived_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var w Workspace
		var createdMs int64
		var archivedMs sql.NullInt64
		if err := rows.Scan(&w.ID, &w.EpicID, &w.Slug, &w.Name, &w.BranchName, &w.WorktreePath, &w.PRURL, &createdMs, &archivedMs); err != nil {
			return nil, err
		}
		w.CreatedAt = time.UnixMilli(createdMs)
		if archivedMs.Valid {
			t := time.UnixMilli(archivedMs.Int64)
			w.ArchivedAt = &t
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ---------- Sessions ----------

func (s *Store) UpsertSession(ctx context.Context, sess Session) error {
	var workspaceID sql.NullInt64
	if sess.WorkspaceID != nil {
		workspaceID = sql.NullInt64{Int64: *sess.WorkspaceID, Valid: true}
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO sessions (
            session_id, project_dir, cwd, git_branch, model, version,
            last_activity, last_user_preview, last_assistant_text,
            file_path, file_size_bytes, num_events, workspace_id
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(session_id) DO UPDATE SET
            project_dir         = excluded.project_dir,
            cwd                 = excluded.cwd,
            git_branch          = excluded.git_branch,
            model               = excluded.model,
            version             = excluded.version,
            last_activity       = excluded.last_activity,
            last_user_preview   = excluded.last_user_preview,
            last_assistant_text = excluded.last_assistant_text,
            file_path           = excluded.file_path,
            file_size_bytes     = excluded.file_size_bytes,
            num_events          = excluded.num_events,
            workspace_id        = excluded.workspace_id
    `,
		sess.SessionID, sess.ProjectDir, sess.Cwd, sess.GitBranch, sess.Model, sess.Version,
		sess.LastActivity.UnixMilli(), sess.LastUserPreview, sess.LastAssistantText,
		sess.FilePath, sess.FileSizeBytes, sess.NumEvents, workspaceID,
	)
	return err
}

func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	return s.querySessions(ctx, ``, nil)
}

func (s *Store) ListSessionsByWorkspace(ctx context.Context, workspaceID int64) ([]Session, error) {
	return s.querySessions(ctx, `WHERE workspace_id = ?`, []any{workspaceID})
}

func (s *Store) ListUnboundSessionsForRepo(ctx context.Context, repoPath string) ([]Session, error) {
	// "Unbound but plausibly related": sessions whose cwd starts with the epic's repo_path
	// and that are not yet bound to any workspace.
	pattern := strings.TrimRight(repoPath, "/") + "%"
	return s.querySessions(ctx, `WHERE workspace_id IS NULL AND cwd LIKE ?`, []any{pattern})
}

func (s *Store) querySessions(ctx context.Context, where string, args []any) ([]Session, error) {
	q := `SELECT session_id, project_dir, cwd, git_branch, model, version,
                 last_activity, last_user_preview, last_assistant_text,
                 file_path, file_size_bytes, num_events, workspace_id
          FROM sessions ` + where + ` ORDER BY last_activity DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var sess Session
		var actMs int64
		var workspaceID sql.NullInt64
		if err := rows.Scan(
			&sess.SessionID, &sess.ProjectDir, &sess.Cwd, &sess.GitBranch, &sess.Model, &sess.Version,
			&actMs, &sess.LastUserPreview, &sess.LastAssistantText,
			&sess.FilePath, &sess.FileSizeBytes, &sess.NumEvents, &workspaceID,
		); err != nil {
			return nil, err
		}
		sess.LastActivity = time.UnixMilli(actMs)
		if workspaceID.Valid {
			id := workspaceID.Int64
			sess.WorkspaceID = &id
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}
