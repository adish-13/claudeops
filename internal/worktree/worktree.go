package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepo returns nil if repoPath is the working tree of a git repository.
func IsGitRepo(repoPath string) error {
	if repoPath == "" {
		return errors.New("repoPath is empty")
	}
	if _, err := os.Stat(repoPath); err != nil {
		return fmt.Errorf("path does not exist: %s", repoPath)
	}
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--git-dir")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("not a git repository: %s (%s)", repoPath, strings.TrimSpace(string(out)))
	}
	return nil
}

// Add creates a new git worktree at targetPath, on a new branch branchName,
// branched from baseBranch. Returns an error if the path already exists or
// the git command fails.
func Add(repoPath, targetPath, branchName, baseBranch string) error {
	if repoPath == "" {
		return errors.New("repoPath is required")
	}
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("worktree path already exists: %s", targetPath)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	args := []string{"-C", repoPath, "worktree", "add", "-b", branchName, targetPath, baseBranch}
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// SuggestPath returns the conventional worktree path for a given epic+workspace.
// Layout: <root>/<repo-basename>/<epic-slug>/<workspace-slug>
func SuggestPath(root, repoPath, epicSlug, workspaceSlug string) string {
	return filepath.Join(root, filepath.Base(strings.TrimRight(repoPath, "/")), epicSlug, workspaceSlug)
}
