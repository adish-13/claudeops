// Package git wraps the small set of git operations the UI needs:
// summary stats and per-file diff status for a worktree against its base branch.
package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"claudeops/internal/domain"
)

// Summary returns aggregate added/removed line counts for worktreePath
// against baseBranch, treating uncommitted changes as part of the diff.
//
// JSON tags are required: this type is returned directly by the HTTP API
// (see workspaceDetailJSON.Diff) and the React client expects snake_case.
type Summary struct {
	FilesChanged int `json:"files_changed"`
	Added        int `json:"added"`
	Removed      int `json:"removed"`
}

// Diff returns the combined diff (committed-vs-base + uncommitted) summary
// and per-file list. baseBranch is something like "master".
func Diff(worktreePath, baseBranch string) (Summary, []domain.DiffStat, error) {
	if worktreePath == "" {
		return Summary{}, nil, fmt.Errorf("worktreePath required")
	}
	// Per-file numstat against base...HEAD covers committed changes.
	committed, err := runNumstat(worktreePath, baseBranch+"...HEAD")
	if err != nil {
		return Summary{}, nil, err
	}
	// Uncommitted (working tree + staged) against HEAD.
	uncommitted, err := runNumstat(worktreePath, "HEAD")
	if err != nil {
		// Empty repo or no HEAD yet — keep committed and continue.
		uncommitted = nil
	}
	// Untracked files: list them as +N adds with status=??.
	untracked, err := runUntracked(worktreePath)
	if err == nil {
		uncommitted = append(uncommitted, untracked...)
	}

	merged := mergeStats(committed, uncommitted)
	var sum Summary
	for _, d := range merged {
		sum.FilesChanged++
		sum.Added += d.Added
		sum.Removed += d.Removed
	}
	return sum, merged, nil
}

func runNumstat(worktreePath, ref string) ([]domain.DiffStat, error) {
	out, err := runGit(worktreePath, "diff", "--numstat", ref)
	if err != nil {
		return nil, err
	}
	var stats []domain.DiffStat
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		added, _ := strconv.Atoi(fields[0])
		removed, _ := strconv.Atoi(fields[1])
		stats = append(stats, domain.DiffStat{
			Path:    fields[2],
			Status:  "M",
			Added:   added,
			Removed: removed,
		})
	}
	return stats, nil
}

func runUntracked(worktreePath string) ([]domain.DiffStat, error) {
	out, err := runGit(worktreePath, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var stats []domain.DiffStat
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		stats = append(stats, domain.DiffStat{Path: line, Status: "??"})
	}
	return stats, nil
}

// mergeStats combines two diff lists, summing per-path counts and preferring
// the more specific status when both sides report the same file.
func mergeStats(a, b []domain.DiffStat) []domain.DiffStat {
	idx := map[string]*domain.DiffStat{}
	order := []string{}
	add := func(d domain.DiffStat) {
		if existing, ok := idx[d.Path]; ok {
			existing.Added += d.Added
			existing.Removed += d.Removed
			if d.Status == "??" || existing.Status == "" {
				existing.Status = d.Status
			}
			return
		}
		copy := d
		idx[d.Path] = &copy
		order = append(order, d.Path)
	}
	for _, d := range a {
		add(d)
	}
	for _, d := range b {
		add(d)
	}
	out := make([]domain.DiffStat, 0, len(order))
	for _, p := range order {
		out = append(out, *idx[p])
	}
	return out
}

func runGit(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
