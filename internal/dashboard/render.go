// internal/dashboard/render.go
package dashboard

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/oxGrad/spicebag/internal/fs"
)

func (s *Server) handleRenderAppCV(w http.ResponseWriter, r *http.Request) {
	s.renderAppDoc(w, r, "cv.html", "cv.md")
}

func (s *Server) handleRenderAppCL(w http.ResponseWriter, r *http.Request) {
	s.renderAppDoc(w, r, "cover-letter.html", "cover-letter.md")
}

// renderAppDoc serves an application document, trying each candidate filename in order.
func (s *Server) renderAppDoc(w http.ResponseWriter, r *http.Request, candidates ...string) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	app, err := s.store.GetApplicationByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	base := filepath.Join(s.root, "applications")
	for _, filename := range candidates {
		resolved := filepath.Join(base, app.FolderPath, filename)
		if !strings.HasPrefix(resolved, filepath.Clean(base)+string(os.PathSeparator)) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		fragment, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		writeDocument(w, string(fragment), s.themeCSS(r.URL.Query().Get("theme")))
		return
	}
	http.NotFound(w, r)
}

// writeDocument writes a complete standalone HTML page wrapping fragment with optional CSS.
func writeDocument(w http.ResponseWriter, fragment, css string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><style>%s</style></head>
<body>%s</body>
</html>`, css, fragment)
}

func (s *Server) handleRenderCV(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	fragment, err := fs.ReadCV(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeDocument(w, fragment, s.themeCSS(r.URL.Query().Get("theme")))
}

func (s *Server) handleRenderCL(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	fragment, err := fs.ReadCoverLetter(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeDocument(w, fragment, s.themeCSS(r.URL.Query().Get("theme")))
}

// themeCSS returns the CSS for a named theme, or "" if name is empty or not found.
func (s *Server) themeCSS(name string) string {
	if name == "" {
		return ""
	}
	css, _ := fs.ReadTheme(s.root, name)
	return css
}
