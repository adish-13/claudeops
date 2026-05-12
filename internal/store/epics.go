package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"claudeops/internal/domain"
)

var ErrNotFound = errors.New("not found")

func (s *Store) CreateEpic(ctx context.Context, e domain.Epic) (int64, error) {
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

func (s *Store) ArchiveEpic(ctx context.Context, slug string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE epics SET archived_at = ? WHERE slug = ?`, time.Now().UnixMilli(), slug)
	return err
}

func (s *Store) ListEpics(ctx context.Context) ([]domain.Epic, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, slug, name, description, repo_path, base_branch, context_md, created_at, archived_at
        FROM epics WHERE archived_at IS NULL
        ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Epic
	for rows.Next() {
		e, err := scanEpic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetEpicBySlug(ctx context.Context, slug string) (*domain.Epic, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT id, slug, name, description, repo_path, base_branch, context_md, created_at, archived_at
        FROM epics WHERE slug = ?`, slug)
	e, err := scanEpic(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanEpic(r rowScanner) (domain.Epic, error) {
	var e domain.Epic
	var createdMs int64
	var archivedMs sql.NullInt64
	if err := r.Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.RepoPath, &e.BaseBranch, &e.ContextMD, &createdMs, &archivedMs); err != nil {
		return e, err
	}
	e.CreatedAt = time.UnixMilli(createdMs)
	if archivedMs.Valid {
		t := time.UnixMilli(archivedMs.Int64)
		e.ArchivedAt = &t
	}
	return e, nil
}
