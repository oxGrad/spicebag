package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
)

type analyticsResponse struct {
	PerPeriod   any    `json:"per_period"`
	PeriodType  string `json:"period_type"` // "day" or "month"
	SourceStats any    `json:"source_stats"`
}

func (s *Server) handleAPIAnalytics(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")    // YYYY-MM, empty = no constraint
	to := r.URL.Query().Get("to")        // YYYY-MM, empty = no constraint
	groupBy := r.URL.Query().Get("groupBy") // "day" or "month", default "day"
	if groupBy != "month" {
		groupBy = "day"
	}

	sources, err := s.store.SourceStats(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sources == nil {
		sources = []db.SourceStat{}
	}

	if groupBy == "day" {
		days, err := s.store.ApplicationsPerDay(from, to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if days == nil {
			days = []db.DayCount{}
		}
		writeJSON(w, analyticsResponse{PerPeriod: days, PeriodType: "day", SourceStats: sources})
		return
	}

	months, err := s.store.ApplicationsPerMonth(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if months == nil {
		months = []db.MonthCount{}
	}
	writeJSON(w, analyticsResponse{PerPeriod: months, PeriodType: "month", SourceStats: sources})
}
