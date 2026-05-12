package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // localhost only
}

// handleTerminalWS bridges the workspace's pty to the browser via xterm.js.
//
// Frame protocol (text frames from client):
//   {"type":"input","data":"..."} or {"type":"resize","rows":N,"cols":M}
// Server → client: raw byte chunks from the pty.
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

	if backlog := sess.Backlog(); len(backlog) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, backlog); err != nil {
			return
		}
	}

	out, cancel := sess.Subscribe(64)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for chunk := range out {
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				return
			}
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if len(raw) == 0 {
			continue
		}
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
