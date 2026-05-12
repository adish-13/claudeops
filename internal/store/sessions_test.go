package store_test

import (
	"context"
	"testing"
	"time"

	"claudeops/internal/domain"
	"claudeops/internal/store/storetest"
)

func TestUpsertAndListSessions(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	now := time.Now()
	for i, sess := range []domain.Session{
		{SessionID: "a", ProjectDir: "p", FilePath: "/tmp/a.jsonl", FileSizeBytes: 1, LastActivity: now.Add(-2 * time.Hour)},
		{SessionID: "b", ProjectDir: "p", FilePath: "/tmp/b.jsonl", FileSizeBytes: 1, LastActivity: now.Add(-1 * time.Hour)},
		{SessionID: "c", ProjectDir: "p", FilePath: "/tmp/c.jsonl", FileSizeBytes: 1, LastActivity: now},
	} {
		if err := s.UpsertSession(ctx, sess); err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}

	got, err := s.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 sessions, got %d", len(got))
	}
	// Ordered by last_activity DESC.
	if got[0].SessionID != "c" || got[2].SessionID != "a" {
		t.Fatalf("wrong order: %v %v %v", got[0].SessionID, got[1].SessionID, got[2].SessionID)
	}
}

func TestUpsertSessionReplaces(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	base := domain.Session{SessionID: "x", ProjectDir: "p", FilePath: "/tmp/x.jsonl", FileSizeBytes: 1, LastActivity: time.Now()}
	if err := s.UpsertSession(ctx, base); err != nil {
		t.Fatal(err)
	}
	base.LastUserPreview = "hello"
	base.NumEvents = 42
	if err := s.UpsertSession(ctx, base); err != nil {
		t.Fatal(err)
	}
	all, _ := s.ListSessions(ctx)
	if len(all) != 1 {
		t.Fatalf("upsert should replace, got %d rows", len(all))
	}
	if all[0].LastUserPreview != "hello" || all[0].NumEvents != 42 {
		t.Fatalf("upsert didn't update: %+v", all[0])
	}
}

func TestListSessionsByWorkspaceAndUnbound(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	epicID := storetest.SeedEpic(t, s, "ep", "/tmp/repo")
	wsID := storetest.SeedWorkspace(t, s, epicID, "ws", "branch", "/tmp/repo/wt")

	bound := domain.Session{SessionID: "b1", ProjectDir: "p", FilePath: "/tmp/b1.jsonl", FileSizeBytes: 1, LastActivity: time.Now(), Cwd: "/tmp/repo/wt", WorkspaceID: &wsID}
	unbound := domain.Session{SessionID: "u1", ProjectDir: "p", FilePath: "/tmp/u1.jsonl", FileSizeBytes: 1, LastActivity: time.Now(), Cwd: "/tmp/repo/elsewhere"}
	if err := s.UpsertSession(ctx, bound); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSession(ctx, unbound); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListSessionsByWorkspace(ctx, wsID)
	if err != nil {
		t.Fatalf("by-ws: %v", err)
	}
	if len(got) != 1 || got[0].SessionID != "b1" {
		t.Fatalf("wrong by-ws result: %+v", got)
	}

	un, err := s.ListUnboundSessionsForRepo(ctx, "/tmp/repo")
	if err != nil {
		t.Fatalf("unbound: %v", err)
	}
	if len(un) != 1 || un[0].SessionID != "u1" {
		t.Fatalf("wrong unbound result: %+v", un)
	}
}
