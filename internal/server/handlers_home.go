package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"claudeops/internal/transcript"
)

type homeData struct {
	basePage
	TotalSessions int
	ProjectCount  int
}

func (srv *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	base, err := srv.buildBasePage(r.Context(), "Home", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	all, _ := srv.store.ListSessions(r.Context())
	projects := map[string]struct{}{}
	for _, s := range all {
		projects[s.ProjectDir] = struct{}{}
	}
	srv.render(w, "home", homeData{basePage: base, TotalSessions: len(all), ProjectCount: len(projects)})
}

type sessionRow struct {
	ID                string
	ShortID           string
	Ago               string
	CwdShort          string
	GitBranch         string
	WorkspaceLink     string
	WorkspaceLabel    string
	LastUserPreview   string
	LastAssistantText string
	NumEvents         int64
}

type sessionsData struct {
	basePage
	Sessions     []sessionRow
	ProjectCount int
}

func (srv *Server) handleSessionsIndex(w http.ResponseWriter, r *http.Request) {
	base, err := srv.buildBasePage(r.Context(), "Sessions", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sessions, err := srv.store.ListSessions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	wsByID := srv.workspaceLookup(r.Context())
	rows := make([]sessionRow, 0, len(sessions))
	projects := map[string]struct{}{}
	now := time.Now()
	for _, s := range sessions {
		projects[s.ProjectDir] = struct{}{}
		row := sessionRow{
			ID:                s.SessionID,
			ShortID:           short(s.SessionID, 8),
			Ago:               humanAgo(now.Sub(s.LastActivity)),
			CwdShort:          srv.shortenCwd(s.Cwd, s.ProjectDir),
			GitBranch:         s.GitBranch,
			LastUserPreview:   s.LastUserPreview,
			LastAssistantText: s.LastAssistantText,
			NumEvents:         s.NumEvents,
		}
		if s.WorkspaceID != nil {
			if e, ok := wsByID[*s.WorkspaceID]; ok {
				row.WorkspaceLink = fmt.Sprintf("/epics/%s/workspaces/%s", e.epicSlug, e.workspaceSlug)
				row.WorkspaceLabel = e.epicName + " / " + e.workspaceName
			}
		}
		rows = append(rows, row)
	}
	srv.render(w, "sessions", sessionsData{basePage: base, Sessions: rows, ProjectCount: len(projects)})
}

type transcriptData struct {
	basePage
	SessionID string
	ShortID   string
	Cwd       string
	Branch    string
	Messages  []renderedMessage
}

type renderedMessage struct {
	Role     string
	Text     string
	ToolName string
	Model    string
	Ago      string
}

func (srv *Server) handleSessionTranscript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := srv.store.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}
	base, err := srv.buildBasePage(r.Context(), "Session "+short(id, 8), "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	msgs, _ := transcript.Render(sess.FilePath, 200)
	now := time.Now()
	rendered := make([]renderedMessage, 0, len(msgs))
	for _, m := range msgs {
		rendered = append(rendered, renderedMessage{
			Role: m.Role, Text: m.Text, ToolName: m.ToolName, Model: m.Model,
			Ago: humanAgo(now.Sub(m.Timestamp)),
		})
	}
	srv.render(w, "transcript", transcriptData{
		basePage:  base,
		SessionID: id,
		ShortID:   short(id, 8),
		Cwd:       srv.shortPath(sess.Cwd),
		Branch:    sess.GitBranch,
		Messages:  rendered,
	})
}

type debugData struct {
	basePage
	DBPath        string
	DBSizeBytes   int64
	WALSizeBytes  int64
	EpicCount     int
	WorkspaceCnt  int
	SessionCount  int
	RecentEpics   []recentRow
	RecentWS      []recentRow
	RecentSession []recentRow
}

type recentRow struct {
	Cells []string
}

func (srv *Server) handleDebug(w http.ResponseWriter, r *http.Request) {
	base, err := srv.buildBasePage(r.Context(), "Debug", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	d := debugData{basePage: base, DBPath: srv.store.Path()}
	if fi, err := osStat(srv.store.Path()); err == nil {
		d.DBSizeBytes = fi.Size()
	}
	if fi, err := osStat(srv.store.Path() + "-wal"); err == nil {
		d.WALSizeBytes = fi.Size()
	}
	d.EpicCount = countTable(r.Context(), srv.store, "epics")
	d.WorkspaceCnt = countTable(r.Context(), srv.store, "workspaces")
	d.SessionCount = countTable(r.Context(), srv.store, "sessions")
	d.RecentEpics = queryRows(r.Context(), srv.store, `SELECT slug, name, repo_path, base_branch FROM epics ORDER BY id DESC LIMIT 10`)
	d.RecentWS = queryRows(r.Context(), srv.store, `SELECT slug, branch_name, worktree_path, pr_url FROM workspaces ORDER BY id DESC LIMIT 10`)
	d.RecentSession = queryRows(r.Context(), srv.store, `SELECT substr(session_id,1,8), cwd, git_branch, num_events FROM sessions ORDER BY last_activity DESC LIMIT 10`)
	srv.render(w, "debug", d)
}

// ---- small support funcs (kept here to keep handlers_home.go self-contained) ----

type wsLookup struct{ epicSlug, epicName, workspaceSlug, workspaceName string }

func (srv *Server) workspaceLookup(ctx context.Context) map[int64]wsLookup {
	out := map[int64]wsLookup{}
	epics, err := srv.store.ListEpics(ctx)
	if err != nil {
		return out
	}
	for _, e := range epics {
		ws, err := srv.store.ListWorkspacesByEpic(ctx, e.ID)
		if err != nil {
			continue
		}
		for _, w := range ws {
			out[w.ID] = wsLookup{epicSlug: e.Slug, epicName: e.Name, workspaceSlug: w.Slug, workspaceName: w.Name}
		}
	}
	return out
}

func (srv *Server) shortenCwd(cwd, projectDir string) string {
	if cwd == "" {
		// derive from project dir like "-Users-Adish-Shah-sdmain"
		s := projectDir
		if len(s) > 0 && s[0] == '-' {
			s = s[1:]
		}
		return "/" + replaceAll(s, "-", "/")
	}
	return srv.shortPath(cwd)
}

func replaceAll(s, old, new string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if string(r) == old {
			out = append(out, new...)
		} else {
			out = append(out, byte(r))
		}
	}
	return string(out)
}
