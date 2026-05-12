package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"claudeops/internal/domain"
	"claudeops/internal/git"
	"claudeops/internal/launcher"
	"claudeops/internal/worktree"
)

type workspaceData struct {
	basePage
	Epic         domain.Epic
	Workspace    domain.Workspace
	WorktreeRel  string
	Sessions     []sessionRow
	CreatedAgo   string
	Diff         git.Summary
	Files        []domain.DiffStat
	TerminalLive bool
}

func (srv *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, epic, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		http.Error(w, "workspace not found", 404)
		return
	}
	base, err := srv.buildBasePage(r.Context(), ws.Name, slug, wsslug)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	base.RightPane = true
	sessions, _ := srv.store.ListSessionsByWorkspace(r.Context(), ws.ID)
	now := time.Now()
	rows := make([]sessionRow, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, sessionRow{
			ID:                s.SessionID,
			ShortID:           short(s.SessionID, 8),
			Ago:               humanAgo(now.Sub(s.LastActivity)),
			LastUserPreview:   s.LastUserPreview,
			LastAssistantText: s.LastAssistantText,
			NumEvents:         s.NumEvents,
		})
	}
	diff, files, _ := git.Diff(ws.WorktreePath, epic.BaseBranch)
	live := srv.terminals.Get(ws.ID) != nil
	srv.render(w, "workspace", workspaceData{
		basePage:     base,
		Epic:         *epic,
		Workspace:    *ws,
		WorktreeRel:  srv.shortPath(ws.WorktreePath),
		Sessions:     rows,
		CreatedAgo:   humanAgo(now.Sub(ws.CreatedAt)),
		Diff:         diff,
		Files:        files,
		TerminalLive: live,
	})
}

type newWorkspaceData struct {
	basePage
	Epic            domain.Epic
	SuggestedBranch string
	SuggestedPath   string
}

func (srv *Server) handleNewWorkspaceForm(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "epic not found", 404)
		return
	}
	base, err := srv.buildBasePage(r.Context(), "New workspace", slug, "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}
	suggestedSlug := "wip-" + time.Now().Format("0102-1504")
	srv.render(w, "new_workspace", newWorkspaceData{
		basePage:        base,
		Epic:            *epic,
		SuggestedBranch: fmt.Sprintf("%s-%s-%s", strings.ToLower(user), epic.Slug, suggestedSlug),
		SuggestedPath:   worktree.SuggestPath(srv.worktreeRoot, epic.RepoPath, epic.Slug, suggestedSlug),
	})
}

func (srv *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "epic not found", 404)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	wsSlug := slugify(r.FormValue("slug"))
	branch := strings.TrimSpace(r.FormValue("branch_name"))
	wtPath := srv.expandHome(strings.TrimSpace(r.FormValue("worktree_path")))
	name := strings.TrimSpace(r.FormValue("name"))
	if wsSlug == "" || branch == "" || wtPath == "" || name == "" {
		http.Error(w, "name, slug, branch_name, worktree_path are required", 400)
		return
	}
	if err := worktree.Add(epic.RepoPath, wtPath, branch, epic.BaseBranch); err != nil {
		http.Error(w, "git worktree add failed: "+err.Error(), 500)
		return
	}
	if _, err := srv.store.CreateWorkspace(r.Context(), domain.Workspace{
		EpicID: epic.ID, Slug: wsSlug, Name: name, BranchName: branch, WorktreePath: wtPath,
	}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/epics/%s/workspaces/%s", epic.Slug, wsSlug), http.StatusSeeOther)
}

func (srv *Server) handleLaunchExternal(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, epic, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		http.Error(w, "workspace not found", 404)
		return
	}
	if err := launcher.LaunchClaude(ws.WorktreePath, ""); err != nil {
		http.Error(w, "launch failed: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/epics/%s/workspaces/%s", epic.Slug, ws.Slug), http.StatusSeeOther)
}

func (srv *Server) handleSavePR(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		http.Error(w, "workspace not found", 404)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := srv.store.UpdateWorkspacePR(r.Context(), ws.ID, strings.TrimSpace(r.FormValue("pr_url"))); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/epics/%s/workspaces/%s", slug, wsslug), http.StatusSeeOther)
}

func (srv *Server) handleArchiveWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		http.Error(w, "workspace not found", 404)
		return
	}
	if err := srv.store.ArchiveWorkspace(r.Context(), ws.ID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	srv.terminals.Kill(ws.ID)
	http.Redirect(w, r, "/epics/"+slug, http.StatusSeeOther)
}

// diffStatsForWorkspace returns aggregated +/- counts for a worktree.
// It is silent on error (returns zeros) — used to populate sidebar badges.
func (srv *Server) diffStatsForWorkspace(repoPath, worktreePath, baseBranch string) (added, removed int) {
	if worktreePath == "" {
		return 0, 0
	}
	sum, _, err := git.Diff(worktreePath, baseBranch)
	if err != nil {
		return 0, 0
	}
	return sum.Added, sum.Removed
}
