package server

import (
	"net/http"
	"strings"
	"time"

	"claudeops/internal/domain"
	"claudeops/internal/worktree"
)

type epicWorkspaceRow struct {
	Slug         string
	Name         string
	BranchName   string
	WorktreePath string
	PRURL        string
	NumSessions  int
	Added        int
	Removed      int
}

type epicData struct {
	basePage
	Epic            domain.Epic
	Workspaces      []epicWorkspaceRow
	UnboundSessions []sessionRow
}

func (srv *Server) handleEpic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "epic not found", 404)
		return
	}
	base, err := srv.buildBasePage(r.Context(), epic.Name, slug, "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	ws, err := srv.store.ListWorkspacesByEpic(r.Context(), epic.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	allSessions, _ := srv.store.ListSessions(r.Context())
	counts := map[int64]int{}
	for _, s := range allSessions {
		if s.WorkspaceID != nil {
			counts[*s.WorkspaceID]++
		}
	}
	rows := make([]epicWorkspaceRow, 0, len(ws))
	for _, w := range ws {
		added, removed := srv.diffStatsForWorkspace(epic.RepoPath, w.WorktreePath, epic.BaseBranch)
		rows = append(rows, epicWorkspaceRow{
			Slug:         w.Slug,
			Name:         w.Name,
			BranchName:   w.BranchName,
			WorktreePath: srv.shortPath(w.WorktreePath),
			PRURL:        w.PRURL,
			NumSessions:  counts[w.ID],
			Added:        added,
			Removed:      removed,
		})
	}
	unbound, _ := srv.store.ListUnboundSessionsForRepo(r.Context(), epic.RepoPath)
	now := time.Now()
	uRows := make([]sessionRow, 0, len(unbound))
	for _, s := range unbound {
		uRows = append(uRows, sessionRow{
			ID:              s.SessionID,
			ShortID:         short(s.SessionID, 8),
			Ago:             humanAgo(now.Sub(s.LastActivity)),
			CwdShort:        srv.shortenCwd(s.Cwd, s.ProjectDir),
			GitBranch:       s.GitBranch,
			LastUserPreview: s.LastUserPreview,
		})
	}
	srv.render(w, "epic", epicData{basePage: base, Epic: *epic, Workspaces: rows, UnboundSessions: uRows})
}

type newEpicData struct{ basePage }

func (srv *Server) handleNewEpicForm(w http.ResponseWriter, r *http.Request) {
	base, err := srv.buildBasePage(r.Context(), "New epic", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	srv.render(w, "new_epic", newEpicData{basePage: base})
}

func (srv *Server) handleCreateEpic(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	e := domain.Epic{
		Slug:        slugify(r.FormValue("slug")),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		RepoPath:    srv.expandHome(strings.TrimSpace(r.FormValue("repo_path"))),
		BaseBranch:  strings.TrimSpace(r.FormValue("base_branch")),
		ContextMD:   r.FormValue("context_md"),
	}
	if e.BaseBranch == "" {
		e.BaseBranch = "master"
	}
	if e.Slug == "" || e.Name == "" || e.RepoPath == "" {
		http.Error(w, "name, slug, and repo_path are required", 400)
		return
	}
	if err := worktree.IsGitRepo(e.RepoPath); err != nil {
		http.Error(w, err.Error()+" — run `git init` in that directory first, or pick a different path", 400)
		return
	}
	if _, err := srv.store.CreateEpic(r.Context(), e); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/epics/"+e.Slug, http.StatusSeeOther)
}

func (srv *Server) handleSaveContext(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := srv.store.UpdateEpicContext(r.Context(), slug, r.FormValue("context_md")); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/epics/"+slug, http.StatusSeeOther)
}

func (srv *Server) handleArchiveEpic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := srv.store.ArchiveEpic(r.Context(), slug); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
