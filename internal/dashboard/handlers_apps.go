// internal/dashboard/handlers_apps.go
package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/oxGrad/spicebag/internal/db"
)

var validStatuses = []string{"applied", "assessment", "interview", "offer", "rejected", "withdrawn", "ghosted"}

func isValidStatus(s string) bool { return slices.Contains(validStatuses, s) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleAPIAppsList(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListApplicationsWithStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if apps == nil {
		apps = []db.ApplicationWithStatus{}
	}
	writeJSON(w, apps)
}

type appDetailResponse struct {
	App           db.Application          `json:"app"`
	History       []db.StatusHistoryEntry `json:"history"`
	ValidStatuses []string                `json:"valid_statuses"`
}

func (s *Server) handleAPIAppDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	app, err := s.store.GetApplicationByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	history, err := s.store.GetStatusHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if history == nil {
		history = []db.StatusHistoryEntry{}
	}
	writeJSON(w, appDetailResponse{App: app, History: history, ValidStatuses: validStatuses})
}

func (s *Server) handleAPIAppStatusUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	status := r.FormValue("status")
	notes := r.FormValue("notes")
	if !isValidStatus(status) {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	if err := s.store.AddStatusHistory(id, status, notes); err != nil {
		http.Error(w, fmt.Sprintf("update status: %v", err), http.StatusInternalServerError)
		return
	}
	history, err := s.store.GetStatusHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if history == nil {
		history = []db.StatusHistoryEntry{}
	}
	writeJSON(w, history)
}
