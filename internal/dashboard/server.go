// internal/dashboard/server.go
package dashboard

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
)

//go:embed ui
var uiFS embed.FS

// Server holds all dependencies and the HTTP mux.
type Server struct {
	root  string
	store *db.Store
	cfg   config.Config
	mux   *http.ServeMux
}

// NewServer creates a Server and registers all routes.
func NewServer(root string, store *db.Store, cfg config.Config) *Server {
	s := &Server{root: root, store: store, cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Serve starts the HTTP server on addr (e.g. ":8080").
func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

// ServeHTTP implements http.Handler so the server can be used with httptest.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Store returns the underlying DB store (used in tests).
func (s *Server) Store() *db.Store { return s.store }

// Root returns the spicebag data root directory (used in tests).
func (s *Server) Root() string { return s.root }

// routes registers all URL patterns.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /render/cv/{name}", s.handleRenderCV)
	s.mux.HandleFunc("GET /render/cl/{name}", s.handleRenderCL)
	s.mux.HandleFunc("GET /render/app/{id}/cv", s.handleRenderAppCV)
	s.mux.HandleFunc("GET /render/app/{id}/cl", s.handleRenderAppCL)

	s.mux.HandleFunc("GET /api/analytics", s.handleAPIAnalytics)

	s.mux.HandleFunc("GET /api/sources", s.handleAPISourcesList)
	s.mux.HandleFunc("POST /api/sources", s.handleAPISourcesCreate)
	s.mux.HandleFunc("DELETE /api/sources/{id}", s.handleAPISourcesDelete)

	s.mux.HandleFunc("GET /api/apps", s.handleAPIAppsList)
	s.mux.HandleFunc("GET /api/apps/{id}", s.handleAPIAppDetail)
	s.mux.HandleFunc("POST /api/apps/{id}/status", s.handleAPIAppStatusUpdate)
	s.mux.HandleFunc("POST /api/apps/{id}/source", s.handleAPIAppSourceUpdate)
	s.mux.HandleFunc("GET /api/apps/{id}/pdfs", s.handleAPIAppListPDFs)
	s.mux.HandleFunc("POST /api/apps/{id}/pdf", s.handleAPIAppExportPDF)
	s.mux.HandleFunc("GET /render/app/{id}/pdf/{filename}", s.handleRenderAppPDF)

	s.mux.HandleFunc("GET /api/apps/{id}/questions", s.handleAPIQuestionsList)
	s.mux.HandleFunc("POST /api/apps/{id}/questions", s.handleAPIQuestionsAdd)
	s.mux.HandleFunc("POST /api/apps/{id}/questions/answers", s.handleAPIQuestionsBulkAnswer)
	s.mux.HandleFunc("PUT /api/apps/{id}/questions/{qid}", s.handleAPIQuestionUpdate)
	s.mux.HandleFunc("PUT /api/apps/{id}/questions/{qid}/bullets", s.handleAPIQuestionBulletsUpdate)
	s.mux.HandleFunc("DELETE /api/apps/{id}/questions/{qid}", s.handleAPIQuestionDelete)

	s.mux.HandleFunc("GET /api/cv", s.handleAPICVList)
	s.mux.HandleFunc("GET /api/cv/{name}/usages", s.handleAPICVUsages)
	s.mux.HandleFunc("GET /api/cl", s.handleAPICLList)
	s.mux.HandleFunc("GET /api/themes", s.handleAPIThemesList)
	s.mux.HandleFunc("POST /api/themes/upload", s.handleThemeUpload)
	s.mux.HandleFunc("POST /api/export", s.handleExport)

	s.mux.HandleFunc("GET /api/stats", s.handleAPIStats)

	s.mux.HandleFunc("GET /api/gotenberg/status", s.handleAPIGotenbergStatus)

	s.mux.HandleFunc("/", s.handleSPA)
}

// parseID extracts an integer path parameter from the URL using Go 1.22 PathValue.
func parseID(r *http.Request, param string) (int64, bool) {
	s := r.PathValue(param)
	if s == "" {
		return 0, false
	}
	var id int64
	if _, err := fmt.Sscanf(s, "%d", &id); err != nil {
		return 0, false
	}
	return id, true
}

// handleSPA serves the embedded Vue SPA, falling back to index.html for
// client-side routes (anything that is not a real file in the ui directory).
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(uiFS, "ui")
	if err != nil {
		http.Error(w, "ui filesystem error", http.StatusInternalServerError)
		return
	}
	// Try serving the exact path as a static file
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "."
	}
	f, err := sub.Open(path)
	if err == nil {
		fi, sterr := f.Stat()
		f.Close()
		if sterr == nil && !fi.IsDir() {
			http.FileServer(http.FS(sub)).ServeHTTP(w, r)
			return
		}
	}
	// SPA fallback — serve index.html for all unmatched routes
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "dashboard not built: run 'just build-frontend'", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

