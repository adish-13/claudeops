package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"claudeops/internal/domain"
	"claudeops/internal/server"
	"claudeops/internal/store"
	"claudeops/internal/store/storetest"
	"claudeops/internal/terminals"
)

// liveServer wires up the real production handler against an httptest server,
// so any route changes are exercised here before they ever ship.
func liveServer(t *testing.T, s *store.Store) *httptest.Server {
	t.Helper()
	srv := server.New(s, terminals.NewManager("echo"), t.TempDir())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestSidebarEmpty(t *testing.T) {
	s := storetest.New(t)
	ts := liveServer(t, s)

	resp, err := http.Get(ts.URL + "/api/sidebar")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		Epics     []any  `json:"epics"`
		IndexedAt string `json:"indexed_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Epics) != 0 {
		t.Fatalf("expected empty sidebar, got %d epics", len(body.Epics))
	}
	if body.IndexedAt == "" {
		t.Fatalf("indexed_at should be populated")
	}
}

func TestHomeWithSession(t *testing.T) {
	s := storetest.New(t)
	if err := s.UpsertSession(context.Background(), domain.Session{
		SessionID: "sx", ProjectDir: "p", FilePath: "/tmp/sx.jsonl", FileSizeBytes: 1,
	}); err != nil {
		t.Fatal(err)
	}
	ts := liveServer(t, s)

	resp, err := http.Get(ts.URL + "/api/home")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Total    int `json:"total_sessions"`
		Projects int `json:"project_count"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Total != 1 || body.Projects != 1 {
		t.Fatalf("got total=%d projects=%d", body.Total, body.Projects)
	}
}

func TestCreateEpicValidation(t *testing.T) {
	s := storetest.New(t)
	ts := liveServer(t, s)

	resp, err := http.Post(ts.URL+"/api/epics", "application/json", strings.NewReader(`{"name":""}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing required fields, got %d", resp.StatusCode)
	}
}

func TestSessionsList(t *testing.T) {
	s := storetest.New(t)
	if err := s.UpsertSession(context.Background(), domain.Session{
		SessionID: "s1", ProjectDir: "p", FilePath: "/tmp/s1.jsonl", FileSizeBytes: 1,
	}); err != nil {
		t.Fatal(err)
	}
	ts := liveServer(t, s)

	resp, err := http.Get(ts.URL + "/api/sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		Sessions []map[string]any `json:"sessions"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(body.Sessions))
	}
}

func TestNonAPIPathsServeSPA(t *testing.T) {
	// SPA fallback must succeed even when there's no dist/ embedded,
	// returning the placeholder error message rather than 404.
	s := storetest.New(t)
	ts := liveServer(t, s)
	resp, err := http.Get(ts.URL + "/some/client/route")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// Either 200 (dist embedded) or 500 (dist missing — clear error).
	if resp.StatusCode != 200 && resp.StatusCode != 500 {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}
