// internal/dashboard/handlers_cl.go
package dashboard

import "net/http"

func (s *Server) handleCLList(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
func (s *Server) handleCLView(w http.ResponseWriter, r *http.Request)  { http.NotFound(w, r) }
