# Notes for Claude

This file is loaded into your context when you work in this repo. Keep it short
and concrete.

## What this project is

A single Go binary that embeds a React SPA and serves a JSON API for managing
many Claude Code sessions. Three things run in-process: HTTP server, terminal
manager (pty-per-workspace, multiplexed over WebSocket), and an indexer
goroutine that scans `~/.claude/projects/*/*.jsonl`.

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) before making non-trivial
changes — it covers the package boundaries, data model, and request flow.

## Build / run

```sh
make build      # vite → internal/server/dist, then go build → ./claudeops
./claudeops     # http://127.0.0.1:7777
```

For dev: backend with `go run ./cmd/claudeops` in one terminal, `cd web && npm run dev`
in another. Vite proxies `/api/*` and `/ws/*` to :7777.

## Known gotchas

- **There is no `cmd/claudeops/main.go` in the repo.** The Makefile and README
  reference it but it has never been committed. `make build` fails at `go build`.
  If you're asked to "fix the build" or "run it", this is almost certainly why.
  ARCHITECTURE.md documents what the entrypoint needs to wire up.
- **No tests.** Don't claim "tests pass" unless you wrote some. `go vet ./...`
  and a `go build ./internal/...` are the only automated checks today.
- **The frontend embed is conditional on `npm run build` having run.** If you
  rebuild Go without rebuilding the SPA, hitting `/` returns a 500 with a clear
  message. This is intentional, not a bug — fix it by running `make web` first.
- **macOS-only:** the "Launch in iTerm/Terminal" path uses `osascript`. Don't
  break it on macOS while making other paths cross-platform; do guard those
  calls with `runtime.GOOS == "darwin"` if you add new ones.

## Package boundaries (don't blur these)

- `internal/domain` — pure types, no I/O, no SQL, no HTTP. Leaf package.
- `internal/store` — only place that touches SQLite.
- `internal/git` and `internal/worktree` — only places that shell out to `git`.
- `internal/server` — only place that knows about HTTP / JSON / WebSocket.
- `internal/terminals` — pty multiplexer; doesn't know about workspaces beyond an opaque `int64` id.
- `internal/indexer` — the only consumer of the on-disk JSONL format (other than `transcript`, which renders one file at a time on demand).

## When making changes

- New API endpoint? Follow the recipe in [CONTRIBUTING.md](CONTRIBUTING.md#add-a-new-json-api-endpoint).
- New persisted field? Schema lives in `internal/store/store.go`. There's no migration framework — append `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` next to the `CREATE TABLE`.
- Touching the React app? Pages in `web/src/pages/`, typed API calls in `web/src/lib/api.ts`. Use SWR for GETs.
- Touching the terminal? `internal/terminals/manager.go` — keep the ring buffer at 64KB unless you have a reason; the size is load-bearing for "reload page, see recent output."

## Style

- Go: handwritten SQL via `database/sql` (no ORM); errors flow up; don't `log.Fatal` from libraries; package-level docs in `doc.go`.
- TS: SWR for reads, hand-rolled `fetch` + `mutate(key)` for writes; Tailwind only; no component library beyond `ui.tsx` primitives.
- Keep handlers thin. Business logic belongs in the domain-aware packages, not in `api_*.go`.
