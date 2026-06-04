package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
)

type analyticsResponse struct {
	PerMonth    any `json:"per_month"`
	SourceStats any `json:"source_stats"`
}

func (s *Server) handleAPIAnalytics(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from") // YYYY-MM, empty = no constraint
	to := r.URL.Query().Get("to")     // YYYY-MM, empty = no constraint

	months, err := s.store.ApplicationsPerMonth(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sources, err := s.store.SourceStats(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if months == nil {
		months = []db.MonthCount{}
	}
	if sources == nil {
		sources = []db.SourceStat{}
	}
	writeJSON(w, analyticsResponse{PerMonth: months, SourceStats: sources})
}
