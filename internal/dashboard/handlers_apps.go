// internal/dashboard/handlers_apps.go
package dashboard

import "net/http"

func (s *Server) handleAppsList(w http.ResponseWriter, r *http.Request) {
	s.render(w, "apps_list.html", nil)
}

func (s *Server) handleAppDetail(w http.ResponseWriter, r *http.Request)       { http.NotFound(w, r) }
func (s *Server) handleAppStatusUpdate(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
