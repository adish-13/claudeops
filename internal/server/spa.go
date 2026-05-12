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
		// Try to serve a real file; if missing, fall through to index.html.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(sub, path); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/index.html"
			files.ServeHTTP(w, r2)
			return
		}
		files.ServeHTTP(w, r)
	})
}
