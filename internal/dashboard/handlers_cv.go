// internal/dashboard/handlers_cv.go
package dashboard

import "net/http"

func (s *Server) handleCVList(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
func (s *Server) handleCVView(w http.ResponseWriter, r *http.Request)  { http.NotFound(w, r) }
