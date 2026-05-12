// Package store is the SQLite persistence layer.
//
// It is the only package in claudeops that touches SQL. Other packages get
// data through the methods on *Store, not by reaching into the *sql.DB
// (Store.DB() exists for the /api/debug introspector — don't use it from
// regular handlers).
//
// The schema lives in store.go and is created on Open with `CREATE TABLE
// IF NOT EXISTS`. There is no migration framework yet; for additive changes,
// place an `ALTER TABLE … ADD COLUMN IF NOT EXISTS …` next to the create.
//
// Per-resource queries are split out:
//
//   - epics.go      — CreateEpic, GetEpicBySlug, ListEpics, UpdateEpicContext, ArchiveEpic
//   - workspaces.go — CreateWorkspace, GetWorkspaceBySlug, ListWorkspacesByEpic, AllWorkspaces, UpdateWorkspacePR, ArchiveWorkspace
//   - sessions.go   — UpsertSession, ListSessions, ListSessionsByWorkspace, ListUnboundSessionsForRepo, GetSession
//
// SQLite is opened in WAL mode with a busy timeout and foreign keys enabled
// so the indexer goroutine and HTTP handlers can read/write concurrently.
//
// ErrNotFound is returned by the Get* helpers when no row matches.
package store
