package server

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"sync"
)

//go:embed templates/layout.html templates/components/*.html templates/pages/*.html
var tmplFS embed.FS

// allPages lists every page template under templates/pages/ (without the .html suffix).
// Used at startup to validate parsing.
var allPages = []string{
	"home", "epic", "workspace", "new_epic", "new_workspace",
	"sessions", "transcript", "debug",
}

type templateCache struct {
	mu sync.Mutex
	m  map[string]*template.Template
}

func newTemplateCache() *templateCache { return &templateCache{m: map[string]*template.Template{}} }

func (c *templateCache) get(page string) (*template.Template, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.m[page]; ok {
		return t, nil
	}
	t, err := template.New("").Funcs(funcMap).ParseFS(tmplFS,
		"templates/layout.html",
		"templates/components/*.html",
		"templates/pages/"+page+".html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", page, err)
	}
	c.m[page] = t
	return t, nil
}

func (srv *Server) render(w http.ResponseWriter, page string, data any) {
	t, err := srv.tmpl.get(page)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		// Body may already be partially written; just log via stdlib http error if possible.
		http.Error(w, err.Error(), 500)
	}
}

var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"dict": func(values ...any) (map[string]any, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict needs even number of args")
		}
		m := make(map[string]any, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			k, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict key must be string")
			}
			m[k] = values[i+1]
		}
		return m, nil
	},
}
