package server

import (
	"context"
	"time"
)

// sidebarWorkspace is what the sidebar template needs per workspace card.
type sidebarWorkspace struct {
	EpicSlug    string
	Slug        string
	Name        string
	BranchName  string
	NumSessions int
	Added       int
	Removed     int
	HasPR       bool
}

type sidebarEpic struct {
	Slug       string
	Name       string
	Workspaces []sidebarWorkspace
}

// basePage is embedded in every page-specific data struct; the layout +
// sidebar templates depend only on these fields.
type basePage struct {
	Title               string
	Epics               []sidebarEpic
	ActiveEpicSlug      string
	ActiveWorkspaceSlug string
	IndexedAt           string
	Flash               string
	RightPane           bool
}

// buildBasePage gathers sidebar data and timestamp. Diff stats per workspace
// are computed lazily — failures are silent (badge just won't show).
func (srv *Server) buildBasePage(ctx context.Context, title, activeEpic, activeWorkspace string) (basePage, error) {
	epics, err := srv.store.ListEpics(ctx)
	if err != nil {
		return basePage{}, err
	}
	allSessions, err := srv.store.ListSessions(ctx)
	if err != nil {
		return basePage{}, err
	}
	wsCounts := map[int64]int{}
	for _, s := range allSessions {
		if s.WorkspaceID != nil {
			wsCounts[*s.WorkspaceID]++
		}
	}
	var sidebar []sidebarEpic
	for _, e := range epics {
		ws, err := srv.store.ListWorkspacesByEpic(ctx, e.ID)
		if err != nil {
			return basePage{}, err
		}
		sw := make([]sidebarWorkspace, 0, len(ws))
		for _, w := range ws {
			added, removed := srv.diffStatsForWorkspace(e.RepoPath, w.WorktreePath, e.BaseBranch)
			sw = append(sw, sidebarWorkspace{
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
		sidebar = append(sidebar, sidebarEpic{Slug: e.Slug, Name: e.Name, Workspaces: sw})
	}
	return basePage{
		Title:               title,
		Epics:               sidebar,
		ActiveEpicSlug:      activeEpic,
		ActiveWorkspaceSlug: activeWorkspace,
		IndexedAt:           time.Now().Format("15:04:05"),
	}, nil
}
