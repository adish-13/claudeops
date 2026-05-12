package server

import (
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

// shortPath shortens an absolute path by collapsing $HOME to "~".
func (srv *Server) shortPath(p string) string {
	if srv.home != "" && strings.HasPrefix(p, srv.home) {
		return "~" + p[len(srv.home):]
	}
	return p
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
