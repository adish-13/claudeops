package store_test

import (
	"context"
	"errors"
	"testing"

	"claudeops/internal/domain"
	"claudeops/internal/store"
	"claudeops/internal/store/storetest"
)

func TestCreateAndGetEpic(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	id, err := s.CreateEpic(ctx, domain.Epic{
		Slug:        "postgres-recovery",
		Name:        "Postgres Recovery",
		Description: "PiTR work",
		RepoPath:    "/tmp/repo",
		BaseBranch:  "master",
		ContextMD:   "spec...",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}

	got, err := s.GetEpicBySlug(ctx, "postgres-recovery")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Postgres Recovery" || got.RepoPath != "/tmp/repo" || got.ContextMD != "spec..." {
		t.Fatalf("unexpected epic: %+v", got)
	}
}

func TestGetEpicNotFound(t *testing.T) {
	s := storetest.New(t)
	_, err := s.GetEpicBySlug(context.Background(), "does-not-exist")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateEpicDuplicateSlug(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	storetest.SeedEpic(t, s, "dup", "/tmp/r")
	if _, err := s.CreateEpic(ctx, domain.Epic{Slug: "dup", Name: "x", RepoPath: "/tmp/r2", BaseBranch: "master"}); err == nil {
		t.Fatalf("expected unique-constraint error")
	}
}

func TestUpdateEpicContextAndArchive(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	storetest.SeedEpic(t, s, "ctx", "/tmp/r")

	if err := s.UpdateEpicContext(ctx, "ctx", "new context"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := s.GetEpicBySlug(ctx, "ctx")
	if got.ContextMD != "new context" {
		t.Fatalf("context not updated: %q", got.ContextMD)
	}

	if err := s.ArchiveEpic(ctx, "ctx"); err != nil {
		t.Fatalf("archive: %v", err)
	}
	all, _ := s.ListEpics(ctx)
	for _, e := range all {
		if e.Slug == "ctx" {
			t.Fatalf("archived epic still in list: %+v", e)
		}
	}
}
