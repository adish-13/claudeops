package server

import (
	"net/http"
	"strings"
	"time"

	"claudeops/internal/domain"
	"claudeops/internal/worktree"
)

type epicJSON struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RepoPath    string `json:"repo_path"`
	BaseBranch  string `json:"base_branch"`
	ContextMD   string `json:"context_md"`
	CreatedAt   string `json:"created_at"`
}

type sessionJSON struct {
	SessionID         string `json:"session_id"`
	ProjectDir        string `json:"project_dir"`
	Cwd               string `json:"cwd"`
	GitBranch         string `json:"git_branch"`
	Model             string `json:"model"`
	LastActivity      string `json:"last_activity"`
	LastUserPreview   string `json:"last_user_preview"`
	LastAssistantText string `json:"last_assistant_text"`
	NumEvents         int64  `json:"num_events"`
	WorkspaceID       *int64 `json:"workspace_id"`
	WorkspaceLink     string `json:"workspace_link,omitempty"`
	WorkspaceLabel    string `json:"workspace_label,omitempty"`
}

type epicWorkspaceJSON struct {
	sidebarWorkspaceJSON
	WorktreePath string `json:"worktree_path"`
	PRURL        string `json:"pr_url"`
}

type epicDetailJSON struct {
	Epic            epicJSON            `json:"epic"`
	Workspaces      []epicWorkspaceJSON `json:"workspaces"`
	UnboundSessions []sessionJSON       `json:"unbound_sessions"`
}

func toEpicJSON(e domain.Epic) epicJSON {
	return epicJSON{
		ID: e.ID, Slug: e.Slug, Name: e.Name, Description: e.Description,
		RepoPath: e.RepoPath, BaseBranch: e.BaseBranch, ContextMD: e.ContextMD,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (srv *Server) toSessionJSON(s domain.Session) sessionJSON {
	return sessionJSON{
		SessionID: s.SessionID, ProjectDir: s.ProjectDir, Cwd: srv.shortPath(s.Cwd),
		GitBranch: s.GitBranch, Model: s.Model,
		LastActivity:      s.LastActivity.UTC().Format(time.RFC3339),
		LastUserPreview:   s.LastUserPreview,
		LastAssistantText: s.LastAssistantText,
		NumEvents:         s.NumEvents,
		WorkspaceID:       s.WorkspaceID,
	}
}

func (srv *Server) apiGetEpic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		writeErr(w, 404, "epic not found")
		return
	}
	ws, _ := srv.store.ListWorkspacesByEpic(r.Context(), epic.ID)
	allSessions, _ := srv.store.ListSessions(r.Context())
	counts := map[int64]int{}
	for _, s := range allSessions {
		if s.WorkspaceID != nil {
			counts[*s.WorkspaceID]++
		}
	}
	wsRows := make([]epicWorkspaceJSON, 0, len(ws))
	for _, x := range ws {
		added, removed := safeDiff(x.WorktreePath, epic.BaseBranch)
		wsRows = append(wsRows, epicWorkspaceJSON{
			sidebarWorkspaceJSON: sidebarWorkspaceJSON{
				EpicSlug:    epic.Slug,
				Slug:        x.Slug,
				Name:        x.Name,
				BranchName:  x.BranchName,
				NumSessions: counts[x.ID],
				Added:       added,
				Removed:     removed,
				HasPR:       x.PRURL != "",
			},
			WorktreePath: srv.shortPath(x.WorktreePath),
			PRURL:        x.PRURL,
		})
	}
	unbound, _ := srv.store.ListUnboundSessionsForRepo(r.Context(), epic.RepoPath)
	uRows := make([]sessionJSON, 0, len(unbound))
	for _, s := range unbound {
		uRows = append(uRows, srv.toSessionJSON(s))
	}
	writeJSON(w, 200, epicDetailJSON{Epic: toEpicJSON(*epic), Workspaces: wsRows, UnboundSessions: uRows})
}

func (srv *Server) apiCreateEpic(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		RepoPath    string `json:"repo_path"`
		BaseBranch  string `json:"base_branch"`
		ContextMD   string `json:"context_md"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	e := domain.Epic{
		Slug:        slugify(body.Slug),
		Name:        strings.TrimSpace(body.Name),
		Description: strings.TrimSpace(body.Description),
		RepoPath:    srv.expandHome(strings.TrimSpace(body.RepoPath)),
		BaseBranch:  strings.TrimSpace(body.BaseBranch),
		ContextMD:   body.ContextMD,
	}
	if e.BaseBranch == "" {
		e.BaseBranch = "master"
	}
	if e.Slug == "" || e.Name == "" || e.RepoPath == "" {
		writeErr(w, 400, "name, slug, and repo_path are required")
		return
	}
	if err := worktree.IsGitRepo(e.RepoPath); err != nil {
		writeErr(w, 400, err.Error()+" — run `git init` in that directory first, or pick a different path")
		return
	}
	id, err := srv.store.CreateEpic(r.Context(), e)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	e.ID = id
	writeJSON(w, 200, toEpicJSON(e))
}

func (srv *Server) apiSaveContext(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var body struct {
		ContextMD string `json:"context_md"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := srv.store.UpdateEpicContext(r.Context(), slug, body.ContextMD); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (srv *Server) apiArchiveEpic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := srv.store.ArchiveEpic(r.Context(), slug); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
