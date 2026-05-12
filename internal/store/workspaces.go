package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"claudeops/internal/domain"
)

func (s *Store) CreateWorkspace(ctx context.Context, w domain.Workspace) (int64, error) {
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

func (s *Store) ArchiveWorkspace(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workspaces SET archived_at = ? WHERE id = ?`, time.Now().UnixMilli(), id)
	return err
}

func (s *Store) ListWorkspacesByEpic(ctx context.Context, epicID int64) ([]domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, epic_id, slug, name, branch_name, worktree_path, pr_url, created_at, archived_at
        FROM workspaces WHERE epic_id = ? AND archived_at IS NULL
        ORDER BY created_at DESC`, epicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Workspace
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// AllWorkspaces returns every active workspace; used by the indexer for cwd binding.
func (s *Store) AllWorkspaces(ctx context.Context) ([]domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, epic_id, slug, name, branch_name, worktree_path, pr_url, created_at, archived_at
        FROM workspaces WHERE archived_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Workspace
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspaceBySlug(ctx context.Context, epicSlug, wsSlug string) (*domain.Workspace, *domain.Epic, error) {
	epic, err := s.GetEpicBySlug(ctx, epicSlug)
	if err != nil {
		return nil, nil, err
	}
	row := s.db.QueryRowContext(ctx, `
        SELECT id, epic_id, slug, name, branch_name, worktree_path, pr_url, created_at, archived_at
        FROM workspaces WHERE epic_id = ? AND slug = ?`, epic.ID, wsSlug)
	w, err := scanWorkspace(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, epic, ErrNotFound
		}
		return nil, epic, err
	}
	return &w, epic, nil
}

func scanWorkspace(r rowScanner) (domain.Workspace, error) {
	var w domain.Workspace
	var createdMs int64
	var archivedMs sql.NullInt64
	if err := r.Scan(&w.ID, &w.EpicID, &w.Slug, &w.Name, &w.BranchName, &w.WorktreePath, &w.PRURL, &createdMs, &archivedMs); err != nil {
		return w, err
	}
	w.CreatedAt = time.UnixMilli(createdMs)
	if archivedMs.Valid {
		t := time.UnixMilli(archivedMs.Int64)
		w.ArchivedAt = &t
	}
	return w, nil
}
