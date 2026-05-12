package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"claudeops/internal/git"
)

// TestDiffOnRealRepo creates a tiny git repo in a tempdir, makes a worktree on
// a new branch, modifies + adds files, and verifies the Diff summary matches.
func TestDiffOnRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	mustGit(t, ".", "init", "-b", "master", repo)

	// Initial commit on master with a simple file.
	must(t, os.WriteFile(filepath.Join(repo, "a.txt"), []byte("one\ntwo\nthree\n"), 0o644))
	mustGit(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "add", "a.txt")
	mustGit(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "init")

	// New worktree on a feature branch.
	mustGit(t, repo, "worktree", "add", "-b", "feature", wt, "master")

	// Modify a.txt (remove 1 line, add 2 lines) and add a brand-new file.
	must(t, os.WriteFile(filepath.Join(wt, "a.txt"), []byte("one\ntwo\nfour\nfive\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(wt, "b.txt"), []byte("brand new\n"), 0o644))

	sum, files, err := git.Diff(wt, "master")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if sum.FilesChanged != 2 {
		t.Fatalf("expected 2 files changed, got %d (files=%+v)", sum.FilesChanged, files)
	}
	// a.txt: -1 +2 (numstat counts lines added/removed); b.txt is untracked, status "??".
	var sawA, sawB bool
	for _, f := range files {
		switch f.Path {
		case "a.txt":
			sawA = true
			if f.Added < 1 || f.Removed < 1 {
				t.Errorf("a.txt unexpected stats: %+v", f)
			}
		case "b.txt":
			sawB = true
			if !strings.Contains(f.Status, "?") {
				t.Errorf("b.txt should be untracked, got status=%q", f.Status)
			}
		}
	}
	if !sawA || !sawB {
		t.Fatalf("missing expected files: sawA=%v sawB=%v", sawA, sawB)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" && dir != "." {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, string(out))
	}
}
