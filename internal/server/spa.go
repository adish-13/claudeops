package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// dist holds the built React app. Run `npm run build` in web/ before
// `go build` to refresh.
//
//go:embed all:dist
var distFS embed.FS

// spaHandler serves files under web/dist; any 404 falls back to index.html
// so client-side routes work on a hard refresh.
func (srv *Server) spaHandler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Build wasn't run — surface a clear message.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "web/dist not embedded — run `npm run build` in web/ then `go build`", 500)
		})
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API + WS routes are matched earlier; if we got here it's an SPA route.
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		// Serve index.html bytes directly for "/" and any unknown path so
		// (a) http.FileServer doesn't bounce /index.html → / (redirect loop)
		// and (b) client-side routes work on hard-refresh.
		if path == "" || statMissing(sub, path) {
			serveIndex(w, sub)
			return
		}
		files.ServeHTTP(w, r)
	})
}

func statMissing(sub fs.FS, path string) bool {
	_, err := fs.Stat(sub, path)
	return err != nil
}

func serveIndex(w http.ResponseWriter, sub fs.FS) {
	body, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "web/dist/index.html missing — run `make web`", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(body)
}
