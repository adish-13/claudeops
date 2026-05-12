package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"claudeops/internal/domain"
)

func (s *Store) UpsertSession(ctx context.Context, sess domain.Session) error {
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

func (s *Store) ListSessions(ctx context.Context) ([]domain.Session, error) {
	return s.querySessions(ctx, ``, nil)
}

func (s *Store) ListSessionsByWorkspace(ctx context.Context, workspaceID int64) ([]domain.Session, error) {
	return s.querySessions(ctx, `WHERE workspace_id = ?`, []any{workspaceID})
}

func (s *Store) ListUnboundSessionsForRepo(ctx context.Context, repoPath string) ([]domain.Session, error) {
	pattern := strings.TrimRight(repoPath, "/") + "%"
	return s.querySessions(ctx, `WHERE workspace_id IS NULL AND cwd LIKE ?`, []any{pattern})
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	rows, err := s.querySessions(ctx, `WHERE session_id = ?`, []any{sessionID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

func (s *Store) querySessions(ctx context.Context, where string, args []any) ([]domain.Session, error) {
	q := `SELECT session_id, project_dir, cwd, git_branch, model, version,
                 last_activity, last_user_preview, last_assistant_text,
                 file_path, file_size_bytes, num_events, workspace_id
          FROM sessions ` + where + ` ORDER BY last_activity DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Session
	for rows.Next() {
		var sess domain.Session
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

// avoid unused import warnings for narrow build configurations
var _ = errors.New
