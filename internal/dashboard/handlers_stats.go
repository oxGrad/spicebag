// internal/dashboard/handlers_stats.go
package dashboard

import "net/http"

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request)     { http.NotFound(w, r) }
func (s *Server) handleStatsSync(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
