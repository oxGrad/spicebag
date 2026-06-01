// internal/dashboard/handlers_themes.go
package dashboard

import "net/http"

func (s *Server) handleThemesList(w http.ResponseWriter, r *http.Request)   { http.NotFound(w, r) }
func (s *Server) handleThemePreview(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
func (s *Server) handleThemeUpload(w http.ResponseWriter, r *http.Request)  { http.NotFound(w, r) }
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request)       { http.NotFound(w, r) }
