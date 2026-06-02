// internal/dashboard/server.go
package dashboard

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
)

//go:embed ui
var uiFS embed.FS

// Server holds all dependencies and the HTTP mux.
type Server struct {
	root            string
	store           *db.Store
	cfg             config.Config
	mux             *http.ServeMux
	startGotenberg  func() error
	stopGotenberg   func() error
}

// NewServer creates a Server and registers all routes.
func NewServer(root string, store *db.Store, cfg config.Config) *Server {
	s := &Server{root: root, store: store, cfg: cfg, mux: http.NewServeMux()}
	s.startGotenberg = s.defaultStartGotenberg
	s.stopGotenberg = s.defaultStopGotenberg
	s.routes()
	return s
}

// SetGotenbergRunners replaces the start/stop executors — used in tests.
func (s *Server) SetGotenbergRunners(start, stop func() error) {
	s.startGotenberg = start
	s.stopGotenberg = stop
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

	s.mux.HandleFunc("GET /api/apps", s.handleAPIAppsList)
	s.mux.HandleFunc("GET /api/apps/{id}", s.handleAPIAppDetail)
	s.mux.HandleFunc("POST /api/apps/{id}/status", s.handleAPIAppStatusUpdate)

	s.mux.HandleFunc("GET /api/cv", s.handleAPICVList)
	s.mux.HandleFunc("GET /api/cl", s.handleAPICLList)
	s.mux.HandleFunc("GET /api/themes", s.handleAPIThemesList)
	s.mux.HandleFunc("POST /api/themes/upload", s.handleThemeUpload)
	s.mux.HandleFunc("POST /api/export", s.handleExport)

	s.mux.HandleFunc("GET /api/stats", s.handleAPIStats)

	s.mux.HandleFunc("GET /api/gotenberg/status", s.handleAPIGotenbergStatus)
	s.mux.HandleFunc("POST /api/gotenberg/start", s.handleAPIGotenbergStart)
	s.mux.HandleFunc("POST /api/gotenberg/stop", s.handleAPIGotenbergStop)

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
	path := "ui" + r.URL.Path
	f, err := uiFS.Open(path)
	if err == nil {
		fi, sterr := f.Stat()
		f.Close()
		if sterr == nil && !fi.IsDir() {
			http.FileServer(http.FS(uiFS)).ServeHTTP(w, r)
			return
		}
	}
	data, err := uiFS.ReadFile("ui/index.html")
	if err != nil {
		http.Error(w, "dashboard not built: run 'just build-frontend'", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

