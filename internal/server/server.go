package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"claudeops/internal/launcher"
	"claudeops/internal/store"
	"claudeops/internal/worktree"
)

//go:embed templates/*.html
var tmplFS embed.FS

type Server struct {
	store        *store.Store
	home         string
	worktreeRoot string

	mu      sync.Mutex
	tmplMap map[string]*template.Template
}

func New(s *store.Store, worktreeRoot string) (*Server, error) {
	home, _ := os.UserHomeDir()
	if worktreeRoot == "" {
		worktreeRoot = filepath.Join(home, "worktrees")
	}
	srv := &Server{store: s, home: home, worktreeRoot: worktreeRoot, tmplMap: map[string]*template.Template{}}
	// Pre-parse to validate templates on startup.
	for _, page := range []string{"home", "epic", "workspace", "new_epic", "new_workspace", "sessions"} {
		if _, err := srv.template(page); err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
	}
	return srv, nil
}

func (srv *Server) template(page string) (*template.Template, error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if t, ok := srv.tmplMap[page]; ok {
		return t, nil
	}
	t, err := template.New("").ParseFS(tmplFS, "templates/layout.html", "templates/"+page+".html")
	if err != nil {
		return nil, err
	}
	srv.tmplMap[page] = t
	return t, nil
}

func (srv *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", srv.handleHome)
	mux.HandleFunc("GET /sessions", srv.handleSessions)
	mux.HandleFunc("GET /epics/new", srv.handleNewEpicForm)
	mux.HandleFunc("POST /epics", srv.handleCreateEpic)
	mux.HandleFunc("GET /epics/{slug}", srv.handleEpic)
	mux.HandleFunc("POST /epics/{slug}/context", srv.handleSaveContext)
	mux.HandleFunc("GET /epics/{slug}/workspaces/new", srv.handleNewWorkspaceForm)
	mux.HandleFunc("POST /epics/{slug}/workspaces", srv.handleCreateWorkspace)
	mux.HandleFunc("GET /epics/{slug}/workspaces/{wsslug}", srv.handleWorkspace)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/launch", srv.handleLaunch)
	mux.HandleFunc("POST /epics/{slug}/workspaces/{wsslug}/pr", srv.handleSavePR)
	return http.ListenAndServe(addr, mux)
}

// ---------- Sidebar / common data ----------

type sidebarWorkspace struct {
	EpicSlug    string
	Slug        string
	Name        string
	BranchName  string
	NumSessions int
}

type sidebarEpic struct {
	Slug       string
	Name       string
	Workspaces []sidebarWorkspace
}

type pageData struct {
	Title               string
	Epics               []sidebarEpic
	ActiveEpicSlug      string
	ActiveWorkspaceSlug string
	IndexedAt           string
	Flash               string

	// Page-specific fields are set by handlers via embedding or direct assignment in subtypes below.
}

func (srv *Server) baseData(ctx context.Context, title, activeEpic, activeWorkspace string) (pageData, error) {
	epics, err := srv.store.ListEpics(ctx)
	if err != nil {
		return pageData{}, err
	}
	allSessions, err := srv.store.ListSessions(ctx)
	if err != nil {
		return pageData{}, err
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
			return pageData{}, err
		}
		sw := make([]sidebarWorkspace, 0, len(ws))
		for _, w := range ws {
			sw = append(sw, sidebarWorkspace{
				EpicSlug:    e.Slug,
				Slug:        w.Slug,
				Name:        w.Name,
				BranchName:  w.BranchName,
				NumSessions: wsCounts[w.ID],
			})
		}
		sidebar = append(sidebar, sidebarEpic{Slug: e.Slug, Name: e.Name, Workspaces: sw})
	}
	return pageData{
		Title:               title,
		Epics:               sidebar,
		ActiveEpicSlug:      activeEpic,
		ActiveWorkspaceSlug: activeWorkspace,
		IndexedAt:           time.Now().Format("15:04:05"),
	}, nil
}

func (srv *Server) render(w http.ResponseWriter, page string, data any) {
	t, err := srv.template(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

// ---------- Handlers ----------

type homeData struct {
	pageData
	TotalSessions int
	ProjectCount  int
}

func (srv *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	base, err := srv.baseData(r.Context(), "Home", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	all, _ := srv.store.ListSessions(r.Context())
	projects := map[string]struct{}{}
	for _, s := range all {
		projects[s.ProjectDir] = struct{}{}
	}
	srv.render(w, "home", homeData{pageData: base, TotalSessions: len(all), ProjectCount: len(projects)})
}

type sessionRow struct {
	Ago               string
	CwdShort          string
	GitBranch         string
	WorkspaceLink     string
	WorkspaceLabel    string
	LastUserPreview   string
	LastAssistantText string
	NumEvents         int64
	ShortID           string
}

type sessionsData struct {
	pageData
	Sessions     []sessionRow
	ProjectCount int
}

func (srv *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	base, err := srv.baseData(r.Context(), "Sessions", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sessions, err := srv.store.ListSessions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	wsByID := srv.workspaceIndex(r.Context())
	rows := make([]sessionRow, 0, len(sessions))
	projects := map[string]struct{}{}
	now := time.Now()
	for _, s := range sessions {
		projects[s.ProjectDir] = struct{}{}
		row := sessionRow{
			Ago:               humanAgo(now.Sub(s.LastActivity)),
			CwdShort:          srv.shortenPath(s.Cwd, s.ProjectDir),
			GitBranch:         s.GitBranch,
			LastUserPreview:   s.LastUserPreview,
			LastAssistantText: s.LastAssistantText,
			NumEvents:         s.NumEvents,
			ShortID:           short(s.SessionID, 8),
		}
		if s.WorkspaceID != nil {
			if e, ok := wsByID[*s.WorkspaceID]; ok {
				row.WorkspaceLink = fmt.Sprintf("/epics/%s/workspaces/%s", e.epicSlug, e.workspaceSlug)
				row.WorkspaceLabel = e.epicName + " / " + e.workspaceName
			}
		}
		rows = append(rows, row)
	}
	srv.render(w, "sessions", sessionsData{pageData: base, Sessions: rows, ProjectCount: len(projects)})
}

type epicWorkspaceRow struct {
	Slug         string
	Name         string
	BranchName   string
	WorktreePath string
	PRURL        string
	NumSessions  int
}

type epicData struct {
	pageData
	Epic            store.Epic
	Workspaces      []epicWorkspaceRow
	UnboundSessions []sessionRow
}

func (srv *Server) handleEpic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		srv.notFound(w, "epic")
		return
	}
	base, err := srv.baseData(r.Context(), epic.Name, slug, "")
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
		rows = append(rows, epicWorkspaceRow{
			Slug:         w.Slug,
			Name:         w.Name,
			BranchName:   w.BranchName,
			WorktreePath: shortPath(w.WorktreePath, srv.home),
			PRURL:        w.PRURL,
			NumSessions:  counts[w.ID],
		})
	}
	unbound, _ := srv.store.ListUnboundSessionsForRepo(r.Context(), epic.RepoPath)
	now := time.Now()
	uRows := make([]sessionRow, 0, len(unbound))
	for _, s := range unbound {
		uRows = append(uRows, sessionRow{
			Ago:             humanAgo(now.Sub(s.LastActivity)),
			CwdShort:        srv.shortenPath(s.Cwd, s.ProjectDir),
			GitBranch:       s.GitBranch,
			LastUserPreview: s.LastUserPreview,
		})
	}
	srv.render(w, "epic", epicData{pageData: base, Epic: *epic, Workspaces: rows, UnboundSessions: uRows})
}

type workspaceData struct {
	pageData
	Epic       store.Epic
	Workspace  store.Workspace
	Sessions   []sessionRow
	CreatedAgo string
}

func (srv *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, epic, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		srv.notFound(w, "workspace")
		return
	}
	base, err := srv.baseData(r.Context(), ws.Name, slug, wsslug)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sessions, _ := srv.store.ListSessionsByWorkspace(r.Context(), ws.ID)
	now := time.Now()
	rows := make([]sessionRow, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, sessionRow{
			Ago:               humanAgo(now.Sub(s.LastActivity)),
			LastUserPreview:   s.LastUserPreview,
			LastAssistantText: s.LastAssistantText,
			NumEvents:         s.NumEvents,
			ShortID:           short(s.SessionID, 8),
		})
	}
	srv.render(w, "workspace", workspaceData{
		pageData:   base,
		Epic:       *epic,
		Workspace:  *ws,
		Sessions:   rows,
		CreatedAgo: humanAgo(now.Sub(ws.CreatedAt)),
	})
}

type newEpicData struct {
	pageData
}

func (srv *Server) handleNewEpicForm(w http.ResponseWriter, r *http.Request) {
	base, err := srv.baseData(r.Context(), "New epic", "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	srv.render(w, "new_epic", newEpicData{pageData: base})
}

func (srv *Server) handleCreateEpic(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	e := store.Epic{
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

type newWorkspaceData struct {
	pageData
	Epic            store.Epic
	SuggestedBranch string
	SuggestedPath   string
}

func (srv *Server) handleNewWorkspaceForm(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		srv.notFound(w, "epic")
		return
	}
	base, err := srv.baseData(r.Context(), "New workspace", slug, "")
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
		pageData:        base,
		Epic:            *epic,
		SuggestedBranch: fmt.Sprintf("%s-%s-%s", strings.ToLower(user), epic.Slug, suggestedSlug),
		SuggestedPath:   worktree.SuggestPath(srv.worktreeRoot, epic.RepoPath, epic.Slug, suggestedSlug),
	})
}

func (srv *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		srv.notFound(w, "epic")
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
	if _, err := srv.store.CreateWorkspace(r.Context(), store.Workspace{
		EpicID: epic.ID, Slug: wsSlug, Name: name, BranchName: branch, WorktreePath: wtPath,
	}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/epics/%s/workspaces/%s", epic.Slug, wsSlug), http.StatusSeeOther)
}

func (srv *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, epic, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		srv.notFound(w, "workspace")
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
		srv.notFound(w, "workspace")
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

// ---------- helpers ----------

type wsLookup struct {
	epicSlug, epicName, workspaceSlug, workspaceName string
}

func (srv *Server) workspaceIndex(ctx context.Context) map[int64]wsLookup {
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

// expandHome turns "~" or "~/foo" into an absolute path under the user's home dir.
// Leaves any other path unchanged.
func (srv *Server) expandHome(p string) string {
	if p == "" || srv.home == "" {
		return p
	}
	if p == "~" {
		return srv.home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(srv.home, p[2:])
	}
	return p
}

func (srv *Server) shortenPath(cwd, projectDir string) string {
	if cwd == "" {
		return strings.ReplaceAll(strings.TrimPrefix(projectDir, "-"), "-", "/")
	}
	return shortPath(cwd, srv.home)
}

func shortPath(p, home string) string {
	if home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func (srv *Server) notFound(w http.ResponseWriter, kind string) {
	http.Error(w, kind+" not found", http.StatusNotFound)
}

func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func humanAgo(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || r == ' ' || r == '/':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// avoid unused-import lint if errors becomes unused after future refactors
var _ = errors.New
