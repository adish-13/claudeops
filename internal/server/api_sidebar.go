package server

import (
	"context"
	"net/http"
	"time"

	"claudeops/internal/git"
)

type sidebarWorkspaceJSON struct {
	EpicSlug    string `json:"epic_slug"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	BranchName  string `json:"branch_name"`
	NumSessions int    `json:"num_sessions"`
	Added       int    `json:"added"`
	Removed     int    `json:"removed"`
	HasPR       bool   `json:"has_pr"`
}

type sidebarEpicJSON struct {
	Slug       string                 `json:"slug"`
	Name       string                 `json:"name"`
	Workspaces []sidebarWorkspaceJSON `json:"workspaces"`
}

func (srv *Server) apiSidebar(w http.ResponseWriter, r *http.Request) {
	epics, err := srv.store.ListEpics(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	allSessions, _ := srv.store.ListSessions(r.Context())
	wsCounts := map[int64]int{}
	for _, s := range allSessions {
		if s.WorkspaceID != nil {
			wsCounts[*s.WorkspaceID]++
		}
	}
	out := make([]sidebarEpicJSON, 0, len(epics))
	for _, e := range epics {
		ws, _ := srv.store.ListWorkspacesByEpic(r.Context(), e.ID)
		row := sidebarEpicJSON{Slug: e.Slug, Name: e.Name}
		for _, w := range ws {
			added, removed := safeDiff(w.WorktreePath, e.BaseBranch)
			row.Workspaces = append(row.Workspaces, sidebarWorkspaceJSON{
				EpicSlug:    e.Slug,
				Slug:        w.Slug,
				Name:        w.Name,
				BranchName:  w.BranchName,
				NumSessions: wsCounts[w.ID],
				Added:       added,
				Removed:     removed,
				HasPR:       w.PRURL != "",
			})
		}
		out = append(out, row)
	}
	writeJSON(w, 200, map[string]any{
		"epics":      out,
		"indexed_at": time.Now().Format("15:04:05"),
	})
}

func (srv *Server) apiHome(w http.ResponseWriter, r *http.Request) {
	all, _ := srv.store.ListSessions(r.Context())
	projects := map[string]struct{}{}
	for _, s := range all {
		projects[s.ProjectDir] = struct{}{}
	}
	writeJSON(w, 200, map[string]any{
		"total_sessions": len(all),
		"project_count":  len(projects),
	})
}

// safeDiff returns +/- counts; on any error returns zeros.
func safeDiff(worktreePath, baseBranch string) (int, int) {
	if worktreePath == "" {
		return 0, 0
	}
	sum, _, err := git.Diff(worktreePath, baseBranch)
	if err != nil {
		return 0, 0
	}
	return sum.Added, sum.Removed
}

// keep imports tidy
var _ = context.Background
