// Package server is the HTTP layer.
//
// Responsibilities:
//   - JSON API under /api/* (handlers split by resource into api_*.go files)
//   - WebSocket terminal under /ws/terminal/{wsid}
//   - Static SPA assets served from web/dist (embedded), with index.html
//     returned for any unknown path so client-side routing works.
package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"claudeops/internal/store"
	"claudeops/internal/terminals"
)

type Server struct {
	store        *store.Store
	terminals    *terminals.Manager
	home         string
	worktreeRoot string
}

func New(s *store.Store, tm *terminals.Manager, worktreeRoot string) *Server {
	home, _ := os.UserHomeDir()
	if worktreeRoot == "" {
		worktreeRoot = filepath.Join(home, "worktrees")
	}
	return &Server{store: s, terminals: tm, home: home, worktreeRoot: worktreeRoot}
}

func (srv *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()

	// JSON API
	mux.HandleFunc("GET /api/sidebar", srv.apiSidebar)
	mux.HandleFunc("GET /api/home", srv.apiHome)
	mux.HandleFunc("GET /api/sessions", srv.apiSessionsList)
	mux.HandleFunc("GET /api/sessions/{id}", srv.apiSessionGet)
	mux.HandleFunc("GET /api/debug", srv.apiDebug)

	mux.HandleFunc("POST /api/epics", srv.apiCreateEpic)
	mux.HandleFunc("GET /api/epics/{slug}", srv.apiGetEpic)
	mux.HandleFunc("POST /api/epics/{slug}/context", srv.apiSaveContext)
	mux.HandleFunc("POST /api/epics/{slug}/archive", srv.apiArchiveEpic)

	mux.HandleFunc("GET /api/epics/{slug}/workspaces/suggest", srv.apiSuggestWorkspace)
	mux.HandleFunc("POST /api/epics/{slug}/workspaces", srv.apiCreateWorkspace)
	mux.HandleFunc("GET /api/epics/{slug}/workspaces/{wsslug}", srv.apiGetWorkspace)
	mux.HandleFunc("POST /api/epics/{slug}/workspaces/{wsslug}/launch", srv.apiLaunchITerm)
	mux.HandleFunc("POST /api/epics/{slug}/workspaces/{wsslug}/pr", srv.apiSavePR)
	mux.HandleFunc("POST /api/epics/{slug}/workspaces/{wsslug}/archive", srv.apiArchiveWorkspace)
	mux.HandleFunc("POST /api/epics/{slug}/workspaces/{wsslug}/term/start", srv.apiTermStart)
	mux.HandleFunc("POST /api/epics/{slug}/workspaces/{wsslug}/term/kill", srv.apiTermKill)

	// WebSocket
	mux.HandleFunc("/ws/terminal/{wsid}", srv.handleTerminalWS)

	// SPA — must be last; matches everything else.
	mux.Handle("/", srv.spaHandler())

	return http.ListenAndServe(addr, mux)
}

// ---------- JSON helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
