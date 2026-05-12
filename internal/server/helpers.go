package server

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// expandHome turns "~" or "~/foo" into an absolute path under the user's home dir.
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

// shortPath collapses $HOME prefix to "~".
func (srv *Server) shortPath(p string) string {
	if p == "" {
		return p
	}
	if srv.home != "" && strings.HasPrefix(p, srv.home) {
		return "~" + p[len(srv.home):]
	}
	return p
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

// workspaceLookup is used by the sessions list to enrich each row with a link
// to its parent workspace, if any.
type wsLookup struct {
	epicSlug, epicName, workspaceSlug, workspaceName string
}

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
			out[w.ID] = wsLookup{
				epicSlug: e.Slug, epicName: e.Name,
				workspaceSlug: w.Slug, workspaceName: w.Name,
			}
		}
	}
	return out
}

// avoid unused import lint when fmt/time become unreferenced in narrow builds
var _ = fmt.Sprintf
var _ = time.Now
