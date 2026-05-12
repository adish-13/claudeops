package store_test

import (
	"context"
	"testing"

	"claudeops/internal/store/storetest"
)

func TestCreateAndListWorkspaces(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	epicID := storetest.SeedEpic(t, s, "epic", "/tmp/repo")
	storetest.SeedWorkspace(t, s, epicID, "ws-a", "user-epic-ws-a", "/tmp/wt-a")
	storetest.SeedWorkspace(t, s, epicID, "ws-b", "user-epic-ws-b", "/tmp/wt-b")

	ws, err := s.ListWorkspacesByEpic(ctx, epicID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ws) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(ws))
	}
}

func TestGetWorkspaceBySlug(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	epicID := storetest.SeedEpic(t, s, "ep", "/tmp/r")
	storetest.SeedWorkspace(t, s, epicID, "wsx", "branch-x", "/tmp/wt-x")

	w, e, err := s.GetWorkspaceBySlug(ctx, "ep", "wsx")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if w.BranchName != "branch-x" || e.Slug != "ep" {
		t.Fatalf("unexpected: ws=%+v epic=%+v", w, e)
	}
}

func TestUpdateWorkspaceNotes(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	epicID := storetest.SeedEpic(t, s, "ep", "/tmp/r")
	wsID := storetest.SeedWorkspace(t, s, epicID, "ws", "branch", "/tmp/wt")

	if err := s.UpdateWorkspaceNotes(ctx, wsID, "decided X over Y"); err != nil {
		t.Fatalf("update notes: %v", err)
	}
	w, _, _ := s.GetWorkspaceBySlug(ctx, "ep", "ws")
	if w.NotesMD != "decided X over Y" {
		t.Fatalf("notes not saved: %q", w.NotesMD)
	}
}

func TestUpdateWorkspacePR(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	epicID := storetest.SeedEpic(t, s, "ep", "/tmp/r")
	wsID := storetest.SeedWorkspace(t, s, epicID, "ws", "branch", "/tmp/wt")

	if err := s.UpdateWorkspacePR(ctx, wsID, "https://github.com/x/y/pull/1"); err != nil {
		t.Fatalf("update PR: %v", err)
	}
	w, _, _ := s.GetWorkspaceBySlug(ctx, "ep", "ws")
	if w.PRURL != "https://github.com/x/y/pull/1" {
		t.Fatalf("PR not saved: %q", w.PRURL)
	}
}

func TestArchiveWorkspaceHidesIt(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	epicID := storetest.SeedEpic(t, s, "ep", "/tmp/r")
	wsID := storetest.SeedWorkspace(t, s, epicID, "ws", "branch", "/tmp/wt")

	if err := s.ArchiveWorkspace(ctx, wsID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	all, _ := s.ListWorkspacesByEpic(ctx, epicID)
	if len(all) != 0 {
		t.Fatalf("archived workspace still listed: %d entries", len(all))
	}
	got, _ := s.AllWorkspaces(ctx)
	if len(got) != 0 {
		t.Fatalf("AllWorkspaces should also exclude archived: %d entries", len(got))
	}
}
