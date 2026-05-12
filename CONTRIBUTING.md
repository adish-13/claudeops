# Contributing

Thanks for hacking on claudeops. This document covers local dev setup, the
shape of the codebase, and the recipes for the most common kinds of changes.
Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) first if you haven't.

## Local setup

```sh
git clone https://github.com/adish-13/claudeops.git
cd claudeops

# Frontend deps
cd web && npm install && cd ..

# First build (will fail until cmd/claudeops/main.go exists — see ARCHITECTURE.md)
make build
```

## Dev loop

Two terminals:

```sh
# Terminal 1 — backend (rebuild + run on every change)
go run ./cmd/claudeops -addr 127.0.0.1:7777

# Terminal 2 — frontend with HMR
cd web && npm run dev
```

Open http://localhost:5173. Vite proxies `/api/*` and `/ws/*` to the Go server,
so you get React HMR without needing to rebuild Go on every UI tweak.

To test the production embed path (SPA served by Go), `make build && ./claudeops`
and hit http://127.0.0.1:7777 directly.

## Code conventions

### Go

- Module path is `claudeops`. Internal packages live under `internal/`.
- Keep `domain` a leaf — no SQL, no HTTP, no git, no I/O.
- Anything that talks to SQLite goes in `store`. Anything that shells out to git goes in `git` or `worktree`. Anything HTTP-aware goes in `server`. Don't blur these lines.
- Prefer `database/sql` queries over an ORM. Queries are short; readability beats abstraction.
- All exported identifiers should have a doc comment. Each package has a `doc.go` with the package overview.
- Errors flow up. Don't `log.Fatal` from a library package.

### TypeScript / React

- Pages live in `web/src/pages/`. Reusable bits in `web/src/components/`. Typed API client in `web/src/lib/api.ts`.
- Use SWR for GETs (`useSWR(key, fetcher)`). For mutations, hand-rolled `fetch` + `mutate(key)` to revalidate.
- Tailwind for styling. There is no design-system component library; the small set of primitives in `ui.tsx` (Button, Card, Pill, …) covers most cases.
- Keep components small. If a page is doing data fetching, transformation, *and* rendering, split out a hook in `lib/`.

## Recipes

### Add a new JSON API endpoint

1. Pick the right file under `internal/server/`. New resource? `api_<thing>.go`. Otherwise add to the existing `api_*.go` for that resource.
2. Write the handler: parse path/body, call into `store` / `git` / etc., return JSON. Keep handlers thin.
3. Register the route in `internal/server/server.go` next to the others. Use the same `mux.HandleFunc("METHOD /path", srv.handler)` style as the existing routes.
4. Add the typed call to `web/src/lib/api.ts`. Mirror the request/response shapes as TS types.
5. Wire it into the page that needs it.

### Add a new page

1. Create `web/src/pages/<Name>.tsx`. Default-export the component.
2. Add the route in `web/src/App.tsx` under the existing `<Routes>`.
3. If the page needs sidebar entries, the sidebar reads from `/api/sidebar` — usually no change needed there.

### Add a new persisted field

1. Add it to the relevant struct in `internal/domain/types.go`.
2. Add the column to the `CREATE TABLE` in `internal/store/store.go` (this codebase recreates schema on each open with `IF NOT EXISTS`, so you'll need a migration step — there's no migration framework yet, so for now write an `ALTER TABLE … ADD COLUMN IF NOT EXISTS …` next to the create).
3. Update the relevant queries in `internal/store/<resource>.go` (insert / select / scan).
4. Plumb through the API layer and React.

### Add a git operation

Put it in `internal/git/git.go` if it's a read (diff, status, log). Put it in
`internal/worktree/worktree.go` if it mutates worktree state. Either way, shell
out via `os/exec`; do not import a git library.

## Tests

There aren't any yet. If you're adding non-trivial logic — particularly anything
in `indexer`, `transcript`, `terminals` (ring buffer), or `git` (numstat parsing) —
please add a `_test.go` next to it. Standard `go test ./...` setup, no extra deps
needed.

## Commit / PR style

- Commits use lowercase imperative subjects: `add session resume: claude --resume <id> from the workspace UI`. Multi-paragraph bodies are welcome for non-trivial changes — see the `v3 rebuild` and `v4` commits for examples.
- Keep PRs scoped. The v3/v4 rebuilds are the exception, not the rule.
- Run `go vet ./...` and `gofmt -w .` before pushing.
- For frontend changes, `cd web && npm run build` should succeed (no TS errors).

## Things on the wishlist

- Tests (any).
- A real migration framework so we can evolve schema without `IF NOT EXISTS` gymnastics.
- A cmd/claudeops/main.go that actually exists.
- Cross-platform launcher (the macOS-only `osascript` path is a lonely island).
- Tauri wrapper for a real `.app`.
