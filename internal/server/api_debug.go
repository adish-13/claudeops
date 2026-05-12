package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type debugJSON struct {
	DBPath           string           `json:"db_path"`
	DBSizeBytes      int64            `json:"db_size_bytes"`
	WALSizeBytes     int64            `json:"wal_size_bytes"`
	Counts           map[string]int   `json:"counts"`
	RecentEpics      []map[string]any `json:"recent_epics"`
	RecentWorkspaces []map[string]any `json:"recent_workspaces"`
	RecentSessions   []map[string]any `json:"recent_sessions"`
}

func (srv *Server) apiDebug(w http.ResponseWriter, r *http.Request) {
	d := debugJSON{
		DBPath: srv.store.Path(),
		Counts: map[string]int{
			"epics":      srv.countTable(r.Context(), "epics"),
			"workspaces": srv.countTable(r.Context(), "workspaces"),
			"sessions":   srv.countTable(r.Context(), "sessions"),
		},
	}
	if fi, err := os.Stat(d.DBPath); err == nil {
		d.DBSizeBytes = fi.Size()
	}
	if fi, err := os.Stat(d.DBPath + "-wal"); err == nil {
		d.WALSizeBytes = fi.Size()
	}
	d.RecentEpics = srv.queryToMap(r.Context(), `SELECT slug, name, repo_path, base_branch FROM epics ORDER BY id DESC LIMIT 10`)
	d.RecentWorkspaces = srv.queryToMap(r.Context(), `SELECT slug, branch_name, worktree_path, pr_url FROM workspaces ORDER BY id DESC LIMIT 10`)
	d.RecentSessions = srv.queryToMap(r.Context(), `SELECT substr(session_id,1,8) AS id, cwd, git_branch, num_events FROM sessions ORDER BY last_activity DESC LIMIT 10`)
	writeJSON(w, 200, d)
}

func (srv *Server) countTable(ctx context.Context, table string) int {
	var n int
	row := srv.store.DB().QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	_ = row.Scan(&n)
	return n
}

func (srv *Server) queryToMap(ctx context.Context, q string) []map[string]any {
	rows, err := srv.store.DB().QueryContext(ctx, q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []map[string]any
	for rows.Next() {
		dest := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			if b, ok := dest[i].([]byte); ok {
				m[c] = string(b)
			} else {
				m[c] = dest[i]
			}
		}
		out = append(out, m)
	}
	return out
}
