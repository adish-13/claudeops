// Package server is the HTTP layer: routing, template rendering, and the
// WebSocket bridge to embedded terminals. Handlers are methods on *Server,
// split across handlers_*.go files by resource.
package server

import (
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

	tmpl *templateCache
}

func New(s *store.Store, tm *terminals.Manager, worktreeRoot string) (*Server, error) {
	home, _ := os.UserHomeDir()
	if worktreeRoot == "" {
		worktreeRoot = filepath.Join(home, "worktrees")
	}
	srv := &Server{
		store:        s,
		terminals:    tm,
		home:         home,
		worktreeRoot: worktreeRoot,
		tmpl:         newTemplateCache(),
	}
	// Validate templates upfront.
	for _, p := range allPages {
		if _, err := srv.tmpl.get(p); err != nil {
			return nil, err
		}
	}
	return srv, nil
}

func (srv *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()

	// Pages
	mux.HandleFunc("GET /{$}", srv.handleHome)
	mux.HandleFunc("GET /sessions", srv.handleSessionsIndex)
	mux.HandleFunc("GET /sessions/{id}", srv.handleSessionTranscript)
	mux.HandleFunc("GET /debug", srv.handleDebug)

	// Epic CRUD
	mux.HandleFunc("GET /epics/new", srv.handleNewEpicForm)
	mux.HandleFunc("POST /epics", srv.handleCreateEpic)
	mux.HandleFunc("GET /epics/{slug}", srv.handleEpic)
	mux.HandleFunc("POST /epics/{slug}/context", srv.handleSaveContext)
	mux.HandleFunc("POST /epics/{slug}/archive", srv.handleArchiveEpic)

	// Workspaces under an epic
	mux.HandleFunc("GET /epics/{slug}/workspaces/new", srv.handleNewWorkspaceForm)
	mux.HandleFunc("POST /epics/{slug}/workspaces", srv.handleCreateWorkspace)
	mux.HandleFunc("GET /epics/{slug}/workspaces/{wsslug}", srv.handleWorkspace)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/launch", srv.handleLaunchExternal)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/pr", srv.handleSavePR)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/archive", srv.handleArchiveWorkspace)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/term/start", srv.handleTerminalStart)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/term/kill", srv.handleTerminalKill)

	// WebSocket terminal stream
	mux.HandleFunc("/ws/terminal/{wsid}", srv.handleTerminalWS)

	return http.ListenAndServe(addr, mux)
}
