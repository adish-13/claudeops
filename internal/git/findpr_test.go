package git_test

import (
	"testing"

	"claudeops/internal/git"
)

// TestFindPREmptyInputs returns "" without invoking gh.
func TestFindPREmptyInputs(t *testing.T) {
	url, err := git.FindPR("", "branch")
	if err != nil || url != "" {
		t.Fatalf("empty path: got (%q, %v); want empty", url, err)
	}
	url, err = git.FindPR("/tmp", "")
	if err != nil || url != "" {
		t.Fatalf("empty branch: got (%q, %v); want empty", url, err)
	}
}

// TestFindPRMissingRemote: in a tempdir without a github remote, gh either
// isn't installed (FindPR short-circuits) or it errors talking to the API
// (FindPR returns ""). Either way the contract is "no PR, no error".
func TestFindPRMissingRemote(t *testing.T) {
	dir := initTinyRepoForFindPR(t)
	url, err := git.FindPR(dir, "feature/x")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if url != "" {
		t.Fatalf("expected empty URL for repo with no PRs, got %q", url)
	}
}

// initTinyRepoForFindPR is a tiny git init helper local to this test file —
// keeps the package boundary clean (test pkg can't reach into the server pkg
// helper of the same name).
func initTinyRepoForFindPR(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, ".", "init", "-b", "master", dir)
	return dir
}
