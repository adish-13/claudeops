// Package git wraps the read-only git operations claudeops needs to render
// workspace diffs.
//
// It shells out to the system `git` binary via os/exec; we deliberately
// don't import a git library — the set of operations is small (numstat,
// untracked listing) and the cost of a vendored git is much larger than
// the cost of a few exec calls per page load.
//
// The single exported function is Diff, which returns aggregate Summary
// counts and per-file domain.DiffStat entries that combine:
//
//  1. Committed changes vs the epic's base branch (`git diff --numstat baseBranch...HEAD`).
//  2. Uncommitted changes (`git diff --numstat HEAD`).
//  3. Untracked files (`git ls-files --others --exclude-standard`).
//
// Worktree-mutating git operations (creating worktrees, switching branches)
// live in package worktree, not here.
package git
