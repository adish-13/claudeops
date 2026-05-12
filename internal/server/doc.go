// Package server is the HTTP layer: JSON API, WebSocket terminal bridge,
// and the embedded React SPA.
//
// All routes are registered in server.go's New(). The shape:
//
//   - /api/* — JSON endpoints. Handlers live in api_*.go grouped by resource
//     (api_epics.go, api_workspaces.go, api_sessions.go, api_terminal.go,
//     api_sidebar.go, api_debug.go). Keep handlers thin; push business
//     logic into the domain-aware packages (store, git, worktree, …).
//   - /ws/terminal/{wsid} — WebSocket bridge to a terminals.Session. Binary
//     frames carry pty output; JSON control frames carry {type:"input"} and
//     {type:"resize"}.
//   - Everything else — falls through to spaHandler, which serves the
//     embedded React build from internal/server/dist/ and 404s back to
//     index.html so client-side routing survives a hard refresh.
//
// The SPA is embedded with `//go:embed all:dist` in spa.go. That directory
// is produced by `cd web && npm run build` (Vite's outDir is configured to
// `../internal/server/dist`). If you `go build` without a fresh SPA build,
// spaHandler returns a 500 with a clear message — it does not silently
// serve stale or empty assets.
//
// Server is constructed once with New(store, terminals.Manager, worktreeRoot)
// and started with ListenAndServe(addr). It owns no goroutines beyond
// what the http package and the WebSocket handlers spawn.
package server
