// Package worktree wraps the small set of `git worktree` operations
// claudeops performs when creating workspaces.
//
// Three functions:
//
//   - IsGitRepo validates that a path is the root of a git repo (used to
//     pre-check the New Epic form so we fail with a clear message instead
//     of a confusing `git worktree add` error later).
//   - Add shells out to `git worktree add -b <branch> <path> <baseBranch>`
//     to create a new worktree on a fresh branch.
//   - SuggestPath builds the conventional path for a new worktree:
//     `<root>/<repo-basename>/<epic-slug>/<workspace-slug>`.
//
// Read-only diff operations live in package git, not here.
package worktree
