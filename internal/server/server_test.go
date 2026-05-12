package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"claudeops/internal/domain"
	"claudeops/internal/server"
	"claudeops/internal/store"
	"claudeops/internal/store/storetest"
	"claudeops/internal/terminals"
)

// initTinyRepo creates a minimal git repo in t.TempDir and returns its path.
// Used by tests that need git.Diff to actually run against something.
func initTinyRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "master", dir},
		{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init"},
	} {
		out, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
}

func readAll(r io.Reader) ([]byte, error) { return io.ReadAll(r) }

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

// TestWorkspaceDetailJSONFieldNames is a regression test against the v4 bug
// where git.Summary and domain.DiffStat had no JSON tags, so the API returned
// PascalCase keys (FilesChanged, Path) but the React client read snake_case
// (files_changed, path) and rendered everything as undefined.
//
// The check is deliberately string-based on the raw response bytes so any
// future loss of the json:"..." tags is caught immediately, regardless of
// whether the Go test struct decode silently tolerates either casing.
func TestWorkspaceDetailJSONFieldNames(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	// We need a real git repo as the worktree path, otherwise git.Diff returns
	// an error and the response body lacks any per-file structure to inspect.
	// Use t.TempDir + a tiny init so the test is self-contained.
	worktree := initTinyRepo(t)
	epicID := storetest.SeedEpic(t, s, "ep", worktree)
	storetest.SeedWorkspace(t, s, epicID, "ws", "branch", worktree)
	// Re-fetch epic so we use the actual repo path stored in the DB.
	epic, _ := s.GetEpicBySlug(ctx, "ep")
	if epic.RepoPath == "" {
		t.Fatal("seed didn't persist repo_path")
	}

	ts := liveServer(t, s)
	resp, err := http.Get(ts.URL + "/api/epics/ep/workspaces/ws")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := readAll(resp.Body)
	bodyStr := string(body)

	for _, want := range []string{`"files_changed"`, `"diff"`, `"files"`} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("response missing expected key %s\nbody: %s", want, bodyStr)
		}
	}
	for _, badPascal := range []string{`"FilesChanged"`, `"Added":`, `"Removed":`, `"Path":`, `"Status":`} {
		if strings.Contains(bodyStr, badPascal) {
			t.Errorf("response leaked PascalCase key %s — JSON tags are missing again", badPascal)
		}
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
