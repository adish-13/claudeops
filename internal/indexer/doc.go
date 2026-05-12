// Package indexer crawls Claude Code's on-disk session logs and indexes their
// metadata into the SQLite store.
//
// Indexer.Run blocks on a ticker (default 5s) and on each tick walks
// `<root>/*/*.jsonl`. For each file it reads the head event for session
// metadata, tail-reads to extract the last user/assistant text and an event
// count, and UpsertSession's the row into the store.
//
// On every tick the indexer also re-binds sessions to workspaces by
// longest-prefix matching the session's `cwd` against every known
// `workspace.worktree_path`. Sessions with no matching workspace are left
// unbound and surface on the epic page as "started in this worktree but
// not tied to a workspace yet".
//
// The indexer polls rather than using fsnotify; JSONL files are append-only
// and small enough that a 5s tick is fine.
package indexer
