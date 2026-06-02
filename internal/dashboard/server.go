// internal/dashboard/server.go
package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
)

//go:embed templates
var templateFS embed.FS

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
	s.mux.HandleFunc("GET /", s.handleAppsList)
	s.mux.HandleFunc("GET /apps/{id}", s.handleAppDetail)
	s.mux.HandleFunc("POST /apps/{id}/status", s.handleAppStatusUpdate)

	s.mux.HandleFunc("GET /cv", s.handleCVList)
	s.mux.HandleFunc("GET /cv/{name}", s.handleCVView)

	s.mux.HandleFunc("GET /cl", s.handleCLList)
	s.mux.HandleFunc("GET /cl/{name}", s.handleCLView)

	s.mux.HandleFunc("POST /export", s.handleExport)

	s.mux.HandleFunc("GET /stats", s.handleStats)
	s.mux.HandleFunc("POST /stats/sync", s.handleStatsSync)

	s.mux.HandleFunc("GET /themes", s.handleThemesList)
	s.mux.HandleFunc("GET /themes/{name}/preview", s.handleThemePreview)
	s.mux.HandleFunc("POST /themes/upload", s.handleThemeUpload)

	s.mux.HandleFunc("GET /gotenberg/status", s.handleGotenbergStatus)
	s.mux.HandleFunc("POST /gotenberg/start", s.handleGotenbergStart)
	s.mux.HandleFunc("POST /gotenberg/stop", s.handleGotenbergStop)
}

// render parses layout.html + the named page template and executes "layout".
func (s *Server) render(w http.ResponseWriter, page string, data any) {
	t, err := template.ParseFS(templateFS, "templates/layout.html", "templates/"+page)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

// renderPartial parses a single partial template file and executes the named template within it.
func (s *Server) renderPartial(w http.ResponseWriter, partial, name string, data any) {
	t, err := template.ParseFS(templateFS, "templates/"+partial)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("renderPartial %s: %v", partial, err)
	}
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

// statusBadgeClass returns a Tailwind class string for a given status value.
func statusBadgeClass(status string) string {
	switch strings.ToLower(status) {
	case "offer":
		return "bg-green-100 text-green-800"
	case "interview", "assessment":
		return "bg-yellow-100 text-yellow-800"
	case "rejected", "withdrawn", "ghosted":
		return "bg-red-100 text-red-800"
	default:
		return "bg-blue-100 text-blue-800"
	}
}
