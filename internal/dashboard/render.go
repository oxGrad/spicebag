// internal/dashboard/render.go
package dashboard

import (
	"fmt"
	"net/http"

	"github.com/oxGrad/spicebag/internal/fs"
)

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
