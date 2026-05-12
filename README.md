# claudeops

Local dashboard for managing many Claude Code sessions: epics → workspaces (git worktrees) → sessions.
React + Vite + Tailwind frontend, Go backend, embedded xterm.js terminal per workspace.

> **Heads up:** the build entrypoint at `cmd/claudeops/main.go` is referenced by the
> Makefile but does not exist in the repo yet. `make build` will fail at the `go build`
> step until it is added. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#entrypoint-status) for what
> the entrypoint needs to wire up.

## Prerequisites

- Go 1.22+ (go.mod declares 1.25)
- Node 18+ and npm
- macOS for the "Launch in iTerm/Terminal" feature (uses `osascript`); the rest works cross-platform
- `claude` CLI on `$PATH` for the embedded terminal to spawn sessions
- `git` on `$PATH` (worktree creation + diff stats)

## Build

```sh
make build      # frontend (vite → internal/server/dist) + backend (go) → ./claudeops
./claudeops     # listens on http://127.0.0.1:7777
```

`make build` runs two steps:

1. `cd web && npm run build` — Vite emits to `internal/server/dist/`.
2. `go build -o claudeops ./cmd/claudeops` — Go embeds `internal/server/dist/` via `//go:embed`.

The two steps must happen in this order: the Go build will refuse to embed an empty/missing `dist/`.

## Dev mode (hot-reload frontend)

```sh
./claudeops &                # backend on :7777
cd web && npm run dev        # vite on :5173 with HMR + API/WS proxy
```

Open http://localhost:5173 — Vite proxies `/api/*` and `/ws/*` to the Go server (see `web/vite.config.ts`).

## Flags

| Flag        | Default                       | Purpose                                       |
|-------------|-------------------------------|-----------------------------------------------|
| `-projects` | `~/.claude/projects`          | Root scanned by the indexer for session JSONL |
| `-db`       | `~/.claude/claudeops.db`      | SQLite database path                          |
| `-addr`     | `127.0.0.1:7777`              | HTTP listen address                           |
| `-scan`     | `5s`                          | Indexer rescan interval                       |

(These are the flags the entrypoint should expose; see ARCHITECTURE.md.)

## Layout

```
cmd/claudeops/             — main() (TODO: not yet committed)
internal/
  domain/                  — pure types (Epic, Workspace, Session, Message, DiffStat)
  store/                   — SQLite (store.go + epics.go + workspaces.go + sessions.go)
  git/                     — git diff stats + per-file list
  transcript/              — JSONL → message stream
  terminals/               — pty lifecycle + multiplexer + 64KB ring buffer
  indexer/                 — crawls ~/.claude/projects/*/*.jsonl on a tick
  worktree/                — wraps `git worktree add`
  launcher/                — osascript → iTerm/Terminal (macOS only)
  server/                  — JSON API (api_*.go), WS terminal bridge, embedded SPA
web/                       — React + Vite + TypeScript + Tailwind
  src/components/          — Sidebar, Layout, Terminal, ui.tsx primitives
  src/pages/               — Home, Epic, Workspace, Sessions, Transcript, Debug, NewEpic, NewWorkspace
  src/lib/                 — api.ts (typed client + SWR hooks), utils.ts
docs/
  ARCHITECTURE.md          — package responsibilities, data model, request flow
CLAUDE.md                  — guidance for Claude Code when working on this repo
CONTRIBUTING.md            — dev setup, coding conventions, how to add a route/page
```

## Routes

JSON API (all under `/api/`):

| Method | Path                                                  | Purpose                                       |
|--------|-------------------------------------------------------|-----------------------------------------------|
| GET    | `/api/sidebar`                                        | Epics + workspaces with diff stats / counts   |
| GET    | `/api/home`                                           | Totals for the home page                      |
| GET    | `/api/sessions`                                       | All indexed sessions                          |
| GET    | `/api/sessions/{id}`                                  | Session metadata + rendered transcript        |
| GET    | `/api/debug`                                          | DB path, file size, row counts, recent rows   |
| POST   | `/api/epics`                                          | Create epic                                   |
| GET    | `/api/epics/{slug}`                                   | Epic detail (workspaces, unbound sessions)    |
| POST   | `/api/epics/{slug}/context`                           | Update epic context markdown                  |
| POST   | `/api/epics/{slug}/archive`                           | Archive epic                                  |
| GET    | `/api/epics/{slug}/workspaces/suggest`                | Auto-fill branch + worktree path              |
| POST   | `/api/epics/{slug}/workspaces`                        | Create workspace (also creates worktree)      |
| GET    | `/api/epics/{slug}/workspaces/{wsslug}`               | Workspace detail (diff, sessions, terminal)   |
| POST   | `/api/epics/{slug}/workspaces/{wsslug}/launch`        | Launch external iTerm/Terminal (macOS)        |
| POST   | `/api/epics/{slug}/workspaces/{wsslug}/pr`            | Save PR URL                                   |
| POST   | `/api/epics/{slug}/workspaces/{wsslug}/archive`       | Archive workspace                             |
| POST   | `/api/epics/{slug}/workspaces/{wsslug}/term/start`    | Spawn embedded `claude` pty                   |
| POST   | `/api/epics/{slug}/workspaces/{wsslug}/term/kill`     | Stop embedded pty                             |

WebSocket: `GET /ws/terminal/{wsid}` — bridges xterm.js ↔ pty (binary frames for output;
JSON control frames for `resize` and `input`).

Anything that doesn't match `/api/*` or `/ws/*` falls through to the embedded SPA, which
serves `index.html` for unknown paths so client-side routing works on hard refresh.

## Troubleshooting

- **`web/dist not embedded`** when hitting `/`: run `npm run build` in `web/` then rebuild Go.
- **Empty terminal / "claude not found"**: ensure `claude` is on the PATH of the user running the server.
- **No sessions show up**: check `-projects` points at the right dir; the indexer scans `*/*.jsonl` under it.
- **DB locked errors**: SQLite is in WAL mode with a busy timeout, but only one server process should hold the DB at a time.

## Mac app (planned)

Wrap the React app + Go server in Tauri to ship a real `.app`. ~200 LOC of Tauri config; the Go backend stays the same.
