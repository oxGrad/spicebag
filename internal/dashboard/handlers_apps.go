// internal/dashboard/handlers_apps.go
package dashboard

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"

	"github.com/graditya/prospector/internal/db"
)

var validStatuses = []string{"applied", "assessment", "interview", "offer", "rejected", "withdrawn", "ghosted"}

func isValidStatus(s string) bool {
	return slices.Contains(validStatuses, s)
}

type appsListData struct {
	Apps []appsListRow
}

type appsListRow struct {
	db.ApplicationWithStatus
	BadgeClass string
}

func (s *Server) handleAppsList(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListApplicationsWithStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]appsListRow, len(apps))
	for i, a := range apps {
		rows[i] = appsListRow{a, statusBadgeClass(a.CurrentStatus)}
	}
	s.render(w, "apps_list.html", appsListData{Apps: rows})
}

type appDetailData struct {
	App           db.Application
	History       []db.StatusHistoryEntry
	ValidStatuses []string
}

func (s *Server) handleAppDetail(w http.ResponseWriter, r *http.Request) {
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

	// app_detail.html uses {{template "status_history" .History}}, so parse both templates together.
	t, err := template.ParseFS(templateFS,
		"templates/layout.html",
		"templates/app_detail.html",
		"templates/status_history.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", appDetailData{App: app, History: history, ValidStatuses: validStatuses}); err != nil {
		log.Printf("handleAppDetail: %v", err)
	}
}

func (s *Server) handleAppStatusUpdate(w http.ResponseWriter, r *http.Request) {
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
	s.renderPartial(w, "status_history.html", "status_history", history)
}
