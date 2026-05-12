package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // localhost only
}

func (srv *Server) handleTerminalStart(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		http.Error(w, "workspace not found", 404)
		return
	}
	if _, err := srv.terminals.Spawn(ws.ID, ws.WorktreePath); err != nil {
		http.Error(w, "spawn failed: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/epics/%s/workspaces/%s", slug, wsslug), http.StatusSeeOther)
}

func (srv *Server) handleTerminalKill(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	wsslug := r.PathValue("wsslug")
	ws, _, err := srv.store.GetWorkspaceBySlug(r.Context(), slug, wsslug)
	if err != nil {
		http.Error(w, "workspace not found", 404)
		return
	}
	srv.terminals.Kill(ws.ID)
	http.Redirect(w, r, fmt.Sprintf("/epics/%s/workspaces/%s", slug, wsslug), http.StatusSeeOther)
}

// handleTerminalWS upgrades to WS and bridges the workspace's pty session.
// Protocol (text frames):
//   - server → client: raw bytes from pty
//   - client → server: either {"type":"input","data":"..."} or {"type":"resize","rows":N,"cols":M}
func (srv *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	wsidStr := r.PathValue("wsid")
	wsid, err := strconv.ParseInt(wsidStr, 10, 64)
	if err != nil {
		http.Error(w, "bad workspace id", 400)
		return
	}
	sess := srv.terminals.Get(wsid)
	if sess == nil {
		http.Error(w, "no terminal session — start it first", 404)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	// Replay backlog so the client sees recent context immediately.
	if backlog := sess.Backlog(); len(backlog) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, backlog); err != nil {
			return
		}
	}

	// Subscribe to subsequent output.
	out, cancel := sess.Subscribe(64)
	defer cancel()

	// Goroutine: pump pty → ws.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for chunk := range out {
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				return
			}
		}
	}()

	// Main loop: ws → pty.
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if len(raw) == 0 {
			continue
		}
		// Try to interpret as JSON control frame; if it parses with type field, route it.
		// Otherwise treat as raw input bytes.
		if raw[0] == '{' {
			var ctrl struct {
				Type string `json:"type"`
				Data string `json:"data"`
				Rows uint16 `json:"rows"`
				Cols uint16 `json:"cols"`
			}
			if err := json.Unmarshal(raw, &ctrl); err == nil && ctrl.Type != "" {
				switch ctrl.Type {
				case "input":
					_, _ = sess.Write([]byte(ctrl.Data))
				case "resize":
					_ = sess.Resize(ctrl.Rows, ctrl.Cols)
				}
				continue
			}
		}
		_, _ = sess.Write(raw)
	}
	<-done
}
