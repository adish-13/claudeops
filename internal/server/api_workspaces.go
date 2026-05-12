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

type workspaceJSON struct {
	ID           int64  `json:"id"`
	EpicID       int64  `json:"epic_id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	BranchName   string `json:"branch_name"`
	WorktreePath string `json:"worktree_path"`
	PRURL        string `json:"pr_url"`
	CreatedAt    string `json:"created_at"`
}

type workspaceDetailJSON struct {
	Workspace     workspaceJSON     `json:"workspace"`
	Epic          epicJSON          `json:"epic"`
	Sessions      []sessionJSON     `json:"sessions"`
	Diff          git.Summary       `json:"diff"`
	Files         []domain.DiffStat `json:"files"`
	TerminalLive  bool              `json:"terminal_live"`
	WorktreeShort string            `json:"worktree_short"`
}

func toWorkspaceJSON(w domain.Workspace) workspaceJSON {
	return workspaceJSON{
		ID: w.ID, EpicID: w.EpicID, Slug: w.Slug, Name: w.Name,
		BranchName: w.BranchName, WorktreePath: w.WorktreePath, PRURL: w.PRURL,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (srv *Server) apiGetWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, epic, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		writeErr(w, 404, "workspace not found")
		return
	}
	sessions, _ := srv.store.ListSessionsByWorkspace(r.Context(), ws.ID)
	rows := make([]sessionJSON, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, srv.toSessionJSON(s))
	}
	diff, files, _ := git.Diff(ws.WorktreePath, epic.BaseBranch)
	live := srv.terminals.Get(ws.ID) != nil
	writeJSON(w, 200, workspaceDetailJSON{
		Workspace:     toWorkspaceJSON(*ws),
		Epic:          toEpicJSON(*epic),
		Sessions:      rows,
		Diff:          diff,
		Files:         files,
		TerminalLive:  live,
		WorktreeShort: srv.shortPath(ws.WorktreePath),
	})
}

func (srv *Server) apiSuggestWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		writeErr(w, 404, "epic not found")
		return
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}
	sug := "wip-" + time.Now().Format("0102-1504")
	writeJSON(w, 200, map[string]string{
		"slug":          sug,
		"branch_name":   fmt.Sprintf("%s-%s-%s", strings.ToLower(user), epic.Slug, sug),
		"worktree_path": worktree.SuggestPath(srv.worktreeRoot, epic.RepoPath, epic.Slug, sug),
	})
}

func (srv *Server) apiCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	epic, err := srv.store.GetEpicBySlug(r.Context(), slug)
	if err != nil {
		writeErr(w, 404, "epic not found")
		return
	}
	var body struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		BranchName   string `json:"branch_name"`
		WorktreePath string `json:"worktree_path"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	wsSlug := slugify(body.Slug)
	branch := strings.TrimSpace(body.BranchName)
	wtPath := srv.expandHome(strings.TrimSpace(body.WorktreePath))
	name := strings.TrimSpace(body.Name)
	if wsSlug == "" || branch == "" || wtPath == "" || name == "" {
		writeErr(w, 400, "name, slug, branch_name, worktree_path are required")
		return
	}
	if err := worktree.Add(epic.RepoPath, wtPath, branch, epic.BaseBranch); err != nil {
		writeErr(w, 500, "git worktree add failed: "+err.Error())
		return
	}
	id, err := srv.store.CreateWorkspace(r.Context(), domain.Workspace{
		EpicID: epic.ID, Slug: wsSlug, Name: name, BranchName: branch, WorktreePath: wtPath,
	})
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, toWorkspaceJSON(domain.Workspace{
		ID: id, EpicID: epic.ID, Slug: wsSlug, Name: name,
		BranchName: branch, WorktreePath: wtPath, CreatedAt: time.Now(),
	}))
}

func (srv *Server) apiLaunchITerm(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		writeErr(w, 404, "workspace not found")
		return
	}
	if err := launcher.LaunchClaude(ws.WorktreePath, ""); err != nil {
		writeErr(w, 500, "launch failed: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (srv *Server) apiSavePR(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		writeErr(w, 404, "workspace not found")
		return
	}
	var body struct {
		PRURL string `json:"pr_url"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := srv.store.UpdateWorkspacePR(r.Context(), ws.ID, strings.TrimSpace(body.PRURL)); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (srv *Server) apiArchiveWorkspace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		writeErr(w, 404, "workspace not found")
		return
	}
	if err := srv.store.ArchiveWorkspace(r.Context(), ws.ID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	srv.terminals.Kill(ws.ID)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (srv *Server) apiTermStart(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		writeErr(w, 404, "workspace not found")
		return
	}
	// Optional body: { "session_id": "..." } — when present, the pty runs
	// `claude --resume <id>` instead of a fresh `claude`.
	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = decodeJSON(r, &body) // body is optional; ignore decode errors
	var extra []string
	if id := strings.TrimSpace(body.SessionID); id != "" {
		extra = []string{"--resume", id}
	}
	if _, err := srv.terminals.Spawn(ws.ID, ws.WorktreePath, extra...); err != nil {
		writeErr(w, 500, "spawn failed: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (srv *Server) apiTermKill(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		writeErr(w, 404, "workspace not found")
		return
	}
	srv.terminals.Kill(ws.ID)
	writeJSON(w, 200, map[string]bool{"ok": true})
}
