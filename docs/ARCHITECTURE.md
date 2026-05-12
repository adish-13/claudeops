# Architecture

claudeops is a single Go binary that embeds a React SPA and serves a JSON API.
Three things run inside the process:

1. An **HTTP server** that serves the SPA and a JSON API.
2. A **terminal manager** that owns a pty per workspace and bridges it to the browser over WebSocket.
3. An **indexer** goroutine that periodically scans `~/.claude/projects/*/*.jsonl` and indexes session metadata into SQLite.

All persistent state lives in a single SQLite file (`~/.claude/claudeops.db` by default).
All git state lives in real worktrees on disk — claudeops shells out to `git`, it does not vendor any git library.

## Entrypoint status

There is currently no `cmd/claudeops/main.go` in the repo. The Makefile, README, and
`go build` target all assume one exists. Until it is added, the project does not
build end-to-end; you can still `go build ./internal/...` to type-check the libraries.

The entrypoint should:

1. Parse the four flags documented in [README.md](../README.md#flags).
2. `store.Open(dbPath)` and `defer store.Close()`.
3. `tm := terminals.NewManager("claude")` (or whatever CLI to spawn).
4. `idx := indexer.New(projectsDir, store)` and `go idx.Run(ctx, scanInterval)`.
5. `srv := server.New(store, tm, worktreeRoot)` and `srv.ListenAndServe(addr)`.

`worktreeRoot` is the parent directory under which new worktrees are created
(`worktree.SuggestPath` builds `<root>/<repo>/<epic>/<workspace>`). A reasonable
default is `~/worktrees/claudeops`.

## Package layout

```
internal/
  domain/      # pure types — leaf, no internal deps
  store/       # SQLite persistence — depends on domain
  git/         # git diff stats — depends on domain
  transcript/  # JSONL → Message stream — depends on domain
  terminals/   # pty multiplexer — no internal deps
  worktree/    # `git worktree add` wrapper — no internal deps
  launcher/    # macOS osascript launcher — no internal deps
  indexer/     # periodic scanner — depends on domain + store
  server/      # JSON API + SPA + WS — depends on everything above
```

### Dependency direction

```
                          domain
                            ▲
        ┌──────────┬────────┼────────┬──────────────┐
        │          │        │        │              │
       git    transcript  store   indexer        (and used by server)
                            ▲        │
                            └────────┘
                            
        terminals    worktree    launcher        (no internal deps)
                            ▲
                            │
                          server  ◄── depends on every other internal package
```

Rule of thumb: anything that knows about HTTP belongs in `server`. Anything that
knows about SQL belongs in `store`. Anything that shells out to `git` belongs in
`git` or `worktree`. The other packages should stay free of those concerns.

## Data model

Three tables in SQLite. Foreign keys are enabled. WAL mode is on with a
busy timeout so the indexer goroutine and the request handlers can both read/write
without stepping on each other.

### `epics`

A long-running theme of work, scoped to one repo and base branch. The user
attaches free-form context markdown that they want every Claude session in this
epic to start with.

Key columns: `id`, `slug` (unique), `name`, `description`, `repo_path`,
`base_branch`, `context_md`, `created_at`, `archived_at`.

### `workspaces`

A single branch + worktree under an epic. One workspace owns one git worktree on
disk and one PR URL (optional). The embedded terminal session is keyed by
workspace id.

Key columns: `id`, `epic_id` (FK), `slug` (unique within epic), `name`,
`branch_name`, `worktree_path`, `pr_url`, `created_at`, `archived_at`.

### `sessions`

A single Claude Code session, derived from one `~/.claude/projects/<proj>/<uuid>.jsonl`
file. The indexer upserts these on every scan tick. A session is bound to a
workspace by matching `cwd` against `workspace.worktree_path` (longest-prefix wins).
Bound sessions show up under their workspace in the UI; unbound ones show up on
the epic page so the user can see "you started a session in a worktree but
haven't tied it to a workspace yet."

Key columns: `session_id` (uuid, PK), `project_dir`, `cwd`, `git_branch`,
`model`, `version`, `last_activity`, `last_user_preview`, `last_assistant_text`,
`file_path`, `file_size_bytes`, `num_events`, `workspace_id` (nullable FK).

## Request flow: viewing a workspace

1. Browser hits `/epics/foo/workspaces/bar` — Vite (dev) or the embedded SPA (prod) loads the React app.
2. React's `WorkspacePage` calls `GET /api/epics/foo/workspaces/bar`.
3. `server.apiGetWorkspace` looks up the workspace, calls `git.Diff(worktreePath, baseBranch)` for diff stats and per-file changes, calls `store.ListSessionsByWorkspace` for bound sessions, and returns it all as JSON.
4. `WorkspacePage` renders the diff panel and mounts `<Terminal workspaceId={...} />`.
5. `<Terminal>` opens `ws://.../ws/terminal/<wsid>`.
6. `server.handleTerminalWS` looks up the `terminals.Session` for that workspace; if missing, returns an error so the client knows to call `POST .../term/start` first.
7. Once attached, the server sends the ring-buffer backlog (last ~64KB) as binary frames, then streams new pty output. Client sends `{type:"input", data}` for keystrokes and `{type:"resize", rows, cols}` on terminal resize.

## Request flow: creating a workspace

1. User submits the New Workspace form. React calls `GET /api/epics/foo/workspaces/suggest` first to auto-fill a branch name and worktree path (`worktree.SuggestPath`).
2. On submit, `POST /api/epics/foo/workspaces`.
3. `apiCreateWorkspace` calls `worktree.Add(repoPath, targetPath, branchName, baseBranch)` which shells out to `git worktree add -b`.
4. On success, `store.CreateWorkspace` inserts the row.
5. The next indexer tick will start binding any sessions whose `cwd` matches the new worktree path.

## Indexer

`indexer.Run(ctx, every)` is a simple polling loop. Each tick:

1. Walks `<root>/*/<uuid>.jsonl`.
2. For each file, reads the head for the session metadata event and tail-reads to extract `last_user_preview`, `last_assistant_text`, `last_activity`, and event count.
3. `UpsertSession` writes/updates the row.
4. `AllWorkspaces` is loaded once per tick; sessions are bound to workspaces by longest-prefix match on `cwd`.

The polling interval defaults to 5s. There is no fsnotify-style watcher — JSONL
files are append-only and short-lived, so polling has been good enough.

## Terminal multiplexer

`terminals.Manager` keeps one `Session` per workspace id. A `Session` owns:

- The `*os.File` returned by `pty.Start(cmd)`.
- A `sync.Mutex`-protected list of subscriber channels.
- A 64KB ring buffer of recent output (`backlog`).

A read goroutine pumps pty output → backlog → every subscriber. Writes (keyboard
input from the browser) go directly to the pty. Resizes go through `pty.Setsize`.

Subscriber channels have a small buffer; if a subscriber falls behind, its writes
drop rather than block the read goroutine. The backlog ring is the safety net —
on reconnect, the client gets the last 64KB so it can re-render context without
the pty restarting.

## Embedded SPA

`internal/server/spa.go` does `//go:embed all:dist` (relative to that file, so it
picks up `internal/server/dist/` produced by `vite build`). The SPA handler
serves files normally and falls back to `index.html` for unknown routes so that
React Router URLs survive a hard refresh.

If the binary is built without first running `npm run build`, the embedded
filesystem is empty and the SPA handler returns a 500 with a clear message
telling you what to do.

## What is intentionally not here

- **No ORM.** `database/sql` + handwritten queries. There are ~10 of them.
- **No git library.** `os/exec` `git` calls. The set of git operations needed is small (`diff --numstat`, `ls-files --others`, `worktree add`, `rev-parse --git-dir`).
- **No fsnotify.** The indexer polls.
- **No auth.** The server binds to `127.0.0.1` by default. Don't expose it.
- **No tests.** This is a known gap; see CONTRIBUTING.md.
