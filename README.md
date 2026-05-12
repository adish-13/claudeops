# claudeops

Local dashboard for managing many Claude Code sessions: epics → workspaces (worktrees) → sessions.
React + Vite + Tailwind frontend, Go backend, embedded xterm.js terminal per workspace.

## Build

```sh
make build      # frontend (vite) + backend (go) → ./claudeops
./claudeops     # listens on http://127.0.0.1:7777
```

Requires Go 1.22+ and Node 18+.

## Dev mode (hot-reload frontend)

```sh
./claudeops &                # backend on :7777
cd web && npm run dev        # vite on :5173 with HMR + API proxy
```

Open http://localhost:5173 — API and WebSocket calls are proxied to the Go server.

## Layout

```
cmd/claudeops/             — main()
internal/
  domain/                  — pure types (Epic, Workspace, Session, Message, DiffStat)
  store/                   — SQLite (epics.go, workspaces.go, sessions.go)
  git/                     — git diff stats + per-file list
  transcript/              — JSONL → message stream
  terminals/               — pty lifecycle + multiplexer + ring buffer
  indexer/                 — crawls ~/.claude/projects/*/*.jsonl every 5s
  worktree/                — wraps git worktree add
  launcher/                — osascript → iTerm/Terminal
  server/                  — JSON API (api_*.go), WS terminal, embedded SPA
web/                       — React + Vite + TypeScript + Tailwind
  src/components/          — ui.tsx (Button/Card/Pill/...), Sidebar, Layout, Terminal
  src/pages/               — Home, Epic, Workspace, Sessions, Transcript, Debug, NewEpic, NewWorkspace
  src/lib/                 — api.ts (typed client), utils.ts
```

## Routes

JSON API:
- `GET /api/sidebar` — epics + workspaces with diff stats and session counts
- `GET /api/home` — totals
- `GET /api/sessions` and `/api/sessions/{id}` — list + transcript
- `POST /api/epics` and `GET /api/epics/{slug}` — create/get epic
- `POST /api/epics/{slug}/context` and `/archive`
- `GET /api/epics/{slug}/workspaces/suggest` — auto-fills branch + path
- `POST /api/epics/{slug}/workspaces` and `GET .../workspaces/{wsslug}`
- `POST .../launch` (iTerm), `.../term/start`, `.../term/kill`, `.../pr`, `.../archive`
- `GET /api/debug` — DB inspector

WebSocket: `/ws/terminal/{wsid}` — bridges xterm.js ↔ pty.

## Mac app (planned)

Wrap the React app + Go server in Tauri to ship a real `.app`. ~200 LOC of Tauri config; the Go backend stays the same.
