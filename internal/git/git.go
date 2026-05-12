// Package git wraps the small set of git operations the UI needs:
// summary stats and per-file diff status for a worktree against its base branch.
package git

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"claudeops/internal/domain"
)

// jsonUnmarshal is aliased so the FindPR function can stay near the top of
// the file without an inline import.
var jsonUnmarshal = json.Unmarshal

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

// FindPR returns the URL of a pull request whose source branch matches
// `branch` in the repo at worktreePath. Returns ("", nil) if:
//   - the `gh` CLI isn't installed,
//   - the user isn't authenticated,
//   - no PR exists for that branch.
//
// When multiple PRs exist for the branch, an OPEN one is preferred; otherwise
// the most recently created PR wins. This matches how a developer would
// naturally interpret "the PR for this branch."
func FindPR(worktreePath, branch string) (string, error) {
	if branch == "" || worktreePath == "" {
		return "", nil
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return "", nil // gh not installed — silently skip
	}
	cmd := exec.Command("gh", "pr", "list",
		"--head", branch,
		"--json", "url,state,createdAt",
		"--state", "all",
		"--limit", "20",
	)
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		// Either no remote, no auth, or branch not pushed yet — all benign.
		return "", nil
	}
	type prRow struct {
		URL       string `json:"url"`
		State     string `json:"state"`
		CreatedAt string `json:"createdAt"`
	}
	var rows []prRow
	if err := jsonUnmarshal(out, &rows); err != nil || len(rows) == 0 {
		return "", nil
	}
	// Prefer OPEN; otherwise pick the latest by createdAt (already sorted desc by gh).
	for _, p := range rows {
		if strings.EqualFold(p.State, "OPEN") {
			return p.URL, nil
		}
	}
	return rows[0].URL, nil
}

func runGit(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
