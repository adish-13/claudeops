// Package storetest provides ephemeral *store.Store instances for tests.
//
// Test code calls New(t) and immediately has a working store backed by a
// SQLite file in t.TempDir(); cleanup is registered via t.Cleanup, so callers
// don't need to Close manually.
package storetest

import (
	"context"
	"path/filepath"
	"testing"

	"claudeops/internal/domain"
	"claudeops/internal/store"
)

func New(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("storetest.New: open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// SeedEpic inserts an epic and returns its ID. Convenience for table tests.
func SeedEpic(t *testing.T, s *store.Store, slug, repoPath string) int64 {
	t.Helper()
	id, err := s.CreateEpic(context.Background(), domain.Epic{
		Slug:       slug,
		Name:       slug,
		RepoPath:   repoPath,
		BaseBranch: "master",
	})
	if err != nil {
		t.Fatalf("SeedEpic: %v", err)
	}
	return id
}

// SeedWorkspace inserts a workspace under epicID and returns its ID.
func SeedWorkspace(t *testing.T, s *store.Store, epicID int64, slug, branch, worktreePath string) int64 {
	t.Helper()
	id, err := s.CreateWorkspace(context.Background(), domain.Workspace{
		EpicID:       epicID,
		Slug:         slug,
		Name:         slug,
		BranchName:   branch,
		WorktreePath: worktreePath,
	})
	if err != nil {
		t.Fatalf("SeedWorkspace: %v", err)
	}
	return id
}
