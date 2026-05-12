package server

import (
	"fmt"
	"net/http"
	"time"

	"claudeops/internal/transcript"
)

type messageJSON struct {
	Role      string `json:"role"`
	Text      string `json:"text"`
	ToolName  string `json:"tool_name,omitempty"`
	Model     string `json:"model,omitempty"`
	Timestamp string `json:"timestamp"`
}

type sessionDetailJSON struct {
	Session  sessionJSON   `json:"session"`
	Messages []messageJSON `json:"messages"`
}

func (srv *Server) apiSessionsList(w http.ResponseWriter, r *http.Request) {
	sessions, err := srv.store.ListSessions(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	wsByID := srv.workspaceLookup(r.Context())
	out := make([]sessionJSON, 0, len(sessions))
	projects := map[string]struct{}{}
	for _, s := range sessions {
		projects[s.ProjectDir] = struct{}{}
		j := srv.toSessionJSON(s)
		if s.WorkspaceID != nil {
			if e, ok := wsByID[*s.WorkspaceID]; ok {
				j.WorkspaceLink = fmt.Sprintf("/epics/%s/workspaces/%s", e.epicSlug, e.workspaceSlug)
				j.WorkspaceLabel = e.epicName + " / " + e.workspaceName
			}
		}
		out = append(out, j)
	}
	writeJSON(w, 200, map[string]any{"sessions": out, "project_count": len(projects)})
}

func (srv *Server) apiSessionGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := srv.store.GetSession(r.Context(), id)
	if err != nil {
		writeErr(w, 404, "session not found")
		return
	}
	msgs, _ := transcript.Render(sess.FilePath, 200)
	mj := make([]messageJSON, 0, len(msgs))
	for _, m := range msgs {
		mj = append(mj, messageJSON{
			Role: m.Role, Text: m.Text, ToolName: m.ToolName, Model: m.Model,
			Timestamp: m.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, 200, sessionDetailJSON{Session: srv.toSessionJSON(*sess), Messages: mj})
}
