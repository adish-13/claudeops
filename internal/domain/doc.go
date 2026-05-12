// Package domain holds the pure data types shared across claudeops.
//
// It is the leaf of the package graph: no SQL, no HTTP, no git, no I/O.
// Adding a dependency here means every other package picks it up — keep
// this package boring.
//
// The four core types are:
//
//   - Epic — a long-running theme of work, scoped to one repo and base branch.
//   - Workspace — a single branch + git worktree under an epic.
//   - Session — a single Claude Code session, derived from one *.jsonl file.
//   - Message — one rendered turn from a session transcript.
//   - DiffStat — per-file added/removed line counts from `git diff --numstat`.
package domain
