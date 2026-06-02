// internal/dashboard/handlers_stats.go
package dashboard

import (
	"net/http"
)

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetExperienceStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats)
}
